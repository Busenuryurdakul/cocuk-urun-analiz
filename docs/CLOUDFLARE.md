# Miyuna — Cloudflare Edge Configuration

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Requirement

**Production public traffic MUST pass through Cloudflare.**

Internal servisler (MongoDB, Redis, Agent Orchestrator, LLM) doğrudan public internete açılmaz.

## 2. DNS & TLS

| Record | Target | Proxy |
|--------|--------|-------|
| `app.{domain}` | Next.js Web origin | Proxied (orange cloud) |
| `api.{domain}` | Go GraphQL API origin | Proxied (orange cloud) |

- TLS termination at Cloudflare edge
- Full (strict) SSL mode between Cloudflare and origin
- Automatic HTTPS redirects
- HSTS enabled

Production domain: **UNRESOLVED** — deploy-time decision.

## 3. Architecture Position

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
```

Cloudflare = **Layer 1** of defense in depth (see [SECURITY.md](./SECURITY.md)).

## 4. WAF (Web Application Firewall)

### Managed Rules

- Cloudflare Managed Ruleset enabled
- OWASP Core Ruleset enabled
- Bot Fight Mode / Super Bot Fight Mode

### Custom Rules (planned)

| Rule | Action |
|------|--------|
| Block known bad IPs | Block |
| Challenge suspicious requests | Managed Challenge |
| Block non-standard HTTP methods on GraphQL | Block |
| Geo-blocking (if required by compliance) | UNRESOLVED |

## 5. Rate Limiting

### Edge Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| `api.{domain}/graphql` | UNRESOLVED | UNRESOLVED |
| `app.{domain}/*` | UNRESOLVED | UNRESOLVED |
| Auth endpoints | Stricter limits | UNRESOLVED |

Rate limit exceeded → 429 response.

Application-layer rate limiting provides secondary defense (Layer 2).

## 6. DDoS Protection

- Cloudflare automatic DDoS mitigation (L3/L4/L7)
- Under Attack Mode available for emergency
- Origin IP not exposed (proxied records only)

## 7. Bot Management

- Bot score evaluation on incoming requests
- Block/challenge automated traffic to auth and API endpoints
- Allow legitimate crawlers on public marketing pages (if any)

## 8. Origin Configuration

### Origin Server Rules

- Origin accepts traffic **only from Cloudflare IP ranges**
- Direct origin IP access blocked via firewall
- Origin certificates for Full (strict) mode

### Internal Services (NOT behind Cloudflare public)

| Service | Access |
|---------|--------|
| MongoDB | Private network only |
| Redis | Private network only |
| Agent Orchestrator | Private network only |
| LLM-1 / LLM-2 | Private network only |
| MinIO/S3 | Private network or signed URLs |

## 9. Caching Policy

| Asset Type | Cache |
|------------|-------|
| Next.js static assets | Cache (with cache busting) |
| GraphQL API responses | **No cache** (dynamic, tenant-scoped) |
| Health/readiness | No cache or short TTL |

GraphQL responses must not be cached at edge (multi-tenant, auth-required).

## 10. Headers & Security

Cloudflare + origin should enforce:

- `Strict-Transport-Security`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY` (or CSP frame-ancestors)
- `Content-Security-Policy` (web app)
- `Referrer-Policy`

## 11. Observability via Cloudflare

- Cloudflare Analytics for traffic patterns
- WAF event logs
- Rate limit trigger logs
- Integration with platform observability stack (Phase 1+)

## 12. Acceptance Criteria

| Criterion | Required |
|-----------|----------|
| Production traffic via Cloudflare | YES |
| WAF enabled | YES |
| DDoS protection active | YES |
| Edge rate limiting configured | YES |
| Internal services not public | YES |
| Direct origin access blocked | YES |

## 13. Related Documents

- [ARCHITECTURE.md](./ARCHITECTURE.md) — system topology
- [SECURITY.md](./SECURITY.md) — defense in depth
- [CI_CD.md](./CI_CD.md) — deployment through Cloudflare

## 14. UNRESOLVED

| Item | Status |
|------|--------|
| Production domain name | UNRESOLVED — deploy-time |
| Specific rate limit thresholds | UNRESOLVED — Phase 1+ |
| Geo-blocking requirements | UNRESOLVED — compliance review |
| Cloudflare plan tier | UNRESOLVED — deploy-time |
