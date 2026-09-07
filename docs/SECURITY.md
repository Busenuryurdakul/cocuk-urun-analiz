# Miyuna — Security

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Defense in Depth

```text
Layer 1: Cloudflare     → WAF, DDoS, bot, edge rate limit, TLS
Layer 2: Application    → Auth, MFA, RBAC, tenant isolation, input validation
Layer 3: Agent          → Tool authorization, sandbox, compliance engine
Layer 4: Outbound       → Fetch security, SSRF prevention, credential encryption
```

Detay Layer 1: [CLOUDFLARE.md](./CLOUDFLARE.md)

## 2. Authentication Lifecycle

```text
Register → Email Verification → MFA → Login → Device Verification → Workspace
```

### Requirements

| Step | Requirement |
|------|-------------|
| Registration | Email + password; email verification mandatory |
| MFA | Required before workspace access |
| Login | MFA challenge on each login (or trusted device) |
| Device Verification | New device requires verification flow |
| Session | Secure, httpOnly cookies (web); OS Keychain (Electron) |
| Account erasure | `requestAccountDeletion` / `confirmAccountDeletion`; sessions revoked on confirm |
| Data export | `exportMyData` — versioned JSON, no credentials |

GraphQL account mutations are self-scoped (authenticated user only; no arbitrary `userId`).

### Electron Hardened Security

- **OS Keychain / Credential Manager** for token storage
- **localStorage token yasak**
- Second active desktop session **blocked**
- Same device rehydrate **OK**
- Hardened Electron configuration (contextIsolation, no nodeIntegration in renderer)

## 3. Authorization (RBAC)

| Role | Capabilities |
|------|-------------|
| OWNER | Full org control, deletion, member management |
| ADMIN | Config, integrations, member invite, analysis |
| ANALYST | Run analysis, import, view reports |
| VIEWER | Read-only access to reports |

Cross-tenant erişim: **REJECT + SECURITY EVENT**

## 4. Phase 5 Analysis Run Security (IMPLEMENTED)

Phase 5 agent analysis runs use canonical GraphQL operations:

| Operation | RBAC |
|-----------|------|
| `startAgentRun` | ANALYST+ (tenant-scoped) |
| `cancelAgentRun` | ADMIN; or ANALYST on own run |
| `agentRun` / `analysisRuns` / `agentRunEvents` | VIEWER+ read within tenant |

**Tenant isolation:** Cross-tenant start/read/list/event access → `FORBIDDEN` + security event.

**Idempotency:** Duplicate `clientRequestId` within an organization returns the same run.

**Internal IPC:** Python orchestrator calls Go-only `/internal/agent/v1/*` endpoints with `X-Miyuna-Internal-Token`. Missing/invalid token → `401`.

## 5. Tool Authorization Chain (Phase 5 — IMPLEMENTED for deterministic tools)

```text
Python Tool Intent (RunManager)
  → Go Tool Policy + Registry version check
  → RBAC / tenant scope (run-bound)
  → Input hash validation
  → Redis grant issue (hashed key, TTL, run/tool/input binding)
  → Go tool execution (single-use atomic consume)
  → Output schema + compliance validation before observation persist
  → Audit event (agent_run_events)
```

- Python orchestrator **cannot** bypass Go authorization or execute tools directly
- LLM tool intent (Phase 6+) still flows through the same chain — **NOT IMPLEMENTED in Phase 5**
- Redis grant replay / binding mismatch / expiry → REJECT + security event; fail-closed when Redis unavailable
- Unavailable registry tools (`import_planner`, etc.) cannot be authorized or executed in Phase 5 analysis runs

## 6. Fetch Security

External fetch (ecommerce_fetcher, URL import) controls:

| Control | Description |
|---------|-------------|
| SSRF prevention | Block private IP ranges, localhost, metadata endpoints |
| DNS rebinding | Resolve-then-connect validation |
| Allow/deny lists | Tenant-configurable permitted domains |
| Redirect limit | Max redirect hops (e.g., 3) |
| Timeout | Configurable per-request timeout |
| Max size | Response size cap |
| Tenant quota | Per-org fetch rate/volume limits |
| Rate limit | Global + per-tenant |
| Retry | Backoff with max attempts |
| Credential encryption | At rest + in transit |
| Input sanitization | URL/path parameter validation |
| Malicious scan | Content-type and payload inspection |
| Audit | All fetch attempts logged |

## 7. Credential Management

Platform store API credential modeli (`EcommerceIntegration`) **iptal edildi (2026-09-04)** — v1'de implement edilmez.

Genel credential kuralları (gelecek CR'ler için referans):
- Never logged, never returned to client
- Revocation support

### General

- Secrets via environment/deploy-time config
- No secrets in source code
- Rotation support architecture

## 8. Multi-Tenant Security

- organizationId from authenticated session only
- Repository layer enforces organizationId on every query
- Tenant escape test mandatory (see [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md))
- Cross-tenant GraphQL ID access → 403 + security event

## 9. Security Events

```text
SecurityEvent {
  eventType: CROSS_TENANT_ACCESS | UNAUTHORIZED_TOOL | FETCH_BLOCKED |
             AUTH_FAILURE | MFA_FAILURE | SESSION_HIJACK_ATTEMPT | ...
  severity: INFO | WARNING | CRITICAL
  organizationId?: string
  userId?: string
  details: object        // no PII, no credentials
  timestamp: ISO8601
}
```

Critical events trigger alerting (implementation Phase 1+).

## 10. GraphQL Security

- Authentication required for all business operations
- Health/readiness dışında public REST business API yok
- Public prod GraphQL playground: **OUT OF SCOPE**
- Query depth/complexity limits
- Rate limiting (application layer + Cloudflare edge)

## 11. PII Protection

- `pii_redactor` in agent pipeline
- PII never in audit logs (plaintext)
- Compliance engine always-on (see [COMPLIANCE.md](./COMPLIANCE.md))

## 12. Account & Organization Deletion Security

- Ownership transfer required before owner deletion
- Org deletion requires OWNER role
- Cascade data purge with audit trail
- Deletion confirmation flow (prevent accidental deletion)

## 13. Internal Service Exposure

**Forbidden:**

- MongoDB public exposure
- Redis public exposure
- Agent Orchestrator public exposure
- LLM runtime public exposure

Internal services accessible only via private network / internal API boundaries.

## 14. Mail Security

MailService abstraction:

- Dev: MailHog (local only)
- Prod: SMTP provider (TLS required)
- No credentials in client-side code
- Email verification tokens: time-limited, single-use

## 15. Related Documents

- [CLOUDFLARE.md](./CLOUDFLARE.md) — edge security
- [COMPLIANCE.md](./COMPLIANCE.md) — KVKK/GDPR
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — tool authorization
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) — tenant isolation
- [CI_CD.md](./CI_CD.md) — secure deployment

## 16. UNRESOLVED

| Item | Status |
|------|--------|
| Encryption algorithm for credentials at rest | UNRESOLVED — Phase 1 |
| Session token TTL values | UNRESOLVED — Phase 1 |
| Security event alerting channel | UNRESOLVED — observability phase |
| Penetration test schedule | UNRESOLVED — pre-production |
