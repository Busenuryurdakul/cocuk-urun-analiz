"""Authoritative Python analysis run lifecycle orchestrator."""

from __future__ import annotations

import logging
import socket
import threading
import time
from dataclasses import dataclass, field
from typing import Callable

from miyuna_agent.go_client import GoAgentClient, RunContext, hash_tool_input
from miyuna_agent.llm_client import GoLLMClient, LLMCompletionResult, LLMGatewayError
from miyuna_agent.planner import build_deterministic_plan

logger = logging.getLogger(__name__)

PHASE_RUN_STARTED = "RUN_STARTED"
PHASE_COMPLIANCE_PRECHECK = "COMPLIANCE_PRECHECK"
PHASE_PLAN_CREATED = "PLAN_CREATED"
PHASE_TOOL_SELECTED = "TOOL_SELECTED"
PHASE_TOOL_EXECUTION_STARTED = "TOOL_EXECUTION_STARTED"
PHASE_TOOL_EXECUTION_COMPLETED = "TOOL_EXECUTION_COMPLETED"
PHASE_OBSERVATION_CREATED = "OBSERVATION_CREATED"
PHASE_LLM_REQUESTED = "LLM_REQUESTED"
PHASE_LLM_COMPLETED = "LLM_COMPLETED"
PHASE_LLM_FAILED = "LLM_FAILED"
PHASE_LLM_FALLBACK_USED = "LLM_FALLBACK_USED"
PHASE_LLM_ESCALATED = "LLM_ESCALATED"
PHASE_RUN_COMPLETED = "RUN_COMPLETED"
PHASE_RUN_FAILED = "RUN_FAILED"
PHASE_RUN_CANCELLED = "RUN_CANCELLED"

DEFAULT_RUN_TIMEOUT_SECONDS = 600
MAX_TOOL_ATTEMPTS = 3
PERSONA_WORKER = "careful_analyst"
PERSONA_REVIEWER = "result_analyst"
ROTATION_RUN_A = "RUN_A"
ROTATION_RUN_B = "RUN_B"
TASK_ANALYSIS = "analysis"
TASK_REVIEW = "review"
TASK_DEEP_ANALYSIS = "deep_analysis"


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
    llm_client: GoLLMClient
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

            worker_persona, reviewer_persona = self._rotation_personas(ctx)

            worker_prompt = self._worker_prompt(ctx)
            worker_result = self._run_llm_step(
                org_id=org_id,
                run_id=run_id,
                trace_id=trace_id,
                ctx=ctx,
                step_key="worker",
                persona_key=worker_persona,
                task_type=TASK_ANALYSIS,
                user_prompt=worker_prompt,
            )
            analysis_result = worker_result
            if self._should_escalate_analysis(worker_prompt, worker_result):
                self._emit(
                    org_id,
                    run_id,
                    trace_id,
                    PHASE_LLM_ESCALATED,
                    "RUNNING",
                    metadata={
                        "fromModel": worker_result.model_key,
                        "reason": "long_or_complex_analysis",
                        "taskType": TASK_DEEP_ANALYSIS,
                    },
                )
                analysis_result = self._run_llm_step(
                    org_id=org_id,
                    run_id=run_id,
                    trace_id=trace_id,
                    ctx=ctx,
                    step_key="worker-escalated",
                    persona_key=self._escalation_persona(worker_persona),
                    task_type=TASK_DEEP_ANALYSIS,
                    user_prompt=worker_prompt,
                )
            reviewer_result = self._run_llm_step(
                org_id=org_id,
                run_id=run_id,
                trace_id=trace_id,
                ctx=ctx,
                step_key="reviewer",
                persona_key=reviewer_persona,
                task_type=TASK_REVIEW,
                user_prompt=self._reviewer_prompt(ctx, analysis_result.content),
            )
            _ = reviewer_result

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

    def _rotation_personas(self, ctx: RunContext) -> tuple[str, str]:
        if ctx.worker_rotation_pattern == ROTATION_RUN_B:
            return PERSONA_REVIEWER, PERSONA_WORKER
        return PERSONA_WORKER, PERSONA_REVIEWER

    @staticmethod
    def _escalation_persona(worker_persona: str) -> str:
        if worker_persona == PERSONA_WORKER:
            return PERSONA_REVIEWER
        return PERSONA_WORKER

    def _worker_prompt(self, ctx: RunContext) -> str:
        return (
            f"Analyze product {ctx.product_id} for organization {ctx.organization_id}. "
            f"Use evidence-backed reasoning for marketplace review count={ctx.marketplace_review_count} "
            f"and ugc count={ctx.ugc_count}. Compliance profile={ctx.compliance_profile or 'default'}."
        )

    def _reviewer_prompt(self, ctx: RunContext, worker_content: str) -> str:
        return (
            f"Review the worker analysis for product {ctx.product_id} and produce a concise decision summary.\n\n"
            f"Worker output:\n{worker_content}"
        )

    def _should_escalate_analysis(self, prompt: str, result: LLMCompletionResult) -> bool:
        if result.escalation_used or result.fallback_used:
            return False
        text = prompt.strip()
        content = result.content.strip()
        if len(text) >= 350:
            return True
        lowered = text.lower()
        hints = (
            "detayli",
            "detaylı",
            "kapsamli",
            "kapsamlı",
            "adim adim",
            "adım adım",
            "uzun surec",
            "uzun süreç",
            "comprehensive",
            "step by step",
        )
        if any(hint in lowered for hint in hints):
            return True
        if len(text) >= 250 and len(content) < 100:
            return True
        return False

    def _run_llm_step(
        self,
        *,
        org_id: str,
        run_id: str,
        trace_id: str,
        ctx: RunContext,
        step_key: str,
        persona_key: str,
        task_type: str,
        user_prompt: str,
    ) -> LLMCompletionResult:
        correlation_id = ctx.correlation_id or trace_id
        idempotency_key = f"{run_id}:llm:{step_key}"
        request_meta = {
            "correlationId": correlation_id,
            "personaKey": persona_key,
            "taskType": task_type,
            "idempotencyKey": idempotency_key,
            "configSnapshotId": ctx.config_snapshot_id,
            "complianceProfile": ctx.compliance_profile,
        }
        if step_key == "worker":
            request_meta["workerRotationPattern"] = ctx.worker_rotation_pattern
        self._emit(
            org_id,
            run_id,
            trace_id,
            PHASE_LLM_REQUESTED,
            "RUNNING",
            metadata=request_meta,
        )
        try:
            result = self.llm_client.complete(
                organization_id=org_id,
                user_prompt=user_prompt,
                correlation_id=correlation_id,
                task_type=task_type,
                persona_key=persona_key,
                analysis_run_id=run_id,
                user_id=ctx.actor_user_id,
                idempotency_key=idempotency_key,
                config_snapshot_id=ctx.config_snapshot_id,
                require_evidence=ctx.require_evidence,
                routing_policy_version=ctx.llm_routing_policy_version,
            )
        except LLMGatewayError as exc:
            fail_meta = {
                "correlationId": exc.correlation_id or correlation_id,
                "personaKey": persona_key,
                "taskType": task_type,
                "error": str(exc),
            }
            if exc.status_code:
                fail_meta["statusCode"] = str(exc.status_code)
            self._emit(
                org_id,
                run_id,
                trace_id,
                PHASE_LLM_FAILED,
                "FAILED",
                metadata=fail_meta,
            )
            raise RuntimeError(f"llm gateway failed for {step_key}: {exc}") from exc

        completion_meta = self._llm_result_metadata(result, task_type=task_type)
        if result.escalation_used:
            self._emit(
                org_id,
                run_id,
                trace_id,
                PHASE_LLM_ESCALATED,
                "COMPLETED",
                metadata=completion_meta,
            )
        elif result.fallback_used:
            self._emit(
                org_id,
                run_id,
                trace_id,
                PHASE_LLM_FALLBACK_USED,
                "COMPLETED",
                metadata=completion_meta,
            )
        self._emit(
            org_id,
            run_id,
            trace_id,
            PHASE_LLM_COMPLETED,
            "COMPLETED",
            metadata=completion_meta,
        )
        return result

    @staticmethod
    def _llm_result_metadata(result: LLMCompletionResult, *, task_type: str) -> dict[str, str]:
        meta = {
            "llmCallId": result.call_id,
            "selectedModel": result.model_key,
            "selectedProvider": result.provider_key,
            "personaKey": result.persona_key,
            "personaVersion": result.persona_version,
            "routingReason": result.routing_reason,
            "fallbackUsed": str(result.fallback_used).lower(),
            "escalationUsed": str(result.escalation_used).lower(),
            "inputTokens": str(result.input_tokens),
            "outputTokens": str(result.output_tokens),
            "correlationId": result.correlation_id,
            "taskType": task_type,
        }
        return meta
