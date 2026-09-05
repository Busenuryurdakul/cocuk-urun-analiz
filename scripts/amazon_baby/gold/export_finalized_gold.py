#!/usr/bin/env python3
"""Export finalized Gold pilot records with truthful human/AI provenance."""

from __future__ import annotations

import json
import sys

from gold_pilot_common import (
    DEFAULT_REVIEW_STATUS,
    FINALIZED_CSV,
    FINALIZED_JSONL,
    GENUINE_HUMAN_GOLD_IDS,
    REVIEWED_JSONL,
    finalized_records,
    provenance_counts,
    read_pilot_jsonl,
    status_counts,
    validate_pilot_records,
    write_pilot_csv,
    write_pilot_jsonl,
)


def main() -> int:
    if not REVIEWED_JSONL.exists():
        print("FINALIZED_EXPORT: FAIL (reviewed JSONL missing)", file=sys.stderr)
        return 1

    records = read_pilot_jsonl(REVIEWED_JSONL)
    errors = validate_pilot_records(records)
    if errors:
        print("FINALIZED_EXPORT: FAIL (validation errors)", file=sys.stderr)
        for err in errors[:10]:
            print(f"  - {err}", file=sys.stderr)
        return 1

    finalized = finalized_records(records)
    pending = [
        record
        for record in records
        if record.get("labelReviewStatus") == DEFAULT_REVIEW_STATUS
    ]
    rejected = [
        record for record in records if record.get("labelReviewStatus") == "REJECTED"
    ]

    if len(finalized) != len(records):
        print(
            f"FINALIZED_EXPORT: FAIL (incomplete: {len(finalized)}/{len(records)})",
            file=sys.stderr,
        )
        return 1
    if pending or rejected:
        print("FINALIZED_EXPORT: FAIL (pending/rejected remain)", file=sys.stderr)
        return 1

    human_count = sum(
        1
        for record in records
        if record.get("goldRecordId") in GENUINE_HUMAN_GOLD_IDS
        and record.get("labelReviewStatus") == "HUMAN_REVIEWED"
        and record.get("labelProvenance") == "HUMAN"
    )
    ai_count = sum(
        1
        for record in records
        if record.get("labelReviewStatus") == "AI_ASSISTED_FINALIZED"
        and record.get("labelProvenance") == "AI_ASSISTED"
    )

    write_pilot_jsonl(finalized, FINALIZED_JSONL)
    write_pilot_csv(finalized, FINALIZED_CSV)

    for record in finalized:
        governance = record.get("governance") or {}
        if governance.get("originalDatasetEligibility") != "QUARANTINED":
            print("FINALIZED_EXPORT: FAIL (governance changed)", file=sys.stderr)
            return 1
        if "TRAINING_APPROVED" in json.dumps(record):
            print("FINALIZED_EXPORT: FAIL (training label detected)", file=sys.stderr)
            return 1

    print(f"FINALIZED_EXPORT_COUNT: {len(finalized)}")
    print(f"GENUINE_HUMAN_REVIEWED: {human_count}")
    print(f"AI_ASSISTED_FINALIZED: {ai_count}")
    print(f"LABEL_REVIEW_STATUS: {status_counts(records)}")
    print(f"LABEL_PROVENANCE: {provenance_counts(records)}")
    print(f"OUTPUT_JSONL: {FINALIZED_JSONL}")
    print(f"OUTPUT_CSV: {FINALIZED_CSV}")
    print("FINALIZED_EXPORT: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
