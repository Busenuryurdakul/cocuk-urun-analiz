# Miyuna

**Çocuk ürünleri için agent destekli analiz ve karar destek platformu.**

> Skoru değil, skorun kanıtını göster.

Architecture source of truth: [`docs/FINAL_MASTER_PROMPT.md`](docs/FINAL_MASTER_PROMPT.md) v1.0.2 FROZEN

## Monorepo layout

```text
apps/
  web/     Next.js 14+ (App Router, TypeScript, Tailwind)
  api/     Go GraphQL API (gqlgen) — health/readiness + GraphQL stub
  agent/   Python Agent Orchestrator (internal, FastAPI stub)
infra/
  docker/  Dev stack: MongoDB, Redis, MinIO, MailHog
  scripts/ XCS internal orchestration (Phase 1+)
docs/      Phase 0 architecture documents
```

## Prerequisites

- Node.js 20+
- Go 1.24+
- Python 3.11+
- Docker (for local infra)

## Quick start

### 1. Infrastructure

```bash
docker compose -f infra/docker/docker-compose.yml up -d
```

### 2. Environment

```bash
cp .env.example .env
```

**Secrets note:** `.env.example` contains **development placeholders only** — not real production secrets. Values such as `S3_SECRET_KEY`, `MINIO_ROOT_PASSWORD` (in `infra/docker/docker-compose.yml`), and `CREDENTIALS_ENCRYPTION_KEY` are **development only** and **must be rotated/replaced before production**.

### 3. API (Go)

```bash
cd apps/api
go run ./cmd/server
# http://localhost:8080/health
# http://localhost:8080/graphql
```

GraphQL playground is enabled in local dev by default (`ALLOW_GRAPHQL_PLAYGROUND=true`). **Production intent:** set `ALLOW_GRAPHQL_PLAYGROUND=false` — public production GraphQL playground is OUT OF SCOPE per master prompt.

### 4. Agent (Python, internal)

```bash
cd apps/agent
pip install -e ".[dev]"
uvicorn miyuna_agent.main:app --host 127.0.0.1 --port 8090
```

### 5. Web (Next.js)

```bash
cd apps/web
npm install
npm run dev
# http://localhost:3000
```

## Phase 1 scope

Foundation scaffold only — no auth, Shopify, LLM, or agent pipeline yet.

Each delivery phase: **PLAN → IMPLEMENT → TEST → VERIFY → REPORT → STOP/APPROVAL**

## Development troubleshooting

### Windows: `go test` blocked by Application Control

Some corporate or Windows Application Control environments block test executables that `go test` builds under temporary paths (for example `%LocalAppData%\Temp\go-build*`).

- **Do not** disable or bypass Windows security policy.
- **Canonical verification:** CI on Linux (`go test ./...` in `.github/workflows/ci.yml`) is the authoritative test result for merges.
- **Local workaround (same tests, project-local binary):**

```powershell
cd apps/api
go test -c -o bin/health.test.exe ./internal/health
.\bin\health.test.exe -test.v
```

`apps/api/bin/` and `*.test.exe` are gitignored. This workaround does **not** replace CI; it only helps local verification when standard `go test` is host-blocked.

## License

Private — see repository owner.
