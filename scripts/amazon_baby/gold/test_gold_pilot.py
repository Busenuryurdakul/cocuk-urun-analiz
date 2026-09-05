"""Tests for Miyuna Gold Dataset Pilot V1."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from gold_pilot_common import (
    APPROVED_STATUSES,
    DEFAULT_REVIEW_STATUS,
    EXPECTED_ELIGIBILITY,
    EXPECTED_SOURCE,
    PILOT_SIZE,
    SEED,
    approved_records,
    build_gold_record,
    build_pilot_records,
    distribution,
    load_source_records,
    rating_distribution,
    select_pilot_records,
    validate_pilot_records,
    write_pilot_jsonl,
)


@pytest.fixture(scope="module")
def source_records():
    return load_source_records()


@pytest.fixture(scope="module")
def pilot_records(source_records):
    return build_pilot_records(source_records, size=PILOT_SIZE, seed=SEED)


def test_source_artifact_non_empty(source_records):
    assert len(source_records) > 0


def test_source_governance(source_records):
    sample = source_records[0]
    assert sample["source"] == EXPECTED_SOURCE
    assert sample["governance"]["datasetEligibility"] == EXPECTED_ELIGIBILITY


def test_deterministic_sampling(source_records):
    first = select_pilot_records(source_records, size=PILOT_SIZE, seed=SEED)
    second = select_pilot_records(source_records, size=PILOT_SIZE, seed=SEED)
    assert [r["sourceReviewId"] for r in first] == [r["sourceReviewId"] for r in second]


def test_pilot_has_exactly_100_records(pilot_records):
    assert len(pilot_records) == PILOT_SIZE


def test_unique_ids(pilot_records):
    gold_ids = [record["goldRecordId"] for record in pilot_records]
    review_ids = [record["sourceReviewId"] for record in pilot_records]
    assert len(set(gold_ids)) == PILOT_SIZE
    assert len(set(review_ids)) == PILOT_SIZE


def test_valid_enums(pilot_records):
    assert validate_pilot_records(pilot_records) == []


def test_pending_default_status(pilot_records):
    assert all(
        record["labelReviewStatus"] == DEFAULT_REVIEW_STATUS for record in pilot_records
    )


def test_manual_labels_prefilled_from_rule(pilot_records):
    for record in pilot_records:
        assert record["manualLabels"] == record["rulePrelabels"]


def test_source_governance_unchanged(pilot_records):
    for record in pilot_records:
        governance = record["governance"]
        assert governance["provenanceStatus"] == "PARTIAL"
        assert governance["licenseStatus"] == "UNKNOWN"
        assert governance["usageRightsStatus"] == "UNKNOWN"
        assert governance["originalDatasetEligibility"] == EXPECTED_ELIGIBILITY


def test_review_text_not_altered(pilot_records, source_records):
    source_by_id = {record["sourceReviewId"]: record for record in source_records}
    for record in pilot_records:
        original = source_by_id[record["sourceReviewId"]]
        assert record["review"]["text"] == original["review"]["text"]


def test_balanced_sentiment_coverage(pilot_records):
    sentiment_dist = distribution(pilot_records, "sentiment", "rulePrelabels")
    assert sentiment_dist.get("POSITIVE", 0) > 0
    assert sentiment_dist.get("NEUTRAL", 0) > 0
    assert sentiment_dist.get("NEGATIVE", 0) > 0


def test_rating_coverage(pilot_records):
    rating_dist = rating_distribution(pilot_records)
    for rating in ("1", "2", "3", "4", "5"):
        assert rating_dist.get(rating, 0) > 0


def test_pending_excluded_from_approved_export(pilot_records, tmp_path: Path):
    path = tmp_path / "pilot.jsonl"
    write_pilot_jsonl(pilot_records, path)
    records = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    assert approved_records(records) == []


def test_rejected_excluded_from_approved_export(pilot_records):
    records = [dict(record) for record in pilot_records]
    records[0]["labelReviewStatus"] = "REJECTED"
    assert len(approved_records(records)) == 0


def test_human_reviewed_included_in_approved_export(pilot_records):
    records = [dict(record) for record in pilot_records]
    records[0]["labelReviewStatus"] = "HUMAN_REVIEWED"
    records[0]["labelProvenance"] = "HUMAN"
    approved = approved_records(records)
    assert len(approved) == 1
    assert approved[0]["labelReviewStatus"] == "HUMAN_REVIEWED"


def test_expert_approved_included_in_approved_export(pilot_records):
    records = [dict(record) for record in pilot_records]
    records[1]["labelReviewStatus"] = "EXPERT_APPROVED"
    records[1]["labelProvenance"] = "HUMAN"
    approved = approved_records(records)
    assert len(approved) == 1
    assert approved[0]["labelReviewStatus"] == "EXPERT_APPROVED"


def test_ai_assisted_finalized_not_in_human_approved_export(pilot_records):
    from gold_pilot_common import finalized_records

    records = [dict(record) for record in pilot_records]
    records[0]["labelReviewStatus"] = "AI_ASSISTED_FINALIZED"
    records[0]["labelProvenance"] = "AI_ASSISTED"
    assert approved_records(records) == []
    assert len(finalized_records(records)) == 1


def test_gold_record_id_format(pilot_records):
    assert pilot_records[0]["goldRecordId"] == "gold_pilot_001"
    assert pilot_records[-1]["goldRecordId"] == "gold_pilot_100"


def test_build_gold_record_structure(source_records):
    record = build_gold_record(source_records[0], 1)
    assert record["goldRecordId"] == "gold_pilot_001"
    assert "rulePrelabels" in record
    assert "manualLabels" in record
    assert record["labelReviewStatus"] == DEFAULT_REVIEW_STATUS
