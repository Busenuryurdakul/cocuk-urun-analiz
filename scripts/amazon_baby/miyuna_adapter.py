"""
Miyuna Amazon Baby dataset adapter.

Transforms Kaggle Amazon Baby raw CSV into canonical Miyuna records with
governance defaults (QUARANTINED, rights unresolved).
"""

from __future__ import annotations

import hashlib
import json
import re
import zipfile
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import pandas as pd

SOURCE = "KAGGLE_AMAZON_BABY_ROOPALIK"
SOURCE_URL = "https://www.kaggle.com/datasets/roopalik/amazon-baby-dataset"
PRODUCT_ID_PREFIX = "amazonbaby_product_"
REVIEW_ID_PREFIX = "amazonbaby_review_"
AMAZON_BABY_REAL_SCHEMA = "AMAZON_BABY_REAL_SCHEMA"
AMAZON_BABY_LEGACY_SCHEMA = "AMAZON_BABY_LEGACY_SCHEMA"
MIN_REVIEW_LEN = 10
MAX_REVIEW_LEN = 10000
MAX_REVIEWS_PER_PRODUCT = 100
RANDOM_SEED = 42

TRAINING_RIGHTS_APPROVED = False

EMAIL_PATTERN = re.compile(r"(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}")
PHONE_PATTERN = re.compile(r"(?:\+?\d[\d\s\-()]{7,}\d)")

ISSUE_RULES: list[tuple[str, list[str]]] = [
    ("DURABILITY", ["durability", "durable", "wear out", "fall apart"]),
    ("BREAKAGE", ["broke", "broken", "break", "crack", "broke during"]),
    ("AGE_SIZE_MISMATCH", ["too small", "too big", "age", "size", "fit"]),
    ("MATERIAL", ["material", "fabric", "plastic", "bpa"]),
    ("ODOR", ["odor", "odour", "smell", "stink"]),
    ("PACKAGING", ["packaging", "package", "box"]),
    ("USABILITY", ["hard to use", "difficult", "confusing"]),
    ("QUALITY", ["quality", "cheap", "defect", "flimsy"]),
]

SAFETY_KEYWORDS = [
    "choking", "choke", "sharp edge", "strangulation", "burn", "overheat",
    "detached part", "locking failure", "restraint failure", "unsafe", "hazard",
]


@dataclass
class GovernanceMetadata:
    provenance_status: str = "PARTIAL"
    license_status: str = "UNKNOWN"
    usage_rights_status: str = "UNKNOWN"
    dataset_eligibility: str = "QUARANTINED"
    training_allowed: bool = False
    evaluation_allowed: bool = False


@dataclass
class SchemaSpec:
    kind: str
    asin_col: str | None = None
    review_text_col: str | None = None
    overall_col: str | None = None
    summary_col: str | None = None
    review_time_col: str | None = None
    unix_review_time_col: str | None = None
    helpful_num_col: str | None = None
    helpful_den_col: str | None = None
    reviewer_id_col: str | None = None
    name_col: str | None = None
    review_col: str | None = None
    rating_col: str | None = None


@dataclass
class QualityReport:
    raw_rows: int = 0
    accepted_rows: int = 0
    rejected_rows: int = 0
    duplicate_rows: int = 0
    unique_products: int = 0
    unique_reviews: int = 0
    schema_kind: str = ""
    rating_distribution: dict[str, int] = field(default_factory=dict)
    sentiment_distribution: dict[str, int] = field(default_factory=dict)
    language_distribution: dict[str, int] = field(default_factory=dict)
    issue_type_distribution: dict[str, int] = field(default_factory=dict)
    safety_related_observation_count: int = 0
    pii_redaction_count: int = 0
    provenance_status: str = "PARTIAL"
    license_status: str = "UNKNOWN"
    usage_rights_status: str = "UNKNOWN"
    dataset_eligibility: str = "QUARANTINED"
    training_allowed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return {
            "rawRows": self.raw_rows,
            "acceptedRows": self.accepted_rows,
            "rejectedRows": self.rejected_rows,
            "duplicateRows": self.duplicate_rows,
            "uniqueProducts": self.unique_products,
            "uniqueReviews": self.unique_reviews,
            "schemaKind": self.schema_kind,
            "ratingDistribution": self.rating_distribution,
            "sentimentDistribution": self.sentiment_distribution,
            "languageDistribution": self.language_distribution,
            "issueTypeDistribution": self.issue_type_distribution,
            "safetyRelatedObservationCount": self.safety_related_observation_count,
            "piiRedactionCount": self.pii_redaction_count,
            "provenanceStatus": self.provenance_status,
            "licenseStatus": self.license_status,
            "usageRightsStatus": self.usage_rights_status,
            "datasetEligibility": self.dataset_eligibility,
            "trainingAllowed": self.training_allowed,
        }


def _column_map(df: pd.DataFrame) -> dict[str, str]:
    return {c.lower().strip(): c for c in df.columns}


def detect_schema(df: pd.DataFrame) -> SchemaSpec:
    colmap = _column_map(df)

    if all(k in colmap for k in ("asin", "reviewtext", "overall")):
        return SchemaSpec(
            kind=AMAZON_BABY_REAL_SCHEMA,
            asin_col=colmap["asin"],
            review_text_col=colmap["reviewtext"],
            overall_col=colmap["overall"],
            summary_col=colmap.get("summary"),
            review_time_col=colmap.get("reviewtime"),
            unix_review_time_col=colmap.get("unixreviewtime"),
            helpful_num_col=colmap.get("helpful_num"),
            helpful_den_col=colmap.get("helpful_den"),
            reviewer_id_col=colmap.get("reviewerid"),
        )

    name_col = next((colmap[k] for k in colmap if k in {"name", "title", "product_title", "product name"}), None)
    review_col = next((colmap[k] for k in colmap if k in {"review", "review_text", "text", "review_body"}), None)
    rating_col = next((colmap[k] for k in colmap if k in {"rating", "star_rating", "stars", "score"}), None)
    if name_col and review_col:
        return SchemaSpec(
            kind=AMAZON_BABY_LEGACY_SCHEMA,
            name_col=name_col,
            review_col=review_col,
            rating_col=rating_col,
        )

    raise ValueError(
        "Unsupported Amazon Baby CSV schema. Expected either real schema "
        "(asin, reviewText, overall) or legacy schema (name, review, rating). "
        f"Found columns: {list(df.columns)}"
    )


def validate_schema(df: pd.DataFrame) -> SchemaSpec:
    """Backward-compatible alias returning detected schema specification."""
    return detect_schema(df)


def normalize_product_key(name: str) -> str:
    s = " ".join(name.lower().split())
    return s[:120]


def dataset_local_product_id(normalized_name: str) -> str:
    digest = hashlib.sha256(normalized_name.encode()).hexdigest()[:16]
    return f"{PRODUCT_ID_PREFIX}{digest}"


def dataset_local_review_id(
    product_id: str,
    rating: float | None,
    review_text: str,
    review_time: str | None = None,
) -> str:
    rating_part = "na" if rating is None else f"{rating:.1f}"
    normalized = " ".join(review_text.split())
    time_part = (review_time or "na").strip()
    payload = f"{product_id}|{rating_part}|{time_part}|{normalized}"
    digest = hashlib.sha256(payload.encode()).hexdigest()[:16]
    return f"{REVIEW_ID_PREFIX}{digest}"


def review_fingerprint(
    source_product_id: str,
    source_review_id: str,
    review_text: str,
    reviewer_id: str | None = None,
) -> str:
    parts = [source_product_id, source_review_id, " ".join(review_text.split())]
    if reviewer_id:
        rid_hash = hashlib.sha256(str(reviewer_id).strip().encode()).hexdigest()[:8]
        parts.append(rid_hash)
    return hashlib.sha256("|".join(parts).encode()).hexdigest()


def redact_pii(text: str) -> tuple[str, bool]:
    redacted = EMAIL_PATTERN.sub("[REDACTED_EMAIL]", text)
    redacted = PHONE_PATTERN.sub("[REDACTED_PHONE]", redacted)
    return redacted, redacted != text


def detect_language(text: str) -> str:
    turkish_chars = set("çğıöşüÇĞİÖŞÜ")
    if any(c in turkish_chars for c in text):
        return "TR"
    if re.search(r"[A-Za-z]", text):
        return "EN"
    return "UNKNOWN"


def derive_sentiment(rating: float | None) -> tuple[str, str]:
    if rating is None:
        return "NEUTRAL", "RATING_DERIVED"
    if rating <= 2:
        return "NEGATIVE", "RATING_DERIVED"
    if rating == 3:
        return "NEUTRAL", "RATING_DERIVED"
    return "POSITIVE", "RATING_DERIVED"


def derive_issue_type(text: str) -> tuple[str, str, str]:
    lower = text.lower()
    for issue, words in ISSUE_RULES:
        for word in words:
            if word in lower:
                conf = "HIGH" if len(word) > 8 else "MEDIUM"
                return issue, "RULE_BASED", conf
    return "OTHER", "RULE_BASED", "LOW"


def derive_safety_observation(text: str) -> bool | None:
    lower = text.lower()
    if any(word in lower for word in SAFETY_KEYWORDS):
        return True
    return False


def derive_quality_signal(sentiment: str, issue_type: str, rating: float | None) -> str:
    has_issue = issue_type != "OTHER"
    if sentiment == "POSITIVE" and not has_issue:
        return "POSITIVE"
    if sentiment == "NEGATIVE" or has_issue:
        if sentiment == "POSITIVE" and has_issue:
            return "MIXED"
        return "NEGATIVE"
    if rating is None:
        return "UNKNOWN"
    return "MIXED"


def _clean_str(value: Any) -> str:
    if value is None or (isinstance(value, float) and pd.isna(value)):
        return ""
    return str(value).strip()


def _optional_int(value: Any) -> int | None:
    if value is None or (isinstance(value, float) and pd.isna(value)):
        return None
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _optional_rating(value: Any) -> float | None:
    if value is None or (isinstance(value, float) and pd.isna(value)):
        return None
    try:
        rating = float(value)
    except (TypeError, ValueError):
        return None
    if rating < 1 or rating > 5:
        return None
    return rating


def validate_training_governance(meta: dict[str, Any]) -> tuple[bool, list[str]]:
    required = {
        "licenseStatus": "APPROVED",
        "usageRightsStatus": "APPROVED",
        "provenanceStatus": "VERIFIED",
        "piiPolicy": "PASS",
        "qualityPolicy": "PASS",
        "datasetApproval": "RECORDED",
    }
    errors = []
    for key, expected in required.items():
        if meta.get(key) != expected:
            errors.append(f"{key} must be {expected}")
    return len(errors) == 0, errors


def build_canonical_record(
    review_text: str,
    rating: float | None,
    source_product_id: str,
    source_review_id: str,
    product_name: str | None = None,
    summary: str | None = None,
    review_date: str | None = None,
    helpful_num: int | None = None,
    helpful_den: int | None = None,
) -> dict[str, Any]:
    redacted, pii_hit = redact_pii(review_text)
    if len(redacted) > MAX_REVIEW_LEN:
        redacted = redacted[:MAX_REVIEW_LEN]
    language = detect_language(redacted)
    sentiment, label_source = derive_sentiment(rating)
    issue_type, label_method, label_conf = derive_issue_type(redacted)
    safety = derive_safety_observation(redacted)
    quality = derive_quality_signal(sentiment, issue_type, rating)
    governance = GovernanceMetadata()

    review_obj: dict[str, Any] = {"rating": rating, "text": redacted, "language": language}
    if summary:
        review_obj["summary"] = summary
    if review_date:
        review_obj["date"] = review_date
    if helpful_num is not None or helpful_den is not None:
        review_obj["helpful"] = {"num": helpful_num, "den": helpful_den}

    return {
        "recordType": "MARKETPLACE_REVIEW",
        "source": SOURCE,
        "sourceType": "MARKETPLACE_REVIEW",
        "sourceProductId": source_product_id,
        "sourceReviewId": source_review_id,
        "product": {"name": product_name, "brand": None, "category": None},
        "review": review_obj,
        "signals": {
            "sentiment": sentiment,
            "sentimentLabelSource": label_source,
            "issueType": issue_type,
            "issueLabelMethod": label_method,
            "issueLabelConfidence": label_conf,
            "safetyRelatedObservation": safety,
            "qualitySignal": quality,
            "shortSummary": None,
            "labelReviewStatus": "RULE_PRELABELED",
        },
        "governance": {
            "provenanceStatus": governance.provenance_status,
            "licenseStatus": governance.license_status,
            "usageRightsStatus": governance.usage_rights_status,
            "datasetEligibility": governance.dataset_eligibility,
            "trainingAllowed": governance.training_allowed,
            "evaluationAllowed": governance.evaluation_allowed,
        },
        "_meta": {"piiRedacted": pii_hit},
    }


def _iter_real_rows(df: pd.DataFrame, schema: SchemaSpec) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for _, row in df.iterrows():
        asin = _clean_str(row[schema.asin_col])
        text = _clean_str(row[schema.review_text_col])
        rating = _optional_rating(row[schema.overall_col])
        summary = _clean_str(row[schema.summary_col]) if schema.summary_col else ""
        review_time = _clean_str(row[schema.review_time_col]) if schema.review_time_col else ""
        if not review_time and schema.unix_review_time_col:
            unix_val = row[schema.unix_review_time_col]
            if not (isinstance(unix_val, float) and pd.isna(unix_val)):
                review_time = str(unix_val).strip()
        helpful_num = _optional_int(row[schema.helpful_num_col]) if schema.helpful_num_col else None
        helpful_den = _optional_int(row[schema.helpful_den_col]) if schema.helpful_den_col else None
        reviewer_id = _clean_str(row[schema.reviewer_id_col]) if schema.reviewer_id_col else None
        rows.append(
            {
                "asin": asin,
                "text": text,
                "rating": rating,
                "summary": summary or None,
                "review_time": review_time or None,
                "helpful_num": helpful_num,
                "helpful_den": helpful_den,
                "reviewer_id": reviewer_id or None,
            }
        )
    return rows


def _iter_legacy_rows(df: pd.DataFrame, schema: SchemaSpec) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for _, row in df.iterrows():
        name = _clean_str(row[schema.name_col])
        text = _clean_str(row[schema.review_col])
        rating = _optional_rating(row[schema.rating_col]) if schema.rating_col else None
        rows.append(
            {
                "name": name,
                "text": text,
                "rating": rating,
            }
        )
    return rows


def adapt_dataframe(df: pd.DataFrame) -> tuple[list[dict[str, Any]], list[dict[str, Any]], QualityReport]:
    schema = detect_schema(df)
    report = QualityReport()
    report.raw_rows = len(df)
    report.schema_kind = schema.kind

    accepted: list[dict[str, Any]] = []
    rejected: list[dict[str, Any]] = []
    seen_review_ids: set[str] = set()
    seen_fingerprints: set[str] = set()
    product_review_counts: dict[str, int] = {}

    if schema.kind == AMAZON_BABY_REAL_SCHEMA:
        source_rows = _iter_real_rows(df, schema)
    else:
        source_rows = _iter_legacy_rows(df, schema)

    for row in source_rows:
        if schema.kind == AMAZON_BABY_REAL_SCHEMA:
            asin = row["asin"]
            text = row["text"]
            rating = row["rating"]
            if not asin or not text:
                report.rejected_rows += 1
                rejected.append({"reason": "missing_asin_or_review", "asin": asin})
                continue
            if len(text) < MIN_REVIEW_LEN:
                report.rejected_rows += 1
                rejected.append({"reason": "review_too_short", "asin": asin})
                continue
            if rating is None:
                report.rejected_rows += 1
                rejected.append({"reason": "invalid_rating", "asin": asin})
                continue

            source_product_id = asin
            source_review_id = dataset_local_review_id(source_product_id, rating, text, row["review_time"])
            reviewer_id = row["reviewer_id"]
            product_name = None
            summary = row["summary"]
            review_date = row["review_time"]
            helpful_num = row["helpful_num"]
            helpful_den = row["helpful_den"]
        else:
            name = row["name"]
            text = row["text"]
            rating = row["rating"]
            if not name or not text:
                report.rejected_rows += 1
                rejected.append({"reason": "missing_name_or_review", "name": name})
                continue
            if len(text) < MIN_REVIEW_LEN:
                report.rejected_rows += 1
                rejected.append({"reason": "review_too_short", "name": name})
                continue
            if rating is not None and (rating < 1 or rating > 5):
                report.rejected_rows += 1
                rejected.append({"reason": "invalid_rating", "name": name, "rating": rating})
                continue

            product_key = normalize_product_key(name)
            source_product_id = dataset_local_product_id(product_key)
            source_review_id = dataset_local_review_id(source_product_id, rating, text)
            reviewer_id = None
            product_name = name
            summary = None
            review_date = None
            helpful_num = None
            helpful_den = None

        if source_review_id in seen_review_ids:
            report.duplicate_rows += 1
            continue
        fp = review_fingerprint(source_product_id, source_review_id, text, reviewer_id)
        if fp in seen_fingerprints:
            report.duplicate_rows += 1
            continue

        count = product_review_counts.get(source_product_id, 0)
        if count >= MAX_REVIEWS_PER_PRODUCT:
            continue

        record = build_canonical_record(
            review_text=text,
            rating=rating,
            source_product_id=source_product_id,
            source_review_id=source_review_id,
            product_name=product_name,
            summary=summary,
            review_date=review_date,
            helpful_num=helpful_num,
            helpful_den=helpful_den,
        )
        if record["_meta"]["piiRedacted"]:
            report.pii_redaction_count += 1
        del record["_meta"]

        seen_review_ids.add(source_review_id)
        seen_fingerprints.add(fp)
        product_review_counts[source_product_id] = count + 1
        accepted.append(record)
        report.accepted_rows += 1

        if rating is not None:
            key = str(int(rating))
            report.rating_distribution[key] = report.rating_distribution.get(key, 0) + 1
        sentiment = record["signals"]["sentiment"]
        report.sentiment_distribution[sentiment] = report.sentiment_distribution.get(sentiment, 0) + 1
        lang = record["review"]["language"]
        report.language_distribution[lang] = report.language_distribution.get(lang, 0) + 1
        issue = record["signals"]["issueType"]
        report.issue_type_distribution[issue] = report.issue_type_distribution.get(issue, 0) + 1
        if record["signals"]["safetyRelatedObservation"] is True:
            report.safety_related_observation_count += 1

    report.unique_products = len(product_review_counts)
    report.unique_reviews = len(accepted)
    return accepted, rejected, report


def resolve_upload_path(uploaded_path: Path, extract_dir: Path | None = None) -> Path:
    """Resolve manual upload to a CSV path. ZIP archives are extracted first."""
    suffix = uploaded_path.suffix.lower()
    if suffix == ".zip":
        target_dir = extract_dir or uploaded_path.parent / f"{uploaded_path.stem}_extracted"
        target_dir.mkdir(parents=True, exist_ok=True)
        with zipfile.ZipFile(uploaded_path) as zf:
            zf.extractall(target_dir)
        csv_files = sorted(target_dir.rglob("*.csv"))
        if not csv_files:
            raise FileNotFoundError(f"No CSV file found inside ZIP: {uploaded_path}")
        return csv_files[0]
    if suffix == ".csv":
        return uploaded_path
    raise ValueError(f"Unsupported upload type '{suffix}'. Upload .csv or .zip only.")


def write_outputs(
    out_dir: Path,
    accepted: list[dict[str, Any]],
    rejected: list[dict[str, Any]],
    report: QualityReport,
) -> dict[str, Path]:
    out_dir.mkdir(parents=True, exist_ok=True)
    import_path = out_dir / "miyuna_amazon_baby_import.jsonl"
    quarantined_path = out_dir / "miyuna_amazon_baby_quarantined.jsonl"
    report_path = out_dir / "dataset_quality_report.json"
    rejected_path = out_dir / "rejected_records.jsonl"

    with import_path.open("w", encoding="utf-8") as f:
        for rec in accepted:
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")
    with quarantined_path.open("w", encoding="utf-8") as f:
        for rec in accepted:
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")
    report_path.write_text(json.dumps(report.to_dict(), ensure_ascii=False, indent=2), encoding="utf-8")
    with rejected_path.open("w", encoding="utf-8") as f:
        for rec in rejected:
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")

    return {
        "import": import_path,
        "quarantined": quarantined_path,
        "report": report_path,
        "rejected": rejected_path,
    }


def find_kaggle_csv(
    search_dirs: list[Path] | None = None,
    manual_path: Path | None = None,
) -> Path:
    """Locate Amazon Baby CSV in Kaggle, Colab, or local dev paths."""
    if manual_path is not None:
        resolved = resolve_upload_path(manual_path) if manual_path.suffix.lower() in {".zip", ".csv"} else manual_path
        if resolved.exists():
            return resolved

    default_dirs = [
        Path("/content/amazon_baby"),
        Path("/kaggle/input"),
        Path("data"),
        Path("scripts/amazon_baby/output"),
    ]
    preferred_names = ("reviews_baby_5_final_dataset.csv",)
    for base in search_dirs or default_dirs:
        if not base.exists():
            continue
        for preferred in preferred_names:
            matches = list(base.rglob(preferred))
            if matches:
                return matches[0]
        matches = sorted(base.rglob("*.csv"))
        if matches:
            return matches[0]
    raise FileNotFoundError(
        "Amazon Baby CSV not found. Download via Kaggle API to /content/amazon_baby "
        "or upload a CSV/ZIP manually."
    )


def validate_import_jsonl(path: Path) -> tuple[bool, list[str], int]:
    """Validate canonical Miyuna import JSONL governance defaults."""
    errors: list[str] = []
    if not path.exists():
        return False, [f"missing output: {path}"], 0

    lines = [line for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]
    if not lines:
        return False, ["output JSONL is empty"], 0

    training_approved = 0
    eval_only = 0
    quarantined = 0

    for idx, line in enumerate(lines, start=1):
        try:
            rec = json.loads(line)
        except json.JSONDecodeError as exc:
            errors.append(f"line {idx}: invalid JSON ({exc})")
            continue

        if rec.get("source") != SOURCE:
            errors.append(f"line {idx}: source must be {SOURCE}")
        if "reviewerID" in rec or "reviewerName" in rec:
            errors.append(f"line {idx}: reviewer identity must not be persisted")
        gov = rec.get("governance") or {}
        checks = {
            "provenanceStatus": "PARTIAL",
            "licenseStatus": "UNKNOWN",
            "usageRightsStatus": "UNKNOWN",
            "datasetEligibility": "QUARANTINED",
        }
        for key, expected in checks.items():
            if gov.get(key) != expected:
                errors.append(f"line {idx}: governance.{key} expected {expected}, got {gov.get(key)}")
        eligibility = gov.get("datasetEligibility")
        if eligibility == "TRAINING_APPROVED":
            training_approved += 1
        elif eligibility == "EVAL_ONLY":
            eval_only += 1
        elif eligibility == "QUARANTINED":
            quarantined += 1

    if training_approved > 0:
        errors.append(f"TRAINING_APPROVED records found: {training_approved}")
    if eval_only > 0:
        errors.append(f"EVAL_ONLY records found: {eval_only}")
    if quarantined == 0:
        errors.append("no QUARANTINED records found")

    return len(errors) == 0, errors, len(lines)


def print_colab_summary(report: QualityReport) -> None:
    """Print concise Colab-friendly quality summary."""
    print("=== MIYUNA DATASET QUALITY SUMMARY ===")
    print(f"SCHEMA_KIND: {report.schema_kind}")
    print(f"RAW_ROWS: {report.raw_rows}")
    print(f"ACCEPTED_ROWS: {report.accepted_rows}")
    print(f"REJECTED_ROWS: {report.rejected_rows}")
    print(f"DUPLICATES_REMOVED: {report.duplicate_rows}")
    print(f"UNIQUE_PRODUCTS: {report.unique_products}")
    print(f"UNIQUE_REVIEWS: {report.unique_reviews}")
    print(f"RATING_DISTRIBUTION: {report.rating_distribution}")
    print(f"SENTIMENT_DISTRIBUTION: {report.sentiment_distribution}")
    print(f"ISSUE_TYPE_DISTRIBUTION: {report.issue_type_distribution}")
    print(f"PII_REDACTIONS: {report.pii_redaction_count}")
    print(f"QUARANTINED_RECORDS: {report.accepted_rows}")
    print("TRAINING_APPROVED_RECORDS: 0")
    print("EVAL_ONLY_RECORDS: 0")
    if not TRAINING_RIGHTS_APPROVED:
        print("TRAINING BLOCKED: license / usage rights not approved.")


def main(out_dir: Path | None = None) -> None:
    out = out_dir or Path("scripts/amazon_baby/output")
    csv_path = find_kaggle_csv()
    print(f"Using CSV: {csv_path}")
    df = pd.read_csv(csv_path)
    accepted, rejected, report = adapt_dataframe(df)
    paths = write_outputs(out, accepted, rejected, report)
    print(f"Accepted: {report.accepted_rows}, Rejected: {report.rejected_rows}, Duplicates: {report.duplicate_rows}")
    print(f"Wrote import JSONL: {paths['import']}")
    print(f"Wrote quality report: {paths['report']}")

    if not TRAINING_RIGHTS_APPROVED:
        print("TRAINING BLOCKED: license / usage rights not approved.")
        return

    ok, errors = validate_training_governance({})
    if not ok:
        print("TRAINING BLOCKED: governance validation failed:")
        for err in errors:
            print(f"  - {err}")


if __name__ == "__main__":
    main()
