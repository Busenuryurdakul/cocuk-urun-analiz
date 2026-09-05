# Gold Pilot Domain Audit

**Seed:** 42

## Counts

- **TOTAL:** 100
- **DOMAIN_RELEVANT_YES:** 100
- **DOMAIN_RELEVANT_NO:** 0
- **DOMAIN_RELEVANT_UNCERTAIN:** 0
- **REPLACEMENTS_MADE:** 4

## Category Distribution

- BABY_CARE_ACCESSORIES: 2
- BOTTLE_FEEDING: 15
- CARRIER: 7
- CAR_SEAT: 4
- CRIB: 2
- DIAPERING: 9
- HIGH_CHAIR: 3
- OTHER_CHILD_PRODUCT: 42
- SAFETY_GATE: 1
- SLEEP_PRODUCTS: 3
- STROLLER: 2
- TEETHING: 1
- TOYS: 9

## Sample Quality Note

Category `OTHER_CHILD_PRODUCT` represents 42% of the pilot (42 records). Consider manual spot-check for over-concentration.

## Replacements Applied (seed 42)

Four pending records classified `domainRelevant=NO` were replaced from the 56,707-record corpus:

| goldRecordId | Original ASIN | New ASIN | Notes |
|--------------|---------------|----------|-------|
| gold_pilot_011 | B002VHBU2W | B0057LUMX2 | Diaper pail accessory → baby-on-board car magnet |
| gold_pilot_031 | B003AM86QU | B000066665 | Nursing pillow (weak signal) → infant bath tub |
| gold_pilot_049 | B002VHBU2W | B0038JDD2C | Diaper pail liner → car seat strap covers |
| gold_pilot_082 | B000NAOH1A | B001PVAXZK | Bumbo tray accessory → baby bottles |

Human-reviewed records (`gold_pilot_001`, `gold_pilot_002`) were preserved unchanged.

Original pilot CSV/JSONL were **not** overwritten. Use `miyuna_gold_pilot_100_domain_corrected.csv` for continued review after replacement.
