#!/usr/bin/env python3
"""Validate Miyuna Gold Dataset Pilot records."""

from __future__ import annotations

import sys

from gold_pilot_common import PILOT_JSONL, read_pilot_jsonl, validate_pilot_records


def main() -> int:
    if not PILOT_JSONL.exists():
        print("GOLD_PILOT_VALIDATION: FAIL (pilot JSONL missing)", file=sys.stderr)
        return 1

    records = read_pilot_jsonl()
    errors = validate_pilot_records(records)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        print("GOLD_PILOT_VALIDATION: FAIL", file=sys.stderr)
        return 1

    print("GOLD_PILOT_VALIDATION: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
