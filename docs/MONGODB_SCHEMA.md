# Miyuna — MongoDB Schema

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
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
              ├── Products[]
              ├── ImportJobs[]
              ├── AnalysisRuns[]
              ├── EcommerceIntegrations[]
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

### ecommerce_integrations

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  platform: SHOPIFY,
  storeDomain: string,
  credentialsEncrypted: Binary,
  status: ACTIVE | REVOKED | ERROR,
  lastVerifiedAt: Date,
  createdAt: Date,
  updatedAt: Date
}
```

**Indexes:** `{ organizationId: 1, platform: 1 }`, `{ organizationId: 1, status: 1 }`

### import_jobs

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  sourceType: SHOPIFY | URL | CSV | JSON | MANUAL,
  integrationId: ObjectId?,
  sourceRef: string,
  status: PENDING | RUNNING | SUFFICIENT | INSUFFICIENT | FAILED,
  rawStorageRef: string,
  qualityDecision: SUFFICIENT | INSUFFICIENT | REFETCH,
  createdBy: ObjectId,
  createdAt: Date,
  completedAt: Date?
}
```

**Indexes:** `{ organizationId: 1, status: 1 }`, `{ organizationId: 1, createdAt: -1 }`

### normalized_products

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  importJobId: ObjectId,
  sourceType: string,
  sourceRef: string,
  fields: { [fieldName]: { value: any, missing: boolean } },
  reviewSample: { totalAvailable: number, sampled: number, reviews: [] },
  priceHistory: [{ recordedAt: Date, price: number, currency: string }],
  normalizedAt: Date
}
```

**Indexes:** `{ organizationId: 1, sourceRef: 1 }`, `{ organizationId: 1, importJobId: 1 }`

### analysis_runs

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  productId: ObjectId,
  configSnapshotId: ObjectId,
  status: PENDING | RUNNING | COMPLETED | FAILED | REJECTED,
  workerRuntime: LLM-1 | LLM-2,
  reviewerRuntime: LLM-1 | LLM-2,
  runPattern: RUN_A | RUN_B,
  startedAt: Date,
  completedAt: Date?,
  reportId: ObjectId?
}
```

**Indexes:** `{ organizationId: 1, status: 1 }`, `{ organizationId: 1, startedAt: -1 }`

**Invariant:** `workerRuntime != reviewerRuntime`

### agent_run_events

```text
{
  _id: ObjectId,
  organizationId: ObjectId,
  runId: ObjectId,
  phase: string,
  toolName: string?,
  status: RUNNING | COMPLETED | FAILED,
  metadata: object,
  timestamp: Date
}
```

**Indexes:** `{ organizationId: 1, runId: 1, timestamp: 1 }`

### reports

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
