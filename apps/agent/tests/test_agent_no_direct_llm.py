from pathlib import Path


def test_agent_package_has_no_direct_provider_calls() -> None:
    root = Path(__file__).resolve().parents[1] / "src" / "miyuna_agent"
    forbidden = (
        "openai",
        "anthropic",
        "api.openai.com",
        "generativelanguage.googleapis.com",
        "LLM_PRIMARY_API_KEY",
        "LLM_SECONDARY_API_KEY",
    )
    violations: list[str] = []
    for path in root.rglob("*.py"):
        if path.name == "llm_client.py":
            continue
        text = path.read_text(encoding="utf-8").lower()
        for token in forbidden:
            if token.lower() in text:
                violations.append(f"{path.name}: {token}")
    assert not violations, f"direct provider usage detected: {violations}"
