# Miyuna — CI/CD Pipeline

> **Source of Truth:** [FINAL_MASTER_PROMPT.md](./FINAL_MASTER_PROMPT.md) v1.0.4 FROZEN (CR-005)  
> Bu doküman master prompt ile çelişemez.

## 1. Pipeline Principles

| Rule | Description |
|------|-------------|
| Feature branch workflow | All work on feature branches |
| PR required | Changes merge via Pull Request |
| Manual prod deploy | No automatic production deployment |
| main direct push forbidden | Protected branch |
| Test gates | Tests must pass before merge |
| Scope freeze | CI does not deploy out-of-scope features |

## 2. Branch Strategy

```text
main (protected)
  ↑
  PR ← feature/phase-N-description
  ↑
  PR ← feature/phase-N-subtask
```

### Branch Protection (main)

- Direct push: **FORBIDDEN**
- Require PR review
- Require CI status checks pass
- No force push

## 3. CI Pipeline Stages

```text
Trigger (PR or push to feature branch)
  ↓
Lint & Type Check
  ↓
Unit Tests
  ↓
Integration Tests
  ↓
Security Scan (dependency + SAST)
  ↓
Build Verification
  ↓
Acceptance Tests (phase-specific)
  ↓
Report → PR status check
```

### Phase-Specific Acceptance Tests

| Phase | Key Tests |
|-------|-----------|
| Auth | MFA flow, device verification |
| Agent | Real AgentRunEvent, no fake progress |
| LLM | 2 LLM routing, 5 concurrent load test |
| Product data import | CSV/JSON/manual + permitted URL (not mock) |
| Evidence | Claim-evidence validation |
| Compliance | Engine always-on verification |
| Security | Tenant escape test |
| Electron | Hardened build, Keychain storage |
| Cloudflare | Production traffic via edge |
| Fine-tune | Eval gate before publish |

## 4. CD Pipeline (Deployment)

```text
PR merged to main
  ↓
Build artifacts
  ↓
Deploy to staging (automatic or manual — UNRESOLVED)
  ↓
Staging verification
  ↓
Manual prod deploy trigger (REQUIRED)
  ↓
Production (via Cloudflare)
```

**Auto production deploy: OUT OF SCOPE v1.0.2**

Production deployment requires explicit manual trigger and approval.

## 5. Environment Strategy

| Environment | Purpose | Deploy Trigger |
|-------------|---------|----------------|
| Local (dev) | Developer machines | Manual |
| CI | Test execution | Automatic on PR |
| Staging | Pre-prod verification | Post-merge (UNRESOLVED: auto vs manual) |
| Production | Live system | **Manual only** |

### Environment Configuration

- Secrets via environment variables (never in repo)
- MailService: MailHog (dev), SMTP provider (prod)
- Storage: MinIO (dev), S3-compatible (prod)
- MongoDB/Redis: environment-specific instances

## 6. XCS Integration

XCS (project-internal orchestration) provides deploy automation scripts in `infra/scripts/`:

- Environment provisioning
- GPU/RAM-based agent runtime setup
- Deploy orchestration helpers

XCS is not a separate CI system; it complements the main pipeline.

## 7. Artifact Management

| Component | Build Output |
|-----------|-------------|
| Next.js Web | Static + server bundle |
| Go GraphQL API | Binary / container |
| Python Agent Orchestrator | Package / container |
| Electron Desktop | Windows + macOS installers |

Linux Electron: bonus only, not required for v1 acceptance.

## 8. Test Requirements (v1.0.2 Acceptance)

Mandatory demonstrable acceptance:

- [ ] Agent pipeline with real events
- [ ] 2 LLM with worker/reviewer rotation
- [ ] Evidence-based reports with disclaimer
- [ ] Security: tenant escape test pass
- [ ] Product import: CSV/JSON/manual + permitted URL validation
- [ ] Fine-tune eval gate
- [ ] Admin LLM control center + snapshot demo
- [ ] KVKK/GDPR compliance engine
- [ ] Electron hardened (Win + Mac)
- [ ] Cloudflare production traffic
- [ ] MFA + device verification
- [ ] Org/account deletion

## 9. Rollback Strategy

- ConfigSnapshot rollback via Manual LLM Control Center
- Application rollback: deploy previous artifact version
- Database: migration rollback scripts (Phase 1+)
- Rollback requires audit trail

## 10. Monitoring Post-Deploy

- Health/readiness endpoints
- Agent run success/failure rates
- LLM routing load metrics
- Security event alerts
- Cloudflare analytics

## 11. Related Documents

- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [CLOUDFLARE.md](./CLOUDFLARE.md) — production edge
- [SECURITY.md](./SECURITY.md) — security gates
- [SCOPE.md](./SCOPE.md) — what's in/out of pipeline scope

## 12. UNRESOLVED

| Item | Status |
|------|--------|
| CI provider (GitHub Actions vs other) | UNRESOLVED — Phase 1 |
| Staging auto-deploy vs manual | UNRESOLVED — Phase 1 |
| Container registry choice | UNRESOLVED — Phase 1 |
| Electron code signing certificates | UNRESOLVED — desktop phase |
