# Ethio Fair Market

An online marketplace for Ethiopia – buy and sell cars, houses, phones, jobs, services and more.
Amharic and English, prices in birr, login with an Ethiopian mobile number, locations down to the sub-city.

Built as a cloud-native microservice system: polyglot services, multi-stage container images,
GitHub Actions CI, Helm + Kustomize, deployed with Argo CD (GitOps) on Kubernetes.

## Services (phase 1)

| Service | Language | Path | Responsibility | Data |
|---|---|---|---|---|
| [web](web) | TypeScript · Next.js 16 | `/` | server-rendered website, Amharic/English | – |
| [identity](services/identity) | Go | `/v1/auth` | phone + one-time code login, EdDSA JWT, JWKS | PostgreSQL `identity` |
| [listings](services/listings) | Go | `/v1/listings` | ads: search (full text, filters), create/edit, expiry | PostgreSQL `listings` |
| [catalog](services/catalog) | TypeScript · NestJS 11 | `/v1/categories` | category tree, attribute schema per category | in code |
| [locations](services/locations) | Python · FastAPI | `/v1/locations` | regions, cities, Addis Ababa sub-cities | in code |

Shared Go platform code: [libs/go/kit](libs/go/kit) (logging, metrics, health, graceful shutdown, JWT, migrations).
Architecture and roadmap: [docs/architecture.md](docs/architecture.md).

## Run locally

```bash
# Go services (needs a PostgreSQL)
cd services/identity && DATABASE_URL=postgres://... JWT_SIGNING_KEY_FILE=key.pem OTP_DEV_MODE=true COOKIE_SECURE=false PORT=8081 go run .
cd services/listings && DATABASE_URL=postgres://... JWKS_URL=http://localhost:8081/v1/auth/.well-known/jwks.json SEED_DEMO_DATA=true PORT=8082 go run .
# catalog / locations
cd services/catalog && npm ci && npm run build && PORT=8083 npm start
cd services/locations && pip install -r requirements.txt && uvicorn app.main:app --port 8084
# web (proxies /v1/* to the services above)
cd web && npm ci && DEV_API_PROXY=1 LISTINGS_URL=http://localhost:8082 CATALOG_URL=http://localhost:8083 LOCATIONS_URL=http://localhost:8084 npm run dev
```

Signing key for local runs: `openssl genpkey -algorithm ed25519 -out key.pem`.

## Tests

```bash
cd libs/go/kit && go test ./...
cd services/identity && TEST_DATABASE_URL=postgres://.../identity go test ./...     # incl. Postgres integration tests
cd services/listings && TEST_DATABASE_URL=postgres://.../listings go test ./...
cd services/catalog && npm test
cd services/locations && pip install -r requirements-dev.txt && pytest
cd web && npm test
```

## Container images

Every service has a multi-stage `Dockerfile`; the build context is always the repository root:

```bash
docker build -f services/listings/Dockerfile -t efm-listings .
docker build -f web/Dockerfile -t efm-web .
```

Go services run on `distroless/static` (no shell, ~10 MB), Node services on `distroless/nodejs22`,
Python on `python:3.13-slim` with a non-root user – all with a read-only root filesystem in Kubernetes.

## CI/CD

`.github/workflows/ci.yml`: changed services only → tests (Go with a real PostgreSQL, Node, Python) →
image build with cache, SBOM and provenance → push to `ghcr.io/ermiaskeno/efm-<service>` → Trivy scan →
commit the new tag to [ethio-fair-market-gitops](https://github.com/ERMIASKENO/ethio-fair-market-gitops) → Argo CD deploys.
