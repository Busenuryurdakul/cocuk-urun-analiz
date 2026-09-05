"""Authoritative Python analysis run lifecycle orchestrator."""

from __future__ import annotations

import logging
import socket
import threading
import time
from dataclasses import dataclass, field
from typing import Callable

from miyuna_agent.go_client import GoAgentClient, hash_tool_input
from miyuna_agent.planner import build_deterministic_plan

logger = logging.getLogger(__name__)

PHASE_RUN_STARTED = "RUN_STARTED"
PHASE_COMPLIANCE_PRECHECK = "COMPLIANCE_PRECHECK"
PHASE_PLAN_CREATED = "PLAN_CREATED"
PHASE_TOOL_SELECTED = "TOOL_SELECTED"
PHASE_TOOL_EXECUTION_STARTED = "TOOL_EXECUTION_STARTED"
PHASE_TOOL_EXECUTION_COMPLETED = "TOOL_EXECUTION_COMPLETED"
PHASE_OBSERVATION_CREATED = "OBSERVATION_CREATED"
PHASE_RUN_COMPLETED = "RUN_COMPLETED"
PHASE_RUN_FAILED = "RUN_FAILED"
PHASE_RUN_CANCELLED = "RUN_CANCELLED"

DEFAULT_RUN_TIMEOUT_SECONDS = 600
MAX_TOOL_ATTEMPTS = 3


@dataclass
class StartRunPayload:
    organization_id: str
    analysis_run_id: str
    product_id: str
    trace_id: str
    actor_user_id: str
    capabilities: dict


@dataclass
class CancelRunPayload:
    organization_id: str
    analysis_run_id: str
    trace_id: str


def _default_worker_factory(target: Callable[[], None]) -> threading.Thread:
    return threading.Thread(target=target, daemon=True)


def _owner_id() -> str:
    return f"{socket.gethostname()}-{threading.get_ident()}"


@dataclass
class RunManager:
    go_client: GoAgentClient
    _lock: threading.Lock = field(default_factory=threading.Lock)
    _active: dict[str, threading.Event] = field(default_factory=dict)
    _terminal: dict[str, bool] = field(default_factory=dict)
    _worker_factory: Callable[[Callable[[], None]], threading.Thread] = field(
        default=_default_worker_factory
    )

    def start_run(self, payload: StartRunPayload) -> None:
        key = payload.analysis_run_id
        with self._lock:
            if key in self._active:
                return
            cancel_event = threading.Event()
            self._active[key] = cancel_event
            self._terminal[key] = False

        def runner() -> None:
            try:
                self._execute_run(payload, cancel_event)
            finally:
                with self._lock:
                    self._active.pop(key, None)

        thread = self._worker_factory(runner)
        if not thread.is_alive() and not thread.ident:
            thread.start()

    def cancel_run(self, payload: CancelRunPayload) -> None:
        with self._lock:
            event = self._active.get(payload.analysis_run_id)
        if event is not None:
            event.set()

    def _execute_run(self, payload: StartRunPayload, cancel_event: threading.Event) -> None:
        org_id = payload.organization_id
        run_id = payload.analysis_run_id
        trace_id = payload.trace_id
        owner = _owner_id()
        started_at = time.monotonic()

        try:
            if not self.go_client.claim_run_lease(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                owner_id=owner,
            ):
                raise RuntimeError("run lease not claimed")

            ctx = self.go_client.fetch_run_context(
                organization_id=org_id,
                analysis_run_id=run_id,
            )
            if ctx.trace_id != trace_id:
                raise ValueError("trace id mismatch")

            self._emit(org_id, run_id, trace_id, PHASE_RUN_STARTED, "RUNNING")
            self.go_client.update_run_status(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                status="RUNNING",
                current_phase=PHASE_RUN_STARTED,
            )

            self._emit(org_id, run_id, trace_id, PHASE_COMPLIANCE_PRECHECK, "COMPLETED")

            available = {
                tool.name: tool.version
                for tool in ctx.capabilities.available_tools
                if tool.availability == "AVAILABLE"
            }
            plan = build_deterministic_plan(
                marketplace_review_count=ctx.marketplace_review_count,
                available_tools=available,
            )
            self._emit(
                org_id,
                run_id,
                trace_id,
                PHASE_PLAN_CREATED,
                "COMPLETED",
                metadata={"planLength": str(len(plan))},
            )

            for step_index, step in enumerate(plan):
                if self._is_terminal(run_id):
                    return
                if time.monotonic()-started_at > DEFAULT_RUN_TIMEOUT_SECONDS:
                    raise TimeoutError("run timeout exceeded")
                if cancel_event.is_set() or self.go_client.cancellation_requested(
                    organization_id=org_id, analysis_run_id=run_id
                ):
                    self._terminate_cancelled(org_id, run_id, trace_id)
                    return
                self.go_client.heartbeat_run_lease(
                    organization_id=org_id,
                    analysis_run_id=run_id,
                    trace_id=trace_id,
                    owner_id=owner,
                )

                self._emit(
                    org_id,
                    run_id,
                    trace_id,
                    PHASE_TOOL_SELECTED,
                    "RUNNING",
                    tool_name=step.tool_name,
                    metadata={"stepIndex": str(step_index)},
                )

                input_hash = hash_tool_input(
                    organization_id=org_id,
                    analysis_run_id=run_id,
                    product_id=ctx.product_id,
                    tool_name=step.tool_name,
                    step_index=step_index,
                )
                auth = self.go_client.authorize_tool(
                    organization_id=org_id,
                    analysis_run_id=run_id,
                    trace_id=trace_id,
                    tool_name=step.tool_name,
                    tool_version=step.tool_version,
                    input_hash=input_hash,
                    step_index=step_index,
                )
                if not auth.allowed:
                    raise RuntimeError(f"tool authorization rejected: {auth.error_code}")

                self._emit(
                    org_id,
                    run_id,
                    trace_id,
                    PHASE_TOOL_EXECUTION_STARTED,
                    "RUNNING",
                    tool_name=step.tool_name,
                )
                result = self._execute_with_retry(
                    org_id=org_id,
                    run_id=run_id,
                    trace_id=trace_id,
                    step=step,
                    input_hash=input_hash,
                    auth=auth,
                    step_index=step_index,
                )
                if result.status in {"REJECTED", "FAILED"}:
                    raise RuntimeError(result.error or "tool execution failed")

                self._emit(
                    org_id,
                    run_id,
                    trace_id,
                    PHASE_TOOL_EXECUTION_COMPLETED,
                    "COMPLETED",
                    tool_name=step.tool_name,
                )
                self._emit(
                    org_id,
                    run_id,
                    trace_id,
                    PHASE_OBSERVATION_CREATED,
                    "COMPLETED",
                    tool_name=step.tool_name,
                    metadata={"stepIndex": str(step_index)},
                )

            if self._is_terminal(run_id):
                return
            self._mark_terminal(run_id)
            self._emit(org_id, run_id, trace_id, PHASE_RUN_COMPLETED, "COMPLETED")
            self.go_client.update_run_status(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                status="COMPLETED",
                current_phase=PHASE_RUN_COMPLETED,
                iteration_count=len(plan),
            )
        except Exception as exc:  # noqa: BLE001
            if self._is_terminal(run_id):
                return
            logger.exception("analysis run failed run_id=%s trace_id=%s", run_id, trace_id)
            self._mark_terminal(run_id)
            self._emit(org_id, run_id, trace_id, PHASE_RUN_FAILED, "FAILED", metadata={"error": str(exc)})
            self.go_client.update_run_status(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                status="FAILED",
                current_phase=PHASE_RUN_FAILED,
                terminal_error=str(exc),
                terminal_reason="ORCHESTRATOR_ERROR",
            )

    def _execute_with_retry(self, *, org_id, run_id, trace_id, step, input_hash, auth, step_index):
        delay = 0.5
        current_auth = auth
        for attempt in range(1, MAX_TOOL_ATTEMPTS + 1):
            result = self.go_client.execute_tool(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                tool_name=step.tool_name,
                tool_version=step.tool_version,
                input_hash=input_hash,
                grant_nonce=current_auth.grant_nonce,
                tool_execution_id=current_auth.tool_execution_id,
                step_index=step_index,
            )
            if result.status not in {"REJECTED", "FAILED"}:
                return result
            last_error = result.error or result.status
            non_retryable = any(
                token in last_error.lower()
                for token in ("schema", "compliance", "authorization", "grant", "forbidden", "cancelled")
            )
            if non_retryable or attempt >= MAX_TOOL_ATTEMPTS:
                return result
            current_auth = self.go_client.authorize_tool(
                organization_id=org_id,
                analysis_run_id=run_id,
                trace_id=trace_id,
                tool_name=step.tool_name,
                tool_version=step.tool_version,
                input_hash=input_hash,
                step_index=step_index,
            )
            if not current_auth.allowed:
                return result
            time.sleep(delay)
            delay = min(delay * 2, 5.0)
        return result

    def _terminate_cancelled(self, org_id: str, run_id: str, trace_id: str) -> None:
        if self._is_terminal(run_id):
            return
        self._mark_terminal(run_id)
        self._emit(org_id, run_id, trace_id, PHASE_RUN_CANCELLED, "COMPLETED")
        self.go_client.update_run_status(
            organization_id=org_id,
            analysis_run_id=run_id,
            trace_id=trace_id,
            status="REJECTED",
            current_phase=PHASE_RUN_CANCELLED,
            terminal_reason="CANCELLED",
        )

    def _mark_terminal(self, run_id: str) -> None:
        with self._lock:
            self._terminal[run_id] = True

    def _is_terminal(self, run_id: str) -> bool:
        with self._lock:
            return self._terminal.get(run_id, False)

    def _emit(
        self,
        org_id: str,
        run_id: str,
        trace_id: str,
        phase: str,
        status: str,
        *,
        tool_name: str = "",
        metadata: dict[str, str] | None = None,
    ) -> None:
        meta = metadata or {}
        meta.setdefault("traceId", trace_id)
        self.go_client.record_event(
            organization_id=org_id,
            analysis_run_id=run_id,
            trace_id=trace_id,
            phase=phase,
            status=status,
            tool_name=tool_name,
            metadata=meta,
        )
