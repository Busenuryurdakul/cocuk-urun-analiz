"""Tests for AI-assisted Gold pre-review."""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

GOLD_DIR = Path(__file__).resolve().parent
if str(GOLD_DIR) not in sys.path:
    sys.path.insert(0, str(GOLD_DIR))
if str(GOLD_DIR / "review_helper") not in sys.path:
    sys.path.insert(0, str(GOLD_DIR / "review_helper"))

from ai_prereview import analyze_record, classify_rule_relation  # noqa: E402
from gold_pilot_common import ISSUE_TYPES, QUALITY_SIGNALS, SENTIMENTS  # noqa: E402
from review_store import ReviewStore, load_ai_prereview  # noqa: E402


@pytest.fixture(scope="module")
def ai_prereview_file():
    from run_ai_prereview import AI_PREREVIEW_JSONL, run_batch

    stats = run_batch()
    assert stats["total"] == 100
    assert AI_PREREVIEW_JSONL.exists()
    return stats, AI_PREREVIEW_JSONL


@pytest.fixture()
def isolated_store(tmp_path, monkeypatch):
    from review_store import (
        AI_PREREVIEW_JSONL,
        BASE_CSV,
        BASE_JSONL,
        DOMAIN_AUDIT_CSV,
        ReviewStore,
    )
    import shutil

    base_csv = tmp_path / "base.csv"
    base_jsonl = tmp_path / "base.jsonl"
    reviewed_csv = tmp_path / "reviewed.csv"
    reviewed_jsonl = tmp_path / "reviewed.jsonl"
    shutil.copy(BASE_JSONL, base_jsonl)
    shutil.copy(BASE_CSV, base_csv)
    monkeypatch.setattr("review_store.DOMAIN_AUDIT_CSV", DOMAIN_AUDIT_CSV)
    monkeypatch.setattr("review_store.AI_PREREVIEW_JSONL", AI_PREREVIEW_JSONL)
    return ReviewStore(
        base_jsonl_path=base_jsonl,
        base_csv_path=base_csv,
        reviewed_csv_path=reviewed_csv,
        reviewed_jsonl_path=reviewed_jsonl,
    )


def test_pending_records_get_ai_suggestions(isolated_store, ai_prereview_file):
    suggestions = load_ai_prereview()
    assert len(suggestions) == 98
    for record in isolated_store.records:
        if record["labelReviewStatus"] == "PENDING_HUMAN_REVIEW":
            assert record["goldRecordId"] in suggestions


def test_human_reviewed_untouched(isolated_store):
    by_id = {record["goldRecordId"]: record for record in isolated_store.records}
    assert by_id["gold_pilot_001"]["labelReviewStatus"] == "HUMAN_REVIEWED"
    assert by_id["gold_pilot_002"]["labelReviewStatus"] == "HUMAN_REVIEWED"
    assert "gold_pilot_001" not in load_ai_prereview()


def test_no_false_human_review_after_batch(isolated_store, ai_prereview_file):
    for record in isolated_store.records:
        gold_id = record["goldRecordId"]
        if gold_id in {"gold_pilot_001", "gold_pilot_002"}:
            assert record["labelReviewStatus"] == "HUMAN_REVIEWED"
        elif gold_id in load_ai_prereview():
            assert record["labelReviewStatus"] == "PENDING_HUMAN_REVIEW"


def test_canonical_enums(ai_prereview_file):
    for entry in load_ai_prereview().values():
        assert entry["aiSuggestedSentiment"] in SENTIMENTS
        assert entry["aiSuggestedIssueType"] in ISSUE_TYPES
        assert entry["aiSuggestedSafetyRelatedObservation"] in {"TRUE", "FALSE"}
        assert entry["aiSuggestedQualitySignal"] in QUALITY_SIGNALS


def test_usability_correction_over_age_mismatch():
    record = {
        "labelReviewStatus": "PENDING_HUMAN_REVIEW",
        "review": {
            "text": (
                "This umbrella is well-made, but it was a pain to have to readjust it "
                "every time I turned a corner with the stroller."
            ),
            "rating": 3.0,
        },
        "rulePrelabels": {
            "sentiment": "NEUTRAL",
            "issueType": "AGE_SIZE_MISMATCH",
            "safetyRelatedObservation": "FALSE",
            "qualitySignal": "NEGATIVE",
        },
    }
    suggestion = analyze_record(record)
    assert suggestion is not None
    assert suggestion.issue_type == "USABILITY"


def test_positive_safe_wording_not_safety_true():
    record = {
        "labelReviewStatus": "PENDING_HUMAN_REVIEW",
        "review": {
            "text": "Good head support and feels safe for my newborn.",
            "rating": 5.0,
        },
        "rulePrelabels": {
            "sentiment": "POSITIVE",
            "issueType": "OTHER",
            "safetyRelatedObservation": "FALSE",
            "qualitySignal": "POSITIVE",
        },
    }
    suggestion = analyze_record(record)
    assert suggestion.safety_related_observation == "FALSE"


def test_approve_ai_marks_human_reviewed(tmp_path, monkeypatch):
    from review_store import BASE_CSV, BASE_JSONL, REVIEWED_CSV, REVIEWED_JSONL, AI_PREREVIEW_JSONL

    monkeypatch.setattr("review_store.BASE_JSONL", BASE_JSONL)
    monkeypatch.setattr("review_store.BASE_CSV", BASE_CSV)
    monkeypatch.setattr("review_store.REVIEWED_CSV", tmp_path / "reviewed.csv")
    monkeypatch.setattr("review_store.REVIEWED_JSONL", tmp_path / "reviewed.jsonl")
    monkeypatch.setattr("review_store.AI_PREREVIEW_JSONL", AI_PREREVIEW_JSONL)

    store = ReviewStore(
        reviewed_csv_path=tmp_path / "reviewed.csv",
        reviewed_jsonl_path=tmp_path / "reviewed.jsonl",
    )
    index = store.first_pending_index()
    store.apply_action(index, "approve_ai")
    assert store.records[index]["labelReviewStatus"] == "AI_ASSISTED_FINALIZED"
    assert store.records[index]["labelProvenance"] == "AI_ASSISTED"


def test_filter_counts_present(isolated_store, ai_prereview_file):
    counts = isolated_store.filter_counts()
    assert counts["ALL_PENDING"] == 98
    assert counts["OTHER_CHILD_PRODUCT"] > 0


def test_completed_review_dataset_integrity():
    from gold_pilot_common import (
        FINALIZED_STATUSES,
        GENUINE_HUMAN_GOLD_IDS,
        finalized_records,
        read_pilot_jsonl,
        validate_pilot_records,
    )

    reviewed_path = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected_reviewed.jsonl"
    records = read_pilot_jsonl(reviewed_path)
    assert len(records) == 100
    assert len(validate_pilot_records(records)) == 0
    assert len(finalized_records(records)) == 100
    human = [
        record
        for record in records
        if record.get("goldRecordId") in GENUINE_HUMAN_GOLD_IDS
    ]
    assert len(human) == 2
    assert all(record["labelReviewStatus"] == "HUMAN_REVIEWED" for record in human)
    assert all(record["labelProvenance"] == "HUMAN" for record in human)
    ai_finalized = [
        record
        for record in records
        if record.get("labelReviewStatus") == "AI_ASSISTED_FINALIZED"
    ]
    assert len(ai_finalized) == 98
    assert all(record["labelProvenance"] == "AI_ASSISTED" for record in ai_finalized)
    by_id = {record["goldRecordId"]: record for record in records}
    assert by_id["gold_pilot_001"]["manualLabels"]["issueType"] == "USABILITY"
    assert by_id["gold_pilot_002"]["manualLabels"]["issueType"] == "USABILITY"
    assert by_id["gold_pilot_007"]["manualLabels"]["issueType"] == "DURABILITY"
    assert all(status in FINALIZED_STATUSES for status in (r["labelReviewStatus"] for r in records))
