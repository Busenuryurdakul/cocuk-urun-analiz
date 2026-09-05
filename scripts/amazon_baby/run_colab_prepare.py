#!/usr/bin/env python3
"""Local helper to exercise Colab prep flow without downloading Kaggle data."""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from miyuna_adapter import (  # noqa: E402
    TRAINING_RIGHTS_APPROVED,
    adapt_dataframe,
    find_kaggle_csv,
    print_colab_summary,
    validate_import_jsonl,
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
    print_colab_summary(report)

    ok, errors, count = validate_import_jsonl(paths["import"])
    print(f"Validated records: {count}")
    if ok:
        print("MIYUNA_COLAB_DATASET_PREP: PASS")
    else:
        print("MIYUNA_COLAB_DATASET_PREP: FAIL")
        for err in errors:
            print(f"  - {err}")
        raise SystemExit(1)

    if not TRAINING_RIGHTS_APPROVED:
        print("TRAINING BLOCKED: license / usage rights not approved.")


if __name__ == "__main__":
    main()
