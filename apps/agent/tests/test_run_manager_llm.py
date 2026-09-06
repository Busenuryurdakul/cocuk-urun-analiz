import threading
from unittest.mock import MagicMock

import pytest

from miyuna_agent.llm_client import LLMCompletionResult, LLMGatewayError
from miyuna_agent.run_manager import (
    PERSONA_REVIEWER,
    PERSONA_WORKER,
    PHASE_LLM_COMPLETED,
    PHASE_LLM_ESCALATED,
    PHASE_LLM_FAILED,
    PHASE_LLM_FALLBACK_USED,
    PHASE_LLM_REQUESTED,
    PHASE_RUN_COMPLETED,
    PHASE_RUN_FAILED,
    RunManager,
    StartRunPayload,
)


def _tool(name: str, version: str = "frozen-v1"):
    tool = MagicMock(name=name, version=version, availability="AVAILABLE")
    tool.name = name
    tool.version = version
    tool.availability = "AVAILABLE"
    return tool


def _run_context(**overrides):
    base = dict(
        organization_id="org",
        trace_id="trace-1",
        product_id="prod-1",
        correlation_id="corr-1",
        actor_user_id="user-1",
        config_snapshot_id="snap-1",
        compliance_profile="KVKK",
        llm_routing_policy_version="1.0.0",
        require_evidence=True,
        marketplace_review_count=0,
        ugc_count=0,
        capabilities=MagicMock(
            available_tools=(
                _tool("policy_evaluator"),
                _tool("compliance_checker"),
                _tool("dataset_validator"),
                _tool("pii_redactor"),
            )
        ),
    )
    base.update(overrides)
    return MagicMock(**base)


def _llm_result(**overrides):
    data = dict(
        call_id="call-1",
        content="analysis draft",
        model_key="careful_analyst",
        provider_key="primary",
        fallback_used=False,
        escalation_used=False,
        routing_reason="task=analysis",
        persona_key=PERSONA_WORKER,
        persona_version="1.0.0",
        correlation_id="corr-1",
        input_tokens=10,
        output_tokens=5,
    )
    data.update(overrides)
    return LLMCompletionResult(**data)


def _start_manager(go_client: MagicMock, llm_client: MagicMock) -> RunManager:
    finished = threading.Event()

    def worker_factory(target):
        def wrapped() -> None:
            try:
                target()
            finally:
                finished.set()

        thread = threading.Thread(target=wrapped, daemon=True)
        thread.start()
        return thread

    manager = RunManager(
        go_client=go_client,
        llm_client=llm_client,
        _worker_factory=worker_factory,
    )
    manager.start_run(
        StartRunPayload(
            organization_id="org",
            analysis_run_id="run",
            product_id="prod",
            trace_id="trace-1",
            actor_user_id="user-1",
            capabilities={},
        )
    )
    assert finished.wait(timeout=5)
    return manager


def test_worker_and_reviewer_use_gateway_client() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context()
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(content="worker output"),
        _llm_result(
            call_id="call-2",
            content="review output",
            model_key="result_analyst",
            provider_key="secondary",
            persona_key=PERSONA_REVIEWER,
            routing_reason="task=review",
        ),
    ]

    _start_manager(go_client, llm_client)

    assert llm_client.complete.call_count == 2
    worker_call = llm_client.complete.call_args_list[0].kwargs
    reviewer_call = llm_client.complete.call_args_list[1].kwargs
    assert worker_call["persona_key"] == PERSONA_WORKER
    assert worker_call["task_type"] == "analysis"
    assert reviewer_call["persona_key"] == PERSONA_REVIEWER
    assert reviewer_call["task_type"] == "review"
    assert worker_call["correlation_id"] == "corr-1"
    assert worker_call["analysis_run_id"] == "run"
    assert worker_call["organization_id"] == "org"
    assert worker_call["user_id"] == "user-1"
    assert worker_call["config_snapshot_id"] == "snap-1"
    assert worker_call["require_evidence"] is True
    assert worker_call["idempotency_key"] == "run:llm:worker"
    assert reviewer_call["idempotency_key"] == "run:llm:reviewer"


def test_gateway_success_emits_llm_events_and_completes_run() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context()
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(fallback_used=True, routing_reason="fallback"),
        _llm_result(call_id="call-2", content="review", persona_key=PERSONA_REVIEWER),
    ]

    _start_manager(go_client, llm_client)
    threading.Event().wait(0.5)

    phases = [call.kwargs["phase"] for call in go_client.record_event.call_args_list]
    assert PHASE_LLM_REQUESTED in phases
    assert PHASE_LLM_FALLBACK_USED in phases
    assert PHASE_LLM_COMPLETED in phases
    assert PHASE_RUN_COMPLETED in phases
    assert PHASE_RUN_FAILED not in phases

    completed = go_client.update_run_status.call_args_list[-1].kwargs
    assert completed["status"] == "COMPLETED"


def test_gateway_error_fails_run_without_completed_status() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context()
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = LLMGatewayError(
        "llm gateway error: 503",
        status_code=503,
        correlation_id="corr-1",
    )

    _start_manager(go_client, llm_client)
    threading.Event().wait(0.5)

    phases = [call.kwargs["phase"] for call in go_client.record_event.call_args_list]
    assert PHASE_LLM_FAILED in phases
    assert PHASE_RUN_FAILED in phases
    assert PHASE_RUN_COMPLETED not in phases

    failed = go_client.update_run_status.call_args_list[-1].kwargs
    assert failed["status"] == "FAILED"
    assert failed["terminal_reason"] == "ORCHESTRATOR_ERROR"


def test_idempotency_keys_are_stable_per_step() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context()
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(),
        _llm_result(call_id="call-2", persona_key=PERSONA_REVIEWER),
    ]

    _start_manager(go_client, llm_client)

    keys = [call.kwargs["idempotency_key"] for call in llm_client.complete.call_args_list]
    assert keys == ["run:llm:worker", "run:llm:reviewer"]
    assert len(set(keys)) == 2


def test_fallback_metadata_recorded_on_event() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context()
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(
            fallback_used=True,
            routing_reason="task=analysis;fallback",
            provider_key="secondary",
        ),
        _llm_result(call_id="call-2", persona_key=PERSONA_REVIEWER),
    ]

    _start_manager(go_client, llm_client)
    threading.Event().wait(0.5)

    fallback_events = [
        call.kwargs
        for call in go_client.record_event.call_args_list
        if call.kwargs["phase"] == PHASE_LLM_FALLBACK_USED
    ]
    assert fallback_events
    meta = fallback_events[0]["metadata"]
    assert meta["fallbackUsed"] == "true"
    assert meta["selectedProvider"] == "secondary"
    assert meta["llmCallId"] == "call-1"


def test_long_analysis_escalates_to_heavy_model() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context(
        compliance_profile="comprehensive-KVKK",
    )
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(content="short draft"),
        _llm_result(
            call_id="call-2",
            content="deep analysis output",
            model_key="result_analyst",
            provider_key="secondary",
            persona_key=PERSONA_REVIEWER,
            routing_reason="task=deep_analysis",
            escalation_used=True,
        ),
        _llm_result(
            call_id="call-3",
            content="review output",
            model_key="result_analyst",
            provider_key="secondary",
            persona_key=PERSONA_REVIEWER,
            routing_reason="task=review",
        ),
    ]

    _start_manager(go_client, llm_client)

    assert llm_client.complete.call_count == 3
    keys = [call.kwargs["idempotency_key"] for call in llm_client.complete.call_args_list]
    assert keys == ["run:llm:worker", "run:llm:worker-escalated", "run:llm:reviewer"]
    escalated_call = llm_client.complete.call_args_list[1].kwargs
    assert escalated_call["task_type"] == "deep_analysis"
    phases = [call.kwargs["phase"] for call in go_client.record_event.call_args_list]
    assert PHASE_LLM_ESCALATED in phases


def test_run_b_rotation_swaps_worker_and_reviewer() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = _run_context(worker_rotation_pattern="RUN_B")
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        _llm_result(content="worker output", persona_key=PERSONA_REVIEWER),
        _llm_result(
            call_id="call-2",
            content="review output",
            model_key="careful_analyst",
            provider_key="primary",
            persona_key=PERSONA_WORKER,
        ),
    ]

    _start_manager(go_client, llm_client)

    worker_call = llm_client.complete.call_args_list[0].kwargs
    reviewer_call = llm_client.complete.call_args_list[1].kwargs
    assert worker_call["persona_key"] == PERSONA_REVIEWER
    assert reviewer_call["persona_key"] == PERSONA_WORKER
    assert worker_call["persona_key"] != reviewer_call["persona_key"]
