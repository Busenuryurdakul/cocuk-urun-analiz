# Miyuna — Production Deployment (Render + Vercel + Cloudflare)

## Architecture

```text
Internet → Cloudflare (DNS, TLS, WAF, Turnstile)
              ├─ app.{domain}  → Vercel (Next.js)
              └─ api.{domain}  → Render (Go GraphQL API)
Render private network:
  miyuna-agent (Python)
  miyuna-llm-primary (Ollama)
  miyuna-llm-secondary (Ollama)
  miyuna-redis (Key Value)
MongoDB Atlas (external, MONGODB_URI)
```

## 1. Render (Backend + LLM + Agent)

### Blueprint

1. Push `render.yaml` and Dockerfiles to GitHub (`main` branch).
2. Open Blueprint:
   `https://dashboard.render.com/blueprint/new?repo=https://github.com/Busenuryurdakul/cocuk-urun-analiz`
3. Create project **Miyuna** → environment **production** → Apply.
4. Fill secrets marked `sync: false`:

| Variable | Description |
|----------|-------------|
| `MONGODB_URI` | MongoDB Atlas connection string |
| `JWT_SECRET` | 32+ char random secret |
| `AGENT_INTERNAL_TOKEN` | Shared internal token (agent + API) |
| `INTERNAL_TOKEN` | Same value for agent service |
| `WEB_BASE_URL` | `https://app.{domain}` |
| `MAIL_SMTP_*` | Production SMTP (SendGrid/SES/Resend) |
| `MAIL_FROM` | noreply@{domain} |
| `TURNSTILE_SECRET_KEY` | Cloudflare Turnstile secret |
| `CREDENTIALS_ENCRYPTION_KEY` | 32-byte key |

5. LLM services use **Standard** plan + 10GB disk (Ollama model cache).
6. Verify: `https://miyuna-api.onrender.com/health` → `ok`

### LLM internal URLs (auto-wired by Blueprint)

- Primary: `http://miyuna-llm-primary:11434/v1` → `qwen2.5:0.5b`
- Secondary: `http://miyuna-llm-secondary:11434/v1` → `llama3.2:1b`

First deploy pulls models (~5–15 min per LLM service).

## 2. Vercel (Web App)

```bash
cd apps/web
npx vercel link
npx vercel env add NEXT_PUBLIC_API_URL production
# Value: https://api.{domain}/graphql  (or Render URL until custom domain)
npx vercel env add NEXT_PUBLIC_TURNSTILE_SITE_KEY production
npx vercel deploy --prod
```

**Root directory:** `apps/web` (set in Vercel Project Settings).

| Env | Value |
|-----|-------|
| `NEXT_PUBLIC_API_URL` | `https://api.{domain}/graphql` |
| `NEXT_PUBLIC_TURNSTILE_SITE_KEY` | Cloudflare Turnstile site key |

## 3. Cloudflare

### DNS

| Type | Name | Target | Proxy |
|------|------|--------|-------|
| CNAME | `app` | `cname.vercel-dns.com` (from Vercel) | Proxied |
| CNAME | `api` | `miyuna-api.onrender.com` | Proxied |

### SSL/TLS

- Mode: **Full (strict)**
- Always Use HTTPS: ON
- HSTS: enabled

### Turnstile

1. Cloudflare Dashboard → Turnstile → Add site
2. Domains: `app.{domain}`
3. Copy Site Key → Vercel `NEXT_PUBLIC_TURNSTILE_SITE_KEY`
4. Copy Secret Key → Render `TURNSTILE_SECRET_KEY`
5. Set `TURNSTILE_ENABLED=true` on API

### WAF & Rate Limits

Apply rules from [`docs/CLOUDFLARE.md`](./CLOUDFLARE.md) §5 (auth login/register/OTP limits).

### Origin (Render)

Render Dashboard → miyuna-api → Settings → Custom Domain → add `api.{domain}`.

## 4. Post-deploy checklist

- [ ] API `/health` and `/ready` return OK
- [ ] Register → email link → MFA → login → email OTP flow
- [ ] Agent run triggers LLM via internal gateway
- [ ] Turnstile visible on auth pages
- [ ] Cloudflare proxy orange-cloud on app + api
- [ ] `COOKIE_SECURE=true`, playground disabled

## 5. Notes

- **MongoDB:** Render does not host MongoDB; use [MongoDB Atlas](https://www.mongodb.com/atlas) free tier (Frankfurt region).
- **LLM RAM:** Ollama Standard plan minimum; upgrade if model pull fails.
- **Branch:** Blueprint defaults to `main`; merge feature branch before apply or change `branch` in `render.yaml`.
