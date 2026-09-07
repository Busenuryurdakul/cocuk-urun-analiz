# Miyuna — Compliance Engine

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Core Rule

**Compliance Engine ALWAYS ON — kapatılamaz.**

Hiçbir admin action, config değişikliği veya runtime flag compliance engine'i devre dışı bırakamaz.

## 2. Policy Profiles

Versioned policy profilleri:

| Profile | Scope |
|---------|-------|
| KVKK | Türkiye Kişisel Verilerin Korunması Kanunu |
| GDPR | EU General Data Protection Regulation |
| BOTH | Combined KVKK + GDPR enforcement |

Policy profilleri **versioned** olarak değiştirilebilir. Değişiklik ConfigSnapshot üzerinden publish edilir.

## 3. Compliance Check Points

Compliance engine üç katmanda çalışır:

```text
┌─────────────────────────────────────────────────────────┐
│  1. Input Compliance Pre-Check                          │
│     Before agent pipeline starts                        │
│     Tools: policy_evaluator, compliance_checker         │
├─────────────────────────────────────────────────────────┤
│  2. Runtime Compliance (Tool Loop)                      │
│     During tool execution                               │
│     Tools: pii_redactor, fetch_policy_checker           │
├─────────────────────────────────────────────────────────┤
│  3. Output Compliance Validation                        │
│     Before report persist/delivery                      │
│     Tools: compliance_checker, pii_redactor             │
└─────────────────────────────────────────────────────────┘
```

## 4. Policy Version Lifecycle

```text
Draft Policy → Review → Publish as ConfigSnapshot
  → Active for new runs
  → Previous version archived (immutable)
  → Rollback available via new snapshot from archived version
```

Audit: changedBy, changedAt, reason, oldVersion, newVersion, fieldDiff

## 5. PII Handling

`pii_redactor` tool:

- Detect PII in input, intermediate, and output data
- Redact before persistence where policy requires
- Log redaction events in audit trail (without exposing PII)

**Rules:**

- PII never logged in plaintext
- Redaction decisions tied to active compliance policy version
- Cross-tenant PII leakage → REJECT + SECURITY EVENT

## 6. Data Subject Rights (Architecture)

Platform architecture must support (implementation in auth/data phases):

| Right | Mechanism |
|-------|-----------|
| Access | `exportMyData` JSON export (schema `1.0.0`) |
| Rectification | Profile/data update flows |
| Erasure | `requestAccountDeletion` + `confirmAccountDeletion` |
| Portability | Structured JSON export (`UserDataExport.payload`) |
| Restriction | Tenant-scoped data isolation |

### 6.1 Account deletion policy (PR-B)

- **Request:** authenticated user receives a single-use, expiring 6-digit code (hashed at rest).
- **Confirm:** revokes all sessions/devices, removes memberships, anonymizes user PII.
- **Owner guard:** if user is `OWNER` of a team org with other active members → `OWNERSHIP_TRANSFER_REQUIRED` (no auto owner assignment).
- **Sole-member org:** personal or team org where user is the only member → membership removed and organization deleted.
- **Analysis records:** org/business records retained; `createdByUserId` / `cancelledByUserId` pseudonymized to `AnonymizedActorUserID`.
- **Retained for audit:** `security_events`, compliance audit trails (no raw deletion codes, passwords, or secrets).

### 6.2 Export scope (PR-B)

User export includes only the authenticated user's data: profile metadata, consents, memberships (no other member PII), devices, own analysis activity metadata, activity log entries. Excludes password hashes, TOTP secrets, token hashes, and other credentials.

Detay: [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md), [SECURITY.md](./SECURITY.md)

## 7. Disclaimer Requirements

Rapor footer **disclaimer zorunlu:**

- Platform decision-support / risk-assessment system'dır
- Resmi güvenlik sertifikasyonu iddiası yok
- Hukuki uygunluk garantisi yok
- Regülasyon onayı iddiası yok
- "Kesinlikle güvenli/güvensiz" hüküm verilmez

## 8. Language & Claims Policy

| Claim Type | Requirement |
|------------|-------------|
| Safety verdict | Evidence required; otherwise UNVERIFIED |
| Regulatory compliance | Never asserted without verified source |
| Scientific certainty | Forbidden without verified evidence |
| Certification | OUT OF SCOPE v1.0.2 |

## 9. Compliance ↔ Evidence Integration

Compliance validation evidence model ile entegre:

- Claim without evidence → blocked or marked UNVERIFIED
- Evidence with CONTRADICTED status → flagged in report
- Compliance policy version recorded in every report metadata

Detay: [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md)

## 10. Tenant Compliance Configuration

Organization-level compliance profile selection:

```text
Organization {
  complianceProfile: KVKK | GDPR | BOTH
  compliancePolicyVersion: string  // pinned published version (e.g. 1.0.0)
}
```

**Policy resolution (Phase 3):**

1. Organization-specific published policy for pinned version (ORG_OVERRIDE)
2. Platform published default for same pinned version (PLATFORM_DEFAULT)
3. Runtime `"latest"` resolution is **forbidden**

Personal workspace default profile: **KVKK** (`PERSONAL_DEFAULT_COMPLIANCE_PROFILE`)

## 11. Phase 5 Analysis Compliance (IMPLEMENTED)

Phase 5 enforces compliance at three verified points for deterministic analysis runs:

| Point | When | Mechanism |
|-------|------|-----------|
| Pre-run | `startAgentRun` | Go `Compliance.Engine` + consent check before orchestrator dispatch |
| Pre-tool | `/internal/agent/v1/tools/authorize` | Registry availability, RBAC, input hash, grant binding |
| Post-tool | `/internal/agent/v1/tools/execute` | Output schema validation + compliance rejection before observation/event persist |

**Observation/event metadata:** Sensitive keys rejected; oversized metadata blocked (`maxEventMetadataBytes`).

**Provenance:** Cross-tenant `marketplaceImportRunId` references rejected at start.

**Phase 6 LLM output compliance validation:** IMPLEMENTED — Go LLM Gateway applies platform Compliance Reflex (KVKK/GDPR/BOTH) on every provider call and validates output text before returning results.

| Point | When | Mechanism |
|-------|------|-----------|
| LLM pre-call | `Gateway.Complete` | Bypass/injection detection, `compliance.RedactPII`, scope reflex instruction injection |
| LLM post-call | After provider response | `ValidateOutputText` + compliance audit (`llm_complete_output`) |

**Phase 6+ report/footer compliance validation (non-LLM persist path):** DEFERRED — not claimed as implemented.

## 12. Audit Requirements

All compliance events logged:

- Policy evaluation results (pass/fail + rule ID)
- PII redaction events (count, field types — no values)
- Output validation blocks
- Profile changes

Retention policy: **DEPLOYMENT_POLICY_REQUIRED** — concrete duration defined at deployment/legal policy time, not invented in code.

## 13. Related Documents

- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md)
- [SECURITY.md](./SECURITY.md)
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md)
- [LLM_ROUTING.md](./LLM_ROUTING.md) — compliance policy versions in ConfigSnapshot

## 14. UNRESOLVED

| Item | Status |
|------|--------|
| Default compliance profile for personal workspace | **KVKK** (Phase 3 canonical) |
| Audit log retention period | **DEPLOYMENT_POLICY_REQUIRED** |
| Data export format specification | UNRESOLVED — deletion phase |
