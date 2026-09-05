#!/usr/bin/env python3
"""Summarize Miyuna Gold Dataset Pilot review progress."""

from __future__ import annotations

import json
import sys

from gold_pilot_common import (
    PILOT_JSONL,
    approved_records,
    distribution,
    rating_distribution,
    read_pilot_jsonl,
    status_counts,
)


def main() -> int:
    if not PILOT_JSONL.exists():
        print("GOLD_REVIEW_SUMMARY: FAIL (pilot JSONL missing)", file=sys.stderr)
        return 1

    records = read_pilot_jsonl()
    counts = status_counts(records)
    approved = approved_records(records)

    print(f"TOTAL: {len(records)}")
    print(f"PENDING: {counts.get('PENDING_HUMAN_REVIEW', 0)}")
    print(f"HUMAN_REVIEWED: {counts.get('HUMAN_REVIEWED', 0)}")
    print(f"EXPERT_APPROVED: {counts.get('EXPERT_APPROVED', 0)}")
    print(f"REJECTED: {counts.get('REJECTED', 0)}")
    print(
        "SENTIMENT_DISTRIBUTION:",
        json.dumps(distribution(records, "sentiment", "manualLabels")),
    )
    print(
        "ISSUE_TYPE_DISTRIBUTION:",
        json.dumps(distribution(records, "issueType", "manualLabels")),
    )
    print(
        "SAFETY_OBSERVATION_DISTRIBUTION:",
        json.dumps(
            distribution(records, "safetyRelatedObservation", "manualLabels")
        ),
    )
    print("RATING_DISTRIBUTION:", json.dumps(rating_distribution(records)))
    print(f"APPROVED_GOLD_COUNT: {len(approved)}")
    print("GOLD_REVIEW_SUMMARY: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
