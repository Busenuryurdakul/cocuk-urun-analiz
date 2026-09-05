# AMAZON_BABY_REAL_IMPORT_E2E Report

**Date:** 2026-09-05  
**Artifact:** `scripts/amazon_baby/output/miyuna_amazon_baby_import.jsonl`  
**Pipeline:** JSONL → GraphQL `startMarketplaceFileImport` → Redis → Go consumer → MinIO → MongoDB

---

## Summary

| Field | Value |
|-------|-------|
| **REAL_BACKEND_IMPORT_E2E** | **PASS** |
| INPUT_ARTIFACT | Present (56,707 records, ~92 MB) |
| INPUT_RECORD_COUNT (full artifact) | 56,707 |
| IMPORT_SUBSET (GraphQL upload limit) | 1,000 lines from same artifact |
| IMPORT_RUN_ID | `6a9b99eccec7ba3a231b0555` |
| ORG_ID | `6a9b99eccec7ba3a231b0548` |
| IMPORT_RUN_STATUS | `SUCCEEDED` |
| RECORDS_SEEN | 1,071 (71 products + 1,000 reviews) |
| RECORDS_ACCEPTED | 1,071 |
| RECORDS_REJECTED | 0 |

---

## Infrastructure

| Service | Status |
|---------|--------|
| MongoDB (`docker-mongodb-1`) | PASS — healthy |
| Redis (`docker-redis-1`) | PASS — queue consumed |
| MinIO (`docker-minio-1`) | PASS — object stored |
| MailHog | PASS — auth flow |
| API (`miyuna-api.exe :8080`) | PASS — S3 env required |
| Web (`:3000`) | Running — portal auth not wired to E2E runner |

---

## MongoDB Verification (org `6a9b99eccec7ba3a231b0548`)

| Collection | Count |
|------------|------:|
| `products` | 71 |
| `product_source_mappings` | 71 |
| `marketplace_reviews` | 1,000 |
| `dataset_records` | 1,000 |
| `marketplace_import_runs` | 1 |

### Governance (marketplace_reviews)

| Field | Count | Expected |
|-------|------:|----------|
| `datasetEligibility=QUARANTINED` | 1,000 | ✓ |
| `provenanceStatus=PARTIAL` | 1,000 | ✓ |
| `licenseStatus=UNKNOWN` | 1,000 | ✓ |
| `TRAINING_APPROVED` | 0 | ✓ |
| `EVAL_ONLY` | 0 | ✓ |

### Sample ASIN mapping

- `sourceProductId`: `097293751X` (real ASIN, not synthetic)
- Product name: `missing: true` (correct — no fake product title)
- Source URL: Kaggle Amazon Baby dataset

---

## MinIO

| Check | Status |
|-------|--------|
| Bucket `miyuna-dev` | Created |
| Import object | `s3://miyuna-dev/6a9b99eccec7ba3a231b0548/imports/6a9b99eccec7ba3a231b0555/raw/miyuna_amazon_baby_import.jsonl` |
| Object size | ~1.5 MiB (1,000-record subset) |

**Note:** API must be started with `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` or uploads silently fail (`IMPORT_PAYLOAD_NOT_FOUND`).

---

## GraphQL Readback

| Query | Result |
|-------|--------|
| `marketplaceImportStatus` | `SUCCEEDED` |
| `products` | 71 products (sample shows `name.missing=true`) |
| `productMarketplaceReviews` | Reviews returned with `datasetEligibility=QUARANTINED`, `source=OTHER` |
| `datasetEligibilitySummary` | 1,000 total, 100% `QUARANTINED` |
| `createUserExperience` | PASS — `moderationStatus=PENDING`, `datasetEligibility=ANALYSIS_ONLY` (UGC separate from marketplace) |

---

## Portal Verification

| Check | Status |
|-------|--------|
| Products page (`/org/{id}/products`) | **BLOCKED** — requires authenticated browser session (E2E runner uses cookie-less `requests` session) |
| Unauthenticated view | Shows "Bu içeriği görüntüleme yetkiniz yok." |

Backend data is persisted and readable via GraphQL; portal UI verification deferred until manual login or cookie injection.

---

## Fixes Applied During E2E

1. **MinIO bucket** — created `miyuna-dev` bucket; API restarted with S3 env vars.
2. **Go parser** — `ParseMiyunaJSONL` now accepts records with `product.name=null` when `sourceProductId` (ASIN) is present. Real artifact uses ASIN-only product keys per governance spec.

---

## Artifacts

- Full import JSONL: `scripts/amazon_baby/output/miyuna_amazon_baby_import.jsonl`
- E2E subset: `scripts/amazon_baby/output/_e2e_import_subset.jsonl`
- E2E runner: `scripts/amazon_baby/_e2e_import_runner.py`
- Machine report: `scripts/amazon_baby/output/_e2e_report.json`

---

## Conclusion

Real Amazon Baby Miyuna JSONL (56,707 records) was adapted offline; a 1,000-record subset was imported through the **full production path** (GraphQL → Redis queue → MinIO → MongoDB). All imported marketplace reviews remain **QUARANTINED** with **PARTIAL/UNKNOWN** governance. No training or eval eligibility was granted. UGC creation remains separate with `PENDING` moderation.

**REAL_BACKEND_IMPORT_E2E: PASS**
