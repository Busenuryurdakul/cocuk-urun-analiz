"""Internal HTTP client for Go authoritative agent boundaries."""

from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any

import httpx

INTERNAL_TOKEN_HEADER = "X-Miyuna-Internal-Token"


@dataclass(frozen=True)
class ToolCapability:
    name: str
    version: str
    availability: str


@dataclass(frozen=True)
class CapabilitySnapshot:
    tool_registry_version: str
    tool_policy_version: str
    planner_version: str
    observation_schema_version: str
    max_tool_loop_iterations: int
    total_run_timeout_seconds: int
    available_tools: tuple[ToolCapability, ...]

    @classmethod
    def from_dict(cls, raw: dict[str, Any]) -> CapabilitySnapshot:
        tools = tuple(
            ToolCapability(
                name=item["name"],
                version=item["version"],
                availability=item["availability"],
            )
            for item in raw.get("availableTools", [])
        )
        return cls(
            tool_registry_version=raw["toolRegistryVersion"],
            tool_policy_version=raw["toolPolicyVersion"],
            planner_version=raw["plannerVersion"],
            observation_schema_version=raw["observationSchemaVersion"],
            max_tool_loop_iterations=int(raw.get("maxToolLoopIterations", 10)),
            total_run_timeout_seconds=int(raw.get("totalRunTimeoutSeconds", 600)),
            available_tools=tools,
        )


@dataclass(frozen=True)
class RunContext:
    organization_id: str
    analysis_run_id: str
    product_id: str
    trace_id: str
    correlation_id: str
    actor_user_id: str
    status: str
    config_snapshot_id: str
    compliance_profile: str
    llm_routing_policy_version: str
    require_evidence: bool
    capabilities: CapabilitySnapshot
    marketplace_review_count: int
    ugc_count: int
    worker_rotation_pattern: str = "RUN_A"

    @classmethod
    def from_dict(cls, raw: dict[str, Any]) -> RunContext:
        return cls(
            organization_id=raw["organizationId"],
            analysis_run_id=raw["analysisRunId"],
            product_id=raw["productId"],
            trace_id=raw["traceId"],
            correlation_id=raw.get("correlationId", raw.get("traceId", "")),
            actor_user_id=raw["actorUserId"],
            status=raw["status"],
            config_snapshot_id=raw.get("configSnapshotId", ""),
            compliance_profile=raw.get("complianceProfile", ""),
            llm_routing_policy_version=raw.get("llmRoutingPolicyVersion", ""),
            require_evidence=bool(raw.get("requireEvidence", False)),
            capabilities=CapabilitySnapshot.from_dict(raw["capabilities"]),
            marketplace_review_count=int(raw.get("marketplaceReviewCount", 0)),
            ugc_count=int(raw.get("ugcCount", 0)),
            worker_rotation_pattern=raw.get("workerRotationPattern", "RUN_A") or "RUN_A",
        )


@dataclass(frozen=True)
class AuthorizeResult:
    allowed: bool
    grant_nonce: str = ""
    tool_execution_id: str = ""
    error_code: str = ""


@dataclass(frozen=True)
class ExecuteResult:
    status: str
    payload: dict[str, Any] | None = None
    error: str = ""


class GoAgentClient:
    def __init__(self, base_url: str, token: str, timeout_seconds: float = 15.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._token = token
        self._timeout = timeout_seconds

    def _headers(self) -> dict[str, str]:
        return {INTERNAL_TOKEN_HEADER: self._token}

    def claim_run_lease(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        owner_id: str,
        attempt: int = 1,
    ) -> bool:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "ownerId": owner_id,
            "attempt": attempt,
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/runs/lease/claim",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()
            return bool(resp.json().get("claimed"))

    def heartbeat_run_lease(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        owner_id: str,
    ) -> None:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "ownerId": owner_id,
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/runs/heartbeat",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()

    def fetch_run_context(self, *, organization_id: str, analysis_run_id: str) -> RunContext:
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.get(
                f"{self._base_url}/internal/agent/v1/runs/context",
                params={"organizationId": organization_id, "analysisRunId": analysis_run_id},
                headers=self._headers(),
            )
            resp.raise_for_status()
            return RunContext.from_dict(resp.json())

    def record_event(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        phase: str,
        status: str,
        tool_name: str = "",
        metadata: dict[str, str] | None = None,
    ) -> None:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "phase": phase,
            "status": status,
            "toolName": tool_name,
            "metadata": metadata or {},
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/events",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()

    def update_run_status(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        status: str,
        current_phase: str = "",
        terminal_error: str = "",
        terminal_reason: str = "",
        iteration_count: int = 0,
    ) -> None:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "status": status,
            "currentPhase": current_phase,
            "terminalError": terminal_error,
            "terminalReason": terminal_reason,
            "iterationCount": iteration_count,
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/runs/status",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()

    def cancellation_requested(self, *, organization_id: str, analysis_run_id: str) -> bool:
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.get(
                f"{self._base_url}/internal/agent/v1/runs/cancellation",
                params={"organizationId": organization_id, "analysisRunId": analysis_run_id},
                headers=self._headers(),
            )
            resp.raise_for_status()
            return bool(resp.json().get("cancellationRequested"))

    def authorize_tool(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        tool_name: str,
        tool_version: str,
        input_hash: str,
        step_index: int,
    ) -> AuthorizeResult:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "toolName": tool_name,
            "toolVersion": tool_version,
            "inputHash": input_hash,
            "stepIndex": step_index,
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/tools/authorize",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()
            raw = resp.json()
            return AuthorizeResult(
                allowed=bool(raw.get("allowed")),
                grant_nonce=raw.get("grantNonce", ""),
                tool_execution_id=raw.get("toolExecutionId", ""),
                error_code=raw.get("errorCode", ""),
            )

    def execute_tool(
        self,
        *,
        organization_id: str,
        analysis_run_id: str,
        trace_id: str,
        tool_name: str,
        tool_version: str,
        input_hash: str,
        grant_nonce: str,
        tool_execution_id: str,
        step_index: int,
    ) -> ExecuteResult:
        payload = {
            "organizationId": organization_id,
            "analysisRunId": analysis_run_id,
            "traceId": trace_id,
            "toolName": tool_name,
            "toolVersion": tool_version,
            "inputHash": input_hash,
            "grantNonce": grant_nonce,
            "toolExecutionId": tool_execution_id,
            "stepIndex": step_index,
        }
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.post(
                f"{self._base_url}/internal/agent/v1/tools/execute",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()
            raw = resp.json()
            return ExecuteResult(
                status=raw.get("status", ""),
                payload=raw.get("payload"),
                error=raw.get("error", ""),
            )


def hash_tool_input(
    *,
    organization_id: str,
    analysis_run_id: str,
    product_id: str,
    tool_name: str,
    step_index: int,
) -> str:
    import hashlib

    payload = {
        "organizationId": organization_id,
        "analysisRunId": analysis_run_id,
        "productId": product_id,
        "tool": tool_name,
        "step": step_index,
    }
    encoded = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()
