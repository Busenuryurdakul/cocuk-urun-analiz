# Miyuna — Amazon Baby Dataset V1

## Source

- **Identifier:** `KAGGLE_AMAZON_BABY_ROOPALIK`
- **Reference:** https://www.kaggle.com/datasets/roopalik/amazon-baby-dataset

Kaggle hosting does **not** prove training or evaluation rights. Default governance treats this source as **QUARANTINED**.

## Corpus size and backend import

| Scope | Records | Notes |
|-------|---------|-------|
| **Amazon Baby source corpus** | **56,707** | Full adapted JSONL from real CSV |
| **Backend E2E import** | **1,000** | Subset only — GraphQL/base64 payload limit |
| **Gold Pilot** | **100** | Domain-corrected manual/AI-assisted review set |

**Do not claim** the full 56,707-record corpus was imported into the backend. E2E verified: **71 products**, **1,000 marketplace reviews**, **1,000 dataset records**, **1,000 quarantined**.

Large generated artifact (local only, gitignored): `scripts/amazon_baby/output/miyuna_amazon_baby_import.jsonl` (~92 MB).

## Real CSV schema

Primary real dataset file: `reviews_Baby_5_final_dataset.csv`

| Raw column | Canonical mapping |
|------------|-------------------|
| `asin` | `sourceProductId` (real ASIN) |
| `reviewText` | `review.text` |
| `overall` | `review.rating` |
| `summary` | `review.summary` (optional) |
| `reviewTime` | `review.date` (optional) |
| `helpful_num` / `helpful_den` | `review.helpful` (optional) |

Not persisted: `reviewerName`, `reviewerID`

Legacy schema (`name`, `review`, `rating`) remains supported for older samples.

## Pre-labels vs gold labels

Rule-based fields (`sentiment`, `issueType`, `safetyRelatedObservation`, `qualitySignal`) are **PRE-LABELS** with:

- `labelReviewStatus = PENDING_HUMAN_REVIEW` (initial)
- `labelProvenance = RULE` (initial)

They are **not** expert-reviewed gold truth until finalized through human or AI-assisted review.

## Default governance (unchanged by Gold finalization)

```
source: KAGGLE_AMAZON_BABY_ROOPALIK
provenanceStatus: PARTIAL
licenseStatus: UNKNOWN
usageRightsStatus: UNKNOWN
datasetEligibility: QUARANTINED
trainingAllowed: false
evaluationAllowed: false
```

Gold finalization does **not** change governance. No `TRAINING_APPROVED`, `EVAL_ONLY`, or `LICENSED_DATASET`.

## Gold Pilot V1 (executed)

Workflow completed for 100 domain-audited records:

1. Raw Amazon Baby reviews → deterministic pre-labels
2. Domain audit: **100/100** child/baby-domain relevant (`YES`)
3. AI-assisted pre-review: 98 suggestions for pending records
4. Manual review helper + bulk AI finalization
5. Truthful provenance applied

### Review provenance (current truth)

| Category | Count | `labelReviewStatus` | `labelProvenance` |
|----------|-------|---------------------|-------------------|
| Genuine human review | **2** | `HUMAN_REVIEWED` | `HUMAN` |
| AI-assisted finalized | **98** | `AI_ASSISTED_FINALIZED` | `AI_ASSISTED` |
| Rejected | **0** | — | — |

Genuine human records: `gold_pilot_001`, `gold_pilot_002`.

Complete label coverage: **100/100**. Replacement ASINs preserved: 011→B0057LUMX2, 031→B000066665, 049→B0038JDD2C, 082→B001PVAXZK.

### Export artifacts (generated, gitignored)

| File | Purpose |
|------|---------|
| `miyuna_gold_pilot_100_domain_corrected_reviewed.jsonl` | Active reviewed/autosave |
| `miyuna_gold_pilot_ai_assisted_finalized.jsonl` | Truthful finalized export (100 records) |
| `miyuna_gold_pilot_ai_assisted_finalized.csv` | CSV equivalent |

Legacy misleading name `miyuna_gold_pilot_approved.jsonl` must not be used.

## Fine-tuning and Phase 5

- **Fine-tuning:** NOT STARTED
- **Training rights:** NOT APPROVED
- **Phase 5:** NOT STARTED

## Backend import

Use generic GraphQL `startMarketplaceFileImport` with:

- `accessMode: CSV_IMPORT` or `JSON_IMPORT`
- `source: OTHER`
- File: subset or full `miyuna_amazon_baby_import.jsonl` (respect payload limits)

Persisted collections: `products`, `product_source_mappings`, `marketplace_reviews`, `dataset_records`.

## Scripts

| Path | Purpose |
|------|---------|
| `scripts/amazon_baby/miyuna_adapter.py` | Core adaptation logic |
| `scripts/amazon_baby/prepare_finetune.py` | CLI wrapper (PREPARE_ONLY) |
| `scripts/amazon_baby/miyuna_amazon_baby_kaggle.ipynb` | Kaggle notebook |
| `scripts/amazon_baby/miyuna_amazon_baby_colab.ipynb` | Google Colab notebook |
| `scripts/amazon_baby/run_colab_prepare.py` | Local Colab-flow helper |
| `scripts/amazon_baby/gold/fix_gold_provenance.py` | Apply truthful human/AI provenance |
| `scripts/amazon_baby/gold/export_finalized_gold.py` | Export finalized Gold pilot |

Training output generation is **blocked** until governance approval is explicitly recorded.
