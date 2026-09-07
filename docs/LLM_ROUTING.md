# Miyuna — LLM Routing & Control

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Two-LLM Architecture

Miyuna v1.0.2'de **tam 2 LLM runtime** bulunur. Üçüncü reviewer LLM oluşturulmaz.

| Runtime | Persona | Characteristics |
|---------|---------|-----------------|
| LLM-1 | Careful Analyst | Açıklayıcı, temkinli, evidence-focused, safety/compliance aware |
| LLM-2 | Result Analyst | Sonuç odaklı, kısa, karar destek, pazar/risk sinyalleri |

## 2. Dual Routing Systems

Automatic load routing ile Admin Manual LLM Control **birbirinden bağımsızdır**.

```text
┌─────────────────────────────┐    ┌─────────────────────────────┐
│  Automatic LLM Routing      │    │  Manual LLM Control Center  │
│  (job dispatch)             │    │  (config/profile management)│
│                             │    │                             │
│  Redis Queue → Router       │    │  Model Registry → Snapshot  │
│  → LLM-1 | LLM-2           │    │  → Rollback                 │
└─────────────────────────────┘    └─────────────────────────────┘
         ↑                                    ↑
         │                                    │
    Independent                         Independent
```

Manual profile değişikliği automatic routing policy'sini override etmez (ve tersi).

## 3. Automatic LLM Routing

```text
Incoming Jobs → Redis Queue → LLM Router → LLM-1 | LLM-2
```

### Routing Policy

1. **Least-loaded:** Select runtime with lowest active job count
2. **Round-robin tie break:** Equal load → alternate selection

### Load State (Redis)

```text
llm:load:LLM-1 → active_job_count
llm:load:LLM-2 → active_job_count
llm:router:round_robin_state → last_selected
```

### Acceptance Test

**5 concurrent load test zorunlu** — routing fairness doğrulanmalı.

## 4. Worker / Reviewer Assignment vs Load Routing

Worker/Reviewer rotation ile automatic load routing **ayrı katmanlardır** ve birbirine karıştırılmaz.

### Run Orchestrator

- Run için Worker ve Reviewer **logical role**'lerini belirler
- Rotation kuralını uygular (RUN A / RUN B)
- `workerRuntime != reviewerRuntime` invariant'ını garanti eder

| Run Pattern | Worker | Reviewer |
|-------------|--------|----------|
| RUN A | LLM-1 | LLM-2 |
| RUN B | LLM-2 | LLM-1 |

**Invariant:** `workerRuntime != reviewerRuntime`

### LLM Router

- Yalnızca uygun **physical runtime**'a dispatch / load-routing yapar
- Persona veya role kararını **oluşturmaz**
- Automatic load routing, Admin Manual LLM Control'den **bağımsızdır**

Rotation kararları Run Orchestrator katmanında; routing kararları LLM Router katmanında verilir.

## 5. Manual LLM Control Center

Admin-only interface for versioned configuration management.

### Configuration Hierarchy

```text
Model Registry
  → Model Version
    → Adapter Version
      → Context Version
        → Knowledge Version
          → Tool Policy Version
            → Compliance Policy Version
              → Runtime Config
                → Config Snapshot
                  → Rollback
```

### Config Snapshot

Immutable publish unit. Her analiz run bir snapshot ID ile ilişkilendirilir.

```text
ConfigSnapshot {
  id: string
  publishedAt: ISO8601
  publishedBy: userId
  modelVersion: string
  adapterVersion: string
  contextVersion: string
  knowledgeVersion: string
  toolPolicyVersion: string
  compliancePolicyVersion: string
  runtimeConfig: object
}
```

### Audit Trail

Her değişiklik audit edilir:

| Field | Description |
|-------|-------------|
| changedBy | Admin user ID |
| changedAt | Timestamp |
| reason | Mandatory change justification |
| oldVersion | Previous version reference |
| newVersion | New version reference |
| fieldDiff | Changed fields diff |

## 6. Admin Live Manipulation Demo

**Scenario:** Admin profile/config değiştirir → ConfigSnapshot publish → yeni run yeni snapshot kullanır → eski rapor değişmez.

**UI requirement:** OLD REPORT vs NEW REPORT comparison view.

Bu demo manual control center'ın immutable snapshot modelini kanıtlar.

## 7. Fine-Tune Integration

Fine-tuned model eval gate:

- Model Version registry'de fine-tuned variant olarak kayıtlı
- Eval gate geçmeden production snapshot'a publish edilemez
- Eval sonuçları audit trail'e bağlanır

Detay implementasyon Phase fine-tune fazında.

## 8. Constraints

| Constraint | Rule |
|------------|------|
| LLM count | Exactly 2; no third reviewer |
| LLM internet | Direct access forbidden |
| Routing independence | Auto routing ≠ manual control |
| Snapshot immutability | Published snapshots never mutated |
| Report binding | Reports bound to snapshot at run time |
| Worker/reviewer persist | PR-A3 stores `workerResult` and `reviewerResult` on `analysis_runs`. Product ALLOW/BLOCK is a policy engine result, not an LLM verdict. |

## 9. Failure Modes

| Condition | Behavior |
|-----------|----------|
| Both LLMs at capacity | Queue backpressure; client notified |
| LLM runtime unreachable | Job retry with backoff; alert |
| Snapshot publish failure | Old snapshot remains active |
| Rollback | New snapshot from previous version; audit logged |

## 10. Related Documents

- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — worker/reviewer pipeline
- [COMPLIANCE.md](./COMPLIANCE.md) — compliance policy versions
- [ARCHITECTURE.md](./ARCHITECTURE.md) — system topology

## 11. Platform Compliance Reflex Layer

Every LLM call through the Go gateway receives a **versioned platform compliance reflex** based on the active compliance scope (KVKK, GDPR, or BOTH):

```text
Analysis Run → ConfigSnapshot (compliancePolicyVersion)
  → Agent POST /internal/llm/v1/complete
    → Enforcer.PreCall (PII redaction + reflex resolve)
    → Persona system instruction + ComplianceReflexInstruction
    → Provider call
    → Enforcer.PostCall (forbidden-claim validation)
    → LLMCall audit (complianceProfile, policyVersion, reflexVersion, safetyResult)
```

Reflex registry version: **1.0.0** (`compliance.ComplianceReflexVersion()`). Scope-specific behavior is injected at runtime; persona seeds remain scope-agnostic.

## 12. UNRESOLVED

| Item | Status |
|------|--------|
| LLM hosting (self-hosted GPU vs external API) | UNRESOLVED — Phase 1+ |
| Fine-tune eval gate criteria thresholds | UNRESOLVED — fine-tune phase |
| Max queue depth before rejection | UNRESOLVED — Phase 1 |
