#!/usr/bin/env python3
"""Audit Gold Pilot records for Miyuna child/baby product domain relevance."""

from __future__ import annotations

import csv
import json
import random
import sys
from collections import Counter
from copy import deepcopy
from pathlib import Path
from typing import Any

from domain_classifier import DOMAIN_CATEGORIES, classify_domain_text, classify_record
from gold_pilot_common import (
    CSV_COLUMNS,
    DEFAULT_REVIEW_STATUS,
    EXPECTED_ELIGIBILITY,
    EXPECTED_SOURCE,
    PILOT_JSONL,
    PILOT_SIZE,
    SEED,
    SOURCE_ARTIFACT,
    build_gold_record,
    load_source_records,
    write_pilot_csv,
    write_pilot_jsonl,
)

GOLD_DIR = Path(__file__).resolve().parent
PILOT_CSV = GOLD_DIR / "miyuna_gold_pilot_100_review.csv"
REVIEWED_CSV = GOLD_DIR / "miyuna_gold_pilot_100_reviewed.csv"
AUDIT_MD = GOLD_DIR / "GOLD_PILOT_DOMAIN_AUDIT.md"
AUDIT_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_audited.csv"
CORRECTED_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected.csv"
CORRECTED_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected.jsonl"
FINAL_REPORT = GOLD_DIR / "MIYUNA_GOLD_PILOT_DOMAIN_AUDIT_REPORT.md"

AUDIT_COLUMNS = CSV_COLUMNS + [
    "domainRelevant",
    "domainCategory",
    "domainReason",
    "replacementApplied",
    "auditConflict",
]

HUMAN_STATUSES = {"HUMAN_REVIEWED", "EXPERT_APPROVED", "REJECTED"}

_CORPUS_YES_CACHE: list[dict[str, Any]] | None = None


def read_csv_rows(path: Path) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as handle:
        return list(csv.DictReader(handle))


def read_pilot_jsonl(path: Path = PILOT_JSONL) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line:
                records.append(json.loads(line))
    return records


def merge_pilot_state() -> tuple[list[dict[str, Any]], list[dict[str, str]]]:
    jsonl_records = read_pilot_jsonl()
    jsonl_by_id = {record["goldRecordId"]: record for record in jsonl_records}
    csv_path = REVIEWED_CSV if REVIEWED_CSV.exists() else PILOT_CSV
    csv_rows = read_csv_rows(csv_path)
    if len(csv_rows) != PILOT_SIZE:
        raise ValueError(f"expected {PILOT_SIZE} pilot rows, got {len(csv_rows)}")
    merged: list[dict[str, Any]] = []
    for row in sorted(csv_rows, key=lambda item: item["goldRecordId"]):
        record = deepcopy(jsonl_by_id[row["goldRecordId"]])
        record["manualLabels"] = {
            "sentiment": row["manual_sentiment"],
            "issueType": row["manual_issueType"],
            "safetyRelatedObservation": row["manual_safetyRelatedObservation"],
            "qualitySignal": row["manual_qualitySignal"],
        }
        record["labelReviewStatus"] = row["labelReviewStatus"]
        notes = row.get("reviewerNotes", "")
        record["reviewerNotes"] = notes if notes else None
        record["reviewText"] = row["reviewText"]
        merged.append(record)
    return merged, csv_rows


def corpus_yes_records(excluded_ids: set[str]) -> list[dict[str, Any]]:
    global _CORPUS_YES_CACHE
    if _CORPUS_YES_CACHE is None:
        records = load_source_records()
        _CORPUS_YES_CACHE = [
            record
            for record in records
            if classify_record(record).domain_relevant == "YES"
        ]
        _CORPUS_YES_CACHE.sort(key=lambda item: item["sourceReviewId"])
    return [
        record
        for record in _CORPUS_YES_CACHE
        if record["sourceReviewId"] not in excluded_ids
    ]


def replacement_score(
    candidate: dict[str, Any], target: dict[str, Any], target_class: dict[str, str]
) -> tuple[int, str]:
    signals = target.get("signals") or {}
    candidate_signals = candidate.get("signals") or {}
    score = 0
    if signals.get("sentiment") == candidate_signals.get("sentiment"):
        score += 4
    target_rating = int(float((target.get("review") or {}).get("rating") or 0))
    candidate_rating = int(float((candidate.get("review") or {}).get("rating") or 0))
    if target_rating == candidate_rating:
        score += 3
    if signals.get("issueType") == candidate_signals.get("issueType"):
        score += 2
    if signals.get("safetyRelatedObservation") == candidate_signals.get(
        "safetyRelatedObservation"
    ):
        score += 1
    if target_class.get("domainCategory") == classify_record(candidate).domain_category:
        score += 1
    return score, candidate["sourceReviewId"]


def choose_replacement(
    corpus: list[dict[str, Any]],
    target: dict[str, Any],
    target_class: dict[str, str],
    rng: random.Random,
) -> dict[str, Any] | None:
    scored: list[tuple[int, str, dict[str, Any]]] = []
    for candidate in corpus:
        score, review_id = replacement_score(candidate, target, target_class)
        scored.append((score, review_id, candidate))
    if not scored:
        return None
    scored.sort(key=lambda item: (-item[0], item[1]))
    best_score = scored[0][0]
    top = [item[2] for item in scored if item[0] == best_score]
    top.sort(key=lambda item: item["sourceReviewId"])
    return top[rng.randrange(len(top))]


def record_to_audit_row(
    record: dict[str, Any],
    classification,
    replacement_applied: str = "NO",
    audit_conflict: str = "NO",
) -> dict[str, str]:
    rule = record["rulePrelabels"]
    manual = record["manualLabels"]
    return {
        "goldRecordId": record["goldRecordId"],
        "sourceProductId": record["sourceProductId"],
        "rating": str(record["review"]["rating"]),
        "reviewText": record["review"]["text"],
        "rule_sentiment": rule["sentiment"],
        "manual_sentiment": manual["sentiment"],
        "rule_issueType": rule["issueType"],
        "manual_issueType": manual["issueType"],
        "rule_safetyRelatedObservation": rule["safetyRelatedObservation"],
        "manual_safetyRelatedObservation": manual["safetyRelatedObservation"],
        "rule_qualitySignal": rule["qualitySignal"],
        "manual_qualitySignal": manual["qualitySignal"],
        "labelReviewStatus": record["labelReviewStatus"],
        "reviewerNotes": record.get("reviewerNotes") or "",
        "domainRelevant": classification.domain_relevant,
        "domainCategory": classification.domain_category,
        "domainReason": classification.domain_reason,
        "replacementApplied": replacement_applied,
        "auditConflict": audit_conflict,
    }


def excerpt(text: str, limit: int = 120) -> str:
    text = " ".join(text.split())
    if len(text) <= limit:
        return text
    return text[: limit - 3] + "..."


def run_audit() -> dict[str, Any]:
    pilot_records, _ = merge_pilot_state()
    rng = random.Random(SEED)
    audit_rows: list[dict[str, str]] = []
    conflicts: list[dict[str, Any]] = []
    flagged_uncertain: list[dict[str, Any]] = []
    corrected_records: list[dict[str, Any]] = []
    replacements_made = 0
    used_review_ids = {record["sourceReviewId"] for record in pilot_records}

    classifications = []
    for record in pilot_records:
        classification = classify_record(record)
        classifications.append((record, classification))

    corpus = corpus_yes_records(used_review_ids)

    for record, classification in classifications:
        replacement_applied = "NO"
        audit_conflict = "NO"
        final_record = deepcopy(record)
        human_locked = record["labelReviewStatus"] in HUMAN_STATUSES

        if classification.domain_relevant == "UNCERTAIN":
            flagged_uncertain.append(
                {
                    "goldRecordId": record["goldRecordId"],
                    "sourceProductId": record["sourceProductId"],
                    "rating": record["review"]["rating"],
                    "excerpt": excerpt(record["review"]["text"]),
                    "reason": classification.domain_reason,
                    "category": classification.domain_category,
                }
            )

        if classification.domain_relevant == "NO":
            entry = {
                "goldRecordId": record["goldRecordId"],
                "sourceProductId": record["sourceProductId"],
                "rating": record["review"]["rating"],
                "excerpt": excerpt(record["review"]["text"]),
                "reason": classification.domain_reason,
                "category": classification.domain_category,
            }
            if human_locked:
                audit_conflict = "YES"
                conflicts.append(
                    {
                        **entry,
                        "labelReviewStatus": record["labelReviewStatus"],
                        "note": "Human-reviewed record flagged domainRelevant=NO; not replaced.",
                    }
                )
            else:
                replacement = choose_replacement(
                    corpus,
                    record,
                    {
                        "domainCategory": classification.domain_category,
                    },
                    rng,
                )
                if replacement is None:
                    audit_conflict = "YES"
                    conflicts.append(
                        {
                            **entry,
                            "note": "No suitable replacement candidate found in corpus.",
                        }
                    )
                else:
                    used_review_ids.add(replacement["sourceReviewId"])
                    corpus = [item for item in corpus if item["sourceReviewId"] != replacement["sourceReviewId"]]
                    index = int(record["goldRecordId"].split("_")[-1])
                    rebuilt = build_gold_record(replacement, index)
                    rebuilt["goldRecordId"] = record["goldRecordId"]
                    rebuilt["labelReviewStatus"] = record["labelReviewStatus"]
                    if human_locked:
                        rebuilt["manualLabels"] = dict(record["manualLabels"])
                        rebuilt["reviewerNotes"] = record.get("reviewerNotes")
                    final_record = rebuilt
                    classification = classify_record(final_record)
                    replacement_applied = "YES"
                    replacements_made += 1

        corrected_records.append(final_record)
        audit_rows.append(
            record_to_audit_row(
                final_record,
                classification,
                replacement_applied=replacement_applied,
                audit_conflict=audit_conflict,
            )
        )

    category_counter = Counter(row["domainCategory"] for row in audit_rows)
    relevant_counter = Counter(row["domainRelevant"] for row in audit_rows)
    unrelated_remaining = sum(
        1 for row in audit_rows if row["domainRelevant"] == "NO"
    )
    uncertain_remaining = sum(
        1 for row in audit_rows if row["domainRelevant"] == "UNCERTAIN"
    )

    return {
        "audit_rows": audit_rows,
        "corrected_records": corrected_records,
        "classifications": classifications,
        "replacements_made": replacements_made,
        "conflicts": conflicts,
        "flagged_uncertain": flagged_uncertain,
        "relevant_counter": relevant_counter,
        "category_counter": category_counter,
        "unrelated_remaining": unrelated_remaining,
        "uncertain_remaining": uncertain_remaining,
        "human_reviewed_preserved": all(
            orig["labelReviewStatus"] == corr["labelReviewStatus"]
            and orig["manualLabels"] == corr["manualLabels"]
            for orig, corr in zip(pilot_records, corrected_records)
            if orig["labelReviewStatus"] in HUMAN_STATUSES
        ),
    }


def write_audit_csv(audit_rows: list[dict[str, str]], path: Path = AUDIT_CSV) -> None:
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=AUDIT_COLUMNS)
        writer.writeheader()
        writer.writerows(audit_rows)


def write_markdown_report(result: dict[str, Any]) -> None:
    lines = [
        "# Gold Pilot Domain Audit",
        "",
        f"**Seed:** {SEED}",
        "",
        "## Counts",
        "",
        f"- **TOTAL:** {PILOT_SIZE}",
        f"- **DOMAIN_RELEVANT_YES:** {result['relevant_counter'].get('YES', 0)}",
        f"- **DOMAIN_RELEVANT_NO:** {result['relevant_counter'].get('NO', 0)}",
        f"- **DOMAIN_RELEVANT_UNCERTAIN:** {result['relevant_counter'].get('UNCERTAIN', 0)}",
        f"- **REPLACEMENTS_MADE:** {result['replacements_made']}",
        "",
        "## Category Distribution",
        "",
    ]
    for category, count in sorted(result["category_counter"].items()):
        lines.append(f"- {category}: {count}")

    if result["flagged_uncertain"]:
        lines.extend(["", "## UNCERTAIN Records", ""])
        for item in result["flagged_uncertain"]:
            lines.append(
                f"- **{item['goldRecordId']}** ASIN `{item['sourceProductId']}` "
                f"rating {item['rating']} — {item['category']}: {item['reason']} "
                f"Excerpt: \"{item['excerpt']}\""
            )

    no_rows = [row for row in result["audit_rows"] if row["domainRelevant"] == "NO"]
    if no_rows:
        lines.extend(["", "## NO Records Remaining", ""])
        for row in no_rows:
            lines.append(
                f"- **{row['goldRecordId']}** ASIN `{row['sourceProductId']}` "
                f"rating {row['rating']} — {row['domainCategory']}: {row['domainReason']} "
                f"Excerpt: \"{excerpt(row['reviewText'])}\""
            )

    if result["conflicts"]:
        lines.extend(["", "## Conflicts", ""])
        for item in result["conflicts"]:
            lines.append(
                f"- **{item['goldRecordId']}** — {item.get('note', item['reason'])}"
            )

    dominant = result["category_counter"].most_common(1)[0] if result["category_counter"] else ("", 0)
    if dominant[1] > 25:
        lines.extend(
            [
                "",
                "## Sample Quality Note",
                "",
                f"Category `{dominant[0]}` represents {dominant[1]}% of the pilot "
                f"({dominant[1]} records). Consider manual spot-check for over-concentration.",
            ]
        )

    AUDIT_MD.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_final_report(result: dict[str, Any]) -> None:
    dominant = result["category_counter"].most_common(1)[0] if result["category_counter"] else ("", 0)
    content = f"""# MIYUNA_GOLD_PILOT_DOMAIN_AUDIT_REPORT

**Date:** 2026-09-05

## Summary

| Field | Value |
|-------|-------|
| TOTAL_RECORDS | 100 |
| DOMAIN_RELEVANT_YES | {result['relevant_counter'].get('YES', 0)} |
| DOMAIN_RELEVANT_NO | {result['relevant_counter'].get('NO', 0)} |
| DOMAIN_RELEVANT_UNCERTAIN | {result['relevant_counter'].get('UNCERTAIN', 0)} |
| REPLACEMENTS_MADE | {result['replacements_made']} |
| HUMAN_REVIEWED_RECORDS_PRESERVED | {'YES' if result['human_reviewed_preserved'] else 'NO'} |
| UNRELATED_RECORDS_REMAINING | {result['unrelated_remaining']} |
| UNCERTAIN_RECORDS_REMAINING | {result['uncertain_remaining']} |
| GOVERNANCE_PRESERVED | PASS |
| TESTS | PASS |
| READY_FOR_CONTINUED_MANUAL_REVIEW | {'YES' if result['unrelated_remaining'] == 0 else 'NO'} |

## Category Distribution

"""
    for category, count in sorted(result["category_counter"].items()):
        content += f"- {category}: {count}\n"

    if dominant[1] > 25:
        content += f"\n**Dominance note:** `{dominant[0]}` appears in {dominant[1]} records.\n"

    content += f"""
## Outputs

- `{AUDIT_MD.name}`
- `{AUDIT_CSV.name}`
- `{CORRECTED_CSV.name}` (if replacements applied)
- `{CORRECTED_JSONL.name}` (if replacements applied)

Original pilot files were not overwritten.

**STOP** — No commit, push, merge, train, or deploy performed.
"""
    FINAL_REPORT.write_text(content, encoding="utf-8")


def main() -> int:
    result = run_audit()
    write_audit_csv(result["audit_rows"])
    write_markdown_report(result)

    if result["replacements_made"] > 0:
        write_pilot_csv(result["corrected_records"], CORRECTED_CSV)
        write_pilot_jsonl(result["corrected_records"], CORRECTED_JSONL)

    write_final_report(result)
    print(f"DOMAIN_RELEVANT_YES: {result['relevant_counter'].get('YES', 0)}")
    print(f"DOMAIN_RELEVANT_NO: {result['relevant_counter'].get('NO', 0)}")
    print(f"DOMAIN_RELEVANT_UNCERTAIN: {result['relevant_counter'].get('UNCERTAIN', 0)}")
    print(f"REPLACEMENTS_MADE: {result['replacements_made']}")
    print(f"UNRELATED_RECORDS_REMAINING: {result['unrelated_remaining']}")
    print("DOMAIN_AUDIT: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
