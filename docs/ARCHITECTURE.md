# Miyuna — High-Level Architecture

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Overview

Miyuna, çocuk ürünlerini çok kaynaklı toplayan, normalize eden ve agent tabanlı analiz pipeline'ından geçiren **multi-tenant, evidence-based decision-support platform**dur.

**Product name:** Miyuna  
**Tagline:** Skoru değil, skorun kanıtını göster.

## 2. Architectural Principles

| Principle | Requirement |
|-----------|-------------|
| Agent-centric | Orchestrator merkezde; UI yalnızca gerçek eventleri gösterir |
| Two LLM only | LLM-1 + LLM-2; üçüncü reviewer yok |
| Evidence-first | Ciddi iddialar kanıt olmadan kesin dil kullanamaz |
| Compliance always-on | Engine kapatılamaz; policy profilleri versioned |
| No LLM internet | External fetch yalnızca Tool Registry + Authorization Chain |
| Product data import | URL/CSV/JSON/manual + marketplace adapters (Phase 4); platform store API cancelled |
| Defense in depth | Cloudflare → Application → Agent → Outbound |

## 3. System Topology

```text
                         INTERNET
                            ↓
                       CLOUDFLARE
          ┌─────────────────┼─────────────────┐
         WAF          Rate Limiting       DDoS/Bot
          └─────────────────┼─────────────────┘
                            ↓
              ┌─────────────┴─────────────┐
              ↓                           ↓
         Next.js Web                Go GraphQL API
       app.{domain}               api.{domain}/graphql
                                          ↓
                                 Auth / MFA / RBAC / Tenant
                                          ↓
              ┌───────────────────────────┴───────────────────────────┐
              │ Phase 4 Data Layer (IMPLEMENTED)                      │
              │ products · mappings · UGC · marketplace reviews       │
              │ async import (Redis) · dataset draft metadata         │
              └───────────────────────────┬───────────────────────────┘
                                          ↓
                                 Agent Orchestrator (Python, internal) ← Phase 5 IMPLEMENTED
                                          ↓
                              Go internal agent boundary (auth, grants, tools, persist)
                                          ↓
                              LLM-1          LLM-2                         ← Phase 6+ DEFERRED
                                          ↓
                           MongoDB / Redis / Object Storage
```

### Public-Facing Components

| Component | Domain | Protocol |
|-----------|--------|----------|
| Web App | `app.{domain}` | HTTPS via Cloudflare |
| GraphQL API | `api.{domain}/graphql` | HTTPS via Cloudflare |
| Health/Readiness | Internal or restricted | Minimal REST only |

### Internal-Only Components

- MongoDB
- Redis (queue + cache)
- Python Agent Orchestrator
- LLM-1 / LLM-2 runtimes
- MinIO (dev) / S3-compatible (prod)

Internal servisler doğrudan public internete açılmaz.

## 4. Technology Stack

| Layer | Technology | Notes |
|-------|------------|-------|
| Web | Next.js 14+ App Router, TypeScript, Tailwind, shadcn/ui | GraphQL client |
| Desktop | Electron (Windows + macOS), hardened | OS Keychain; no localStorage tokens |
| Backend | Go, gqlgen | Resolver → Service → Repository → MongoDB |
| Database | MongoDB | Multi-tenant, org-scoped |
| Queue/Cache | Redis | Job queue + LLM routing state |
| Object Storage | MinIO (dev) / S3 (prod) | Raw import blobs, artifacts |
| Agent | Python Orchestrator | Internal; not public API |
| Edge | Cloudflare | DNS, TLS, WAF, DDoS, bot, rate limit |
| Mail | MailService abstraction | MailHog (dev), SMTP provider (prod) |

Health/readiness dışında **public REST business API yok**.

### UI State Requirements (FINAL_MASTER_PROMPT.md §3)

Her kullanıcı-facing async/data-driven view aşağıdaki durumları desteklemelidir:

- **loading**
- **error**
- **empty**
- **unauthorized**
- **retry**

Bu gereksinim Web (Next.js) ve Desktop (Electron) istemcileri için geçerlidir.

## 5. Service Boundaries

```text
┌─────────────┐     GraphQL      ┌─────────────┐
│  Next.js    │ ◄──────────────► │  Go API     │
│  Web/Electron│                 │  (gqlgen)   │
└─────────────┘                  └──────┬──────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ↓                   ↓                   ↓
             AgentClient          MailService         StorageClient
                    ↓
            Agent Orchestrator
                    ↓
              LLM-1 / LLM-2
                    ↓
         MongoDB / Redis / S3
```

**Boundary contracts:**

- **GraphQL (Phase 5):** `startAgentRun`, `cancelAgentRun`, `agentRun`, `analysisRuns`, `agentRunEvents` — Go RBAC/tenant/compliance/persistence
- **AgentClient:** Go → Python orchestrator IPC (`/internal/v1/runs/start|cancel`) with shared internal token
- **Go internal agent API:** Python → Go authoritative tool authorization, execution, events, status, lease (`/internal/agent/v1/*`)
- **MailService:** Abstract interface; provider swappable at deploy time
- **StorageClient:** S3-compatible; tenant-scoped prefixes
- **Security/Audit:** Cross-cutting; all sensitive operations logged

**Phase 5 ownership split:**

| Layer | Owner | Responsibility |
|-------|-------|----------------|
| Python `RunManager` | Authoritative lifecycle | Plan, loop, cancel/timeout, heartbeat |
| Go GraphQL + services | Security boundary | RBAC, tenant, compliance pre-check, persistence |
| Redis | Ephemeral coordination | Hashed single-use tool grants, run leases |
| MongoDB | Source of truth | `analysis_runs`, `agent_run_events`, `tool_executions` |

Phase 4 marketplace **import consumer remains Go-only** and is isolated from Phase 5 analysis queue.

## 6. Multi-Tenancy Model

```text
Register → Personal Workspace
Create Organization → Invite Members → Organization Workspace
```

**Roles:** OWNER, ADMIN, ANALYST, VIEWER

Cross-tenant erişim: **REJECT + SECURITY EVENT**

Tüm data access org-scoped; compound indexes ile tenant isolation. Detay: [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md)

## 7. Agent Pipeline (Summary)

**Phase 5 (IMPLEMENTED — deterministic, no LLM):**

```text
GraphQL startAgentRun
  → Go compliance pre-check + ConfigSnapshot
  → Python RunManager (lease claim, heartbeat)
  → Deterministic planner (available tools only)
  → Go authorize → Redis grant → Go execute → schema/compliance output validation
  → agent_run_events + terminal status (single write)
```

**Phase 6+ (DEFERRED):** Worker LLM → Reviewer LLM → Evidence → Report Generator

Detay: [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md)

## 8. XCS — Internal Orchestration

XCS kullanıcıya görünmez. Repo içi automation katmanı (`infra/scripts/`):

- GPU/RAM'e göre agent runtime setup
- Import / analysis / deletion job orchestration
- Deploy automation
- Environment provisioning

XCS ayrı UI ürünü değildir; third-party framework değildir.

## 9. Authentication Lifecycle

```text
Register → Email Verification → MFA → Login → Device Verification → Workspace
```

Electron: second active desktop session blocked; same device rehydrate OK.

Detay: [SECURITY.md](./SECURITY.md)

## 10. Data Flow — Phase 4 Marketplace + UGC Foundation (IMPLEMENTED)

```text
Sources: Hepsiburada | Trendyol | CSV | JSON | Manual | Miyuna UGC Portal
  ↓
MarketplaceAdapter (source detection + policy)
  ↓
Fetch Policy / SSRF guard
  ↓
Async Import Request → MarketplaceImportRun (PENDING)
  ↓
Redis Queue → Go API consumer (in-process, not separate worker)
  ↓
Fetch or Parse (live marketplace fetch DEFERRED pending authorized API)
  ↓
Raw Storage (S3/MinIO via StorageClient) + raw_source_payloads metadata
  ↓
Canonical Normalization (single pipeline)
  ↓
Deduplication · PII · Provenance · License/Rights · DatasetEligibility
  ↓
Persist: products | product_source_mappings | marketplace_reviews | user_experiences
  ↓
DatasetRecord candidates → DatasetVersion (DRAFT metadata only)
```

**Boundaries:**

- **Phase 5 (IMPLEMENTED):** Python authoritative orchestrator; Go GraphQL/security/persistence/tool boundary; Redis grants/leases
- **P0 PR-A1 (IMPLEMENTED):** tenant-scoped `evidences` + `evidence_claim_validations`; `evidence_validator` available via Executor and planner-wired in PR-A3.
- **P0 PR-A2 (IMPLEMENTED):** safety findings + official CPSC/GÜBİS recall adapters + recall matcher; `safety_analyzer` available via Executor and planner-wired in PR-A3.
- **P0 PR-A3 (IMPLEMENTED):** `review_analyzer` AVAILABLE; planner/orchestrator wiring; worker/reviewer persist; versioned `finalResult` on `analysis_runs`; hallucination guard; deterministic confidence + product decision; GraphQL + minimum Product analysis UI.
- **P0 PR-B (IMPLEMENTED):** KVKK/GDPR account deletion + JSON data export (`requestAccountDeletion`, `confirmAccountDeletion`, `exportMyData`).
- **P0 PR-C (THIS PR):** Real dual-LLM validation harness (`scripts/phase6/run_p0_final_validation.ps1`), gateway fallback/routing regression tests, optional `-tags=real_llm` E2E when credentials + distinct models are available. See [REAL_LLM_FINAL_VERIFICATION.md](./REAL_LLM_FINAL_VERIFICATION.md).
- Fine-tuning remains **DEFERRED**.
- **Phase 10:** Fine-tune training, dataset publication export — NOT in Phase 4
- **Live Hepsiburada/Trendyol fetch:** DEFERRED_WITH_REASON until verified authorized/permitted API

Portal UI (Next.js): products list/create/detail, Miyuna experience form, separated marketplace review view.

Detay: [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md), [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md)

## 11. Data Flow — Future Agentic Import (Phase 5+)

```text
Source → Adapter → Fetch Policy → Import Planner → Tool Auth → Fetch
  ↓
Raw Storage → review_sampler → normalize → validate → price_history
  ↓
Quality Check → Decision (Sufficient / Re-fetch / Insufficient)
  ↓
Agent Pipeline (Planner → Worker LLM → Reviewer LLM → Evidence)
```

## 12. Observability

- Gerçek `AgentRunEvent` stream (fake progress yasak)
- Structured audit logs (Security/Audit boundary)
- Config snapshot correlation per analysis run
- Health/readiness endpoints

## 13. Production Requirements

- Public traffic **must** pass through Cloudflare
- Real domain access in production
- Manual prod deploy (no auto deploy to prod)
- Feature branch → PR → manual prod deploy
- `main` direct push yasak

Detay: [CLOUDFLARE.md](./CLOUDFLARE.md), [CI_CD.md](./CI_CD.md)

## 14. Related Documents

| Document | Scope |
|----------|-------|
| [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) | Orchestrator, tools, worker/reviewer |
| [LLM_ROUTING.md](./LLM_ROUTING.md) | Automatic routing + manual control center |
| [COMPLIANCE.md](./COMPLIANCE.md) | KVKK/GDPR engine |
| [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md) | Claims, evidence, verification |
| [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md) | Marketplace/CSV/JSON import + UGC (platform store API cancelled) |
| [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) | Collections, indexes, tenancy |
| [SECURITY.md](./SECURITY.md) | Auth, fetch security, defense in depth |
| [CLOUDFLARE.md](./CLOUDFLARE.md) | Edge configuration |
| [CI_CD.md](./CI_CD.md) | Pipeline, deploy gates |
| [SCOPE.md](./SCOPE.md) | IN/OUT scope, CR registry |

## 15. UNRESOLVED

| Item | Status |
|------|--------|
| Production domain name | UNRESOLVED — deploy-time decision |
| LLM hosting topology (self-hosted vs API) | UNRESOLVED — Phase 1+ |
| SMTP provider selection | UNRESOLVED — deploy-time config |
