# Miyuna — E-Commerce Import

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
> Bu doküman master prompt ile çelişemez.

## 1. Supported Input Sources

| Source | v1.0.2 Status |
|--------|---------------|
| Authorized Platform API (Shopify) | **PRIMARY — E2E required** |
| Permitted Product URL | Supported |
| CSV upload | Supported |
| JSON upload | Supported |
| Manual product entry | Supported |

## 2. Shopify Integration (Primary)

**Platform:** Shopify Admin GraphQL API (API-first)

**Credential model:**

```text
EcommerceIntegration {
  id: string
  organizationId: string
  platform: SHOPIFY
  credentialsEncrypted: Binary   // encrypted at rest; API key, store domain, etc.
  status: ACTIVE | REVOKED | ERROR
  createdAt: ISO8601
  lastVerifiedAt?: ISO8601
}
```

- `credentialsEncrypted`: encrypted at rest
- Raw secret **response/log içine girmez**
- Raw secret **GraphQL response'da dönmez**
- Import çağrıları yalnızca `integrationId` kullanır
- Mock acceptance **geçmez**
- Real E2E verification zorunlu

**Current status:** `SHOPIFY_E2E_STATUS: NOT_VERIFIED` — FAZ 4 blocked until verified.

## 3. WooCommerce

**v1.0.2:** Adapter interface only — **implement edilmez.**

Adapter interface tanımı future marketplace expansion için hazırlanır; v1'de kod yazılmaz.

## 4. Agentic Import Pipeline

```text
Source
  ↓
Adapter                    ← platform-specific (Shopify GraphQL)
  ↓
Fetch Policy               ← fetch_policy_checker
  ↓
Import Planner             ← import_planner
  ↓
Tool Authorization
  ↓
Fetch                      ← ecommerce_fetcher
  ↓
Raw Storage                ← S3/MinIO tenant-scoped prefix
  ↓
review_sampler             ← max 100 reviews
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

## 5. Quality Decision Matrix

| Decision | Meaning | Action |
|----------|---------|--------|
| Sufficient | Required fields present; analysis can proceed | Continue to agent pipeline |
| Re-fetch | Partial data; retry with adjusted params | Re-fetch (within policy limits) |
| Insufficient | Cannot proceed with meaningful analysis | **No fake success** |

Insufficient durumda kullanıcıya explicit failure/insufficient status döner.

## 6. Normalized Product Schema

Kaynakta olmayan alan uydurulmaz — `missing: true`.

**Core fields (normalized):**

```text
NormalizedProduct {
  organizationId: string
  sourceType: SHOPIFY | URL | CSV | JSON | MANUAL
  sourceRef: string
  integrationId?: string
  
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
  
  reviews: ReviewSample        // max 100
  priceHistory: PriceHistoryEntry[]
  
  importJobId: string
  rawStorageRef: string
  normalizedAt: ISO8601
  qualityDecision: SUFFICIENT | INSUFFICIENT | REFETCH
}
```

## 7. Review Sampling

`review_sampler`:

- Maximum **100 reviews** per product (v1.0.2 limit)
- Sampling strategy: UNRESOLVED — Phase 4 (likely recent + rating distribution)
- Unlimited review collection = OUT OF SCOPE

```text
ReviewSample {
  totalAvailable: number
  sampled: number              // ≤ 100
  reviews: ReviewEntry[]
  samplingMethod: string
}
```

## 8. Price History

`price_history_analyzer` — deterministic analytics:

```text
PriceHistoryEntry {
  recordedAt: ISO8601
  price: number
  currency: string
  source: string
}
```

Deterministic calculations (not LLM):

- Min/max/avg over period
- Trend direction
- Price/performance signal flags

## 9. Fetch Security

Import fetch operations [SECURITY.md](./SECURITY.md) fetch security kurallarına tabidir:

- SSRF protection
- DNS rebinding prevention
- Allow/deny URL lists
- Redirect limit
- Timeout
- Max response size
- Tenant quota
- Rate limit
- Retry with backoff
- Credential encryption
- Input sanitization
- Malicious content scan
- Audit logging

## 10. Import Diff

`import_diff_generator`:

- Compare current import with previous version for same product
- Track field-level changes
- Feed into analysis as Derived Data layer

## 11. Storage Layout

```text
s3://{bucket}/{organizationId}/imports/{importJobId}/raw/
s3://{bucket}/{organizationId}/imports/{importJobId}/normalized/
```

Tenant-scoped prefixes; cross-tenant access forbidden.

## 12. Acceptance Criteria (E-Commerce)

| Criterion | Required |
|-----------|----------|
| Real Shopify Admin GraphQL E2E | YES |
| Encrypted credentials | YES |
| Mock-only acceptance | NO (fails) |
| Insufficient → no fake success | YES |
| Review cap 100 | YES |
| Missing fields explicit | YES |
| Fetch security controls | YES |

## 13. Related Documents

- [SECURITY.md](./SECURITY.md) — fetch security
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) — persistence
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — tool pipeline
- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md) — source data layer

## 14. UNRESOLVED

| Item | Status |
|------|--------|
| Review sampling algorithm | UNRESOLVED — Phase 4 |
| Re-fetch max attempts | UNRESOLVED — Phase 4 |
| Permitted URL allowlist management UI | UNRESOLVED — Phase 4 |
| Shopify webhook vs poll strategy | UNRESOLVED — Phase 4 |
