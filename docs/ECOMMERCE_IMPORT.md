# Miyuna — E-Commerce Import

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.3 FROZEN  
> Bu doküman master prompt ile çelişemez.

## 1. Supported Input Sources

| Source | v1.0.3 Status |
|--------|---------------|
| Authorized Platform API (WooCommerce) | **PRIMARY — real E2E required** |
| Permitted Product URL | Supported |
| CSV upload | Supported |
| JSON upload | Supported |
| Manual product entry | Supported |

**Shopify:** OPTIONAL / FUTURE — not mandatory; not implemented in v1.0.3 (CR-004).

## 2. WooCommerce Integration (Primary)

**Platform:** WooCommerce REST API (API-first)

**CR-004:** Primary e-commerce integration. Generic ecommerce adapter boundary preserved.

**Credential model:**

```text
EcommerceIntegration {
  id: string
  organizationId: string
  platform: WOOCOMMERCE
  storeBaseUrl: string              // authorized store base URL
  credentialsEncrypted: Binary      // Consumer Key + Consumer Secret; encrypted at rest
  status: ACTIVE | REVOKED | ERROR
  createdAt: ISO8601
  updatedAt: ISO8601
  revokedAt?: ISO8601
  lastVerifiedAt?: ISO8601
}
```

**Credentials (encrypted, never exposed):**

- Consumer Key
- Consumer Secret

**Security rules:**

- Encrypted at rest
- Raw secret **never logged**
- Raw secret **never returned in GraphQL response**
- Import calls use `integrationId` only
- No raw credential input persisted outside encrypted storage
- Prefer read-only API key permissions for Phase 4

**Real E2E acceptance:**

WooCommerce E2E is **VERIFIED** only if:

1. Real WooCommerce store exists
2. REST API credentials are valid
3. Product endpoint is reachable
4. Real product payload is fetched (not mock)
5. Product is normalized into Miyuna canonical model

**Minimum acceptance probe:**

```http
GET {storeBaseUrl}/wp-json/wc/v3/products
Authorization: Basic (Consumer Key + Consumer Secret)
```

At least one real/test-store product must be fetched through the actual WooCommerce API.

**Current status:** `WOOCOMMERCE_E2E_STATUS: NOT_VERIFIED` — ecommerce phase blocked until verified.

Do not fake success when credentials or store are unavailable.

## 3. Shopify (Optional / Future)

**v1.0.3:** Not mandatory. Not implemented.

Generic adapter architecture may allow a future Shopify adapter via Change Request; no Shopify code in v1.0.3.

Historical note: v1.0.2 designated Shopify as primary; **CR-004** supersedes that decision.

## 4. Agentic Import Pipeline

```text
Source
  ↓
Adapter                    ← platform-specific (WooCommerce REST)
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
review_sampler             ← max 100 reviews (when available)
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

## 6. Normalized Product Schema (Canonical)

Kaynakta olmayan alan uydurulmaz — `missing: true`.

**Canonical fields (unchanged — WooCommerce adapter maps into this model):**

| Field | Notes |
|-------|-------|
| name | Product title |
| brand | |
| category | |
| description | |
| currentPrice | |
| originalPrice | Regular price when distinct from sale |
| currency | |
| seller | Store/vendor when available |
| rating | Average rating when available |
| reviewCount | |
| reviews | Sampled reviews; `missing: true` if unavailable |
| attributes | Platform-specific attributes |
| targetAge | |
| materials | |
| safetyWarnings | |
| imageRefs | |
| stockStatus | |
| sku | |
| sourceUrl | Canonical product URL on store |
| fetchedAt | ISO8601 fetch timestamp |

**Persistence wrapper (MongoDB):**

```text
NormalizedProduct {
  organizationId: string
  sourceType: WOOCOMMERCE | URL | CSV | JSON | MANUAL
  sourceRef: string
  integrationId?: string

  fields: { [fieldName]: { value: any, missing: boolean } }

  reviewSample: ReviewSample        // max 100 when available
  priceHistory: PriceHistoryEntry[]

  importJobId: string
  rawStorageRef: string
  normalizedAt: ISO8601
  qualityDecision: SUFFICIENT | INSUFFICIENT | REFETCH
}
```

### WooCommerce field mapping (adapter responsibility)

WooCommerce REST product fields adapt into canonical fields above. Examples:

- `name` ← WooCommerce `name`
- `currentPrice` ← `price` / sale price
- `originalPrice` ← `regular_price` when applicable
- `sku` ← `sku`
- `stockStatus` ← `stock_status`
- `imageRefs` ← `images[].src`
- `category` ← `categories[].name` (primary or joined)
- `attributes` ← `attributes[]`

Missing source values → `missing: true`. Do not invent data.

## 7. Review Data

Do not assume WooCommerce core product response always contains full review content.

- If review endpoint/API support exists and credentials permit: use adapter + `review_sampler`
- If unavailable: `reviews = missing`
- **WooCommerce E2E acceptance does not depend on reviews**

`review_sampler` (when reviews available):

- Maximum **100 reviews** per product (v1.0.3 limit)
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
- Allowed store URL validation (connected/authorized stores only)
- Redirect limit
- Timeout
- Max response size
- Tenant quota
- Rate limit
- Retry with backoff
- Credential encryption
- Response sanitization
- Malicious content scan
- Audit logging

**No unrestricted generic crawler.** Only connected/authorized WooCommerce stores are supported.

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
| Real WooCommerce REST API E2E | YES |
| At least one real product fetched via API | YES |
| Product normalized to canonical model | YES |
| Encrypted credentials | YES |
| Mock-only acceptance | NO (fails) |
| Insufficient → no fake success | YES |
| Review cap 100 (when reviews available) | YES |
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
| WooCommerce webhook vs poll strategy | UNRESOLVED — Phase 4 |
| Shopify adapter (optional/future) | OUT — CR-004; future CR if needed |
