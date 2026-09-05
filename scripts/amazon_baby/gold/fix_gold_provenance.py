"""Correct Gold pilot review provenance: 2 human, 98 AI-assisted finalized."""

from __future__ import annotations

import json
import sys
from pathlib import Path

from gold_pilot_common import (
    FINALIZED_JSONL,
    FINALIZED_CSV,
    REVIEWED_CSV,
    REVIEWED_JSONL,
    apply_truthful_provenance,
    read_pilot_jsonl,
    status_counts,
    provenance_counts,
    validate_pilot_records,
    write_pilot_csv,
    write_pilot_jsonl,
)

GOLD_DIR = Path(__file__).resolve().parent
LEGACY_APPROVED = GOLD_DIR / "miyuna_gold_pilot_approved.jsonl"


def main() -> int:
    records = read_pilot_jsonl(REVIEWED_JSONL)
    if len(records) != 100:
        print(f"ERROR: expected 100 reviewed records, got {len(records)}")
        return 1

    for record in records:
        apply_truthful_provenance(record)

    errors = validate_pilot_records(records)
    if errors:
        print("Validation errors:")
        for err in errors[:20]:
            print(f"  - {err}")
        return 1

    write_pilot_jsonl(records, REVIEWED_JSONL)
    write_pilot_csv(records, REVIEWED_CSV)

    export_records = [dict(record) for record in records]
    write_pilot_jsonl(export_records, FINALIZED_JSONL)
    write_pilot_csv(export_records, FINALIZED_CSV)

    if LEGACY_APPROVED.exists():
        LEGACY_APPROVED.unlink()
        print(f"Removed misleading export: {LEGACY_APPROVED.name}")

    status = status_counts(records)
    prov = provenance_counts(records)
    print("Provenance fix applied.")
    print(f"  labelReviewStatus: {status}")
    print(f"  labelProvenance: {prov}")
    print(f"  reviewed: {REVIEWED_JSONL.name}")
    print(f"  export: {FINALIZED_JSONL.name}, {FINALIZED_CSV.name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
