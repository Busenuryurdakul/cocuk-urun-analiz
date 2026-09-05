import threading
from unittest.mock import MagicMock

from miyuna_agent.run_manager import CancelRunPayload, RunManager, StartRunPayload


def test_run_manager_starts_lifecycle_thread() -> None:
    go_client = MagicMock()
    go_client.fetch_run_context.return_value = MagicMock(
        trace_id="trace-1",
        product_id="prod-1",
        marketplace_review_count=0,
        capabilities=MagicMock(
            available_tools=(
                MagicMock(name="policy_evaluator", version="frozen-v1", availability="AVAILABLE"),
                MagicMock(name="compliance_checker", version="frozen-v1", availability="AVAILABLE"),
                MagicMock(name="dataset_validator", version="frozen-v1", availability="AVAILABLE"),
                MagicMock(name="pii_redactor", version="frozen-v1", availability="AVAILABLE"),
            )
        ),
    )
    go_client.cancellation_requested.return_value = False
    go_client.authorize_tool.return_value = MagicMock(
        allowed=True, grant_nonce="nonce", tool_execution_id="exec"
    )
    go_client.execute_tool.return_value = MagicMock(status="OK", payload={})

    llm_client = MagicMock()
    llm_client.complete.side_effect = [
        MagicMock(
            call_id="call-1",
            content="worker",
            model_key="careful_analyst",
            provider_key="primary",
            fallback_used=False,
            routing_reason="task=analysis",
            persona_key="careful_analyst",
            persona_version="1.0.0",
            correlation_id="trace-1",
            input_tokens=1,
            output_tokens=1,
        ),
        MagicMock(
            call_id="call-2",
            content="review",
            model_key="result_analyst",
            provider_key="secondary",
            fallback_used=False,
            routing_reason="task=review",
            persona_key="result_analyst",
            persona_version="1.0.0",
            correlation_id="trace-1",
            input_tokens=1,
            output_tokens=1,
        ),
    ]

    started = threading.Event()

    def worker_factory(target):
        started.set()
        return threading.Thread(target=target, daemon=True)

    manager = RunManager(go_client=go_client, llm_client=llm_client, _worker_factory=worker_factory)
    manager.start_run(
        StartRunPayload(
            organization_id="org",
            analysis_run_id="run",
            product_id="prod",
            trace_id="trace-1",
            actor_user_id="user",
            capabilities={},
        )
    )
    assert started.wait(timeout=2)
    threading.Event().wait(0.5)
    assert go_client.update_run_status.called
    assert go_client.record_event.called


def test_cancel_run_signals_active_worker() -> None:
    go_client = MagicMock()
    cancel_seen = threading.Event()

    def runner() -> None:
        cancel_seen.wait(timeout=2)

    manager = RunManager(go_client=go_client, llm_client=MagicMock())
    with manager._lock:
        manager._active["run"] = cancel_seen

    manager.cancel_run(
        CancelRunPayload(organization_id="org", analysis_run_id="run", trace_id="trace")
    )
    assert cancel_seen.is_set()
