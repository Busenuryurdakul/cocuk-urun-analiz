# Miyuna — Agent Orchestration

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Purpose

Python Agent Orchestrator, Miyuna'nın merkezi execution engine'idir. Public API değildir; Go GraphQL API üzerinden **AgentClient** boundary ile internal erişilir.

## 2. Core Agent Flow

```text
Understand → Plan → Select Tool → Authorize → Execute → Observe → Decide
  → Worker Analysis → Reviewer Verification → Evidence Assembly
  → Compliance Validation → Report
```

## 3. Full Pipeline

```text
Input
  ↓
Compliance Pre-Check          ← policy_evaluator, compliance_checker
  ↓
Planner                       ← import_planner (when applicable)
  ↓
Tool Selection
  ↓
Tool Authorization            ← Tool Authorization Chain (§14)
  ↓
Tool Execution
  ↓
Observation
  ↓
[Tool Loop]                   ← repeat until planner decides sufficient
  ↓
Worker LLM                    ← LLM-1 or LLM-2 (per rotation)
  ↓
Reviewer LLM                  ← opposite runtime (invariant: worker != reviewer)
  ↓
Evidence Assembler            ← evidence_validator
  ↓
Report Generator
  ↓
Output Compliance Validation  ← compliance_checker, pii_redactor
  ↓
Persist                       ← MongoDB + audit events
```

## 4. Worker / Reviewer Rotation

| Run | Worker | Reviewer |
|-----|--------|----------|
| RUN A | LLM-1 (Careful Analyst) | LLM-2 (Result Analyst) |
| RUN B | LLM-2 (Result Analyst) | LLM-1 (Careful Analyst) |

**Invariant:** `workerRuntime != reviewerRuntime`

**Reviewer decisions:**

| Decision | Meaning |
|----------|---------|
| APPROVE | Worker output accepted |
| CHALLENGE | Send back to worker (max 2 cycles) |
| REJECT | Run fails; no fake success |

**Max challenge cycles:** 2

## 5. LLM Personas

| LLM | Persona | Focus |
|-----|---------|-------|
| LLM-1 | Careful Analyst | Explanatory, cautious, evidence-focused, safety/compliance aware |
| LLM-2 | Result Analyst | Concise, decision-support, market/risk signals |

**Note:** Persona/behavior ≠ physical worker routing. Rotation ensures both personas serve both roles.

## 6. Tool Registry (v1.0.2)

| Tool | Purpose |
|------|---------|
| `dataset_validator` | Validate incoming CSV/JSON/manual datasets |
| `ecommerce_fetcher` | Fetch permitted product URLs (platform API cancelled) |
| `product_normalizer` | Normalize product fields to canonical schema |
| `import_planner` | Plan multi-step import strategy |
| `fetch_policy_checker` | Enforce fetch security policies |
| `import_diff_generator` | Diff between import versions |
| `review_sampler` | Sample reviews (max 100) |
| `review_analyzer` | Analyze sampled reviews (AVAILABLE; planner-wired in PR-A3) |
| `price_history_analyzer` | Deterministic price analytics |
| `safety_analyzer` | Deterministic safety findings + official recall matching (AVAILABLE; planner-wired in PR-A3) |
| `age_analyzer` | Target age group assessment |
| `material_analyzer` | Material composition analysis |
| `market_analyzer` | Market/risk signal assessment |
| `compliance_checker` | KVKK/GDPR compliance validation |
| `pii_redactor` | PII detection and redaction |
| `policy_evaluator` | Evaluate against versioned policy profiles |
| `evidence_validator` | Validate claim-evidence linkage (AVAILABLE; planner-wired in PR-A3) |
| `report_generator` | Generate final analysis report |

**New tool = Change Request.** Registry genişletmesi master prompt değişikliği gerektirir.

P0 analysis planner (PR-A3, `python-deterministic-planner-v2`):

```text
policy_evaluator → compliance_checker → review_sampler?
  → review_analyzer → safety_analyzer → evidence_validator
  → dataset_validator → pii_redactor
  → worker LLM → reviewer LLM → finalize (hallucination/confidence/decision)
```

Unavailable or excluded tools never enter the plan. `report_generator`, age/material/market/price analyzers remain excluded. Real dual-LLM final E2E is deferred to PR-C.

## 7. Tool Authorization Chain

```text
LLM Tool Intent
  → Tool Policy (versioned)
  → User Permission (RBAC)
  → Org Permission (tenant scope)
  → Input Validation
  → Execution Sandbox
  → Tool Runtime
  → Audit Event
```

**Rules:**

- LLM doğrudan tool execute edemez
- Yetkisiz intent → REJECT + SECURITY EVENT
- LLM doğrudan internete erişemez
- External fetch yalnızca authorized tools üzerinden

## 8. Safety Analysis Model

`safety_analyzer` = **Verified Knowledge + Deterministic Rules + Normalized Data**

- P0 PR-A3: `review_analyzer`, `safety_analyzer`, and `evidence_validator` are AVAILABLE and included in the Python planner (`python-deterministic-planner-v2`) when the registry marks them AVAILABLE.
- Official CPSC/GÜBİS records are normalized and matched deterministically. Weak/fuzzy matches are `requiresReview` and cannot become CRITICAL confirmed findings.
- Official CPSC/GÜBİS records are normalized and matched deterministically. Weak/fuzzy matches are `requiresReview` and cannot become CRITICAL confirmed findings.
- LLM role: explain/summarize — **not** verdict engine
- Evidence yoksa status: **UNVERIFIED** / `INSUFFICIENT_EVIDENCE`
- Platform asla "kesinlikle güvenli/güvensiz" hüküm vermez
- Detay: [SAFETY_RECALL.md](./SAFETY_RECALL.md)

## 9. Progress Events

**Fake agent progress oluşturulamaz.**

UI progress eventleri gerçek backend/orchestrator eventlerinden gelir:

```text
AgentRunEvent {
  runId: string
  organizationId: string       // canonical tenant identifier (incl. PERSONAL org)
  configSnapshotId: string
  phase: string
  toolName?: string
  status: RUNNING | COMPLETED | FAILED
  timestamp: ISO8601
  metadata: object
}
```

Events Redis pub/sub veya equivalent internal channel üzerinden web/desktop client'a iletilir.

## 10. Config Snapshot Binding

Her analiz run'ı immutable **ConfigSnapshot** ile ilişkilendirilir:

- Model Version
- Adapter Version
- Context Version
- Knowledge Version
- Tool Policy Version
- Compliance Policy Version
- Runtime Config

Admin config değişikliği yalnızca **yeni run**ları etkiler; eski raporlar değişmez.

## 11. Orchestrator ↔ Go API Contract

**AgentClient** (internal):

| Operation | Direction |
|-----------|-----------|
| `StartAnalysisRun` | Go → Orchestrator |
| `StartImportJob` | Go → Orchestrator |
| `CancelRun` | Go → Orchestrator |
| `GetRunStatus` | Go → Orchestrator |
| `StreamRunEvents` | Orchestrator → Go → Client |

Orchestrator public internet'e expose edilmez.

**Internal HTTP (implemented):**

| Go internal route | Purpose |
|-------------------|---------|
| `POST /internal/agent/v1/events` | lifecycle event persistence |
| `POST /internal/agent/v1/runs/status` | status transitions |
| `GET /internal/agent/v1/runs/context` | run context + capability snapshot |
| `POST /internal/agent/v1/tools/authorize` | tool authorization + grant issue |
| `POST /internal/agent/v1/tools/execute` | authorized tool execution |
| `GET /internal/agent/v1/runs/cancellation` | cancellation signal |
| `POST /internal/agent/v1/runs/heartbeat` | run lease heartbeat |
| `POST /internal/agent/v1/runs/lease/claim` | atomic run lease claim |

**Python internal routes:** `POST /internal/v1/runs/start`, `POST /internal/v1/runs/cancel`

**Redis keys (Phase 5 hardening):**

| Key pattern | Purpose |
|-------------|---------|
| `agent:grant:{grantIdHash}` | TTL single-use execution grant (atomic consume) |
| `agent:run:lease:{organizationId}:{analysisRunId}` | orchestrator run lease + heartbeat |

Grant plaintext/token asla persist/log edilmez; yalnızca hash saklanır.

## 12. Error Handling

| Condition | Behavior |
|-----------|----------|
| Tool authorization failure | REJECT + SECURITY EVENT |
| Insufficient import data | Explicit Insufficient decision; no fake success |
| Reviewer REJECT | Run fails with audit trail |
| Compliance validation failure | Output blocked; audit logged |
| Missing source field | `missing: true`; never fabricated |

## 13. Related Documents

- [LLM_ROUTING.md](./LLM_ROUTING.md) — automatic routing + manual control
- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md) — evidence assembly
- [COMPLIANCE.md](./COMPLIANCE.md) — pre/post compliance checks
- [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md) — import-specific orchestration

## 14. UNRESOLVED

| Item | Status |
|------|--------|
| Orchestrator ↔ Go IPC protocol (HTTP vs gRPC vs message queue) | UNRESOLVED — Phase 1 |
| Tool sandbox isolation mechanism | UNRESOLVED — Phase 1 |
| Event streaming transport (SSE vs WebSocket vs polling) | UNRESOLVED — Phase 1 |
