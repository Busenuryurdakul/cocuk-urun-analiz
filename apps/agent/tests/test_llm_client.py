from miyuna_agent.llm_client import GoLLMClient, LLMCompletionResult


def test_llm_completion_result_from_dict_camel_case() -> None:
    raw = {
        "callId": "abc",
        "content": "hello",
        "modelKey": "careful_analyst",
        "providerKey": "primary",
        "fallbackUsed": False,
        "routingReason": "task=analysis",
        "personaKey": "careful_analyst",
        "personaVersion": "1.0.0",
        "correlationId": "corr-1",
        "inputTokens": 10,
        "outputTokens": 5,
    }
    result = LLMCompletionResult.from_dict(raw)
    assert result.call_id == "abc"
    assert result.model_key == "careful_analyst"
    assert result.provider_key == "primary"
    assert result.correlation_id == "corr-1"
    assert result.routing_reason.startswith("task=")


def test_go_llm_client_headers() -> None:
    client = GoLLMClient("http://127.0.0.1:8080", "token")
    assert client._headers()["X-Miyuna-Internal-Token"] == "token"
