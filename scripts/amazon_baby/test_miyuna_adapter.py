"""Tests for Miyuna Amazon Baby adapter (offline)."""

from __future__ import annotations

import json
import zipfile
from pathlib import Path

import pandas as pd
import pytest

from miyuna_adapter import (
    AMAZON_BABY_LEGACY_SCHEMA,
    AMAZON_BABY_REAL_SCHEMA,
    TRAINING_RIGHTS_APPROVED,
    adapt_dataframe,
    build_canonical_record,
    dataset_local_product_id,
    dataset_local_review_id,
    derive_issue_type,
    derive_safety_observation,
    derive_sentiment,
    detect_language,
    detect_schema,
    normalize_product_key,
    redact_pii,
    resolve_upload_path,
    validate_import_jsonl,
    validate_schema,
    validate_training_governance,
)


REAL_COLUMNS = [
    "reviewerID",
    "asin",
    "reviewerName",
    "helpful",
    "helpful_num",
    "helpful_den",
    "reviewText",
    "overall",
    "summary",
    "unixReviewTime",
    "reviewTime",
]


def make_real_row(**overrides):
    base = {
        "reviewerID": "R001",
        "asin": "B000123456",
        "reviewerName": "Jane Doe",
        "helpful": "[1,2]",
        "helpful_num": 1,
        "helpful_den": 2,
        "reviewText": "This baby product is soft and durable enough for daily use.",
        "overall": 4,
        "summary": "Soft and durable",
        "unixReviewTime": 1363392000,
        "reviewTime": "03 16, 2013",
    }
    base.update(overrides)
    return base


def test_real_schema_detection():
    df = pd.DataFrame([make_real_row()])
    schema = detect_schema(df)
    assert schema.kind == AMAZON_BABY_REAL_SCHEMA
    assert schema.asin_col == "asin"
    assert schema.review_text_col == "reviewText"
    assert schema.overall_col == "overall"


def test_legacy_schema_detection():
    df = pd.DataFrame({"name": ["Toy"], "review": ["Good product for babies"], "rating": [5]})
    schema = detect_schema(df)
    assert schema.kind == AMAZON_BABY_LEGACY_SCHEMA


def test_validate_schema_failure():
    df = pd.DataFrame({"title_only": ["Toy"]})
    with pytest.raises(ValueError):
        validate_schema(df)


def test_real_schema_adaptation():
    df = pd.DataFrame([make_real_row()])
    accepted, _, report = adapt_dataframe(df)
    assert report.schema_kind == AMAZON_BABY_REAL_SCHEMA
    assert len(accepted) == 1
    rec = accepted[0]
    assert rec["sourceProductId"] == "B000123456"
    assert rec["review"]["text"].startswith("This baby product")
    assert rec["review"]["rating"] == 4.0
    assert rec["review"]["summary"] == "Soft and durable"
    assert rec["review"]["date"] == "03 16, 2013"
    assert rec["review"]["helpful"] == {"num": 1, "den": 2}
    assert rec["product"]["name"] is None


def test_reviewer_fields_excluded():
    df = pd.DataFrame([make_real_row()])
    accepted, _, _ = adapt_dataframe(df)
    rec = accepted[0]
    assert "reviewerID" not in rec
    assert "reviewerName" not in rec
    assert "reviewerID" not in json.dumps(rec)


def test_asin_used_as_source_product_id():
    df = pd.DataFrame([make_real_row(asin="B000999999")])
    accepted, _, _ = adapt_dataframe(df)
    assert accepted[0]["sourceProductId"] == "B000999999"


def test_deterministic_review_id_real_schema():
    df = pd.DataFrame([make_real_row()])
    accepted1, _, _ = adapt_dataframe(df)
    accepted2, _, _ = adapt_dataframe(df)
    assert accepted1[0]["sourceReviewId"] == accepted2[0]["sourceReviewId"]
    assert accepted1[0]["sourceReviewId"].startswith("amazonbaby_review_")


def test_legacy_schema_still_supported():
    df = pd.DataFrame({"name": ["Toy"], "review": ["Great product for babies"], "rating": [5]})
    accepted, _, report = adapt_dataframe(df)
    assert report.schema_kind == AMAZON_BABY_LEGACY_SCHEMA
    assert accepted[0]["product"]["name"] == "Toy"
    assert accepted[0]["sourceProductId"].startswith("amazonbaby_product_")


def test_stable_product_id_legacy():
    key = normalize_product_key("Planetwise Flannel Wipes")
    assert dataset_local_product_id(key) == dataset_local_product_id(key)
    assert dataset_local_product_id(key).startswith("amazonbaby_product_")


def test_stable_review_id():
    pid = dataset_local_product_id("planetwise flannel wipes")
    assert dataset_local_review_id(pid, 4.0, "Soft wipe") == dataset_local_review_id(pid, 4.0, "Soft wipe")
    assert dataset_local_review_id(pid, 4.0, "Soft wipe").startswith("amazonbaby_review_")


def test_sentiment_from_rating():
    assert derive_sentiment(1.0)[0] == "NEGATIVE"
    assert derive_sentiment(3.0)[0] == "NEUTRAL"
    assert derive_sentiment(5.0)[0] == "POSITIVE"


def test_issue_rule_labeling():
    issue, method, conf = derive_issue_type("It broke during first use")
    assert issue == "BREAKAGE"
    assert method == "RULE_BASED"
    assert conf in {"LOW", "MEDIUM", "HIGH"}


def test_safety_observation():
    assert derive_safety_observation("choking hazard with small parts") is True
    assert derive_safety_observation("soft blanket") is False


def test_pii_redaction():
    text, hit = redact_pii("email me at user@example.com")
    assert hit is True
    assert "user@example.com" not in text


def test_language_tagging():
    assert detect_language("Great baby product") == "EN"
    assert detect_language("çok güzel ürün") == "TR"


def test_governance_defaults():
    rec = build_canonical_record(
        review_text="Great product for babies",
        rating=5.0,
        source_product_id="B000123456",
        source_review_id="amazonbaby_review_y",
    )
    gov = rec["governance"]
    assert gov["provenanceStatus"] == "PARTIAL"
    assert gov["licenseStatus"] == "UNKNOWN"
    assert gov["usageRightsStatus"] == "UNKNOWN"
    assert gov["datasetEligibility"] == "QUARANTINED"
    assert gov["trainingAllowed"] is False


def test_deduplication():
    df = pd.DataFrame(
        {
            "name": ["Toy", "Toy"],
            "review": ["Great product for babies", "Great product for babies"],
            "rating": [5, 5],
        }
    )
    accepted, _, report = adapt_dataframe(df)
    assert len(accepted) == 1
    assert report.duplicate_rows == 1


def test_training_gate_default_false():
    assert TRAINING_RIGHTS_APPROVED is False


def test_training_governance_validation_blocks_by_default():
    ok, errors = validate_training_governance({})
    assert ok is False
    assert len(errors) > 0


def test_backend_import_output_allowed():
    df = pd.DataFrame({"name": ["Toy"], "review": ["Great product for babies"], "rating": [5]})
    accepted, _, report = adapt_dataframe(df)
    assert len(accepted) == 1
    assert report.accepted_rows == 1
    assert accepted[0]["recordType"] == "MARKETPLACE_REVIEW"


def test_validate_import_jsonl(tmp_path):
    rec = build_canonical_record(
        review_text="Great product for babies",
        rating=5.0,
        source_product_id="B000123456",
        source_review_id="amazonbaby_review_y",
    )
    path = tmp_path / "import.jsonl"
    path.write_text(json.dumps(rec) + "\n", encoding="utf-8")
    ok, errors, count = validate_import_jsonl(path)
    assert ok is True
    assert count == 1
    assert errors == []


def test_validate_import_jsonl_rejects_training(tmp_path):
    rec = build_canonical_record(
        review_text="Great product for babies",
        rating=5.0,
        source_product_id="B000123456",
        source_review_id="amazonbaby_review_y",
    )
    rec["governance"]["datasetEligibility"] = "TRAINING_APPROVED"
    path = tmp_path / "import.jsonl"
    path.write_text(json.dumps(rec) + "\n", encoding="utf-8")
    ok, errors, _ = validate_import_jsonl(path)
    assert ok is False
    assert any("TRAINING_APPROVED" in err for err in errors)


REAL_CSV_HEADER = (
    "reviewerID,asin,reviewerName,helpful,helpful_num,helpful_den,"
    "reviewText,overall,summary,unixReviewTime,reviewTime\n"
)
REAL_CSV_ROW = (
    'R1,B0001,Jane,"[1,1]",1,1,'
    '"This baby product is soft and durable for daily use.",5,Good,1363392000,"03 16, 2013"\n'
)


def test_zip_support_extracts_csv(tmp_path):
    csv_path = tmp_path / "reviews_Baby_5_final_dataset.csv"
    csv_path.write_text(REAL_CSV_HEADER + REAL_CSV_ROW, encoding="utf-8")
    zip_path = tmp_path / "dataset.zip"
    with zipfile.ZipFile(zip_path, "w") as zf:
        zf.write(csv_path, arcname="reviews_Baby_5_final_dataset.csv")

    resolved = resolve_upload_path(zip_path)
    assert resolved.suffix == ".csv"
    df = pd.read_csv(resolved)
    accepted, _, report = adapt_dataframe(df)
    assert report.schema_kind == AMAZON_BABY_REAL_SCHEMA
    assert len(accepted) == 1


def test_zip_not_passed_to_pandas_directly(tmp_path):
    zip_path = tmp_path / "dataset.zip"
    with zipfile.ZipFile(zip_path, "w") as zf:
        zf.writestr("reviews_Baby_5_final_dataset.csv", REAL_CSV_HEADER + REAL_CSV_ROW)

    resolved = resolve_upload_path(zip_path)
    assert resolved.suffix.lower() == ".csv"
    assert resolved.name.endswith(".csv")
    df = pd.read_csv(resolved)
    assert len(df) == 1
