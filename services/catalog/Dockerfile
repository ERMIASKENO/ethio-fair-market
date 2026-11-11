# syntax=docker/dockerfile:1.7
# ---------------------------------------------------------------------------------------------
# ethio-fair-market / catalog (NestJS)   build context: repository root
#   docker build -f services/catalog/Dockerfile -t efm-catalog .
# ---------------------------------------------------------------------------------------------
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS deps
WORKDIR /app
COPY services/catalog/package.json services/catalog/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund

FROM deps AS build
COPY services/catalog/tsconfig.json services/catalog/tsconfig.build.json services/catalog/jest.config.js ./
COPY services/catalog/src ./src
COPY services/catalog/test ./test
RUN npm run lint && npm test && npm run build \
 && npm prune --omit=dev --no-audit --no-fund

# distroless Node.js: no shell or npm in the final image; runs as user 65532
FROM gcr.io/distroless/nodejs${NODE_VERSION}-debian12:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.title="efm-catalog" \
      org.opencontainers.image.description="ethio-fair-market catalog service" \
      org.opencontainers.image.source="https://github.com/ermiaskeno/ethio-fair-market" \
      org.opencontainers.image.version="${VERSION}"
WORKDIR /app
ENV NODE_ENV=production PORT=8080 APP_VERSION=${VERSION}
COPY --from=build --chown=65532:65532 /app/package.json ./
COPY --from=build --chown=65532:65532 /app/node_modules ./node_modules
COPY --from=build --chown=65532:65532 /app/dist ./dist
USER 65532:65532
EXPOSE 8080
CMD ["dist/main.js"]
