# Miyuna — Scope Definition

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Product Identity

| Field | Value |
|-------|-------|
| Product Name | **Miyuna** |
| Full Name | Miyuna — Çocuk Ürünleri Akıllı Analiz ve Karar Destek Platformu |
| Tagline | Skoru değil, skorun kanıtını göster. |
| Version | 1.0.4 FROZEN (CR-005) |
| Positioning | Decision-support / risk-assessment system |

## 2. Scope Freeze Rule

- Bu dokümanda olmayan yeni özellikler **kendiliğinden eklenmez**
- Yeni fikirler ayrı **Change Request** olarak değerlendirilir
- Master prompt'a sürekli ekleme yapılmaz
- Scope sessizce değiştirilmez
- Belirsiz gereksinimler → **UNRESOLVED** olarak raporlanır

## 3. IN SCOPE (v1.0.2)

### Platform

- [x] Web application (Next.js 14+ App Router)
- [x] Desktop application (Electron — Windows + macOS, hardened)
- [x] Go GraphQL API backend
- [x] MongoDB multi-tenant database
- [x] Redis queue/cache
- [x] S3-compatible object storage
- [x] Python Agent Orchestrator (internal)
- [x] Cloudflare edge (production)
- [x] Production real domain access

### Identity & Tenancy

- [x] Multi-tenancy (personal + organization workspaces)
- [x] RBAC (OWNER, ADMIN, ANALYST, VIEWER)
- [x] Authentication with MFA
- [x] Device verification
- [x] Account deletion
- [x] Organization deletion

### Agent & AI

- [x] Agent-centric architecture
- [x] Exactly 2 LLM runtimes (no third reviewer)
- [x] Worker/Reviewer rotation
- [x] Automatic LLM load routing
- [x] Manual LLM Control Center
- [x] Config Snapshot + rollback
- [x] Admin live manipulation demo (OLD vs NEW report)
- [x] Tool Registry (18 tools, v1 set)
- [x] Tool Authorization Chain
- [x] Fine-tuned model with eval gate
- [x] Real AgentRunEvent (no fake progress)

### Evidence & Compliance

- [x] Evidence-based analysis model
- [x] Compliance engine (always-on, non-disableable)
- [x] KVKK / GDPR / BOTH policy profiles (versioned)
- [x] Report footer disclaimer (mandatory)
- [x] PII redaction

### Product Data Import

- [x] Permitted product URL import
- [x] CSV / JSON / manual import
- [x] Agentic import pipeline
- [x] Review sampling (max 100, when present in source)
- [x] Price history (deterministic analytics)

**CANCELLED (2026-09-04):** Shopify, WooCommerce ve diğer platform store API entegrasyonları.

### Phase 4 — Marketplace + UGC + Dataset Foundation (CR-005, IMPLEMENTED)

- [x] Canonical `products` collection (single product truth)
- [x] ProductSourceMapping + deterministic dedup
- [x] Miyuna UserExperience (UGC) + consent/compliance reuse
- [x] UGC portal UI (products + experience form)
- [x] MarketplaceAdapter architecture (Hepsiburada + Trendyol boundaries)
- [x] MarketplaceReview + async Redis import (Go API consumer)
- [x] CSV/JSON import fallback
- [x] Raw object storage (MinIO/S3 StorageClient)
- [x] DatasetEligibility + provenance/license/rights metadata
- [x] DatasetRecord + DatasetVersion (DRAFT only)
- [ ] Live Hepsiburada/Trendyol fetch — **DEFERRED_WITH_REASON** (authorized API pending)

### Security & Infrastructure

- [x] Defense in depth (4 layers)
- [x] Fetch security (SSRF, DNS rebinding, etc.)
- [x] Tenant escape test
- [x] Internal services not public
- [x] MailService abstraction (MailHog dev / SMTP prod)
- [x] XCS internal orchestration scripts

### Operations

- [x] Observability (real events, audit trails)
- [x] CI/CD (feature branch → PR → manual prod deploy)
- [x] Protected main branch

## 4. OUT OF SCOPE (v1.0.2)

| Feature | Notes |
|---------|-------|
| Mobile app | Future CR |
| 3rd LLM / third reviewer | Explicitly forbidden |
| Unrestricted web crawler | LLM has no direct internet |
| Platform store API (Shopify, WooCommerce) | **CANCELLED** — 2026-09-04 |
| Multi-marketplace adapters (Hepsiburada, Trendyol) | **CR-005 IN** — live fetch deferred |
| Encrypted EcommerceIntegration (platform API) | CANCELLED with platform API |
| Unlimited review collection | Max 100 per product |
| LLM direct internet access | Tool Registry only |
| Mock import acceptance | Real validation required |
| Public internal services | Mongo, Redis, Agent, LLM private |
| Public prod GraphQL playground | Forbidden |
| Auto production deploy | Manual only |
| Certification claims | Platform never certifies |
| Unverified scientific claims | Evidence required |
| Linux Electron | Bonus only, not required |
| Self-hosted mail server | Not required v1 |

## 5. Future Change Requests (NOT IN v1.0.2)

Bu özellikler master prompt'a **eklenmez**. Ayrı CR ile onaylanır:

### CR-001: ChildFit + Product Watch

- ChildFit Profile / "Çocuğuma Uygun mu?"
- Product Watch / Miyuna Watch

### CR-002: Miyuna Compare

- Product comparison features

### CR-003: AudienceFit / Miyuna Business

- Business/audience fit analysis

## 6. Change Request Process

```text
1. CR açıklaması (scope, rationale, acceptance criteria)
2. Etki analizi (architecture, security, compliance, timeline)
3. Explicit onay (stakeholder sign-off)
4. Version bump (e.g., 1.0.2 → 1.1.0)
5. Docs sync (master prompt + affected architecture docs)
```

Onay olmadan FROZEN prompt değiştirilmez.

## 7. Delivery Phases

```text
PHASE 0 (APPROVED, docs pending) → PHASE 1 → ... → PHASE 13
```

Each phase: **PLAN → IMPLEMENT → TEST → VERIFY → REPORT → STOP/APPROVAL**

Phase 1 requires explicit command: **"FAZ 1'E GEÇ"**

## 8. Current Status

| Field | Value |
|-------|-------|
| MASTER_PROMPT_VERSION | 1.0.2 FROZEN |
| ARCHITECTURE_SCOPE | LOCKED |
| PRODUCT_NAME | Miyuna |
| PHASE_0_ARCHITECTURE | APPROVED |
| PHASE_0_FILES | COMPLETE |
| PLATFORM_ECOMMERCE_API_STATUS | CANCELLED |
| IMPLEMENTATION_STATUS | PHASE_4_COMPLETE_PENDING_REVIEW |
| NEXT_ALLOWED_ACTION | AWAIT_NEXT_PHASE_APPROVAL |
| PHASE_1_REQUIRES_EXPLICIT_COMMAND | "FAZ 1'E GEÇ" (completed) |
| PHASE_2_REQUIRES_EXPLICIT_APPROVAL | YES — approved 2026-09-03 (completed) |

## 9. Success Criteria (Final)

Demonstrable **EVET** for all:

| Capability | Required |
|------------|----------|
| Agent | YES |
| 2 LLM | YES |
| Evidence | YES |
| Security | YES |
| Product data import (URL/CSV/JSON/manual) | YES |
| Fine-tune | YES |
| Admin LLM Control | YES |
| KVKK/GDPR | YES |
| Electron | YES |
| Cloudflare | YES |
| Production | YES |

## 10. Related Documents

All Phase 0 architecture documents reference and defer to [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md):

- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md)
- [LLM_ROUTING.md](./LLM_ROUTING.md)
- [COMPLIANCE.md](./COMPLIANCE.md)
- [EVIDENCE_MODEL.md](./EVIDENCE_MODEL.md)
- [ECOMMERCE_IMPORT.md](./ECOMMERCE_IMPORT.md)
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md)
- [SECURITY.md](./SECURITY.md)
- [CLOUDFLARE.md](./CLOUDFLARE.md)
- [CI_CD.md](./CI_CD.md)
