# Miyuna — High-Level Architecture

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN  
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
| Product data import | URL/CSV/JSON/manual; platform API cancelled |
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
                                 Agent Orchestrator (Python, internal)
                                          ↓
                              LLM-1          LLM-2
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

- **AgentClient:** Go → Python orchestrator IPC/HTTP (internal network)
- **MailService:** Abstract interface; provider swappable at deploy time
- **StorageClient:** S3-compatible; tenant-scoped prefixes
- **Security/Audit:** Cross-cutting; all sensitive operations logged

## 6. Multi-Tenancy Model

```text
Register → Personal Workspace
Create Organization → Invite Members → Organization Workspace
```

**Roles:** OWNER, ADMIN, ANALYST, VIEWER

Cross-tenant erişim: **REJECT + SECURITY EVENT**

Tüm data access org-scoped; compound indexes ile tenant isolation. Detay: [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md)

## 7. Agent Pipeline (Summary)

```text
Input
  → Compliance Pre-Check
  → Planner
  → Tool Selection → Tool Authorization → Tool Execution → Observation
  → [Tool Loop]
  → Worker LLM → Reviewer LLM
  → Evidence Assembler → Report Generator
  → Output Compliance Validation
  → Persist
```

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

## 10. Data Flow — E-Commerce Import

```text
Source → Adapter → Fetch Policy → Import Planner → Tool Auth → Fetch
  → Raw Storage → review_sampler → normalize → validate → price_history
  → Quality Check → Decision (Sufficient / Re-fetch / Insufficient)
```

Detay: [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md)

## 11. Observability

- Gerçek `AgentRunEvent` stream (fake progress yasak)
- Structured audit logs (Security/Audit boundary)
- Config snapshot correlation per analysis run
- Health/readiness endpoints

## 12. Production Requirements

- Public traffic **must** pass through Cloudflare
- Real domain access in production
- Manual prod deploy (no auto deploy to prod)
- Feature branch → PR → manual prod deploy
- `main` direct push yasak

Detay: [CLOUDFLARE.md](./CLOUDFLARE.md), [CI_CD.md](./CI_CD.md)

## 13. Related Documents

| Document | Scope |
|----------|-------|
| [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) | Orchestrator, tools, worker/reviewer |
| [LLM_ROUTING.md](./LLM_ROUTING.md) | Automatic routing + manual control center |
| [COMPLIANCE.md](./COMPLIANCE.md) | KVKK/GDPR engine |
| [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md) | Claims, evidence, verification |
| [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md) | Product import pipeline (platform API cancelled) |
| [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) | Collections, indexes, tenancy |
| [SECURITY.md](./SECURITY.md) | Auth, fetch security, defense in depth |
| [CLOUDFLARE.md](./CLOUDFLARE.md) | Edge configuration |
| [CI_CD.md](./CI_CD.md) | Pipeline, deploy gates |
| [SCOPE.md](./SCOPE.md) | IN/OUT scope, CR registry |

## 14. UNRESOLVED

| Item | Status |
|------|--------|
| Production domain name | UNRESOLVED — deploy-time decision |
| LLM hosting topology (self-hosted vs API) | UNRESOLVED — Phase 1+ |
| SMTP provider selection | UNRESOLVED — deploy-time config |
