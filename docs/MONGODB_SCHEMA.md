# Miyuna — MongoDB Schema

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Design Principles

- **Multi-tenant:** All business data org-scoped
- **Tenant isolation:** Cross-tenant access → REJECT + SECURITY EVENT
- **Compound indexes:** Every query includes organizationId
- **Tenant escape test:** Mandatory acceptance test
- **Immutable snapshots:** ConfigSnapshot and published reports never mutated

## 2. Tenancy Model

```text
User
  ├── Personal Workspace (organizationId = personalOrgId)
  └── Organization Memberships[]
        └── Organization (organizationId)
              ├── Members[] (role: OWNER | ADMIN | ANALYST | VIEWER)
              ├── Products[]                  ← canonical `products`
              ├── ProductSourceMappings[]
              ├── UserExperiences[]
              ├── MarketplaceReviews[]
              ├── MarketplaceImportRuns[]
              ├── RawSourcePayloads[]
              ├── DatasetRecords[]
              ├── DatasetVersions[]           ← DRAFT only (Phase 4)
              ├── AnalysisRuns[]              ← Phase 5 IMPLEMENTED
              ├── Evidences[]                 ← P0 PR-A1
              ├── EvidenceClaimValidations[]  ← P0 PR-A1
              └── ConfigSnapshots[]
```

## 3. Core Collections

### users

```text
{
  _id: ObjectId,
  email: string,
  emailVerified: boolean,
  mfaEnabled: boolean,
  personalOrgId: ObjectId,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ email: 1 }` unique

### organizations

```text
{
  _id: ObjectId,
  name: string,
  type: PERSONAL | ORGANIZATION,
  complianceProfile: KVKK | GDPR | BOTH,
  compliancePolicyVersion: string,        // pinned published version; no runtime "latest"
  ownerId: ObjectId,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ ownerId: 1 }`, `{ type: 1, ownerId: 1 }`

### organization_members

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  userId: ObjectId,
  role: OWNER | ADMIN | ANALYST | VIEWER,
  invitedAt: Date,
  joinedAt: Date
}
```

**Indexes:** `{ organizationId: 1, userId: 1 }` unique, `{ userId: 1 }`

### ecommerce_integrations — DEFERRED / CANCELLED

Platform store API (Shopify, WooCommerce) iptal edildi (2026-09-04). Bu koleksiyon v1'de **implement edilmez**.

### import_jobs — NOT IMPLEMENTED (future agent import lifecycle)

Legacy/future agentic import run tracking. Phase 4 uses `marketplace_import_runs` for async marketplace/CSV/JSON imports.

### normalized_products — SUPERSEDED (do not use as canonical truth)

**Phase 4 canonical product truth is `products` only.**

Historical schema reference only. Normalization output persists to `products`; there is no competing canonical product collection in Phase 4.

### products — Phase 4 canonical product truth

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  name: ProductFieldMeta,
  brand: ProductFieldMeta,
  category: ProductFieldMeta,
  description: ProductFieldMeta,
  targetAge: ProductFieldMeta,
  materials: ProductFieldMeta,
  safetyWarnings: ProductFieldMeta,
  currentPrice: ProductFieldMeta,
  originalPrice: ProductFieldMeta,
  currency: ProductFieldMeta,
  seller: ProductFieldMeta,
  rating: ProductFieldMeta,
  reviewCount: ProductFieldMeta,
  attributes: ProductFieldMeta,
  imageRefs: ProductFieldMeta,
  stockStatus: ProductFieldMeta,
  sku: ProductFieldMeta,
  createdBy: ObjectId,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, createdAt: -1 }`

### product_source_mappings — Phase 4

Maps one canonical product to external marketplace/source identities.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  productId: ObjectId,
  source: HEPSIBURADA | TRENDYOL | AMAZON | N11 | OTHER | MIYUNA,
  sourceProductId: string,
  sourceUrl: string?,
  gtin, ean, upc, sku, brand, model: string?,
  matchStatus: VERIFIED | PARTIAL | UNVERIFIED,
  matchMethod: IDENTIFIER | EXPLICIT | MANUAL | FUZZY_CANDIDATE,
  confidence: number?,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, source: 1, sourceProductId: 1 }` unique

### user_experiences — Phase 4 Miyuna UGC

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  productId: ObjectId,
  userId: ObjectId,
  usageStatus: USING | USED,
  satisfactionLevel: VERY_SATISFIED | SATISFIED | NEUTRAL | UNSATISFIED | VERY_UNSATISFIED,
  rating: number?,
  issueType: DURABILITY | BREAKAGE | ... | OTHER?,
  narrative: string,
  marketplaceUrl: string?,
  sourceType: MIYUNA_UGC,
  consentRecordId: ObjectId,
  piiStatus: CLEAN | REDACTED | QUARANTINED,
  moderationStatus: PENDING | APPROVED | REJECTED,   // default PENDING
  qualityStatus: PENDING | APPROVED | LOW_QUALITY | REJECTED,
  datasetEligibility: TRAINING_APPROVED | EVAL_ONLY | ANALYSIS_ONLY | QUARANTINED | REJECTED,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, productId: 1 }`

### marketplace_reviews — Phase 4

Persisted marketplace review records (collection name in code: `marketplace_reviews`).

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  productId: ObjectId,
  productSourceMappingId: ObjectId?,
  sourceType: MARKETPLACE_REVIEW,
  source: HEPSIBURADA | TRENDYOL | AMAZON | N11 | OTHER,
  sourceUrl: string,
  sourceProductId: string,
  sourceReviewId: string?,
  rating: number?,
  reviewText: string,
  reviewDate: Date?,
  fetchedAt: Date,
  language: TR | EN | UNKNOWN,
  fingerprint: string,
  piiStatus, moderationStatus, spamStatus, duplicateStatus, qualityStatus,
  provenanceStatus: VERIFIED | PARTIAL | UNKNOWN | REJECTED,
  licenseStatus: APPROVED | RESTRICTED | UNKNOWN | REJECTED,
  usageRightsStatus: APPROVED | RESTRICTED | UNKNOWN | REJECTED,
  datasetEligibility,
  rawStorageRef: string?,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, fingerprint: 1 }` unique

**Note:** Marketplace reviews are never auto `TRAINING_APPROVED`.

### marketplace_import_runs — Phase 4 async import lifecycle

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  requestedBy: ObjectId,
  source: HEPSIBURADA | TRENDYOL | ...,
  sourceUrl: string?,
  accessMode: AUTHORIZED_API | PERMITTED_PUBLIC_FETCH | CSV_IMPORT | JSON_IMPORT | ...,
  status: PENDING | RUNNING | SUCCEEDED | PARTIAL | FAILED | REJECTED_BY_POLICY,
  importRef: string?,
  startedAt: Date?,
  finishedAt: Date?,
  recordsSeen, recordsAccepted, recordsRejected, recordsQuarantined: number,
  errorCode, errorMessage: string?,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, createdAt: -1 }`

Consumer runs inside Go API process (Redis queue); separate worker binary is out of scope for Phase 4.

### raw_source_payloads — Phase 4

Large raw payloads stored in S3/MinIO; Mongo holds metadata only.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  importRunId: ObjectId?,
  source, sourceUrl, fetchMode,
  fetchedAt: Date,
  contentType: string,
  contentHash: string,
  objectStorageRef: string,
  sizeBytes: number,
  status: string,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, importRunId: 1 }`

### dataset_records — Phase 4

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  recordType: MARKETPLACE_REVIEW | MIYUNA_UGC | PRODUCT | OFFICIAL_SOURCE | OTHER,
  sourceRecordId: ObjectId,
  productId: ObjectId?,
  normalizedPayload: object?,
  datasetEligibility,
  provenanceStatus, licenseStatus, usageRightsStatus, qualityStatus, piiStatus,
  datasetVersionId: ObjectId?,
  split: TRAIN | VALIDATION | HELD_OUT_EVAL?,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, datasetEligibility: 1 }`, `{ organizationId: 1, datasetVersionId: 1 }`

**Invariant:** `HELD_OUT_EVAL` must never enter training selection.

### dataset_versions — Phase 4 DRAFT metadata only

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  version: string,
  status: DRAFT,                         // publication workflow NOT Phase 4
  sourceCounts, sourceDistribution, categoryDistribution?,
  languageDistribution?, eligibilityDistribution: object,
  normalizerVersion, piiPolicyVersion, licensePolicyVersion: string,
  contentHash: string?,
  createdBy: ObjectId,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, createdAt: -1 }`

Publication / fine-tune export belongs to later phases (Phase 10).

### analysis_runs — IMPLEMENTED (Phase 5 Agent Core)

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  createdByUserId: ObjectId,
  productId: ObjectId,
  marketplaceImportRunId: ObjectId?,     // provenance ref only
  clientRequestId: string,               // idempotent start key
  status: PENDING | RUNNING | COMPLETED | FAILED | REJECTED,
  currentPhase: string?,
  traceId: string,
  correlationId: string,
  configSnapshotId: ObjectId,
  toolRegistryVersion: string,
  toolPolicyVersion: string,
  compliancePolicyVersion: string,
  plannerVersion: string,
  observationSchemaVersion: string,
  iterationCount: number,
  retryCount: number,
  cancellationRequested: boolean,
  cancelledByUserId: ObjectId?,
  recoveryAttempt: number?,
  leaseOwnerId: string?,
  lastHeartbeatAt: Date?,
  terminalError: string?,
  terminalReason: string?,
  startedAt: Date?,
  completedAt: Date?,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, status: 1 }`, `{ organizationId: 1, createdAt: -1 }`, unique `{ organizationId: 1, clientRequestId: 1 }`, `{ status: 1, lastHeartbeatAt: 1 }`

**Invariant:** Python orchestrator owns lifecycle; Go persists authoritative state only.

### agent_run_events — IMPLEMENTED (Phase 5)

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  runId: ObjectId,
  sequence: number,                      // monotonic per run
  phase: RUN_STARTED | COMPLIANCE_PRECHECK | PLAN_CREATED | TOOL_SELECTED |
         TOOL_EXECUTION_STARTED | TOOL_EXECUTION_COMPLETED | OBSERVATION_CREATED |
         RUN_COMPLETED | RUN_FAILED | RUN_CANCELLED,
  toolName: string?,
  status: RUNNING | COMPLETED | FAILED,
  metadata: object,
  traceId: string,
  timestamp: Date
}
```

**Indexes:** unique `{ organizationId: 1, runId: 1, sequence: 1 }`

**Invariant:** append-only; no duplicate sequence per run.

### evidences — IMPLEMENTED (P0 PR-A1)

Tenant-scoped persisted evidence. LLM-only URLs/citations in text are not evidence.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  analysisRunId: ObjectId,
  productId: ObjectId,
  claimId: string?,
  source: string,
  sourceType: string,
  claim: string,
  snippet: string?,
  reference: string?,
  reliability: number,                 // 0.0 – 1.0
  freshness: number,                   // 0.0 – 1.0
  retrievedAt: Date,
  createdAt: Date,
  updatedAt: Date,
  metadata: object?
}
```

**Indexes:** `{ organizationId: 1, analysisRunId: 1 }`, `{ organizationId: 1, productId: 1 }`, `{ organizationId: 1, claimId: 1 }`, `{ organizationId: 1, createdAt: -1 }`

**Invariant:** every query includes organizationId. Create path validates analysis run and product belong to the same organization.

### evidence_claim_validations — IMPLEMENTED (P0 PR-A1)

Persisted `evidence_validator` result. Upserted on `(organizationId, analysisRunId, claimId)`.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  analysisRunId: ObjectId,
  claimId: string,
  claimText: string,
  evidenceIds: ObjectId[],
  supportStatus: SUPPORTED | PARTIALLY_SUPPORTED | UNSUPPORTED | CONTRADICTED,
  issues: string[],
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** unique `{ organizationId: 1, analysisRunId: 1, claimId: 1 }`, `{ organizationId: 1, analysisRunId: 1 }`

### safety_findings — IMPLEMENTED (P0 PR-A2)

Tenant-scoped safety analyzer findings. CRITICAL/HIGH rows must carry evidence IDs or are persisted as `INSUFFICIENT_EVIDENCE`.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  analysisRunId: ObjectId,
  productId: ObjectId,
  type: CHOKING | SUFFOCATION | STRANGULATION | CHEMICAL | FIRE | ELECTRICAL | STRUCTURAL | AGE_SUITABILITY | HYGIENE | INJURY | RECALL | MISLEADING_CLAIM | OTHER | INSUFFICIENT_EVIDENCE,
  severity: LOW | MEDIUM | HIGH | CRITICAL,
  confidence: number,
  evidenceIds: ObjectId[],
  rationale: string,
  source: string,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, analysisRunId: 1 }`, `{ organizationId: 1, productId: 1 }`, `{ organizationId: 1, severity: 1 }`, `{ organizationId: 1, type: 1 }`

### recall_matches — IMPLEMENTED (P0 PR-A2)

Run-specific recall match signals. External CPSC/GÜBİS records are not duplicated as tenant-owned canonical data; only the match result is tenant-scoped.

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  analysisRunId: ObjectId,
  productId: ObjectId,
  source: CPSC | GUBIS,
  sourceRecordId: string,
  matched: boolean,
  confidence: number,
  method: string,
  reference: string?,
  requiresReview: boolean,
  hazard: string?,
  title: string?,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, analysisRunId: 1 }`, `{ organizationId: 1, productId: 1 }`

### tool_executions — IMPLEMENTED (Phase 5)

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  analysisRunId: ObjectId,
  toolName: string,
  toolVersion: string,
  attempt: number,
  authorizationDecision: string,
  inputHash: string,
  outputHash: string?,
  grantNonceHash: string?,               // sha256 of grant token, never plaintext
  status: PENDING | RUNNING | COMPLETED | FAILED | REJECTED | UNAVAILABLE,
  errorCode: string?,
  errorClass: string?,
  startedAt: Date?,
  completedAt: Date?,
  createdAt: Date
}
```

**Indexes:** `{ organizationId: 1, analysisRunId: 1 }`

### reports — NOT IMPLEMENTED (Phase 6+)

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  runId: ObjectId,
  configSnapshotId: ObjectId,
  claims: [],
  evidence: [],
  confidence: [],
  summary: string,
  disclaimer: string,
  generatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, runId: 1 }`, `{ organizationId: 1, generatedAt: -1 }`

**Immutable after creation.**

### config_snapshots

```text
{
  _id: ObjectId,
  organizationId: ObjectId?,              // null = global/platform-level
  modelVersion: string,
  adapterVersion: string,
  contextVersion: string,
  knowledgeVersion: string,
  toolPolicyVersion: string,
  compliancePolicyVersion: string,
  runtimeConfig: object,
  publishedBy: ObjectId,
  publishedAt: Date,
  reason: string
}
```

**Indexes:** `{ publishedAt: -1 }`, `{ organizationId: 1, publishedAt: -1 }`

### config_audit_log

```text
{
  _id: ObjectId,
  changedBy: ObjectId,
  changedAt: Date,
  reason: string,
  oldVersion: string,
  newVersion: string,
  fieldDiff: object
}
```

**Indexes:** `{ changedAt: -1 }`, `{ changedBy: 1, changedAt: -1 }`

### security_events

```text
{
  _id: ObjectId,
  organizationId: ObjectId?,
  userId: ObjectId?,
  eventType: string,
  severity: INFO | WARNING | CRITICAL,
  details: object,
  timestamp: Date
}
```

**Indexes:** `{ organizationId: 1, timestamp: -1 }`, `{ eventType: 1, timestamp: -1 }`

### devices

```text
{
  _id: ObjectId,
  userId: ObjectId,
  deviceFingerprint: string,
  platform: WEB | ELECTRON_WIN | ELECTRON_MAC,
  verified: boolean,
  lastActiveAt: Date,
  createdAt: Date
}
```

**Indexes:** `{ userId: 1, deviceFingerprint: 1 }` unique

### organization_invitations

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  email: string,
  role: OWNER | ADMIN | ANALYST | VIEWER,
  inviterId: ObjectId,
  tokenHash: string,
  status: PENDING | ACCEPTED | REVOKED | EXPIRED,
  expiresAt: Date,
  acceptedAt: Date?,
  acceptedByUserId: ObjectId?,
  createdAt: Date
}
```

**Indexes:** `{ tokenHash: 1 }` unique, `{ organizationId: 1, email: 1, status: 1 }`, TTL on `expiresAt`

### compliance_policy_versions

```text
{
  _id: ObjectId,
  organizationId: ObjectId?,              // null = platform default
  profile: KVKK | GDPR | BOTH,
  version: string,
  status: DRAFT | PUBLISHED | ARCHIVED,
  effectiveAt: Date,
  rules: object,
  reason: string?,
  createdBy: ObjectId,
  createdAt: Date,
  publishedAt: Date?
}
```

**Indexes:** `{ organizationId: 1, profile: 1, version: -1 }`, `{ organizationId: 1, status: 1, effectiveAt: -1 }`

**Published versions are immutable.**

### consents

```text
{
  _id: ObjectId,
  userId: ObjectId,
  organizationId: ObjectId?,
  purpose: REGISTRATION | ORG_MEMBERSHIP | DATA_PROCESSING,
  policyVersion: string,
  lawfulBasis: string?,
  grantedAt: Date,
  withdrawnAt: Date?,
  source: WEB | ADMIN | API,
  auditRequestId: string?
}
```

**Indexes:** `{ userId: 1, organizationId: 1, purpose: 1, withdrawnAt: 1 }`, `{ organizationId: 1, purpose: 1 }`

### compliance_events

```text
{
  _id: ObjectId,
  organizationId: ObjectId?,
  userId: ObjectId?,
  eventType: string,
  ruleId: string?,
  result: string,
  details: object,
  timestamp: Date
}
```

**Indexes:** `{ organizationId: 1, timestamp: -1 }`

## 4. Tenant Isolation Enforcement

### Repository Layer Rules

1. Every query MUST include `organizationId` filter from authenticated context
2. organizationId NEVER accepted from client input directly
3. Cross-tenant resource ID lookup → REJECT + SECURITY EVENT
4. Compound indexes on `(organizationId, ...)` for all business collections

### Tenant Escape Test (Mandatory)

Test scenarios:

- User A token → User B organizationId resource → 403 + security event
- GraphQL ID manipulation → cross-tenant reject
- Import job organizationId mismatch → reject
- Report access without org membership → reject

## 5. Deletion Model

### Personal Workspace Deletion

- User account deletion triggers personal org data purge
- Cascade: products, runs, reports, integrations, events

### Organization Deletion

- Requires ownership transfer OR explicit org deletion policy
- All org-scoped data purged
- Member access revoked
- Audit trail of deletion event retained (no PII)

### Data Separation

Personal vs org data strictly separated by organizationId scope.

## 6. Related Documents

- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [SECURITY.md](./SECURITY.md)
- [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md)
- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md)

## 7. UNRESOLVED

| Item | Status |
|------|--------|
| Soft delete vs hard delete policy | UNRESOLVED — deletion phase |
| Audit log retention after org deletion | UNRESOLVED — deletion phase |
| Global vs org-scoped ConfigSnapshot precedence | UNRESOLVED — LLM control phase |
| Personal workspace default compliance profile | **KVKK** (Phase 3) |
| Compliance event retention duration | **DEPLOYMENT_POLICY_REQUIRED** |
