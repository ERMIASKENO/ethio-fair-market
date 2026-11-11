# syntax=docker/dockerfile:1.7
# ---------------------------------------------------------------------------------------------
# ethio-fair-market / web (Next.js, standalone output)   build context: repository root
#   docker build -f web/Dockerfile -t efm-web .
# ---------------------------------------------------------------------------------------------
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund

FROM node:${NODE_VERSION}-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY web/ ./
RUN npm run lint && npm test && npm run build

# standalone output = only the files the server needs (~20 MB instead of the full node_modules)
FROM gcr.io/distroless/nodejs${NODE_VERSION}-debian12:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.title="efm-web" \
      org.opencontainers.image.description="ethio-fair-market website" \
      org.opencontainers.image.source="https://github.com/ermiaskeno/ethio-fair-market" \
      org.opencontainers.image.version="${VERSION}"
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0 APP_VERSION=${VERSION}
COPY --from=build --chown=65532:65532 /app/.next/standalone ./
COPY --from=build --chown=65532:65532 /app/.next/static ./.next/static
COPY --from=build --chown=65532:65532 /app/public ./public
USER 65532:65532
EXPOSE 3000
CMD ["server.js"]
