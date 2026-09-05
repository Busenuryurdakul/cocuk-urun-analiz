from miyuna_agent.planner import EXCLUDED_FROM_ANALYSIS_PLAN, build_deterministic_plan


def _available(*names: str) -> dict[str, str]:
    return {name: "frozen-v1" for name in names}


def test_planner_excludes_import_planner() -> None:
    plan = build_deterministic_plan(
        marketplace_review_count=0,
        available_tools=_available(
            "policy_evaluator",
            "compliance_checker",
            "dataset_validator",
            "pii_redactor",
        ),
    )
    names = {step.tool_name for step in plan}
    assert "import_planner" not in names
    assert "ecommerce_fetcher" not in names
    assert "product_normalizer" not in names


def test_planner_adds_review_sampler_when_reviews() -> None:
    plan = build_deterministic_plan(
        marketplace_review_count=2,
        available_tools=_available(
            "policy_evaluator",
            "compliance_checker",
            "review_sampler",
            "dataset_validator",
            "pii_redactor",
        ),
    )
    assert "review_sampler" in {step.tool_name for step in plan}


def test_unavailable_tool_not_in_plan() -> None:
    plan = build_deterministic_plan(
        marketplace_review_count=0,
        available_tools=_available("policy_evaluator", "compliance_checker", "pii_redactor"),
    )
    assert "dataset_validator" not in {step.tool_name for step in plan}


def test_excluded_tools_constant() -> None:
    assert "import_planner" in EXCLUDED_FROM_ANALYSIS_PLAN
    assert "product_normalizer" in EXCLUDED_FROM_ANALYSIS_PLAN
