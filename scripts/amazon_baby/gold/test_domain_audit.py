"""Tests for Gold Pilot domain audit."""

from __future__ import annotations

import csv
import sys
from pathlib import Path

import pytest

GOLD_DIR = Path(__file__).resolve().parent
if str(GOLD_DIR) not in sys.path:
    sys.path.insert(0, str(GOLD_DIR))

from domain_classifier import DOMAIN_CATEGORIES, classify_domain_text  # noqa: E402
from gold_pilot_common import EXPECTED_ELIGIBILITY, PILOT_SIZE, SEED  # noqa: E402
import run_domain_audit as audit_module  # noqa: E402
from run_domain_audit import (  # noqa: E402
    choose_replacement,
    corpus_yes_records,
    merge_pilot_state,
    run_audit,
    write_audit_csv,
)


@pytest.fixture(scope="module")
def audit_result():
    audit_module._CORPUS_YES_CACHE = None
    return run_audit()


def test_classifier_valid_category_enum():
    result = classify_domain_text(
        "Great stroller for my toddler. Easy to fold and push.",
        "Lightweight stroller",
    )
    assert result.domain_relevant == "YES"
    assert result.domain_category in DOMAIN_CATEGORIES


def test_classifier_marks_non_child_as_no():
    result = classify_domain_text(
        "Excellent drill for home renovation projects.",
        "Power tool",
    )
    assert result.domain_relevant == "NO"


def test_classifier_marks_ambiguous_short_text_uncertain():
    result = classify_domain_text("Good quality.", "")
    assert result.domain_relevant == "UNCERTAIN"


def test_audit_loads_100_records():
    records, _ = merge_pilot_state()
    assert len(records) == PILOT_SIZE


def test_audit_exactly_100_after_run(audit_result):
    assert len(audit_result["audit_rows"]) == PILOT_SIZE
    assert len(audit_result["corrected_records"]) == PILOT_SIZE


def test_no_duplicate_source_review_ids_after_audit(audit_result):
    review_ids = [record["sourceReviewId"] for record in audit_result["corrected_records"]]
    assert len(review_ids) == len(set(review_ids))


def test_governance_preserved_after_audit(audit_result):
    for record in audit_result["corrected_records"]:
        governance = record["governance"]
        assert governance["provenanceStatus"] == "PARTIAL"
        assert governance["licenseStatus"] == "UNKNOWN"
        assert governance["usageRightsStatus"] == "UNKNOWN"
        assert governance["originalDatasetEligibility"] == EXPECTED_ELIGIBILITY


def test_human_reviewed_records_preserved(audit_result):
    assert audit_result["human_reviewed_preserved"] is True
    original, _ = merge_pilot_state()
    corrected_by_id = {
        record["goldRecordId"]: record for record in audit_result["corrected_records"]
    }
    for record in original:
        if record["labelReviewStatus"] in {"HUMAN_REVIEWED", "EXPERT_APPROVED", "REJECTED"}:
            corrected = corrected_by_id[record["goldRecordId"]]
            assert corrected["labelReviewStatus"] == record["labelReviewStatus"]
            assert corrected["manualLabels"] == record["manualLabels"]
            assert corrected["review"]["text"] == record["review"]["text"]


def test_replacements_are_domain_relevant(audit_result):
    for row in audit_result["audit_rows"]:
        if row["replacementApplied"] == "YES":
            assert row["domainRelevant"] == "YES"


def test_audit_csv_written(audit_result, tmp_path: Path):
    out = tmp_path / "audited.csv"
    write_audit_csv(audit_result["audit_rows"], out)
    rows = list(csv.DictReader(out.open(encoding="utf-8")))
    assert len(rows) == PILOT_SIZE
    assert "domainRelevant" in rows[0]


def test_corpus_yes_records_exclude_used_ids():
    records, _ = merge_pilot_state()
    used = {record["sourceReviewId"] for record in records}
    corpus = corpus_yes_records(used)
    assert all(record["sourceReviewId"] not in used for record in corpus)
    assert len(corpus) > 0


def test_deterministic_replacement_choice():
    records, _ = merge_pilot_state()
    used = {record["sourceReviewId"] for record in records}
    corpus = corpus_yes_records(used)
    target = records[0]
    first = choose_replacement(corpus, target, {"domainCategory": "STROLLER"}, __import__("random").Random(SEED))
    second = choose_replacement(corpus, target, {"domainCategory": "STROLLER"}, __import__("random").Random(SEED))
    assert first is not None
    assert first["sourceReviewId"] == second["sourceReviewId"]
