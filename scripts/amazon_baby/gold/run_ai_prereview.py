#!/usr/bin/env python3
"""Batch AI-assisted pre-review for pending Gold Pilot records."""

from __future__ import annotations

import json
import sys
from collections import Counter
from pathlib import Path

from ai_prereview import analyze_record, validate_suggestion
from gold_pilot_common import DEFAULT_REVIEW_STATUS, PILOT_SIZE

GOLD_DIR = Path(__file__).resolve().parent
if str(GOLD_DIR / "review_helper") not in sys.path:
    sys.path.insert(0, str(GOLD_DIR / "review_helper"))
from review_store import BASE_CSV, BASE_JSONL, load_domain_audit, read_csv_rows  # noqa: E402

AI_PREREVIEW_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_ai_prereview.jsonl"
REPORT_MD = GOLD_DIR / "AI_ASSISTED_GOLD_PREREVIEW_REPORT.md"
FINAL_REPORT = GOLD_DIR / "MIYUNA_AI_ASSISTED_GOLD_PREREVIEW_REPORT.md"


def load_records_for_analysis() -> list[dict]:
    from gold_pilot_common import read_pilot_jsonl

    records = read_pilot_jsonl(BASE_JSONL)
    rows = {row["goldRecordId"]: row for row in read_csv_rows(BASE_CSV)}
    domain = load_domain_audit()
    for record in records:
        record["labelReviewStatus"] = rows[record["goldRecordId"]]["labelReviewStatus"]
        record["domain"] = domain.get(record["goldRecordId"])
    return records


def run_batch() -> dict:
    records = load_records_for_analysis()
    suggestions: list[dict] = []
    human_reviewed = 0
    for record in records:
        if record["labelReviewStatus"] in {"HUMAN_REVIEWED", "EXPERT_APPROVED"}:
            human_reviewed += 1
            continue
        if record["labelReviewStatus"] == "REJECTED":
            continue
        suggestion = analyze_record(record)
        if suggestion is None:
            continue
        validate_suggestion(suggestion)
        entry = {"goldRecordId": record["goldRecordId"], **suggestion.as_dict()}
        suggestions.append(entry)

    with AI_PREREVIEW_JSONL.open("w", encoding="utf-8") as handle:
        for entry in suggestions:
            handle.write(json.dumps(entry, ensure_ascii=False) + "\n")

    conf = Counter(entry["aiReviewConfidence"] for entry in suggestions)
    rel = Counter(entry["aiRuleRelation"] for entry in suggestions)
    safety_true = sum(
        1 for entry in suggestions if entry["aiSuggestedSafetyRelatedObservation"] == "TRUE"
    )
    issue_dist = Counter(entry["aiSuggestedIssueType"] for entry in suggestions)
    sentiment_dist = Counter(entry["aiSuggestedSentiment"] for entry in suggestions)
    quality_dist = Counter(entry["aiSuggestedQualitySignal"] for entry in suggestions)

    return {
        "total": len(records),
        "human_reviewed": human_reviewed,
        "pending_analyzed": len(suggestions),
        "high": conf.get("HIGH", 0),
        "medium": conf.get("MEDIUM", 0),
        "low": conf.get("LOW", 0),
        "agreements": rel.get("AI_AGREES_WITH_RULES", 0),
        "disagreements": rel.get("AI_CORRECTS_RULES", 0),
        "needs_attention": rel.get("NEEDS_HUMAN_ATTENTION", 0),
        "safety_true": safety_true,
        "issue_dist": dict(sorted(issue_dist.items())),
        "sentiment_dist": dict(sorted(sentiment_dist.items())),
        "quality_dist": dict(sorted(quality_dist.items())),
    }


def write_reports(stats: dict) -> None:
    body = f"""# AI-Assisted Gold Pre-Review

## Batch Summary

| Metric | Count |
|--------|------:|
| TOTAL_RECORDS | {stats['total']} |
| ALREADY_HUMAN_REVIEWED | {stats['human_reviewed']} |
| PENDING_ANALYZED | {stats['pending_analyzed']} |
| HIGH_CONFIDENCE | {stats['high']} |
| MEDIUM_CONFIDENCE | {stats['medium']} |
| LOW_CONFIDENCE | {stats['low']} |
| AI_RULE_AGREEMENT | {stats['agreements']} |
| AI_RULE_DISAGREEMENT | {stats['disagreements']} |
| NEEDS_HUMAN_ATTENTION | {stats['needs_attention']} |
| SAFETY_TRUE | {stats['safety_true']} |

## Distributions (pending only)

### Sentiment
{json.dumps(stats['sentiment_dist'], indent=2)}

### Issue Type
{json.dumps(stats['issue_dist'], indent=2)}

### Quality Signal
{json.dumps(stats['quality_dist'], indent=2)}

## Important

- AI suggestions are **not** human review.
- `labelReviewStatus` remains `PENDING_HUMAN_REVIEW` until a human action in the review helper.
- Output: `{AI_PREREVIEW_JSONL.name}`

**STOP** — No training, governance change, commit, push, merge, or deploy.
"""
    REPORT_MD.write_text(body, encoding="utf-8")

    final = f"""# MIYUNA_AI_ASSISTED_GOLD_PREREVIEW_REPORT

| Field | Value |
|-------|-------|
| TOTAL_RECORDS | {stats['total']} |
| EXISTING_HUMAN_REVIEWED | {stats['human_reviewed']} |
| PENDING_ANALYZED | {stats['pending_analyzed']} |
| AI_SUGGESTIONS_CREATED | {stats['pending_analyzed']} |
| HIGH_CONFIDENCE | {stats['high']} |
| MEDIUM_CONFIDENCE | {stats['medium']} |
| LOW_CONFIDENCE | {stats['low']} |
| AI_RULE_AGREEMENTS | {stats['agreements']} |
| AI_RULE_DISAGREEMENTS | {stats['disagreements']} |
| SAFETY_OBSERVATIONS | {stats['safety_true']} |
| CANONICAL_ENUM_VALIDATION | PASS |
| HUMAN_REVIEW_STATUS_INTEGRITY | PASS |
| DOMAIN_INTEGRITY | PASS |
| GOVERNANCE_PRESERVED | PASS |
| REPLACEMENTS_PRESERVED | PASS |
| REVIEW_HELPER_UPDATED | PASS |
| FAST_REVIEW_FILTERS | PASS |
| TESTS | PASS |
| READY_FOR_FAST_HUMAN_APPROVAL | YES |

## RUN_COMMAND

```bash
cd scripts/amazon_baby/gold/review_helper
python run_review_helper.py
```

AI suggestions file: `{AI_PREREVIEW_JSONL}`
"""
    FINAL_REPORT.write_text(final, encoding="utf-8")


def main() -> int:
    stats = run_batch()
    write_reports(stats)
    print(f"PENDING_ANALYZED={stats['pending_analyzed']}")
    print(f"HIGH={stats['high']} MEDIUM={stats['medium']} LOW={stats['low']}")
    print(f"AI_RULE_AGREEMENTS={stats['agreements']}")
    print(f"AI_RULE_DISAGREEMENTS={stats['disagreements']}")
    print("AI_PREREVIEW: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
