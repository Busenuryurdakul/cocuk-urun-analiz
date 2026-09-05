#!/usr/bin/env python3
"""Build Miyuna Gold Dataset Pilot (100 records)."""

from __future__ import annotations

import json
import sys

from gold_pilot_common import (
    PILOT_SIZE,
    SEED,
    SOURCE_ARTIFACT,
    build_pilot_records,
    distribution,
    load_source_records,
    rating_distribution,
    write_pilot_csv,
    write_pilot_jsonl,
)


def main() -> int:
    try:
        source_records = load_source_records()
        pilot_records = build_pilot_records(source_records, size=PILOT_SIZE, seed=SEED)
        write_pilot_jsonl(pilot_records)
        write_pilot_csv(pilot_records)
    except ValueError as exc:
        print(f"BUILD_FAILED: {exc}", file=sys.stderr)
        return 1

    print(f"SOURCE_RECORDS_AVAILABLE: {len(source_records)}")
    print(f"PILOT_RECORDS: {len(pilot_records)}")
    print(f"SEED: {SEED}")
    print(
        "SENTIMENT_DISTRIBUTION:",
        json.dumps(distribution(pilot_records, "sentiment", "rulePrelabels")),
    )
    print(
        "ISSUE_TYPE_DISTRIBUTION:",
        json.dumps(distribution(pilot_records, "issueType", "rulePrelabels")),
    )
    print(
        "SAFETY_OBSERVATION_DISTRIBUTION:",
        json.dumps(
            distribution(pilot_records, "safetyRelatedObservation", "rulePrelabels")
        ),
    )
    print("RATING_DISTRIBUTION:", json.dumps(rating_distribution(pilot_records)))
    print("GOLD_PILOT_BUILD: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
