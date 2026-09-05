# MIYUNA_GOLD_REVIEW_CORRECTED_INPUT_REPORT

**Date:** 2026-09-05

## Preflight (before migration)

| Check | Result |
|-------|--------|
| Corrected CSV exists | YES — 100 records |
| Corrected JSONL exists | YES — 100 records |
| Duplicate sourceReviewId | NONE |
| goldRecordId range | gold_pilot_001 … gold_pilot_100 |
| **Previous helper input** | `miyuna_gold_pilot_100_review.csv` + `miyuna_gold_pilot_100.jsonl` |
| **Previous autosave** | `miyuna_gold_pilot_100_reviewed.csv` |
| Human-reviewed before migration | 2 (`gold_pilot_001`, `gold_pilot_002`) |

Nothing deleted.

---

## Active Paths After Migration

| Role | Path |
|------|------|
| **ACTIVE_BASE_INPUT** | `scripts/amazon_baby/gold/miyuna_gold_pilot_100_domain_corrected.csv` |
| Base JSONL (read-only) | `scripts/amazon_baby/gold/miyuna_gold_pilot_100_domain_corrected.jsonl` |
| Domain audit metadata | `scripts/amazon_baby/gold/miyuna_gold_pilot_100_domain_audited.csv` |
| **ACTIVE_REVIEWED_OUTPUT** | `scripts/amazon_baby/gold/miyuna_gold_pilot_100_domain_corrected_reviewed.csv` |
| Reviewed JSONL sync | `scripts/amazon_baby/gold/miyuna_gold_pilot_100_domain_corrected_reviewed.jsonl` |

Base corrected CSV/JSONL are **not** overwritten during review/autosave.

---

## Summary

| Field | Value |
|-------|-------|
| TOTAL_RECORDS | 100 |
| DOMAIN_YES | 100 |
| DOMAIN_NO | 0 |
| DOMAIN_UNCERTAIN | 0 |
| HUMAN_REVIEWED_BEFORE_MIGRATION | 2 |
| HUMAN_REVIEWED_AFTER_MIGRATION | 2 |
| GOLD_PILOT_001_PRESERVED | **PASS** |
| GOLD_PILOT_002_PRESERVED | **PASS** |
| OLD_REPLACED_RECORD_LABEL_LEAK | **NONE** |
| RESUME_RECORD | `gold_pilot_003` |
| KEYBOARD_SHORTCUTS | **PASS** (A/S/R + arrows) |
| FOCUS_PROTECTION | **PASS** |
| AUTOSAVE | **PASS** (reviewed copy only) |
| BASE_FILE_IMMUTABLE | **PASS** |
| GOVERNANCE_PRESERVED | **PASS** |
| TESTS | **PASS** (14/14) |
| LOCAL_HELPER_SMOKE | **PASS** |
| READY_FOR_CONTINUED_MANUAL_REVIEW | **YES** |

---

## Replacement Verification

| Slot | Expected ASIN | Actual | Status |
|------|---------------|--------|--------|
| REPLACEMENT_011 | B0057LUMX2 | B0057LUMX2 | **PASS** |
| REPLACEMENT_031 | B000066665 | B000066665 | **PASS** |
| REPLACEMENT_049 | B0038JDD2C | B0038JDD2C | **PASS** |
| REPLACEMENT_082 | B001PVAXZK | B001PVAXZK | **PASS** |

Replacement slots remain `PENDING_HUMAN_REVIEW` with corrected rule/manual pre-labels (no legacy label leak).

---

## Merge Rules Applied

- **Corrected pilot** = authoritative source text, ASIN, rule pre-labels, domain audit results
- **Legacy reviewed file** = human fields merged **only** when `goldRecordId` + `sourceProductId` match
- Replacement slots ignored stale legacy rows (different ASIN)

Preserved for `gold_pilot_001` / `gold_pilot_002`:
- `labelReviewStatus = HUMAN_REVIEWED`
- Manual labels (`USABILITY` corrections retained)

---

## UI Updates

- Domain block shown when audit fields available (`YES · CATEGORY` + reason)
- Keyboard shortcuts: **A** approve · **S** save · **R** reject · **←/→** navigate
- Shortcuts disabled while focus is in input/textarea/select

---

## RUN_COMMAND

```bash
cd C:\Users\MOSTER\Documents\GitHub\cocuk-urun-analiz\scripts\amazon_baby\gold\review_helper
python run_review_helper.py
```

Open: **http://127.0.0.1:8765**

---

## Files Changed

| File | Change |
|------|--------|
| `review_helper/review_store.py` | Switched to domain-corrected paths; immutable base; merge logic |
| `review_helper/index.html` | Domain display + keyboard shortcuts |
| `review_helper/test_review_helper.py` | Updated/expanded tests for corrected input |

**STOP** — No commit, push, merge, deploy, or fine-tuning.
