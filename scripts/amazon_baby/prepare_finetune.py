#!/usr/bin/env python3
"""
Miyuna — Amazon Baby Dataset preparation (PREPARE_ONLY / NOT_FOR_TRAINING).

Default mode blocks training artifacts until explicit governance approval exists.
"""

from __future__ import annotations

import sys
from pathlib import Path

# Allow running from repo root or scripts/amazon_baby
sys.path.insert(0, str(Path(__file__).resolve().parent))

from miyuna_adapter import (  # noqa: E402
    TRAINING_RIGHTS_APPROVED,
    adapt_dataframe,
    find_kaggle_csv,
    validate_training_governance,
    write_outputs,
)

import pandas as pd  # noqa: E402

OUT_DIR = Path(__file__).resolve().parent / "output"


def main() -> None:
    csv_path = find_kaggle_csv()
    print(f"Using CSV: {csv_path}")
    df = pd.read_csv(csv_path)
    accepted, rejected, report = adapt_dataframe(df)
    paths = write_outputs(OUT_DIR, accepted, rejected, report)

    print(f"Accepted rows: {report.accepted_rows}")
    print(f"Rejected rows: {report.rejected_rows}")
    print(f"Duplicate rows: {report.duplicate_rows}")
    print(f"Unique products: {report.unique_products}")
    print(f"Import JSONL: {paths['import']}")
    print(f"Quality report: {paths['report']}")

    if not TRAINING_RIGHTS_APPROVED:
        print("TRAINING BLOCKED: license / usage rights not approved.")
        return

    governance_meta = {
        "licenseStatus": "APPROVED",
        "usageRightsStatus": "APPROVED",
        "provenanceStatus": "VERIFIED",
        "piiPolicy": "PASS",
        "qualityPolicy": "PASS",
        "datasetApproval": "RECORDED",
    }
    ok, errors = validate_training_governance(governance_meta)
    if not ok:
        print("TRAINING BLOCKED: governance validation failed:")
        for err in errors:
            print(f"  - {err}")
        return

    print("Training rights approved — Phase 10 split generation is still not implemented in this script.")


if __name__ == "__main__":
    main()
