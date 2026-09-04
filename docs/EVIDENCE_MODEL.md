# Miyuna — Evidence Model

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
> Bu doküman master prompt ile çelişemez.

## 1. Purpose

Miyuna evidence-based decision-support platformdur. Ciddi iddialar kanıt olmadan kesin dil kullanamaz.

**Tagline:** Skoru değil, skorun kanıtını göster.

## 2. Data Layers

```text
Source Data
  ↓
Derived Data          ← deterministic transforms, analyzers
  ↓
Agent Assessment      ← LLM worker/reviewer output
  ↓
Evidence              ← validated claim-evidence pairs
  ↓
Confidence            ← aggregated verification status
```

## 3. Core Entities

### Claim

```text
Claim {
  id: string
  text: string
  category: SAFETY | AGE | MATERIAL | PRICE | MARKET | QUALITY | OTHER
  severity: INFO | WARNING | CRITICAL
  createdBy: WORKER | REVIEWER | ANALYZER
  runId: string
}
```

### Evidence

```text
Evidence {
  id: string
  claimId: string
  source: SourceReference
  sourceVersion: string
  excerpt: string              // relevant data excerpt
  derivationMethod: string     // tool or rule that produced this
  collectedAt: ISO8601
}
```

### SourceReference

```text
SourceReference {
  type: ECOMMERCE_API | URL | CSV | JSON | MANUAL | VERIFIED_KNOWLEDGE | DETERMINISTIC_RULE
  identifier: string           // productId, url, fileId, ruleId
  fetchedAt?: ISO8601
  integrationId?: string       // for ecommerce sources
}
```

### Confidence

```text
Confidence {
  claimId: string
  score: float                 // 0.0 – 1.0
  verificationStatus: VerificationStatus
  reviewerNotes?: string
}
```

## 4. Verification Status

| Status | Meaning |
|--------|---------|
| VERIFIED | Evidence fully supports claim; reviewer approved |
| PARTIAL | Some evidence supports claim; gaps exist |
| UNVERIFIED | Insufficient or no evidence |
| CONTRADICTED | Evidence contradicts claim |

## 5. Evidence Validation Rules

`evidence_validator` tool enforces:

1. Every CRITICAL claim must have ≥1 evidence reference
2. Evidence must trace to a valid SourceReference
3. SourceVersion must match data at time of analysis
4. Derived evidence must document derivationMethod
5. Missing source fields → claim cannot be VERIFIED (max PARTIAL or UNVERIFIED)
6. Reviewer CHALLENGE on unsupported claims → worker must add evidence or downgrade

## 6. Missing Data Policy

Kaynakta olmayan alanlar **uydurulmaz:**

```text
ProductField {
  name: string
  value: any | null
  missing: true | false       // explicit missing flag
}
```

Missing field ile ilgili claim → maximum status: **UNVERIFIED**

## 7. Safety Evidence Model

`safety_analyzer` output:

```text
SafetySignal {
  signal: string
  status: VERIFIED | PARTIAL | UNVERIFIED | CONTRADICTED
  evidence: Evidence[]
  ruleId?: string              // deterministic rule reference
  knowledgeBaseRef?: string    // verified knowledge reference
}
```

LLM safety açıklaması evidence'ye bağlıdır. LLM verdict engine değildir.

## 8. Review Evidence Flow

```text
Worker LLM produces claims + draft evidence
  ↓
Reviewer LLM validates:
  APPROVE   → claims + evidence accepted
  CHALLENGE → worker revises (max 2 cycles)
  REJECT    → run fails
  ↓
Evidence Assembler consolidates final evidence set
  ↓
Output Compliance Validation
  ↓
Report with evidence appendix
```

## 9. Report Structure (Evidence Section)

Her rapor evidence appendix içerir:

```text
Report {
  summary: string
  claims: Claim[]
  evidence: Evidence[]
  confidence: Confidence[]
  configSnapshotId: string
  disclaimer: string           // mandatory footer
  generatedAt: ISO8601
}
```

## 10. Admin Demo: OLD vs NEW Report

ConfigSnapshot değişikliği sonrası:

- OLD REPORT: previous snapshot evidence set (immutable)
- NEW REPORT: new snapshot evidence set
- UI side-by-side comparison

Aynı ürün, farklı config → farklı evidence/confidence mümkün.

## 11. Prohibited Patterns

| Pattern | Status |
|---------|--------|
| Claim without any evidence marked VERIFIED | FORBIDDEN |
| Fabricated source references | FORBIDDEN |
| LLM-only safety verdict | FORBIDDEN |
| "Kesinlikle güvenli/güvensiz" language | FORBIDDEN |

## 12. Related Documents

- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — evidence assembly in pipeline
- [COMPLIANCE.md](./COMPLIANCE.md) — output validation
- [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md) — source data layer
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) — persistence schema

## 13. UNRESOLVED

| Item | Status |
|------|--------|
| Confidence score calculation formula | UNRESOLVED — Phase 1+ |
| Verified knowledge base format | UNRESOLVED — safety phase |
| Evidence retention vs source data retention | UNRESOLVED — Phase 1+ |
