"""Miyuna child/baby product domain taxonomy and classification."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Iterable

DOMAIN_CATEGORIES = (
    "STROLLER",
    "CAR_SEAT",
    "CRIB",
    "BASSINET",
    "BOTTLE_FEEDING",
    "DIAPERING",
    "BABY_MONITOR",
    "TOYS",
    "SAFETY_GATE",
    "CARRIER",
    "BATH",
    "NURSERY",
    "TEETHING",
    "HIGH_CHAIR",
    "PACIFIER",
    "FORMULA_PREP",
    "STERILIZER",
    "PLAYARD",
    "WALKER",
    "SLEEP_PRODUCTS",
    "BABY_CARE_ACCESSORIES",
    "OTHER_CHILD_PRODUCT",
)

CHILD_CONTEXT = (
    "baby",
    "babies",
    "infant",
    "infants",
    "newborn",
    "newborns",
    "toddler",
    "toddlers",
    "child",
    "children",
    "kid",
    "kids",
    "granddaughter",
    "grandson",
    "week old",
    "month old",
    "months old",
    "year old",
    "breastfed",
    "breastfeed",
    "daycare",
    "nursery",
    "potty",
    "diaper",
    "bebek",
    "çocuk",
)

NON_CHILD_SIGNALS = (
    "for my husband",
    "for my wife",
    "my dog",
    "my cat",
    "pet ",
    "pets ",
    "adult only",
    "men's",
    "women's",
    "wrinkle cream",
    "anti-aging",
    "lawn mower",
    "power drill",
    "drill for",
    "home renovation",
    "power tool",
)

CATEGORY_PATTERNS: dict[str, tuple[str, ...]] = {
    "STROLLER": (
        r"\bstroller\b",
        r"\bpram\b",
        r"\bbuggy\b",
        r"snap n go",
        r"travel system",
        r"bebek arabası",
    ),
    "CAR_SEAT": (
        r"\bcar seat\b",
        r"\bcarseat\b",
        r"\binfant seat\b",
        r"maxi cosi",
        r"britax",
        r"chicco keyfit",
        r"convertible seat",
        r"infant carrier",
    ),
    "CRIB": (
        r"\bcrib\b",
        r"\bcot\b",
        r"crib mattress",
        r"crib sheet",
    ),
    "BASSINET": (
        r"\bbassinet\b",
        r"bedside sleeper",
        r"moses basket",
    ),
    "BOTTLE_FEEDING": (
        r"\bbottle\b",
        r"sippy cup",
        r"\bnipple\b",
        r"tommee tippee",
        r"dr brown",
        r"breast pump",
        r"nursing pillow",
        r"bottle warmer",
        r"formula",
    ),
    "DIAPERING": (
        r"\bdiaper\b",
        r"\bnappy\b",
        r"cloth diaper",
        r"diaper pail",
        r"diaper cover",
        r"changing pad",
        r"wet bag",
        r"\bwipe\b",
        r"diaper cream",
        r"potty training",
        r"training pant",
    ),
    "BABY_MONITOR": (
        r"baby monitor",
        r"\bmonitor camera\b",
        r"video monitor",
        r"audio monitor",
        r"baby cam",
    ),
    "TOYS": (
        r"\btoy\b",
        r"\btoys\b",
        r"rattle",
        r"play mat",
        r"activity center",
        r"jumperoo",
        r"exersaucer",
        r"stacking cup",
        r"water toy",
        r"play gym",
    ),
    "SAFETY_GATE": (
        r"safety gate",
        r"baby gate",
        r"cabinet lock",
        r"cupboard",
        r"childproof",
        r"child proof",
        r"outlet cover",
        r"corner guard",
    ),
    "CARRIER": (
        r"baby carrier",
        r"\bcarrier\b",
        r"\bsling\b",
        r"\bwrap\b",
        r"\bergobaby\b",
        r"baby wearing",
        r"babywearing",
        r"kanguru",
    ),
    "BATH": (
        r"baby bath",
        r"infant tub",
        r"bath tub",
        r"hooded towel",
        r"bath seat",
        r"bath toy",
    ),
    "NURSERY": (
        r"nursery",
        r"changing table",
        r"rocking chair",
        r"glider",
        r"night light",
    ),
    "TEETHING": (
        r"teething",
        r"teether",
        r"chew toy",
        r"teething ring",
    ),
    "HIGH_CHAIR": (
        r"high chair",
        r"highchair",
        r"hook on chair",
        r"booster seat",
        r"mama sandalyesi",
    ),
    "PACIFIER": (
        r"pacifier",
        r"soothie",
        r"binky",
        r"dummy",
        r"emzik",
    ),
    "FORMULA_PREP": (
        r"formula maker",
        r"formula dispenser",
        r"formula pro",
    ),
    "STERILIZER": (
        r"sterilizer",
        r"steriliser",
        r"uv sanitizer",
    ),
    "PLAYARD": (
        r"playard",
        r"play yard",
        r"pack n play",
        r"pack and play",
        r"playpen",
        r"play pen",
    ),
    "WALKER": (
        r"\bwalker\b",
        r"walkers",
        r"yürüteç",
    ),
    "SLEEP_PRODUCTS": (
        r"swaddle",
        r"sleep sack",
        r"bouncer",
        r"bouncy seat",
        r"rocker",
        r"swing",
        r"white noise",
        r"sound machine",
    ),
    "BABY_CARE_ACCESSORIES": (
        r"baby lotion",
        r"baby shampoo",
        r"nasal aspirator",
        r"nose frida",
        r"thermometer",
        r"nail clipper",
        r"baby brush",
        r"baby tracker",
        r"childcare journal",
        r"feeding log",
        r"baby book",
        r"receiving blanket",
    ),
}

GENERIC_CHILD_PATTERNS = (
    r"\bbaby\b",
    r"\binfant\b",
    r"\bnewborn\b",
    r"\btoddler\b",
    r"\bchild\b",
    r"little one",
    r"my son",
    r"my daughter",
    r"our son",
    r"our daughter",
)


@dataclass(frozen=True)
class DomainClassification:
    domain_relevant: str
    domain_category: str
    domain_reason: str
    matched_category: str | None = None


def _normalize_text(*parts: str | None) -> str:
    return " ".join(str(part or "").strip() for part in parts if part).lower()


def _find_matches(text: str, patterns: Iterable[str]) -> list[str]:
    hits: list[str] = []
    for pattern in patterns:
        if re.search(pattern, text, flags=re.IGNORECASE):
            hits.append(pattern)
    return hits


def _has_child_context(text: str) -> bool:
    return any(token in text for token in CHILD_CONTEXT) or any(
        re.search(pattern, text, flags=re.IGNORECASE) for pattern in GENERIC_CHILD_PATTERNS
    )


def _has_non_child_signal(text: str) -> bool:
    return any(signal in text for signal in NON_CHILD_SIGNALS)


def classify_domain_text(text: str, summary: str | None = None) -> DomainClassification:
    combined = _normalize_text(text, summary)

    if not combined.strip():
        return DomainClassification(
            domain_relevant="UNCERTAIN",
            domain_category="OTHER_CHILD_PRODUCT",
            domain_reason="No review text available to assess child-product domain.",
        )

    if _has_non_child_signal(combined) and not _has_child_context(combined):
        return DomainClassification(
            domain_relevant="NO",
            domain_category="OTHER_CHILD_PRODUCT",
            domain_reason="Review wording indicates a non-child product context.",
        )

    category_scores: list[tuple[str, list[str]]] = []
    for category, patterns in CATEGORY_PATTERNS.items():
        hits = _find_matches(combined, patterns)
        if hits:
            category_scores.append((category, hits))

    # Disambiguate car booster vs dining booster
    if any(cat == "CAR_SEAT" for cat, _ in category_scores) and re.search(
        r"high chair|hook on chair|table|dining", combined
    ):
        category_scores = [
            (cat, hits) for cat, hits in category_scores if cat != "CAR_SEAT"
        ]

    if category_scores:
        category_scores.sort(key=lambda item: (-len(item[1]), item[0]))
        best_category, best_hits = category_scores[0]
        return DomainClassification(
            domain_relevant="YES",
            domain_category=best_category,
            domain_reason=(
                f"Review explicitly references {best_category.lower().replace('_', ' ')} "
                f"via wording such as '{best_hits[0]}'."
            ),
            matched_category=best_category,
        )

    if _has_child_context(combined):
        if len(combined.split()) < 12:
            return DomainClassification(
                domain_relevant="UNCERTAIN",
                domain_category="OTHER_CHILD_PRODUCT",
                domain_reason=(
                    "Child/baby context present but product category cannot be determined "
                    "from explicit review wording."
                ),
            )
        return DomainClassification(
            domain_relevant="YES",
            domain_category="OTHER_CHILD_PRODUCT",
            domain_reason=(
                "Review discusses baby/child use but no specific taxonomy keyword matched; "
                "treated as general child-product context."
            ),
        )

    if len(combined.split()) <= 8 and not _has_non_child_signal(combined):
        return DomainClassification(
            domain_relevant="UNCERTAIN",
            domain_category="OTHER_CHILD_PRODUCT",
            domain_reason="Insufficient explicit product or child-domain wording in review text.",
        )

    return DomainClassification(
        domain_relevant="NO",
        domain_category="OTHER_CHILD_PRODUCT",
        domain_reason=(
            "No explicit child/baby product or usage wording found in review text."
        ),
    )


def classify_record(record: dict) -> DomainClassification:
    review = record.get("review") or {}
    return classify_domain_text(
        str(review.get("text") or record.get("reviewText") or ""),
        review.get("summary"),
    )
