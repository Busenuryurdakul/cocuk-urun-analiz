"""Internal client for Miyuna LLM Gateway (Go authoritative boundary)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import httpx

INTERNAL_TOKEN_HEADER = "X-Miyuna-Internal-Token"
LLM_COMPLETE_PATH = "/internal/llm/v1/complete"


class LLMGatewayError(Exception):
    """Normalized gateway failure — agent must not retry or call providers directly."""

    def __init__(
        self,
        message: str,
        *,
        status_code: int = 0,
        correlation_id: str = "",
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.correlation_id = correlation_id


@dataclass(frozen=True)
class LLMCompletionResult:
    call_id: str
    content: str
    model_key: str
    provider_key: str
    fallback_used: bool
    escalation_used: bool
    routing_reason: str
    persona_key: str
    persona_version: str
    correlation_id: str
    input_tokens: int
    output_tokens: int

    @classmethod
    def from_dict(cls, raw: dict[str, Any]) -> LLMCompletionResult:
        return cls(
            call_id=raw.get("CallID") or raw.get("callId") or raw.get("call_id", ""),
            content=raw.get("Content") or raw.get("content", ""),
            model_key=raw.get("ModelKey") or raw.get("modelKey", ""),
            provider_key=raw.get("ProviderKey") or raw.get("providerKey", ""),
            fallback_used=bool(raw.get("FallbackUsed") or raw.get("fallbackUsed")),
            escalation_used=bool(raw.get("EscalationUsed") or raw.get("escalationUsed")),
            routing_reason=raw.get("RoutingReason") or raw.get("routingReason", ""),
            persona_key=raw.get("PersonaKey") or raw.get("personaKey", ""),
            persona_version=raw.get("PersonaVersion") or raw.get("personaVersion", ""),
            correlation_id=raw.get("CorrelationID") or raw.get("correlationId", ""),
            input_tokens=int(raw.get("InputTokens") or raw.get("inputTokens") or 0),
            output_tokens=int(raw.get("OutputTokens") or raw.get("outputTokens") or 0),
        )


class GoLLMClient:
    def __init__(self, base_url: str, token: str, timeout_seconds: float = 30.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._token = token
        self._timeout = timeout_seconds

    def _headers(self) -> dict[str, str]:
        return {INTERNAL_TOKEN_HEADER: self._token}

    def complete(
        self,
        *,
        organization_id: str,
        user_prompt: str,
        correlation_id: str,
        task_type: str = "analysis",
        persona_key: str = "",
        analysis_run_id: str = "",
        user_id: str = "",
        idempotency_key: str = "",
        config_snapshot_id: str = "",
        require_evidence: bool = False,
        routing_policy_version: str = "",
        external_content: list[str] | None = None,
    ) -> LLMCompletionResult:
        payload: dict[str, Any] = {
            "organizationId": organization_id,
            "userPrompt": user_prompt,
            "correlationId": correlation_id,
            "taskType": task_type,
            "personaKey": persona_key,
            "analysisRunId": analysis_run_id,
            "userId": user_id,
            "idempotencyKey": idempotency_key,
            "requireEvidence": require_evidence,
            "externalContent": external_content or [],
        }
        if config_snapshot_id:
            payload["configSnapshotId"] = config_snapshot_id
        if routing_policy_version:
            payload["routingPolicyVersion"] = routing_policy_version
        try:
            with httpx.Client(timeout=self._timeout) as client:
                resp = client.post(
                    f"{self._base_url}{LLM_COMPLETE_PATH}",
                    json=payload,
                    headers=self._headers(),
                )
                resp.raise_for_status()
                return LLMCompletionResult.from_dict(resp.json())
        except httpx.HTTPStatusError as exc:
            raise LLMGatewayError(
                f"llm gateway error: {exc.response.status_code}",
                status_code=exc.response.status_code,
                correlation_id=correlation_id,
            ) from exc
        except httpx.HTTPError as exc:
            raise LLMGatewayError(
                f"llm gateway transport error: {exc}",
                correlation_id=correlation_id,
            ) from exc

    def health(self) -> dict[str, Any]:
        with httpx.Client(timeout=self._timeout) as client:
            resp = client.get(
                f"{self._base_url}/internal/llm/v1/health",
                headers=self._headers(),
            )
            resp.raise_for_status()
            return resp.json()
