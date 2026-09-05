"""AI-assisted pre-review label suggestions for Gold Pilot (not human review)."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

from gold_pilot_common import ISSUE_TYPES, QUALITY_SIGNALS, SENTIMENTS

CONFIDENCE_LEVELS = ("HIGH", "MEDIUM", "LOW")
RULE_RELATIONS = (
    "AI_AGREES_WITH_RULES",
    "AI_CORRECTS_RULES",
    "NEEDS_HUMAN_ATTENTION",
)

POSITIVE_CUES = (
    "love",
    "great",
    "excellent",
    "perfect",
    "recommend",
    "happy",
    "wonderful",
    "fantastic",
    "best",
    "awesome",
    "highly suggest",
    "worth every penny",
    "works well",
    "works great",
)

NEGATIVE_CUES = (
    "hate",
    "terrible",
    "awful",
    "horrible",
    "disappointed",
    "disappointing",
    "waste of money",
    "do not buy",
    "don't buy",
    "returned",
    "broke",
    "broken",
    "failed",
    "useless",
    "poor quality",
    "not worth",
    "worst",
    "leaky",
    "leaks",
)

NEUTRAL_CUES = (
    "okay",
    "ok ",
    "fine",
    "average",
    "mixed",
    "but ",
    "however",
    "although",
    "wish",
    "expected",
)

ISSUE_PATTERNS: list[tuple[str, tuple[str, ...]]] = [
    ("USABILITY", (
        r"\bhard to use\b", r"\bdifficult\b", r"\bconfus", r"\bawkward\b",
        r"\binconvenient\b", r"\bannoying\b", r"\bhave to adjust\b",
        r"\breadjust\b", r"\bconstantly adjust\b", r"\bevery time\b",
        r"\bpain to\b", r"\bstruggl", r"\bfrustrat",
    )),
    ("AGE_SIZE_MISMATCH", (
        r"\btoo small\b", r"\btoo big\b", r"\btoo large\b", r"\bruns small\b",
        r"\bruns large\b", r"\bdoesn'?t fit\b", r"\bdidn'?t fit\b", r"\boutgrow\b",
        r"\bage\b", r"\bsize\b", r"\bsnug\b", r"\btight on\b",
    )),
    ("BREAKAGE", (
        r"\bbroke\b", r"\bbroken\b", r"\bbreak\b", r"\bcrack", r"\bsnap(ped)?\b",
        r"\bfell apart\b", r"\bcame off\b", r"\bcame apart\b", r"\bchewed the foot off\b",
    )),
    ("DURABILITY", (
        r"\bdurability\b", r"\bdurable\b", r"\bwear out\b", r"\bfall apart\b",
        r"\blasted\b", r"\bheld up\b", r"\bstill going\b",
    )),
    ("MATERIAL", (
        r"\bmaterial\b", r"\bfabric\b", r"\bplastic\b", r"\bbpa\b", r"\bcheap plastic\b",
        r"\bflimsy\b", r"\bthin\b",
    )),
    ("ODOR", (r"\bodor\b", r"\bodour\b", r"\bsmell\b", r"\bstink\b", r"\bscent\b")),
    ("PACKAGING", (r"\bpackaging\b", r"\bpackage\b", r"\bbox\b", r"\barrived damaged\b")),
    ("QUALITY", (
        r"\bquality\b", r"\bcheap\b", r"\bdefect\b", r"\bflimsy\b", r"\bpoorly made\b",
        r"\bwell made\b", r"\bwell-made\b", r"\bsturdy\b",
    )),
]

SAFETY_OBSERVATION_PATTERNS = (
    r"\bchok(e|ing|ed)\b",
    r"\bstrangul",
    r"\bsuffoc",
    r"\bbaby fell\b",
    r"\bchild fell\b",
    r"\bfell down\b",
    r"\bfell off\b.*\b(baby|child|kid|infant|toddler|seat|chair|stroller|crib|rail|gate)\b",
    r"\bfell out\b",
    r"\btipp(ed|ing) over\b",
    r"\bunstable\b",
    r"\bsharp\b",
    r"\bbroken part\b",
    r"\bdetach(ed|ment)?\b.*\b(baby|child|hazard|danger|unsafe|small)\b",
    r"\bburn(ed|t)\b.*\b(baby|child|skin|overheat|hot)\b",
    r"\boverheat",
    r"\belectrical\b",
    r"\bharness\b.*\b(fail|broke|loose|unsafe|danger)\b",
    r"\brestraint\b.*\b(fail|broke|loose|unsafe|danger)\b",
    r"\bunsafe\b",
    r"\bhazard",
    r"\bingest",
    r"\binjur",
    r"\bnear[- ]miss\b",
    r"\bscared\b.*\b(baby|child|chok|strang|suffoc|unsafe|danger|hurt)\b",
    r"\b(baby|child|kid|infant|toddler)\b.*\b(scared|hurt|injur|chok|strang)\b",
)

SAFETY_FALSE_POSITIVE = (
    r"\bfeels safe\b",
    r"\bgood head support\b",
    r"\bprotects the baby\b",
    r"\bpeace of mind\b",
    r"\bsafe and secure\b",
    r"\bkept my baby safe\b",
)


@dataclass(frozen=True)
class AISuggestion:
    sentiment: str
    issue_type: str
    safety_related_observation: str
    quality_signal: str
    confidence: str
    reason: str
    rule_relation: str

    def as_dict(self) -> dict[str, str]:
        return {
            "aiSuggestedSentiment": self.sentiment,
            "aiSuggestedIssueType": self.issue_type,
            "aiSuggestedSafetyRelatedObservation": self.safety_related_observation,
            "aiSuggestedQualitySignal": self.quality_signal,
            "aiReviewConfidence": self.confidence,
            "aiReviewReason": self.reason,
            "aiRuleRelation": self.rule_relation,
        }


def _normalize_text(text: str, summary: str | None = None) -> str:
    return " ".join(part.strip() for part in (text, summary or "") if part).lower()


def _count_hits(text: str, patterns: tuple[str, ...]) -> int:
    return sum(1 for pattern in patterns if re.search(pattern, text))


def infer_sentiment(text: str, rating: float | None) -> str:
    pos = _count_hits(text, POSITIVE_CUES)
    neg = _count_hits(text, NEGATIVE_CUES)
    neu = _count_hits(text, NEUTRAL_CUES)

    if neg > pos and neg >= 1:
        return "NEGATIVE"
    if pos > neg and pos >= 1:
        if neu >= 2 and neg >= 1:
            return "NEUTRAL"
        return "POSITIVE"
    if neu >= 2 or (pos >= 1 and neg >= 1):
        return "NEUTRAL"
    if rating is not None:
        if rating <= 2:
            return "NEGATIVE"
        if rating == 3:
            return "NEUTRAL"
        if rating >= 4:
            return "POSITIVE"
    return "NEUTRAL"


def infer_issue_type(text: str) -> tuple[str, int]:
    scores: list[tuple[int, str]] = []
    priority = {issue: index for index, (issue, _) in enumerate(ISSUE_PATTERNS)}
    for issue, patterns in ISSUE_PATTERNS:
        score = sum(2 if re.search(pattern, text) else 0 for pattern in patterns)
        if score:
            scores.append((score, issue))
    if not scores:
        return "OTHER", 0
    scores.sort(key=lambda item: (-item[0], priority[item[1]]))
    return scores[0][1], scores[0][0]


def infer_safety_observation(text: str) -> bool:
    if any(re.search(pattern, text) for pattern in SAFETY_FALSE_POSITIVE):
        if not any(re.search(pattern, text) for pattern in SAFETY_OBSERVATION_PATTERNS):
            return False
    return any(re.search(pattern, text) for pattern in SAFETY_OBSERVATION_PATTERNS)


def infer_quality_signal(sentiment: str, issue_type: str, text: str) -> str:
    if issue_type in {"BREAKAGE", "MATERIAL", "ODOR", "QUALITY"} and sentiment != "POSITIVE":
        return "NEGATIVE"
    if sentiment == "POSITIVE" and issue_type == "OTHER":
        if re.search(r"\bwell made\b|\bquality product\b|\bworth every penny\b", text):
            return "POSITIVE"
        return "POSITIVE"
    if sentiment == "NEGATIVE":
        return "NEGATIVE"
    if sentiment == "NEUTRAL" or issue_type not in {"OTHER", "USABILITY"}:
        return "MIXED"
    if issue_type != "OTHER":
        return "MIXED"
    return "UNKNOWN"


def classify_rule_relation(
    ai: dict[str, str],
    rule: dict[str, str],
    confidence: str,
    domain_category: str | None,
) -> str:
    fields = ("sentiment", "issueType", "safetyRelatedObservation", "qualitySignal")
    agrees = all(ai[field] == rule[field] for field in fields)
    if confidence == "LOW" or domain_category == "OTHER_CHILD_PRODUCT":
        return "NEEDS_HUMAN_ATTENTION"
    if ai["safetyRelatedObservation"] == "TRUE":
        return "NEEDS_HUMAN_ATTENTION"
    if agrees:
        return "AI_AGREES_WITH_RULES"
    return "AI_CORRECTS_RULES"


def analyze_record(record: dict[str, Any]) -> AISuggestion | None:
    if record.get("labelReviewStatus") != "PENDING_HUMAN_REVIEW":
        return None

    review = record.get("review") or {}
    text = _normalize_text(str(review.get("text") or ""), review.get("summary"))
    rating_raw = review.get("rating")
    rating = float(rating_raw) if rating_raw is not None else None
    rule = record.get("rulePrelabels") or {}

    if len(text.split()) < 8:
        sentiment = infer_sentiment(text, rating)
        issue_type = infer_issue_type(text)[0]
        safety = "TRUE" if infer_safety_observation(text) else "FALSE"
        quality = infer_quality_signal(sentiment, issue_type, text)
        domain = (record.get("domain") or {}).get("domainCategory")
        ai_labels = {
            "sentiment": sentiment,
            "issueType": issue_type,
            "safetyRelatedObservation": safety,
            "qualitySignal": quality,
        }
        rule_labels = {
            "sentiment": rule.get("sentiment", "NEUTRAL"),
            "issueType": rule.get("issueType", "OTHER"),
            "safetyRelatedObservation": rule.get("safetyRelatedObservation", "FALSE"),
            "qualitySignal": rule.get("qualitySignal", "UNKNOWN"),
        }
        relation = classify_rule_relation(ai_labels, rule_labels, "LOW", domain)
        return AISuggestion(
            sentiment=sentiment,
            issue_type=issue_type,
            safety_related_observation=safety,
            quality_signal=quality,
            confidence="LOW",
            reason="Review text is short or ambiguous; labels are best-effort and need human confirmation.",
            rule_relation=relation,
        )

    sentiment = infer_sentiment(text, rating)
    issue_type, issue_score = infer_issue_type(text)
    safety = "TRUE" if infer_safety_observation(text) else "FALSE"
    quality = infer_quality_signal(sentiment, issue_type, text)

    # Correct common rule mislabels
    if issue_type == "USABILITY" and rule.get("issueType") == "AGE_SIZE_MISMATCH":
        if re.search(r"\badjust\b|\binconvenient\b|\bpain\b|\bevery time\b", text):
            issue_type = "USABILITY"

    ambiguous_fields = 0
    if sentiment == "NEUTRAL":
        ambiguous_fields += 1
    if issue_score == 0:
        ambiguous_fields += 1
    if quality == "UNKNOWN":
        ambiguous_fields += 1

    if ambiguous_fields >= 2:
        confidence = "LOW"
    elif ambiguous_fields == 1 or sentiment == "NEUTRAL":
        confidence = "MEDIUM"
    else:
        confidence = "HIGH"

    ai_labels = {
        "sentiment": sentiment,
        "issueType": issue_type,
        "safetyRelatedObservation": safety,
        "qualitySignal": quality,
    }
    rule_labels = {
        "sentiment": rule.get("sentiment", "NEUTRAL"),
        "issueType": rule.get("issueType", "OTHER"),
        "safetyRelatedObservation": rule.get("safetyRelatedObservation", "FALSE"),
        "qualitySignal": rule.get("qualitySignal", "UNKNOWN"),
    }
    domain = (record.get("domain") or {}).get("domainCategory")
    relation = classify_rule_relation(ai_labels, rule_labels, confidence, domain)

    reason_parts = [
        f"Sentiment reads {sentiment.lower()} from review wording",
        f"primary issue appears to be {issue_type.lower().replace('_', ' ')}",
    ]
    if safety == "TRUE":
        reason_parts.append("review mentions a potentially safety-related observation")
    elif relation == "AI_CORRECTS_RULES":
        reason_parts.append("AI adjusts one or more rule pre-labels based on review meaning")
    reason = "; ".join(reason_parts) + "."

    return AISuggestion(
        sentiment=sentiment,
        issue_type=issue_type,
        safety_related_observation=safety,
        quality_signal=quality,
        confidence=confidence,
        reason=reason,
        rule_relation=relation,
    )


def validate_suggestion(suggestion: AISuggestion) -> None:
    if suggestion.sentiment not in SENTIMENTS:
        raise ValueError(f"invalid sentiment: {suggestion.sentiment}")
    if suggestion.issue_type not in ISSUE_TYPES:
        raise ValueError(f"invalid issueType: {suggestion.issue_type}")
    if suggestion.safety_related_observation not in {"TRUE", "FALSE"}:
        raise ValueError("invalid safety suggestion")
    if suggestion.quality_signal not in QUALITY_SIGNALS:
        raise ValueError(f"invalid qualitySignal: {suggestion.quality_signal}")
    if suggestion.confidence not in CONFIDENCE_LEVELS:
        raise ValueError(f"invalid confidence: {suggestion.confidence}")
    if suggestion.rule_relation not in RULE_RELATIONS:
        raise ValueError(f"invalid rule relation: {suggestion.rule_relation}")
