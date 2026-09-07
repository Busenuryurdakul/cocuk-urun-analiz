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
    assert "review_analyzer" not in EXCLUDED_FROM_ANALYSIS_PLAN
    assert "evidence_validator" not in EXCLUDED_FROM_ANALYSIS_PLAN
    assert "safety_analyzer" not in EXCLUDED_FROM_ANALYSIS_PLAN


def test_planner_includes_available_p0_analyzers() -> None:
    plan = build_deterministic_plan(
        marketplace_review_count=1,
        available_tools=_available(
            "policy_evaluator",
            "compliance_checker",
            "review_sampler",
            "review_analyzer",
            "safety_analyzer",
            "evidence_validator",
            "dataset_validator",
            "pii_redactor",
        ),
    )
    names = [step.tool_name for step in plan]
    assert names.index("review_sampler") < names.index("review_analyzer")
    assert names.index("review_analyzer") < names.index("safety_analyzer")
    assert names.index("safety_analyzer") < names.index("evidence_validator")
    assert "review_analyzer" in names
    assert "evidence_validator" in names
    assert "safety_analyzer" in names
    assert "age_analyzer" not in names
    assert "market_analyzer" not in names


def test_planner_skips_unavailable_analyzers() -> None:
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
    assert "review_analyzer" not in names
    assert "safety_analyzer" not in names
    assert "evidence_validator" not in names
