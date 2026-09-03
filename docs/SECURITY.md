# Miyuna — Security

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.3 FROZEN  
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

## 4. Tool Authorization Chain

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

- LLM doğrudan tool execute edemez
- LLM doğrudan internete erişemez
- Yetkisiz → REJECT + SECURITY EVENT

## 5. Fetch Security

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

## 6. Credential Management

### EcommerceIntegration

- API keys/tokens encrypted at rest (MongoDB)
- Decryption only in secure runtime context
- Never logged, never returned to client
- Revocation support

### General

- Secrets via environment/deploy-time config
- No secrets in source code
- Rotation support architecture

## 7. Multi-Tenant Security

- organizationId from authenticated session only
- Repository layer enforces organizationId on every query
- Tenant escape test mandatory (see [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md))
- Cross-tenant GraphQL ID access → 403 + security event

## 8. Security Events

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

## 9. GraphQL Security

- Authentication required for all business operations
- Health/readiness dışında public REST business API yok
- Public prod GraphQL playground: **OUT OF SCOPE**
- Query depth/complexity limits
- Rate limiting (application layer + Cloudflare edge)

## 10. PII Protection

- `pii_redactor` in agent pipeline
- PII never in audit logs (plaintext)
- Compliance engine always-on (see [COMPLIANCE.md](./COMPLIANCE.md))

## 11. Account & Organization Deletion Security

- Ownership transfer required before owner deletion
- Org deletion requires OWNER role
- Cascade data purge with audit trail
- Deletion confirmation flow (prevent accidental deletion)

## 12. Internal Service Exposure

**Forbidden:**

- MongoDB public exposure
- Redis public exposure
- Agent Orchestrator public exposure
- LLM runtime public exposure

Internal services accessible only via private network / internal API boundaries.

## 13. Mail Security

MailService abstraction:

- Dev: MailHog (local only)
- Prod: SMTP provider (TLS required)
- No credentials in client-side code
- Email verification tokens: time-limited, single-use

## 14. Related Documents

- [CLOUDFLARE.md](./CLOUDFLARE.md) — edge security
- [COMPLIANCE.md](./COMPLIANCE.md) — KVKK/GDPR
- [AGENT_ORCHESTRATION.md](./AGENT_ORCHESTRATION.md) — tool authorization
- [MONGODB_SCHEMA.md](./MONGODB_SCHEMA.md) — tenant isolation
- [CI_CD.md](./CI_CD.md) — secure deployment

## 15. UNRESOLVED

| Item | Status |
|------|--------|
| Encryption algorithm for credentials at rest | UNRESOLVED — Phase 1 |
| Session token TTL values | UNRESOLVED — Phase 1 |
| Security event alerting channel | UNRESOLVED — observability phase |
| Penetration test schedule | UNRESOLVED — pre-production |
