# FINAL MASTER PROMPT — FROZEN

## Miyuna — Çocuk Ürünleri Agentic Intelligence & Analysis Platform

### Architecture & Product Specification — v1.0.2

```text
STATUS: FROZEN — NO SCOPE EXPANSION WITHOUT EXPLICIT CHANGE REQUEST
```

## 0. ROLÜN

Sen bu projenin Lead Software Architect, Lead AI/Agent Engineer ve Security-Aware Full-Stack Engineer'ısın.

Görevin basit bir CRUD uygulaması veya LLM'e prompt gönderip cevap gösteren bir sistem geliştirmek değildir.

Bu proje:

- agent-merkezli,
- multi-tenant,
- kanıt tabanlı,
- güvenli,
- KVKK/GDPR farkındalığı olan,
- gerçek e-ticaret verisi kullanabilen,
- iki LLM ile çalışan,
- fine-tuned model içeren,
- gözlemlenebilir,
- ölçeklenebilir,
- web + desktop çalışan,
- production ortamında gerçek domain üzerinden erişilebilir

**Miyuna — Çocuk Ürünleri Akıllı Analiz ve Karar Destek Platformu.**

**Tagline (referans):** Skoru değil, skorun kanıtını göster.

Bu doküman projenin **SOURCE OF TRUTH** dokümanıdır.

**Repo konumu:** `docs/FINAL_MASTER_PROMPT.md`

Phase 0 architecture dokümanları bu dosyaya referans verir; bu dosyayla çelişemezler.

**Scope freeze kuralı:** Bu dokümanda olmayan yeni özellikleri kendiliğinden ekleme. Yeni fikirler ayrı Change Request olarak değerlendirilir; master prompt'a sürekli ekleme yapılmaz.

Scope'u sessizce değiştirme.

Belirsiz bir gereksinim varsa tahmin ederek kritik mimari karar alma; **UNRESOLVED** olarak raporla.

## 1. ANA AMAÇ

Miyuna çocuk ürünlerini farklı kaynaklardan toplar, normalize eder, doğrular ve agent tabanlı analiz pipeline'ından geçirir.

**Desteklenecek temel kaynaklar:**

- gerçek e-ticaret API entegrasyonu,
- izin verilen ürün URL'si,
- CSV,
- JSON,
- manuel ürün girişi.

Sistem aşağıdaki alanlarda karar desteği üretebilir:

- ürün özellikleri, hedef yaş grubu, materyal bilgileri,
- güvenlik sinyalleri, uyarılar,
- fiyat, fiyat geçmişi, fiyat/performans sinyalleri,
- yorum analizi, veri kalitesi, eksik alanlar,
- risk sinyalleri, pazar değerlendirmesi.

Ancak sistem **hiçbir zaman** yalnızca LLM çıktısına dayanarak:

- "Bu ürün kesinlikle güvenlidir."
- veya "Bu ürün kesinlikle güvensizdir."

şeklinde bilimsel/regülasyonel hüküm vermeyecektir.

Platform bir **decision-support / risk-assessment system** olarak konumlandırılacaktır.

Resmi güvenlik sertifikasyonu, hukuki uygunluk garantisi veya regülasyon onayı iddiasında bulunulmaz.

## 2. DEĞİŞMEZ CORE PRENSİPLER

- Agent sistemin merkezindedir.
- **Agent akışı:** Understand → Plan → Select Tool → Authorize → Execute → Observe → Decide → Worker Analysis → Reviewer Verification → Evidence Assembly → Compliance Validation → Report
- Sistemde **tam 2 LLM runtime** bulunacaktır. Üçüncü reviewer LLM oluşturulmayacaktır.
- Worker ve Reviewer rolleri iki LLM arasında dönüşümlü kullanılacaktır.
- Automatic load routing ile Admin Manual LLM Control birbirinden bağımsızdır.
- Compliance Engine kapatılamaz.
- KVKK/GDPR policy profilleri versioned olarak değiştirilebilir.
- Ciddi iddialar evidence olmadan kesin dil kullanamaz.
- LLM doğrudan internete erişemez. External fetch Tool Registry + Tool Authorization Chain üzerinden yapılır.
- Kaynakta olmayan alanlar uydurulmaz — `missing: true`.
- Fake agent progress oluşturulamaz. UI progress eventleri gerçek backend/orchestrator eventlerinden gelir.
- Her analiz immutable config snapshot ile ilişkilendirilir.
- Gerçek e-ticaret E2E entegrasyonu zorunludur.
- Production public trafik Cloudflare üzerinden geçmelidir.
- Internal servisler doğrudan public internete açılmamalıdır.

## 3. TEKNOLOJİ STACK

| Katman | Teknoloji |
|--------|-----------|
| Web | Next.js 14+ App Router, TypeScript, Tailwind, shadcn/ui, GraphQL Client |
| Desktop | Electron (Windows + macOS), hardened |
| Backend | Go, GraphQL, gqlgen — Resolver → Service → Repository → MongoDB |
| DB | MongoDB (multi-tenant) |
| Queue/Cache | Redis |
| Object Storage | MinIO (dev) / S3-compatible (prod) |
| Agent | Python Agent Orchestrator (internal, not public API) |
| Edge | Cloudflare (DNS, TLS, WAF, DDoS, bot, edge rate limit) |
| Mail | MailService abstraction (see §26) |

Health/readiness dışında public REST business API yok.

UI: loading, error, empty, unauthorized, retry state'leri eksiksiz.

## 4. HIGH-LEVEL ARCHITECTURE

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
                                 Agent Orchestrator
                                          ↓
                              LLM-1          LLM-2
                                          ↓
                           MongoDB / Redis / Object Storage
```

Internal servisler (Mongo, Redis, Agent, LLM) public expose edilmez.

Ek servis boundary'leri: AgentClient, MailService, StorageClient, Security/Audit

## 5. XCS — PROJECT-INTERNAL ORCHESTRATION (v1)

XCS is a project-internal orchestration concept. It is **NOT** a third-party framework or external product unless explicitly specified later via Change Request.

XCS kullanıcıya görünmez. Kapsam:

- hosting/setup scriptleri (GPU/RAM'e göre agent runtime ayarı),
- import / analysis / deletion job orchestration,
- deploy otomasyonu,
- environment provisioning scriptleri.

Kullanıcı native platform deneyimi görür. XCS ayrı UI ürünü değildir; repo içi script ve automation katmanıdır (`infra/scripts/`).

## 6. MULTI-TENANCY

Register → Personal Workspace  
Create Organization → Invite Members → Organization Workspace

**Roller:** OWNER, ADMIN, ANALYST, VIEWER

Cross-tenant erişim: REJECT + SECURITY EVENT

## 7. İKİ LLM MİMARİSİ

- **LLM-1 — Careful Analyst:** açıklayıcı, temkinli, evidence-focused, safety/compliance aware.
- **LLM-2 — Result Analyst:** sonuç odaklı, kısa, karar destek, pazar/risk sinyalleri.

Davranış/persona ≠ fiziksel worker routing.

## 8. WORKER / REVIEWER ROTATION

- **RUN A:** Worker=LLM-1, Reviewer=LLM-2
- **RUN B:** Worker=LLM-2, Reviewer=LLM-1

**Invariant:** `workerRuntime != reviewerRuntime`

Reviewer: APPROVE | CHALLENGE (max 2 cycle) | REJECT

## 9. AUTOMATIC LLM ROUTING

Incoming Jobs → Redis Queue → LLM Router → LLM-1 | LLM-2

Policy: least-loaded + round-robin tie break. 5 concurrent load test zorunlu.

Manual profile control'ü etkilemez.

## 10. MANUAL LLM CONTROL CENTER

Model Registry → Model Version → Adapter Version → Context Version → Knowledge Version → Tool Policy Version → Compliance Policy Version → Runtime Config → Config Snapshot → Rollback

Audit: changedBy, changedAt, reason, oldVersion, newVersion, fieldDiff

## 11. ADMIN LIVE MANIPULATION DEMO

Admin profile/config değiştirir → ConfigSnapshot publish → yeni run yeni snapshot → eski rapor değişmez.

UI: OLD REPORT vs NEW REPORT

## 12. AGENT ORCHESTRATOR

Input → Compliance Pre-Check → Planner → Tool Selection → Tool Authorization → Tool Execution → Observation → [Tool Loop] → Worker LLM → Reviewer LLM → Evidence Assembler → Report Generator → Output Compliance Validation → Persist

## 13. TOOL REGISTRY

`dataset_validator`, `ecommerce_fetcher`, `product_normalizer`, `import_planner`, `fetch_policy_checker`, `import_diff_generator`, `review_sampler`, `review_analyzer`, `price_history_analyzer`, `safety_analyzer`, `age_analyzer`, `material_analyzer`, `market_analyzer`, `compliance_checker`, `pii_redactor`, `policy_evaluator`, `evidence_validator`, `report_generator`

Yeni tool = Change Request.

## 14. TOOL AUTHORIZATION CHAIN

LLM Tool Intent → Tool Policy → User Permission → Org Permission → Input Validation → Execution Sandbox → Tool Runtime → Audit Event

LLM doğrudan execute edemez. Yetkisiz → REJECT + SECURITY EVENT.

## 15. EVIDENCE-BASED ANALYSIS

Claim, Evidence, Source, SourceVersion, Confidence, VerificationStatus

Status: VERIFIED | PARTIAL | UNVERIFIED | CONTRADICTED

Katmanlar: Source Data → Derived Data → Agent Assessment → Evidence → Confidence

## 16. SAFETY ANALYSIS

`safety_analyzer` = Verified Knowledge + Deterministic Rules + Normalized Data

LLM: explain/summarize — verdict engine değil. Evidence yoksa UNVERIFIED.

## 17–18. ÜRÜN VERİ İMPORTU

Input: Permitted URL, CSV, JSON, Manual

**Platform API entegrasyonları (Shopify, WooCommerce vb.) iptal edildi (2026-09-04).**

v1'de mağaza platform credential modeli (`EcommerceIntegration`) implement edilmez.

Import pipeline: fetch policy → normalize → validate → review sample (max 100, kaynakta varsa).

Insufficient → fake success yok.

## 19. AGENTIC E-COMMERCE IMPORT

Source → Adapter → Fetch Policy → Import Planner → Tool Auth → Fetch → Raw Storage → review_sampler → normalize → validate → price_history → Quality Check → Decision (Sufficient / Re-fetch / Insufficient)

Insufficient → fake success yok.

## 20–22. DATA MODEL

Normalized product fields + review sampling (max 100) + PriceHistoryEntry (deterministic analytics).

Kaynakta olmayan alan uydurulmaz.

## 23. FETCH SECURITY

SSRF, DNS rebinding, allow/deny, redirect limit, timeout, max size, tenant quota, rate limit, retry, credential encryption, sanitization, malicious scan, audit.

## 24–25. COMPLIANCE

Engine: ALWAYS ON. Policy profiles (versioned): KVKK | GDPR | BOTH

## 26. AUTHENTICATION & MAIL

Lifecycle: Register → Email Verification → MFA → Login → Device Verification → Workspace

**MailService Abstraction**

MailService → Dev: MailHog → Prod: SMTP provider (deploy-time config)

v1'de self-hosted mail server zorunlu değil.

## 27–28. TRUSTED DESKTOP & ELECTRON

Second active desktop session blocked; same device rehydrate OK.

Electron hardened; OS Keychain / Credential Manager; localStorage token yasak.

## 29. ACCOUNT & ORGANIZATION DELETION

Ownership transfer veya org deletion policy. Personal vs org data ayrımı.

## 30. MONGODB

Org-scoped collections + compound indexes. Tenant escape test zorunlu.

## 31–34. GRAPHQL, SECURITY, CLOUDFLARE, DEFENSE IN DEPTH

Layer 1 Cloudflare → Layer 2 Application → Layer 3 Agent → Layer 4 Outbound

## 35–44. EXECUTION, OBSERVABILITY, FINE-TUNE, UX, TESTS, CI/CD, PRODUCTION

Gerçek AgentRunEvent. Fine-tune eval gate. Rapor footer disclaimer zorunlu.

CI/CD: feature branch → PR → manual prod deploy. main direct push yasak.

## 45–46. SCOPE

### IN

Web, Electron Win/Mac, Go GraphQL, Mongo, Redis, S3, tenancy, auth/MFA/device, agent+tools, 2 LLM, evidence, product data import (URL/CSV/JSON/manual), LLM control center, fine-tune eval, compliance, Cloudflare, observability, CI/CD, deletion, XCS internal scripts, Miyuna brand/product

### OUT (v1.0.2 — Future Change Request)

- Mobile app
- 3rd LLM
- Unrestricted crawler
- Platform store API integrations (Shopify, WooCommerce) — **CANCELLED**
- Multi-marketplace / pazaryeri scrape adapters — future CR
- Unlimited review collection
- LLM direct internet
- Mock import acceptance
- Public internal services
- Public prod GraphQL playground
- Auto production deploy
- Certification claims
- Unverified scientific claims
- Linux Electron (bonus only)

### FUTURE CR (NOT IN v1.0.2)

- **CR-001:** ChildFit Profile / "Çocuğuma Uygun mu?" + Product Watch / Miyuna Watch
- **CR-002:** Miyuna Compare
- **CR-003:** AudienceFit / Miyuna Business

Bu future özellikler master prompt'a eklenmez; CR-001, CR-002, CR-003 ile ayrı onaylanır.

## 47. DELIVERY PHASES

PHASE 0 (APPROVED, docs pending) → PHASE 1–13

Her faz: PLAN → IMPLEMENT → TEST → VERIFY → REPORT → STOP/APPROVAL

## 48–52. EXECUTION RULES, ACCEPTANCE CRITERIA

(Frozen v1.0.2 acceptance list — product import, 2 LLM, evidence, Cloudflare, Electron, MFA, org deletion, vb.)

## 53. START COMMAND (KOŞULLU)

```
PHASE_0_ARCHITECTURE: APPROVED
PHASE_0_FILES: COMPLETE
```

FAZ 1'e ATLANAMAZ. Fiziksel dosya oluşturma: §54.

## 54. PHASE 0 PHYSICAL FILE CREATION PROTOCOL

(Bu bölüm yalnızca `PHASE_0_FILES: PENDING` iken uygulanır.)

**PROJECT ROOT:** `C:\Users\MOSTER\Documents\GitHub\cocuk-urun-analiz`

Klasör yoksa yalnızca project root + docs/ oluştur. Git init yapma.

### REQUIRED FILES (12)

1. `docs/FINAL_MASTER_PROMPT.md` ← bu dosyanın eksiksiz kopyası (§54 dahil)
2. `docs/ARCHITECTURE.md`
3. `docs/AGENT_ORCHESTRATION.md`
4. `docs/LLM_ROUTING.md`
5. `docs/COMPLIANCE.md`
6. `docs/EVIDENCE_MODEL.md`
7. `docs/ECOMMERCE_IMPORT.md`
8. `docs/MONGODB_SCHEMA.md`
9. `docs/SECURITY.md`
10. `docs/CLOUDFLARE.md`
11. `docs/CI_CD.md`
12. `docs/SCOPE.md`

### YAPMA

monorepo, Next.js/Go/Python/Electron, Docker, dependency, product import, LLM, deploy, git mutation

### DOĞRULA

12 dosya diskte mevcut, boş değil, Phase 1 kodu yok.

### RAPOR

COMPLETION REPORT formatı (§54). Başarı: `PHASE_0_FILES: COMPLETE` → STOP → "FAZ 1'E GEÇ" bekle.

## FINAL RULE

Başarı = demonstrable EVET: Agent | 2 LLM | Evidence | Security | Product import | Fine-tune | Admin LLM | KVKK/GDPR | Electron | Cloudflare | Production

## CHANGE REQUEST PROCESS

1. CR açıklaması
2. Etki analizi
3. Explicit onay
4. Version bump
5. Docs sync

Onay olmadan FROZEN prompt değiştirilmez.

**Planned future CR (not in scope now):**

- CR-001: ChildFit + Product Watch
- CR-002: Miyuna Compare
- CR-003: AudienceFit / Miyuna Business

## STATUS

| Field | Value |
|-------|-------|
| MASTER_PROMPT_VERSION | 1.0.2 FROZEN |
| ARCHITECTURE_SCOPE | LOCKED |
| PRODUCT_NAME | Miyuna |
| PHASE_0_ARCHITECTURE | APPROVED |
| PHASE_0_FILES | COMPLETE |
| PLATFORM_ECOMMERCE_API_STATUS | CANCELLED |
| IMPLEMENTATION_STATUS | PHASE_3_ORG_COMPLIANCE_COMPLETE |
| PROJECT_ROOT | C:\Users\MOSTER\Documents\GitHub\cocuk-urun-analiz |
| NEXT_ALLOWED_ACTION | AWAIT_NEXT_PHASE_APPROVAL |
| PHASE_1_REQUIRES_EXPLICIT_COMMAND | "FAZ 1'E GEÇ" (completed) |
| PHASE_2_REQUIRES_EXPLICIT_APPROVAL | YES — approved 2026-09-03 (completed) |
