#!/usr/bin/env python3
"""Legacy wrapper — use export_finalized_gold.py (approved name was misleading)."""

from __future__ import annotations

import sys

from export_finalized_gold import main


if __name__ == "__main__":
    print(
        "NOTE: export_approved_gold.py is deprecated; use export_finalized_gold.py",
        file=sys.stderr,
    )
    sys.exit(main())
