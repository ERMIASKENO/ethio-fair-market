# syntax=docker/dockerfile:1.7
# ---------------------------------------------------------------------------------------------
# ethio-fair-market / listings (Go)   build context: repository root
#   docker build -f services/listings/Dockerfile -t efm-listings .
# Stages: deps (module cache) -> test (optional target) -> build (static binary) -> runtime (distroless)
# ---------------------------------------------------------------------------------------------
ARG GO_VERSION=1.25

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS deps
WORKDIR /src
# go.mod/go.sum first: dependency download is cached until they change
COPY libs/go/kit/go.mod libs/go/kit/go.sum libs/go/kit/
COPY services/listings/go.mod services/listings/go.sum services/listings/
RUN --mount=type=cache,target=/go/pkg/mod cd services/listings && go mod download && go mod verify

FROM deps AS source
COPY libs/go/kit libs/go/kit
COPY services/listings services/listings

# docker build --target test ...  (CI runs the tests separately; this target is for local use)
FROM source AS test
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd services/listings && go vet ./... && go test ./...

FROM source AS build
ARG TARGETOS TARGETARCH
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd services/listings && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -buildvcs=false \
      -ldflags="-s -w -X github.com/ermiaskeno/ethio-fair-market/libs/go/kit.Version=${VERSION}" \
      -o /out/app .

# distroless/static: no shell, no package manager, CA certificates + tzdata only (~2 MB)
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.title="efm-listings" \
      org.opencontainers.image.description="ethio-fair-market listings service" \
      org.opencontainers.image.source="https://github.com/ermiaskeno/ethio-fair-market" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="UNLICENSED"
COPY --from=build /out/app /app
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app"]
