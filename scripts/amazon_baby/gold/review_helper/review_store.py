"""Persistence layer for Gold Pilot manual review helper."""

from __future__ import annotations

import csv
import json
import shutil
import sys
from collections import Counter
from copy import deepcopy
from pathlib import Path
from typing import Any

GOLD_DIR = Path(__file__).resolve().parent.parent
if str(GOLD_DIR) not in sys.path:
    sys.path.insert(0, str(GOLD_DIR))

from gold_pilot_common import (  # noqa: E402
    APPROVED_STATUSES,
    CSV_COLUMNS,
    DEFAULT_LABEL_PROVENANCE,
    DEFAULT_REVIEW_STATUS,
    FINALIZED_STATUSES,
    ISSUE_TYPES,
    PILOT_SIZE,
    QUALITY_SIGNALS,
    SAFETY_VALUES,
    SENTIMENTS,
    write_pilot_csv,
    write_pilot_jsonl,
)

# Domain-corrected pilot (immutable base during manual review)
BASE_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected.jsonl"
BASE_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected.csv"
DOMAIN_AUDIT_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_audited.csv"

# Active reviewed/autosave outputs
REVIEWED_CSV = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected_reviewed.csv"
REVIEWED_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_domain_corrected_reviewed.jsonl"
AI_PREREVIEW_JSONL = GOLD_DIR / "miyuna_gold_pilot_100_ai_prereview.jsonl"

# Legacy paths (read-only fallback for human fields when ASIN matches)
LEGACY_REVIEWED_CSV = GOLD_DIR / "miyuna_gold_pilot_100_reviewed.csv"
LEGACY_SOURCE_CSV = GOLD_DIR / "miyuna_gold_pilot_100_review.csv"

HUMAN_STATUSES = {"HUMAN_REVIEWED", "EXPERT_APPROVED", "REJECTED"}
REVIEW_FILTERS = (
    "ALL_PENDING",
    "SAFE_BULK_CANDIDATES",
    "LOW_MEDIUM_CONFIDENCE",
    "AI_RULE_DISAGREEMENTS",
    "SAFETY_OBSERVATIONS",
    "OTHER_CHILD_PRODUCT",
)

RESTORE_REJECTED_IDS = (
    "gold_pilot_003",
    "gold_pilot_004",
    "gold_pilot_005",
    "gold_pilot_006",
)


def load_ai_prereview() -> dict[str, dict[str, str]]:
    if not AI_PREREVIEW_JSONL.exists():
        return {}
    suggestions: dict[str, dict[str, str]] = {}
    with AI_PREREVIEW_JSONL.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            entry = json.loads(line)
            gold_id = entry["goldRecordId"]
            suggestions[gold_id] = entry
    return suggestions


def ai_labels_from_entry(entry: dict[str, str]) -> dict[str, str]:
    return {
        "sentiment": entry["aiSuggestedSentiment"],
        "issueType": entry["aiSuggestedIssueType"],
        "safetyRelatedObservation": entry["aiSuggestedSafetyRelatedObservation"],
        "qualitySignal": entry["aiSuggestedQualitySignal"],
    }


def read_csv_rows(path: Path) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as handle:
        return list(csv.DictReader(handle))


def csv_row_to_manual_labels(row: dict[str, str]) -> dict[str, str]:
    return {
        "sentiment": row["manual_sentiment"],
        "issueType": row["manual_issueType"],
        "safetyRelatedObservation": row["manual_safetyRelatedObservation"],
        "qualitySignal": row["manual_qualitySignal"],
    }


def record_to_csv_row(record: dict[str, Any]) -> dict[str, str]:
    rule = record["rulePrelabels"]
    manual = record["manualLabels"]
    notes = record.get("reviewerNotes")
    return {
        "goldRecordId": record["goldRecordId"],
        "sourceProductId": record["sourceProductId"],
        "rating": str(record["review"]["rating"]),
        "reviewText": record["review"]["text"],
        "rule_sentiment": rule["sentiment"],
        "manual_sentiment": manual["sentiment"],
        "rule_issueType": rule["issueType"],
        "manual_issueType": manual["issueType"],
        "rule_safetyRelatedObservation": rule["safetyRelatedObservation"],
        "manual_safetyRelatedObservation": manual["safetyRelatedObservation"],
        "rule_qualitySignal": rule["qualitySignal"],
        "manual_qualitySignal": manual["qualitySignal"],
        "labelReviewStatus": record["labelReviewStatus"],
        "labelProvenance": record.get("labelProvenance", DEFAULT_LABEL_PROVENANCE),
        "reviewerNotes": "" if notes is None else str(notes),
    }


def load_domain_audit() -> dict[str, dict[str, str]]:
    if not DOMAIN_AUDIT_CSV.exists():
        return {}
    rows = read_csv_rows(DOMAIN_AUDIT_CSV)
    return {
        row["goldRecordId"]: {
            "domainRelevant": row.get("domainRelevant", ""),
            "domainCategory": row.get("domainCategory", ""),
            "domainReason": row.get("domainReason", ""),
        }
        for row in rows
    }


def merge_human_fields(
    gold_id: str,
    baseline_asin: str,
    base_row: dict[str, str],
    candidate_row: dict[str, str] | None,
) -> dict[str, str]:
    """Apply human fields only when candidate matches the same goldRecordId + ASIN."""
    if candidate_row is None:
        return base_row
    if candidate_row.get("goldRecordId") != gold_id:
        return base_row
    if candidate_row.get("sourceProductId") != baseline_asin:
        return base_row
    merged = dict(base_row)
    merged["manual_sentiment"] = candidate_row["manual_sentiment"]
    merged["manual_issueType"] = candidate_row["manual_issueType"]
    merged["manual_safetyRelatedObservation"] = candidate_row[
        "manual_safetyRelatedObservation"
    ]
    merged["manual_qualitySignal"] = candidate_row["manual_qualitySignal"]
    merged["labelReviewStatus"] = candidate_row["labelReviewStatus"]
    merged["labelProvenance"] = candidate_row.get("labelProvenance", DEFAULT_LABEL_PROVENANCE)
    merged["reviewerNotes"] = candidate_row.get("reviewerNotes", "")
    return merged


class ReviewStore:
    """Loads domain-corrected pilot; persists reviewed copy only."""

    def __init__(
        self,
        base_jsonl_path: Path = BASE_JSONL,
        base_csv_path: Path = BASE_CSV,
        reviewed_csv_path: Path = REVIEWED_CSV,
        reviewed_jsonl_path: Path = REVIEWED_JSONL,
        legacy_reviewed_csv_path: Path = LEGACY_REVIEWED_CSV,
    ) -> None:
        self.base_jsonl_path = base_jsonl_path
        self.base_csv_path = base_csv_path
        self.reviewed_csv_path = reviewed_csv_path
        self.reviewed_jsonl_path = reviewed_jsonl_path
        self.legacy_reviewed_csv_path = legacy_reviewed_csv_path
        self.domain_by_id = load_domain_audit()
        self.ai_by_id = load_ai_prereview()
        self.active_filter = "ALL_PENDING"
        self._baseline_text: dict[str, str] = {}
        self._baseline_governance: dict[str, dict[str, Any]] = {}
        self.records: list[dict[str, Any]] = []
        self.current_index = 0
        self.reload()

    def reload(self) -> None:
        self.domain_by_id = load_domain_audit()
        self.ai_by_id = load_ai_prereview()
        self.records = self._load_records()
        self.current_index = self.first_pending_index(self.active_filter)

    def _human_overlay_rows(self) -> dict[str, dict[str, str]]:
        overlay: dict[str, dict[str, str]] = {}
        if self.reviewed_csv_path.exists():
            for row in read_csv_rows(self.reviewed_csv_path):
                overlay[row["goldRecordId"]] = row
            return overlay
        if self.base_csv_path.exists():
            for row in read_csv_rows(self.base_csv_path):
                overlay[row["goldRecordId"]] = row
        if self.legacy_reviewed_csv_path.exists():
            legacy = {row["goldRecordId"]: row for row in read_csv_rows(self.legacy_reviewed_csv_path)}
            merged: dict[str, dict[str, str]] = {}
            for gold_id, base_row in overlay.items():
                merged[gold_id] = merge_human_fields(
                    gold_id,
                    base_row["sourceProductId"],
                    base_row,
                    legacy.get(gold_id),
                )
            return merged
        return overlay

    def _load_records(self) -> list[dict[str, Any]]:
        if not self.base_jsonl_path.exists():
            raise FileNotFoundError(f"missing base JSONL: {self.base_jsonl_path}")
        if not self.base_csv_path.exists():
            raise FileNotFoundError(f"missing base CSV: {self.base_csv_path}")

        records: list[dict[str, Any]] = []
        with self.base_jsonl_path.open(encoding="utf-8") as handle:
            for line in handle:
                line = line.strip()
                if line:
                    records.append(json.loads(line))

        if len(records) != PILOT_SIZE:
            raise ValueError(f"expected {PILOT_SIZE} records, got {len(records)}")

        self._baseline_text = {
            record["goldRecordId"]: record["review"]["text"] for record in records
        }
        self._baseline_governance = {
            record["goldRecordId"]: deepcopy(record.get("governance") or {})
            for record in records
        }

        overlay_rows = self._human_overlay_rows()
        if len(overlay_rows) != PILOT_SIZE:
            raise ValueError(f"expected {PILOT_SIZE} overlay rows, got {len(overlay_rows)}")

        for record in records:
            gold_id = record["goldRecordId"]
            row = overlay_rows.get(gold_id)
            if row is None:
                raise ValueError(f"missing overlay row for {gold_id}")
            if row["sourceProductId"] != record["sourceProductId"]:
                raise ValueError(
                    f"ASIN mismatch for {gold_id}: overlay has stale replacement data"
                )
            if record["review"]["text"] != row["reviewText"]:
                raise ValueError(
                    f"review text mismatch for {gold_id} — corrected source is immutable"
                )
            record["manualLabels"] = csv_row_to_manual_labels(row)
            record["labelReviewStatus"] = row["labelReviewStatus"]
            record["labelProvenance"] = row.get("labelProvenance", DEFAULT_LABEL_PROVENANCE)
            notes = row.get("reviewerNotes", "")
            record["reviewerNotes"] = notes if notes else None
            record["domain"] = self.domain_by_id.get(gold_id)
            record["aiPrereview"] = self.ai_by_id.get(gold_id)

        records.sort(key=lambda item: item["goldRecordId"])
        return records

    def is_malformed_record(self, record: dict[str, Any]) -> bool:
        review = record.get("review") or {}
        text = str(review.get("text") or "").strip()
        if len(text.split()) < 8:
            return True
        ai = record.get("aiPrereview")
        if not ai:
            return True
        try:
            self._validate_manual_labels(ai_labels_from_entry(ai))
        except ValueError:
            return True
        return False

    def is_safe_bulk_candidate(self, record: dict[str, Any]) -> bool:
        if record.get("labelReviewStatus") != DEFAULT_REVIEW_STATUS:
            return False
        ai = record.get("aiPrereview")
        if not ai:
            return False
        if ai.get("aiReviewConfidence") != "HIGH":
            return False
        if ai.get("aiSuggestedSafetyRelatedObservation") != "FALSE":
            return False
        domain = record.get("domain") or {}
        if domain.get("domainRelevant") != "YES":
            return False
        if ai.get("aiRuleRelation") in {"AI_CORRECTS_RULES", "NEEDS_HUMAN_ATTENTION"}:
            return False
        if self.is_malformed_record(record):
            return False
        return True

    def safe_bulk_indices(self) -> list[int]:
        review_ids = [record["sourceReviewId"] for record in self.records]
        duplicate_ids = {rid for rid, count in Counter(review_ids).items() if count > 1}
        indices: list[int] = []
        for index, record in enumerate(self.records):
            if not self.is_safe_bulk_candidate(record):
                continue
            if record["sourceReviewId"] in duplicate_ids:
                continue
            indices.append(index)
        return indices

    def safe_bulk_summary(self, sample_size: int = 8) -> dict[str, Any]:
        indices = self.safe_bulk_indices()
        sentiments: Counter[str] = Counter()
        issue_types: Counter[str] = Counter()
        samples: list[dict[str, Any]] = []
        for index in indices:
            record = self.records[index]
            ai = record["aiPrereview"]
            labels = ai_labels_from_entry(ai)
            sentiments[labels["sentiment"]] += 1
            issue_types[labels["issueType"]] += 1
            if len(samples) < sample_size:
                samples.append(
                    {
                        "index": index,
                        "goldRecordId": record["goldRecordId"],
                        "sourceProductId": record["sourceProductId"],
                        "rating": record["review"]["rating"],
                        "reviewTextPreview": record["review"]["text"][:160],
                        "aiSuggestion": labels,
                        "aiReviewConfidence": ai.get("aiReviewConfidence"),
                        "domainCategory": (record.get("domain") or {}).get("domainCategory"),
                    }
                )
        return {
            "count": len(indices),
            "goldRecordIds": [self.records[i]["goldRecordId"] for i in indices],
            "sentimentDistribution": dict(sorted(sentiments.items())),
            "issueTypeDistribution": dict(sorted(issue_types.items())),
            "samples": samples,
            "warning": (
                "Bulk approval is a reviewer action. Review samples before confirming."
            ),
        }

    def queue_counts(self) -> dict[str, int]:
        pending_indices = self.filtered_pending_indices("ALL_PENDING")
        safe_bulk = set(self.safe_bulk_indices())
        low_medium = set(self.filtered_pending_indices("LOW_MEDIUM_CONFIDENCE"))
        safety = set(self.filtered_pending_indices("SAFETY_OBSERVATIONS"))
        disagreements = set(self.filtered_pending_indices("AI_RULE_DISAGREEMENTS"))
        needs_manual = {
            index
            for index in pending_indices
            if index not in safe_bulk
        }
        return {
            "SAFE_BULK_CANDIDATES": len(safe_bulk),
            "LOW_MEDIUM": len(low_medium),
            "SAFETY": len(safety),
            "AI_RULE_DISAGREEMENTS": len(disagreements),
            "NEEDS_MANUAL_REVIEW": len(needs_manual),
        }

    def restore_accidental_rejections_if_needed(self) -> int:
        restored = 0
        for record in self.records:
            if record["goldRecordId"] not in RESTORE_REJECTED_IDS:
                continue
            if record["labelReviewStatus"] != "REJECTED":
                continue
            record["labelReviewStatus"] = DEFAULT_REVIEW_STATUS
            restored += 1
        if restored:
            for record in self.records:
                self._assert_immutable_fields(record)
            self.persist()
        return restored

    def approve_safe_bulk_candidates(self, expected_count: int, confirmed: bool) -> dict[str, Any]:
        if not confirmed:
            raise ValueError("bulk approval requires explicit confirmation")
        indices = self.safe_bulk_indices()
        if len(indices) != expected_count:
            raise ValueError(
                f"candidate count mismatch: expected {expected_count}, current {len(indices)}"
            )
        approved_ids: list[str] = []
        for index in indices:
            record = self.records[index]
            ai = record.get("aiPrereview")
            if not ai or not self.is_safe_bulk_candidate(record):
                raise ValueError(f"record {record['goldRecordId']} is no longer a safe bulk candidate")
            labels = ai_labels_from_entry(ai)
            self._validate_manual_labels(labels)
            record["manualLabels"] = labels
            record["labelReviewStatus"] = "AI_ASSISTED_FINALIZED"
            record["labelProvenance"] = "AI_ASSISTED"
            self._assert_immutable_fields(record)
            approved_ids.append(record["goldRecordId"])
        self.persist()
        self.current_index = self.first_pending_index(self.active_filter)
        return {
            "approvedCount": len(approved_ids),
            "approvedIds": approved_ids,
        }

    def _record_matches_filter(self, record: dict[str, Any], review_filter: str) -> bool:
        if record.get("labelReviewStatus") != DEFAULT_REVIEW_STATUS:
            return False
        ai = record.get("aiPrereview") or {}
        if review_filter == "ALL_PENDING":
            return True
        if review_filter == "SAFE_BULK_CANDIDATES":
            return self.is_safe_bulk_candidate(record)
        if review_filter == "LOW_MEDIUM_CONFIDENCE":
            return ai.get("aiReviewConfidence") in {"LOW", "MEDIUM"}
        if review_filter == "AI_RULE_DISAGREEMENTS":
            return ai.get("aiRuleRelation") in {
                "AI_CORRECTS_RULES",
                "NEEDS_HUMAN_ATTENTION",
            }
        if review_filter == "SAFETY_OBSERVATIONS":
            return ai.get("aiSuggestedSafetyRelatedObservation") == "TRUE"
        if review_filter == "OTHER_CHILD_PRODUCT":
            return (record.get("domain") or {}).get("domainCategory") == "OTHER_CHILD_PRODUCT"
        return True

    def filtered_pending_indices(self, review_filter: str | None = None) -> list[int]:
        review_filter = review_filter or self.active_filter
        if review_filter == "SAFE_BULK_CANDIDATES":
            return self.safe_bulk_indices()
        return [
            index
            for index, record in enumerate(self.records)
            if self._record_matches_filter(record, review_filter)
        ]

    def filter_counts(self) -> dict[str, int]:
        return {
            review_filter: len(self.filtered_pending_indices(review_filter))
            for review_filter in REVIEW_FILTERS
        }

    def first_pending_index(self, review_filter: str | None = None) -> int:
        indices = self.filtered_pending_indices(review_filter)
        if indices:
            return indices[0]
        for index, record in enumerate(self.records):
            if record.get("labelReviewStatus") == DEFAULT_REVIEW_STATUS:
                return index
        return 0 if self.records else 0

    def next_pending_index(self, from_index: int, review_filter: str | None = None) -> int:
        indices = self.filtered_pending_indices(review_filter)
        if not indices:
            return from_index
        for idx in indices:
            if idx > from_index:
                return idx
        return indices[0]

    def progress(self) -> dict[str, Any]:
        reviewed = sum(
            1 for record in self.records if record["labelReviewStatus"] in FINALIZED_STATUSES
        )
        pending = sum(
            1
            for record in self.records
            if record.get("labelReviewStatus") == DEFAULT_REVIEW_STATUS
        )
        rejected = sum(
            1 for record in self.records if record.get("labelReviewStatus") == "REJECTED"
        )
        complete = pending == 0
        return {
            "total": len(self.records),
            "reviewed": reviewed,
            "pending": pending,
            "rejected": rejected,
            "progressPercent": round((reviewed / len(self.records)) * 100) if self.records else 0,
            "complete": complete,
        }

    def record_view(self, index: int) -> dict[str, Any]:
        if index < 0 or index >= len(self.records):
            raise IndexError("record index out of range")
        record = self.records[index]
        view = {
            "index": index,
            "position": index + 1,
            "total": len(self.records),
            "goldRecordId": record["goldRecordId"],
            "sourceProductId": record["sourceProductId"],
            "rating": record["review"]["rating"],
            "reviewText": record["review"]["text"],
            "rulePrelabels": record["rulePrelabels"],
            "manualLabels": record["manualLabels"],
            "labelReviewStatus": record["labelReviewStatus"],
            "reviewerNotes": record.get("reviewerNotes"),
            "governance": record.get("governance"),
        }
        if record.get("domain"):
            view["domain"] = record["domain"]
        if record.get("aiPrereview"):
            ai = record["aiPrereview"]
            view["aiSuggestion"] = ai_labels_from_entry(ai)
            view["aiReviewConfidence"] = ai.get("aiReviewConfidence")
            view["aiReviewReason"] = ai.get("aiReviewReason")
            view["aiRuleRelation"] = ai.get("aiRuleRelation")
        return view

    def _validate_manual_labels(self, labels: dict[str, str]) -> None:
        if labels.get("sentiment") not in SENTIMENTS:
            raise ValueError("invalid sentiment")
        if labels.get("issueType") not in ISSUE_TYPES:
            raise ValueError("invalid issueType")
        if labels.get("safetyRelatedObservation") not in SAFETY_VALUES:
            raise ValueError("invalid safetyRelatedObservation")
        if labels.get("qualitySignal") not in QUALITY_SIGNALS:
            raise ValueError("invalid qualitySignal")

    def _assert_immutable_fields(self, record: dict[str, Any]) -> None:
        gold_id = record["goldRecordId"]
        if record["review"]["text"] != self._baseline_text[gold_id]:
            raise ValueError("review text changed — source text is immutable")
        if record.get("governance") != self._baseline_governance[gold_id]:
            raise ValueError("governance changed — source governance is immutable")

    def apply_action(
        self,
        index: int,
        action: str,
        manual_labels: dict[str, str] | None = None,
        reviewer_notes: str | None = None,
    ) -> dict[str, Any]:
        if index < 0 or index >= len(self.records):
            raise IndexError("record index out of range")

        record = self.records[index]
        if action == "approve_ai":
            ai = record.get("aiPrereview")
            if not ai:
                raise ValueError("no AI suggestion available for this record")
            status = "AI_ASSISTED_FINALIZED"
            provenance = "AI_ASSISTED"
            labels = ai_labels_from_entry(ai)
        elif action == "approve":
            status = "HUMAN_REVIEWED"
            provenance = "HUMAN"
            labels = dict(record["manualLabels"])
        elif action == "save":
            if manual_labels is None:
                raise ValueError("manual labels required for save")
            status = "HUMAN_REVIEWED"
            provenance = "HUMAN"
            labels = dict(manual_labels)
        elif action == "reject":
            status = "REJECTED"
            provenance = "UNKNOWN"
            labels = dict(manual_labels or record["manualLabels"])
        else:
            raise ValueError(f"unknown action: {action}")

        self._validate_manual_labels(labels)
        record["manualLabels"] = labels
        record["labelReviewStatus"] = status
        record["labelProvenance"] = provenance
        record["reviewerNotes"] = reviewer_notes if reviewer_notes else None
        self._assert_immutable_fields(record)
        self.persist()
        return self.record_view(index)

    def persist(self) -> None:
        for record in self.records:
            self._assert_immutable_fields(record)

        tmp_csv = self.reviewed_csv_path.with_suffix(".csv.tmp")
        with tmp_csv.open("w", encoding="utf-8", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=CSV_COLUMNS)
            writer.writeheader()
            for record in self.records:
                writer.writerow(record_to_csv_row(record))
        shutil.move(str(tmp_csv), str(self.reviewed_csv_path))

        write_pilot_jsonl(self.records, self.reviewed_jsonl_path)

    def set_current_index(self, index: int) -> None:
        if index < 0 or index >= len(self.records):
            raise IndexError("record index out of range")
        self.current_index = index

    def set_filter(self, review_filter: str) -> None:
        if review_filter not in REVIEW_FILTERS:
            raise ValueError(f"unknown filter: {review_filter}")
        self.active_filter = review_filter
        self.current_index = self.first_pending_index(review_filter)

    def state_payload(self) -> dict[str, Any]:
        progress = self.progress()
        payload: dict[str, Any] = {
            "progress": progress,
            "currentIndex": self.current_index,
            "record": self.record_view(self.current_index),
            "activeFilter": self.active_filter,
            "filterCounts": self.filter_counts(),
            "queueCounts": self.queue_counts(),
            "safeBulk": self.safe_bulk_summary(),
            "filteredPendingIndices": self.filtered_pending_indices(self.active_filter),
            "inputPaths": {
                "baseCsv": str(self.base_csv_path),
                "baseJsonl": str(self.base_jsonl_path),
                "reviewedCsv": str(self.reviewed_csv_path),
                "reviewedJsonl": str(self.reviewed_jsonl_path),
            },
            "enums": {
                "sentiment": list(SENTIMENTS),
                "issueType": list(ISSUE_TYPES),
                "safetyRelatedObservation": list(SAFETY_VALUES),
                "qualitySignal": list(QUALITY_SIGNALS),
            },
        }
        if progress["complete"]:
            payload["completion"] = {
                "message": "GOLD PILOT REVIEW COMPLETE",
                "commands": [
                    "python validate_gold_pilot.py",
                    "python gold_review_summary.py",
                    "python export_finalized_gold.py",
                ],
            }
        return payload
