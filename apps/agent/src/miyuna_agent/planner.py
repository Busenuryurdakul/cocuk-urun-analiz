"""Canonical deterministic analysis planner for Python orchestrator."""

from __future__ import annotations

from dataclasses import dataclass

EXCLUDED_FROM_ANALYSIS_PLAN = frozenset(
    {
        "import_planner",
        "ecommerce_fetcher",
        "product_normalizer",
        "import_diff_generator",
        "review_analyzer",
        "price_history_analyzer",
        "safety_analyzer",
        "age_analyzer",
        "material_analyzer",
        "market_analyzer",
        "evidence_validator",
        "report_generator",
    }
)


@dataclass(frozen=True)
class PlanStep:
    tool_name: str
    tool_version: str


def build_deterministic_plan(
    *,
    marketplace_review_count: int,
    available_tools: dict[str, str],
) -> list[PlanStep]:
    """Build plan from Go-provided capability snapshot — unavailable tools never enter plan."""
    candidates: list[tuple[str, str]] = [
        ("policy_evaluator", available_tools.get("policy_evaluator", "")),
        ("compliance_checker", available_tools.get("compliance_checker", "")),
    ]
    if marketplace_review_count > 0:
        candidates.append(("review_sampler", available_tools.get("review_sampler", "")))
    candidates.extend(
        [
            ("dataset_validator", available_tools.get("dataset_validator", "")),
            ("pii_redactor", available_tools.get("pii_redactor", "")),
        ]
    )

    steps: list[PlanStep] = []
    for tool_name, tool_version in candidates:
        if tool_name in EXCLUDED_FROM_ANALYSIS_PLAN:
            raise ValueError(f"excluded tool in analysis plan: {tool_name}")
        if not tool_version:
            continue
        steps.append(PlanStep(tool_name=tool_name, tool_version=tool_version))

    if not steps:
        raise ValueError("no available tools for analysis plan")
    return steps
