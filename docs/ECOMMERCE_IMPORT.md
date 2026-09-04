# Miyuna — Product Data Import

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
> Bu doküman master prompt ile çelişemez.

## 1. Supported Input Sources

| Source | v1 Status |
|--------|-----------|
| Permitted product URL | Supported |
| CSV upload | Supported |
| JSON upload | Supported |
| Manual product entry | Supported |

### Platform API integrations — CANCELLED

**Shopify, WooCommerce ve diğer mağaza platform API entegrasyonları iptal edildi (2026-09-04).**

- `SHOPIFY_E2E_STATUS` / platform E2E gate kaldırıldı
- `EcommerceIntegration` credential modeli **implement edilmez** (v1)
- Phase 4 platform adapter işi yok
- Gelecekte marketplace/pazaryeri verisi ayrı Change Request ile değerlendirilir

## 2. Agentic Import Pipeline

```text
Source (URL | CSV | JSON | Manual)
  ↓
Fetch Policy               ← fetch_policy_checker
  ↓
Import Planner             ← import_planner
  ↓
Tool Authorization
  ↓
Fetch / Parse              ← ecommerce_fetcher (permitted URL only) or dataset_validator
  ↓
Raw Storage                ← S3/MinIO tenant-scoped prefix
  ↓
review_sampler             ← max 100 reviews (when present in source)
  ↓
product_normalizer
  ↓
dataset_validator
  ↓
price_history_analyzer
  ↓
Quality Check
  ↓
Decision: Sufficient | Re-fetch | Insufficient
```

## 3. Quality Decision Matrix

| Decision | Meaning | Action |
|----------|---------|--------|
| Sufficient | Required fields present; analysis can proceed | Continue to agent pipeline |
| Re-fetch | Partial data; retry with adjusted params | Re-fetch (within policy limits) |
| Insufficient | Cannot proceed with meaningful analysis | **No fake success** |

Insufficient durumda kullanıcıya explicit failure/insufficient status döner.

## 4. Normalized Product Schema

Kaynakta olmayan alan uydurulmaz — `missing: true`.

```text
NormalizedProduct {
  organizationId: string
  sourceType: URL | CSV | JSON | MANUAL
  sourceRef: string

  title: ProductField
  description: ProductField
  brand: ProductField
  category: ProductField
  targetAgeGroup: ProductField
  materials: ProductField[]
  safetyInfo: ProductField
  price: ProductField
  currency: ProductField
  images: ProductField[]

  reviews: ReviewSample        // max 100 when present in source
  priceHistory: PriceHistoryEntry[]

  importJobId: string
  rawStorageRef: string
  normalizedAt: ISO8601
  qualityDecision: SUFFICIENT | INSUFFICIENT | REFETCH
}
```

## 5. Review Sampling

`review_sampler` (when reviews present in imported data):

- Maximum **100 reviews** per product (v1 limit)
- Unlimited review collection = OUT OF SCOPE

```text
ReviewSample {
  totalAvailable: number
  sampled: number              // ≤ 100
  reviews: ReviewEntry[]
  samplingMethod: string
}
```

## 6. Price History

`price_history_analyzer` — deterministic analytics:

```text
PriceHistoryEntry {
  recordedAt: ISO8601
  price: number
  currency: string
  source: string
}
```

## 7. Fetch Security

Import fetch operations [SECURITY.md](./SECURITY.md) fetch security kurallarına tabidir:

- SSRF protection
- DNS rebinding prevention
- Allow/deny URL lists (permitted URLs only)
- Redirect limit
- Timeout
- Max response size
- Tenant quota
- Rate limit
- Retry with backoff
- Input sanitization
- Malicious content scan
- Audit logging

**No unrestricted generic crawler.**

## 8. Import Diff

`import_diff_generator`:

- Compare current import with previous version for same product
- Track field-level changes
- Feed into analysis as Derived Data layer

## 9. Storage Layout

```text
s3://{bucket}/{organizationId}/imports/{importJobId}/raw/
s3://{bucket}/{organizationId}/imports/{importJobId}/normalized/
```

Tenant-scoped prefixes; cross-tenant access forbidden.

## 10. Acceptance Criteria (Product Import)

| Criterion | Required |
|-----------|----------|
| CSV / JSON / manual import path | YES |
| Permitted URL fetch (policy-controlled) | YES |
| Platform API E2E (Shopify/WooCommerce) | **NO — CANCELLED** |
| Insufficient → no fake success | YES |
| Review cap 100 (when reviews in source) | YES |
| Missing fields explicit | YES |
| Fetch security controls | YES |

## 11. Related Documents

- [SECURITY.md](./SECURITY.md) — fetch security
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) — persistence
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — tool pipeline
- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md) — source data layer

## 12. UNRESOLVED

| Item | Status |
|------|--------|
| Review sampling algorithm | UNRESOLVED |
| Re-fetch max attempts | UNRESOLVED |
| Permitted URL allowlist management UI | UNRESOLVED |
| Marketplace dataset import (Hepsiburada/Trendyol vb.) | Future CR |
