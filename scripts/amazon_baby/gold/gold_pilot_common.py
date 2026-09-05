"""Shared logic for Miyuna Gold Dataset Pilot V1."""

from __future__ import annotations

import csv
import json
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

SOURCE_ARTIFACT = Path(__file__).resolve().parent.parent / "output" / "miyuna_amazon_baby_import.jsonl"
GOLD_DIR = Path(__file__).resolve().parent
PILOT_JSONL = GOLD_DIR / "miyuna_gold_pilot_100.jsonl"
PILOT_CSV = GOLD_DIR / "miyuna_gold_pilot_100_review.csv"
APPROVED_JSONL = GOLD_DIR / "miyuna_gold_pilot_approved.jsonl"  # legacy misleading name
FINALIZED_JSONL = GOLD_DIR / "miyuna_gold_pilot_ai_assisted_finalized.jsonl"
FINALIZED_CSV = GOLD_DIR / "miyuna_gold_pilot_ai_assisted_finalized.csv"
REVIEWED_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected_reviewed.jsonl"
REVIEWED_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected_reviewed.csv"
GENUINE_HUMAN_GOLD_IDS = frozenset({"gold_pilot_001", "gold_pilot_002"})

PILOT_SIZE = 100
SEED = 42
EXPECTED_SOURCE = "KAGGLE_AMAZON_BABY_ROOPALIK"
EXPECTED_ELIGIBILITY = "QUARANTINED"

SENTIMENTS = ("POSITIVE", "NEUTRAL", "NEGATIVE")
ISSUE_TYPES = (
    "DURABILITY",
    "BREAKAGE",
    "AGE_SIZE_MISMATCH",
    "MATERIAL",
    "ODOR",
    "PACKAGING",
    "USABILITY",
    "QUALITY",
    "SAFETY_RELATED_OBSERVATION",
    "OTHER",
)
SAFETY_VALUES = ("TRUE", "FALSE", "UNKNOWN")
QUALITY_SIGNALS = ("POSITIVE", "NEGATIVE", "MIXED", "UNKNOWN")
LABEL_REVIEW_STATUSES = (
    "PENDING_HUMAN_REVIEW",
    "HUMAN_REVIEWED",
    "EXPERT_APPROVED",
    "AI_ASSISTED_FINALIZED",
    "REJECTED",
)
LABEL_PROVENANCE_VALUES = ("HUMAN", "AI_ASSISTED", "RULE", "UNKNOWN")
APPROVED_STATUSES = ("HUMAN_REVIEWED", "EXPERT_APPROVED")
FINALIZED_STATUSES = ("HUMAN_REVIEWED", "EXPERT_APPROVED", "AI_ASSISTED_FINALIZED")
DEFAULT_REVIEW_STATUS = "PENDING_HUMAN_REVIEW"
DEFAULT_LABEL_PROVENANCE = "RULE"

CSV_COLUMNS = [
    "goldRecordId",
    "sourceProductId",
    "rating",
    "reviewText",
    "rule_sentiment",
    "manual_sentiment",
    "rule_issueType",
    "manual_issueType",
    "rule_safetyRelatedObservation",
    "manual_safetyRelatedObservation",
    "rule_qualitySignal",
    "manual_qualitySignal",
    "labelReviewStatus",
    "labelProvenance",
    "reviewerNotes",
]


def normalize_safety(value: Any) -> str:
    if value is True:
        return "TRUE"
    if value is False:
        return "FALSE"
    return "UNKNOWN"


def normalize_rating(value: Any) -> int:
    if value is None:
        return 0
    try:
        rating = int(float(value))
    except (TypeError, ValueError):
        return 0
    if 1 <= rating <= 5:
        return rating
    return 0


def load_source_records(path: Path = SOURCE_ARTIFACT) -> list[dict[str, Any]]:
    if not path.exists() or path.stat().st_size == 0:
        raise ValueError("source artifact missing or empty")
    records: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"invalid JSONL at line {line_no}: {exc}") from exc
            records.append(record)
    if not records:
        raise ValueError("source artifact contains no records")
    return records


def validate_source_record(record: dict[str, Any]) -> None:
    if record.get("source") != EXPECTED_SOURCE:
        raise ValueError(f"unexpected source: {record.get('source')}")
    governance = record.get("governance") or {}
    if governance.get("datasetEligibility") != EXPECTED_ELIGIBILITY:
        raise ValueError(
            f"unexpected datasetEligibility: {governance.get('datasetEligibility')}"
        )
    text = (record.get("review") or {}).get("text", "")
    if not str(text).strip():
        raise ValueError("review text is empty")


def strat_key(record: dict[str, Any]) -> tuple[str, str, int, str]:
    signals = record.get("signals") or {}
    sentiment = signals.get("sentiment") or "NEUTRAL"
    issue_type = signals.get("issueType") or "OTHER"
    rating = normalize_rating((record.get("review") or {}).get("rating"))
    safety = normalize_safety(signals.get("safetyRelatedObservation"))
    return sentiment, issue_type, rating, safety


def select_pilot_records(
    records: list[dict[str, Any]], size: int = PILOT_SIZE, seed: int = SEED
) -> list[dict[str, Any]]:
    """Deterministic best-effort stratified sampling."""
    _ = seed  # reserved for future tie-break extensions; selection is fully deterministic
    eligible: list[dict[str, Any]] = []
    for record in records:
        validate_source_record(record)
        eligible.append(record)

    by_sentiment: dict[str, list[dict[str, Any]]] = defaultdict(list)
    by_issue: dict[str, list[dict[str, Any]]] = defaultdict(list)
    by_rating: dict[int, list[dict[str, Any]]] = defaultdict(list)
    by_safety: dict[str, list[dict[str, Any]]] = defaultdict(list)
    composite: dict[tuple[str, str, int, str], list[dict[str, Any]]] = defaultdict(list)

    for record in eligible:
        sentiment, issue_type, rating, safety = strat_key(record)
        by_sentiment[sentiment].append(record)
        by_issue[issue_type].append(record)
        by_rating[rating].append(record)
        by_safety[safety].append(record)
        composite[(sentiment, issue_type, rating, safety)].append(record)

    for bucket in (
        *by_sentiment.values(),
        *by_issue.values(),
        *by_rating.values(),
        *by_safety.values(),
        *composite.values(),
    ):
        bucket.sort(key=lambda item: item["sourceReviewId"])

    selected: list[dict[str, Any]] = []
    selected_ids: set[str] = set()

    def take_next(group: dict[Any, list[dict[str, Any]]], key: Any) -> bool:
        for candidate in group.get(key, []):
            review_id = candidate["sourceReviewId"]
            if review_id in selected_ids:
                continue
            selected.append(candidate)
            selected_ids.add(review_id)
            return True
        return False

    for sentiment in SENTIMENTS:
        if len(selected) >= size:
            break
        take_next(by_sentiment, sentiment)

    for rating in (1, 2, 3, 4, 5):
        if len(selected) >= size:
            break
        take_next(by_rating, rating)

    for issue_type in ISSUE_TYPES:
        if len(selected) >= size:
            break
        take_next(by_issue, issue_type)

    for safety in SAFETY_VALUES:
        if len(selected) >= size:
            break
        take_next(by_safety, safety)

    bucket_keys = sorted(composite.keys())
    while len(selected) < size and bucket_keys:
        progress = False
        for key in bucket_keys:
            if len(selected) >= size:
                break
            if take_next(composite, key):
                progress = True
        if not progress:
            break

    if len(selected) < size:
        remaining = sorted(
            (r for r in eligible if r["sourceReviewId"] not in selected_ids),
            key=lambda item: item["sourceReviewId"],
        )
        for record in remaining:
            if len(selected) >= size:
                break
            selected.append(record)
            selected_ids.add(record["sourceReviewId"])

    selected.sort(key=lambda item: item["sourceReviewId"])
    return selected[:size]


def rule_prelabels_from_record(record: dict[str, Any]) -> dict[str, str]:
    signals = record.get("signals") or {}
    return {
        "sentiment": signals.get("sentiment") or "NEUTRAL",
        "issueType": signals.get("issueType") or "OTHER",
        "safetyRelatedObservation": normalize_safety(
            signals.get("safetyRelatedObservation")
        ),
        "qualitySignal": signals.get("qualitySignal") or "UNKNOWN",
    }


def build_gold_record(record: dict[str, Any], index: int) -> dict[str, Any]:
    review = record.get("review") or {}
    governance = record.get("governance") or {}
    rule = rule_prelabels_from_record(record)

    review_payload: dict[str, Any] = {
        "rating": review.get("rating"),
        "text": review.get("text", ""),
    }
    if review.get("summary"):
        review_payload["summary"] = review["summary"]
    if review.get("date"):
        review_payload["date"] = review["date"]

    return {
        "goldRecordId": f"gold_pilot_{index:03d}",
        "source": record.get("source", EXPECTED_SOURCE),
        "sourceProductId": record.get("sourceProductId", ""),
        "sourceReviewId": record.get("sourceReviewId", ""),
        "review": review_payload,
        "rulePrelabels": dict(rule),
        "manualLabels": dict(rule),
        "labelReviewStatus": DEFAULT_REVIEW_STATUS,
        "labelProvenance": DEFAULT_LABEL_PROVENANCE,
        "reviewerNotes": None,
        "governance": {
            "provenanceStatus": governance.get("provenanceStatus", "PARTIAL"),
            "licenseStatus": governance.get("licenseStatus", "UNKNOWN"),
            "usageRightsStatus": governance.get("usageRightsStatus", "UNKNOWN"),
            "originalDatasetEligibility": governance.get(
                "datasetEligibility", EXPECTED_ELIGIBILITY
            ),
        },
    }


def build_pilot_records(
    records: list[dict[str, Any]], size: int = PILOT_SIZE, seed: int = SEED
) -> list[dict[str, Any]]:
    selected = select_pilot_records(records, size=size, seed=seed)
    return [build_gold_record(record, idx + 1) for idx, record in enumerate(selected)]


def write_pilot_jsonl(records: list[dict[str, Any]], path: Path = PILOT_JSONL) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        for record in records:
            handle.write(json.dumps(record, ensure_ascii=False) + "\n")


def write_pilot_csv(records: list[dict[str, Any]], path: Path = PILOT_CSV) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=CSV_COLUMNS)
        writer.writeheader()
        for record in records:
            rule = record["rulePrelabels"]
            manual = record["manualLabels"]
            writer.writerow(
                {
                    "goldRecordId": record["goldRecordId"],
                    "sourceProductId": record["sourceProductId"],
                    "rating": record["review"]["rating"],
                    "reviewText": record["review"]["text"],
                    "rule_sentiment": rule["sentiment"],
                    "manual_sentiment": manual["sentiment"],
                    "rule_issueType": rule["issueType"],
                    "manual_issueType": manual["issueType"],
                    "rule_safetyRelatedObservation": rule["safetyRelatedObservation"],
                    "manual_safetyRelatedObservation": manual[
                        "safetyRelatedObservation"
                    ],
                    "rule_qualitySignal": rule["qualitySignal"],
                    "manual_qualitySignal": manual["qualitySignal"],
                    "labelReviewStatus": record["labelReviewStatus"],
                    "labelProvenance": record.get("labelProvenance", DEFAULT_LABEL_PROVENANCE),
                    "reviewerNotes": record.get("reviewerNotes") or "",
                }
            )


def provenance_counts(records: list[dict[str, Any]]) -> dict[str, int]:
    counter: Counter[str] = Counter()
    for record in records:
        counter[record.get("labelProvenance", "UNKNOWN")] += 1
    return dict(counter)


def finalized_records(records: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return [
        record
        for record in records
        if record.get("labelReviewStatus") in FINALIZED_STATUSES
    ]


def apply_truthful_provenance(record: dict[str, Any]) -> None:
    gold_id = record.get("goldRecordId", "")
    status = record.get("labelReviewStatus")
    if gold_id in GENUINE_HUMAN_GOLD_IDS and status in APPROVED_STATUSES:
        record["labelReviewStatus"] = "HUMAN_REVIEWED"
        record["labelProvenance"] = "HUMAN"
        return
    if status in {"HUMAN_REVIEWED", "EXPERT_APPROVED", "AI_ASSISTED_FINALIZED"}:
        if gold_id in GENUINE_HUMAN_GOLD_IDS:
            record["labelReviewStatus"] = "HUMAN_REVIEWED"
            record["labelProvenance"] = "HUMAN"
        else:
            record["labelReviewStatus"] = "AI_ASSISTED_FINALIZED"
            record["labelProvenance"] = "AI_ASSISTED"
    elif status == DEFAULT_REVIEW_STATUS:
        record["labelProvenance"] = DEFAULT_LABEL_PROVENANCE
    elif status == "REJECTED":
        record["labelProvenance"] = record.get("labelProvenance") or "UNKNOWN"


def read_pilot_jsonl(path: Path = PILOT_JSONL) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line:
                records.append(json.loads(line))
    return records


def distribution(records: list[dict[str, Any]], field: str, nested: str) -> dict[str, int]:
    counter: Counter[str] = Counter()
    for record in records:
        bucket = record.get(nested) or {}
        counter[str(bucket.get(field, "UNKNOWN"))] += 1
    return dict(sorted(counter.items()))


def rating_distribution(records: list[dict[str, Any]]) -> dict[str, int]:
    counter: Counter[str] = Counter()
    for record in records:
        rating = normalize_rating((record.get("review") or {}).get("rating"))
        key = str(rating) if rating else "UNKNOWN"
        counter[key] += 1
    return dict(sorted(counter.items()))


def status_counts(records: list[dict[str, Any]]) -> dict[str, int]:
    counter: Counter[str] = Counter()
    for record in records:
        counter[record.get("labelReviewStatus", "UNKNOWN")] += 1
    return dict(counter)


def approved_records(records: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return [
        record
        for record in records
        if record.get("labelReviewStatus") in APPROVED_STATUSES
    ]


def validate_pilot_records(records: list[dict[str, Any]]) -> list[str]:
    errors: list[str] = []
    if len(records) != PILOT_SIZE:
        errors.append(f"expected {PILOT_SIZE} records, got {len(records)}")

    gold_ids = [record.get("goldRecordId") for record in records]
    review_ids = [record.get("sourceReviewId") for record in records]
    if len(set(gold_ids)) != len(gold_ids):
        errors.append("duplicate goldRecordId")
    if len(set(review_ids)) != len(review_ids):
        errors.append("duplicate sourceReviewId")

    for record in records:
        if not str((record.get("review") or {}).get("text", "")).strip():
            errors.append(f"{record.get('goldRecordId')}: empty review text")

        status = record.get("labelReviewStatus")
        if status not in LABEL_REVIEW_STATUSES:
            errors.append(f"{record.get('goldRecordId')}: invalid labelReviewStatus")

        provenance = record.get("labelProvenance", DEFAULT_LABEL_PROVENANCE)
        if provenance not in LABEL_PROVENANCE_VALUES:
            errors.append(f"{record.get('goldRecordId')}: invalid labelProvenance")
        if status == "HUMAN_REVIEWED" and provenance != "HUMAN":
            errors.append(
                f"{record.get('goldRecordId')}: HUMAN_REVIEWED requires labelProvenance=HUMAN"
            )
        if status == "AI_ASSISTED_FINALIZED" and provenance != "AI_ASSISTED":
            errors.append(
                f"{record.get('goldRecordId')}: AI_ASSISTED_FINALIZED requires labelProvenance=AI_ASSISTED"
            )
        gold_id = record.get("goldRecordId")
        if gold_id in GENUINE_HUMAN_GOLD_IDS and status in FINALIZED_STATUSES:
            if status != "HUMAN_REVIEWED" or provenance != "HUMAN":
                errors.append(
                    f"{gold_id}: genuine human record must be HUMAN_REVIEWED with labelProvenance=HUMAN"
                )
        if gold_id not in GENUINE_HUMAN_GOLD_IDS and status == "HUMAN_REVIEWED":
            errors.append(
                f"{gold_id}: only genuine human records may use HUMAN_REVIEWED"
            )

        for label_set in ("rulePrelabels", "manualLabels"):
            labels = record.get(label_set) or {}
            if not labels:
                errors.append(f"{record.get('goldRecordId')}: missing {label_set}")
                continue
            if labels.get("sentiment") not in SENTIMENTS:
                errors.append(f"{record.get('goldRecordId')}: invalid sentiment")
            if labels.get("issueType") not in ISSUE_TYPES:
                errors.append(f"{record.get('goldRecordId')}: invalid issueType")
            if labels.get("safetyRelatedObservation") not in SAFETY_VALUES:
                errors.append(
                    f"{record.get('goldRecordId')}: invalid safetyRelatedObservation"
                )
            if labels.get("qualitySignal") not in QUALITY_SIGNALS:
                errors.append(f"{record.get('goldRecordId')}: invalid qualitySignal")

        governance = record.get("governance") or {}
        if governance.get("provenanceStatus") != "PARTIAL":
            errors.append(f"{record.get('goldRecordId')}: provenanceStatus changed")
        if governance.get("licenseStatus") != "UNKNOWN":
            errors.append(f"{record.get('goldRecordId')}: licenseStatus changed")
        if governance.get("usageRightsStatus") != "UNKNOWN":
            errors.append(f"{record.get('goldRecordId')}: usageRightsStatus changed")
        if governance.get("originalDatasetEligibility") != EXPECTED_ELIGIBILITY:
            errors.append(
                f"{record.get('goldRecordId')}: originalDatasetEligibility changed"
            )

    pending_or_rejected = [
        record
        for record in records
        if record.get("labelReviewStatus")
        in (DEFAULT_REVIEW_STATUS, "REJECTED")
    ]
    if finalized_records(pending_or_rejected):
        errors.append("pending/rejected records incorrectly marked finalized")

    return errors
