# Miyuna — Compliance Engine

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
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
| Access | User/org data export |
| Rectification | Profile/data update flows |
| Erasure | Account & org deletion (see §29 master prompt) |
| Portability | Structured export format |
| Restriction | Tenant-scoped data isolation |

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

## 11. Audit Requirements

All compliance events logged:

- Policy evaluation results (pass/fail + rule ID)
- PII redaction events (count, field types — no values)
- Output validation blocks
- Profile changes

Retention policy: **DEPLOYMENT_POLICY_REQUIRED** — concrete duration defined at deployment/legal policy time, not invented in code.

## 12. Related Documents

- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md)
- [SECURITY.md](./SECURITY.md)
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md)
- [LLM_ROUTING.md](./LLM_ROUTING.md) — compliance policy versions in ConfigSnapshot

## 13. UNRESOLVED

| Item | Status |
|------|--------|
| Default compliance profile for personal workspace | **KVKK** (Phase 3 canonical) |
| Audit log retention period | **DEPLOYMENT_POLICY_REQUIRED** |
| Data export format specification | UNRESOLVED — deletion phase |
