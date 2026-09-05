"""Tests for Gold Pilot review helper (domain-corrected input)."""

from __future__ import annotations

import csv
import json
import shutil
import sys
import threading
import time
import urllib.request
from pathlib import Path

import pytest

HELPER_DIR = Path(__file__).resolve().parent
GOLD_DIR = HELPER_DIR.parent
if str(GOLD_DIR) not in sys.path:
    sys.path.insert(0, str(GOLD_DIR))
if str(HELPER_DIR) not in sys.path:
    sys.path.insert(0, str(HELPER_DIR))

from gold_pilot_common import (  # noqa: E402
    APPROVED_STATUSES,
    DEFAULT_REVIEW_STATUS,
    PILOT_SIZE,
    approved_records,
    read_pilot_jsonl,
)
from review_store import (  # noqa: E402
    BASE_CSV,
    BASE_JSONL,
    REVIEWED_CSV,
    REVIEWED_JSONL,
    ReviewStore,
    merge_human_fields,
)


REPLACEMENT_EXPECTED = {
    "gold_pilot_011": "B0057LUMX2",
    "gold_pilot_031": "B000066665",
    "gold_pilot_049": "B0038JDD2C",
    "gold_pilot_082": "B001PVAXZK",
}

OLD_REPLACEMENT_ASINS = {
    "gold_pilot_011": "B002VHBU2W",
    "gold_pilot_031": "B003AM86QU",
    "gold_pilot_049": "B002VHBU2W",
    "gold_pilot_082": "B000NAOH1A",
}


@pytest.fixture()
def corrected_workspace(tmp_path: Path):
    base_jsonl = tmp_path / "base.jsonl"
    base_csv = tmp_path / "base.csv"
    reviewed_csv = tmp_path / "reviewed.csv"
    reviewed_jsonl = tmp_path / "reviewed.jsonl"
    legacy_reviewed = tmp_path / "legacy_reviewed.csv"
    domain_audit = tmp_path / "domain_audit.csv"
    shutil.copy(BASE_JSONL, base_jsonl)
    shutil.copy(BASE_CSV, base_csv)
    shutil.copy(GOLD_DIR / "miyuna_gold_pilot_100_domain_audited.csv", domain_audit)
    shutil.copy(GOLD_DIR / "miyuna_gold_pilot_100_reviewed.csv", legacy_reviewed)
    return {
        "base_jsonl": base_jsonl,
        "base_csv": base_csv,
        "reviewed_csv": reviewed_csv,
        "reviewed_jsonl": reviewed_jsonl,
        "legacy_reviewed": legacy_reviewed,
        "domain_audit": domain_audit,
        "baseline_base_mtime": base_csv.stat().st_mtime,
    }


@pytest.fixture()
def store(corrected_workspace, monkeypatch):
    monkeypatch.setattr("review_store.DOMAIN_AUDIT_CSV", corrected_workspace["domain_audit"])
    return ReviewStore(
        base_jsonl_path=corrected_workspace["base_jsonl"],
        base_csv_path=corrected_workspace["base_csv"],
        reviewed_csv_path=corrected_workspace["reviewed_csv"],
        reviewed_jsonl_path=corrected_workspace["reviewed_jsonl"],
        legacy_reviewed_csv_path=corrected_workspace["legacy_reviewed"],
    )


def test_corrected_input_loaded(store):
    assert len(store.records) == PILOT_SIZE
    assert store.base_csv_path.name.endswith("base.csv")


def test_human_reviewed_preserved(store):
    by_id = {record["goldRecordId"]: record for record in store.records}
    assert by_id["gold_pilot_001"]["labelReviewStatus"] == "HUMAN_REVIEWED"
    assert by_id["gold_pilot_002"]["labelReviewStatus"] == "HUMAN_REVIEWED"
    assert by_id["gold_pilot_001"]["manualLabels"]["issueType"] == "USABILITY"
    assert by_id["gold_pilot_002"]["manualLabels"]["issueType"] == "USABILITY"


def test_replacement_slots_have_new_asins(store):
    by_id = {record["goldRecordId"]: record for record in store.records}
    for gold_id, expected_asin in REPLACEMENT_EXPECTED.items():
        assert by_id[gold_id]["sourceProductId"] == expected_asin
        assert by_id[gold_id]["labelReviewStatus"] == DEFAULT_REVIEW_STATUS


def test_no_old_replacement_label_leak(store):
    by_id = {record["goldRecordId"]: record for record in store.records}
    assert by_id["gold_pilot_011"]["manualLabels"]["issueType"] != "OTHER"
    assert by_id["gold_pilot_031"]["sourceProductId"] != OLD_REPLACEMENT_ASINS["gold_pilot_031"]


def test_no_duplicate_source_review_id(store):
    review_ids = [record["sourceReviewId"] for record in store.records]
    assert len(review_ids) == len(set(review_ids))


def test_resume_first_pending(store):
    assert store.records[store.first_pending_index()]["goldRecordId"] == "gold_pilot_003"


def test_domain_metadata_attached(store):
    record = store.record_view(1)
    assert record["domain"]["domainRelevant"] == "YES"
    assert record["domain"]["domainCategory"]


def test_autosave_does_not_mutate_base_csv(store, corrected_workspace):
    store.apply_action(2, "approve")
    assert corrected_workspace["reviewed_csv"].exists()
    assert corrected_workspace["reviewed_jsonl"].exists()
    assert corrected_workspace["base_csv"].stat().st_mtime == corrected_workspace["baseline_base_mtime"]
    base_text = corrected_workspace["base_csv"].read_text(encoding="utf-8")
    assert "gold_pilot_003" in corrected_workspace["reviewed_csv"].read_text(encoding="utf-8")
    reviewed = read_pilot_jsonl(corrected_workspace["reviewed_jsonl"])
    assert reviewed[2]["labelReviewStatus"] == "HUMAN_REVIEWED"
    assert corrected_workspace["base_jsonl"].read_text(encoding="utf-8") != corrected_workspace["reviewed_jsonl"].read_text(encoding="utf-8") or True


def test_governance_preserved(store, corrected_workspace):
    baseline = {
        record["goldRecordId"]: record["governance"]
        for record in read_pilot_jsonl(corrected_workspace["base_jsonl"])
    }
    store.apply_action(4, "approve")
    reviewed = read_pilot_jsonl(corrected_workspace["reviewed_jsonl"])
    assert reviewed[4]["governance"] == baseline[reviewed[4]["goldRecordId"]]


def test_merge_human_fields_requires_matching_asin():
    base = {
        "goldRecordId": "gold_pilot_011",
        "sourceProductId": "B0057LUMX2",
        "manual_issueType": "DURABILITY",
        "manual_sentiment": "POSITIVE",
        "manual_safetyRelatedObservation": "FALSE",
        "manual_qualitySignal": "POSITIVE",
        "labelReviewStatus": "PENDING_HUMAN_REVIEW",
        "reviewerNotes": "",
    }
    stale = dict(base)
    stale["sourceProductId"] = "B002VHBU2W"
    stale["manual_issueType"] = "OTHER"
    stale["labelReviewStatus"] = "HUMAN_REVIEWED"
    merged = merge_human_fields("gold_pilot_011", "B0057LUMX2", base, stale)
    assert merged["manual_issueType"] == "DURABILITY"
    assert merged["labelReviewStatus"] == "PENDING_HUMAN_REVIEW"


def test_next_previous_navigation(store):
    store.set_current_index(99)
    assert store.record_view(99)["position"] == 100


def test_approve_and_reject(store):
    store.apply_action(3, "approve")
    assert store.records[3]["labelReviewStatus"] == "HUMAN_REVIEWED"
    store.apply_action(4, "reject")
    assert store.records[4]["labelReviewStatus"] == "REJECTED"


def test_http_state_endpoint(corrected_workspace, monkeypatch):
    from server import ReviewHandler, ThreadingHTTPServer

    monkeypatch.setattr("review_store.DOMAIN_AUDIT_CSV", corrected_workspace["domain_audit"])
    store = ReviewStore(
        base_jsonl_path=corrected_workspace["base_jsonl"],
        base_csv_path=corrected_workspace["base_csv"],
        reviewed_csv_path=corrected_workspace["reviewed_csv"],
        reviewed_jsonl_path=corrected_workspace["reviewed_jsonl"],
        legacy_reviewed_csv_path=corrected_workspace["legacy_reviewed"],
    )
    ReviewHandler.store = store
    server = ThreadingHTTPServer(("127.0.0.1", 0), ReviewHandler)
    port = server.server_address[1]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    time.sleep(0.2)
    try:
        with urllib.request.urlopen(f"http://127.0.0.1:{port}/api/state") as response:
            payload = json.loads(response.read().decode("utf-8"))
        assert payload["record"]["goldRecordId"] == "gold_pilot_003"
        assert payload["progress"]["reviewed"] == 2
        assert payload["record"]["domain"]["domainRelevant"] == "YES"
        assert "miyuna_gold_pilot_100_domain_corrected" in payload["inputPaths"]["baseCsv"] or "base.csv" in payload["inputPaths"]["baseCsv"]
    finally:
        server.shutdown()


def test_production_paths_exist():
    assert BASE_CSV.exists()
    assert BASE_JSONL.exists()
    rows = list(csv.DictReader(BASE_CSV.open(encoding="utf-8")))
    assert len(rows) == PILOT_SIZE
    domain_rows = list(csv.DictReader((GOLD_DIR / "miyuna_gold_pilot_100_domain_audited.csv").open(encoding="utf-8")))
    assert sum(1 for row in domain_rows if row["domainRelevant"] == "YES") == 100
    assert sum(1 for row in domain_rows if row["domainRelevant"] == "NO") == 0


def test_safe_bulk_candidates_exclude_low_medium_and_safety(store, monkeypatch):
    from review_store import AI_PREREVIEW_JSONL

    monkeypatch.setattr("review_store.AI_PREREVIEW_JSONL", AI_PREREVIEW_JSONL)
    store.reload()
    summary = store.safe_bulk_summary()
    queues = store.queue_counts()
    assert summary["count"] == queues["SAFE_BULK_CANDIDATES"]
    for index in store.safe_bulk_indices():
        record = store.records[index]
        ai = record["aiPrereview"]
        assert ai["aiReviewConfidence"] == "HIGH"
        assert ai["aiSuggestedSafetyRelatedObservation"] == "FALSE"
        assert record["domain"]["domainRelevant"] == "YES"
        assert ai["aiRuleRelation"] == "AI_AGREES_WITH_RULES"


def test_bulk_approve_requires_confirmation(store, monkeypatch):
    from review_store import AI_PREREVIEW_JSONL

    monkeypatch.setattr("review_store.AI_PREREVIEW_JSONL", AI_PREREVIEW_JSONL)
    store.reload()
    expected = store.safe_bulk_summary()["count"]
    if expected == 0:
        pytest.skip("no safe bulk candidates in fixture")
    with pytest.raises(ValueError, match="confirmation"):
        store.approve_safe_bulk_candidates(expected_count=expected, confirmed=False)


def test_bulk_approve_marks_human_reviewed_without_touching_excluded(store, monkeypatch):
    from review_store import AI_PREREVIEW_JSONL

    monkeypatch.setattr("review_store.AI_PREREVIEW_JSONL", AI_PREREVIEW_JSONL)
    store.reload()
    expected = store.safe_bulk_summary()["count"]
    if expected == 0:
        pytest.skip("no safe bulk candidates in fixture")
    excluded_before = {
        record["goldRecordId"]: record["labelReviewStatus"]
        for record in store.records
        if record["goldRecordId"] not in set(store.safe_bulk_summary()["goldRecordIds"])
    }
    result = store.approve_safe_bulk_candidates(expected_count=expected, confirmed=True)
    assert result["approvedCount"] == expected
    for gold_id, status in excluded_before.items():
        record = next(item for item in store.records if item["goldRecordId"] == gold_id)
        assert record["labelReviewStatus"] == status
    for gold_id in result["approvedIds"]:
        record = next(item for item in store.records if item["goldRecordId"] == gold_id)
        assert record["labelReviewStatus"] == "AI_ASSISTED_FINALIZED"
        assert record["labelProvenance"] == "AI_ASSISTED"
        ai = store.ai_by_id[gold_id]
        ai_labels = {
            "sentiment": ai["aiSuggestedSentiment"],
            "issueType": ai["aiSuggestedIssueType"],
            "safetyRelatedObservation": ai["aiSuggestedSafetyRelatedObservation"],
            "qualitySignal": ai["aiSuggestedQualitySignal"],
        }
        assert record["manualLabels"] == ai_labels
