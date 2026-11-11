# Architecture

## Request flow

```
Browser ──HTTPS──▶ Envoy Gateway (192.168.1.240, *.shop.internal certificate from your own CA)
                    │  efm.shop.internal/              → web        (Next.js, server-side rendering)
                    │  efm.shop.internal/v1/auth       → identity
                    │  efm.shop.internal/v1/listings   → listings
                    │  efm.shop.internal/v1/categories → catalog
                    │  efm.shop.internal/v1/locations  → locations
                    ▼
web ──(cluster network)──▶ listings, catalog, locations     (page rendering on the server)
listings ──▶ identity (JWKS, cached 10 min) · catalog + locations (validate category / city)
identity, listings ──▶ PostgreSQL efm-pg (namespace efm-data, one database + role per service)
Prometheus ──▶ /metrics of every service          Argo CD ──▶ watches ethio-fair-market-gitops
```

The browser only ever talks to one origin (`https://efm.shop.internal`), so there is no CORS and the
session cookie (`efm_token`, HttpOnly, Secure, SameSite=Lax) works for both the website and the APIs.

## Authentication

1. `POST /v1/auth/otp/request {phone}` – the number is normalised to E.164 (`+2519…`/`+2517…`), a 6-digit
   code is stored only as a salted SHA-256 hash, valid 5 minutes, max 3 codes per 10 minutes per number.
2. `POST /v1/auth/otp/verify {phone, code}` – max 5 attempts, single use; returns a JWT and sets the cookie.
3. Tokens are **EdDSA (Ed25519)** JWTs. Other services verify them offline with the public keys from
   `/v1/auth/.well-known/jwks.json` – no call to identity per request.

Until the SMS integration exists (phase 2), `OTP_DEV_MODE=true` returns the code to the login page.

## Conventions every service follows

| Topic | Rule |
|---|---|
| Health | `/healthz` = process alive (liveness), `/readyz` = dependencies OK and not shutting down (readiness) |
| Metrics | `/metrics` in Prometheus format; `http_requests_total{route,code}`, `http_request_duration_seconds{route}` with the route *pattern* (low cardinality) |
| Logs | one JSON object per line to stdout: `time, level, msg, service, version, request_id …` |
| Errors | `{"error":{"code":"…","message":"…"}}`, validation errors add `fields` |
| Request id | `X-Request-Id` accepted or generated, returned, logged |
| Shutdown | SIGTERM → readiness false → short delay → finish open requests (+ Kubernetes `preStop` sleep) |
| Config | environment variables only; secrets from Kubernetes Secrets |
| Data | each service owns its database; no shared tables |
| Images | multi-stage, non-root, read-only filesystem, no shell where possible |

## Data model (listings)

`listings` table: price in **birr** (`numeric(14,2)`), category slug (`vehicles/cars`), region code + city +
optional sub-city, condition, category specific `attributes` (JSONB), status `active|sold|expired|removed`,
30-day expiry, featured flag (for paid promotion later). Full-text search uses PostgreSQL `tsvector` with the
`simple` configuration, which works for Amharic (Ge'ez script) and English alike.

## Roadmap

| Phase | Services / features |
|---|---|
| **1 (this)** | web, identity, listings, catalog, locations · Helm chart · Kustomize platform · CI/CD · Argo CD |
| 2 | **media** (Go: image upload + resize), **search** (Rust: dedicated search index), **chat** (Node.js WebSocket + Redis), **notifications** (Python: SMS/e-mail, real OTP delivery), **NATS** event bus (`listing.created` → search, moderation, notifications), Sealed Secrets |
| 3 | **payments** (Java Spring Boot: Chapa / Telebirr, featured ads), **reviews** (Python), **favorites** + saved-search alerts (Go), **moderation** (Python), Argo Rollouts canary releases, Istio ambient mesh (mTLS), Afaan Oromo |
