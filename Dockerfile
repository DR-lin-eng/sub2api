# =============================================================================
# Sub2API Multi-Stage Dockerfile
# =============================================================================
# Stage 1: Build frontend
# Stage 2: Build Rust runtime artifacts
# Stage 3: Build Go backend with embedded frontend
# Stage 4: Copy PostgreSQL client tools
# Stage 5: Final minimal image
# =============================================================================

ARG NODE_IMAGE=node:24-bookworm-slim
ARG GOLANG_IMAGE=golang:1.26.2-bookworm
ARG RUST_IMAGE=rust:1-bookworm
ARG BASE_IMAGE=debian:bookworm-slim
ARG POSTGRES_IMAGE=postgres:18
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ARG RELEASE_REPO=DR-lin-eng/sub2api
ARG REPO_URL=https://github.com/DR-lin-eng/sub2api

# -----------------------------------------------------------------------------
# Stage 1: Frontend Builder
# -----------------------------------------------------------------------------
FROM ${NODE_IMAGE} AS frontend-builder

WORKDIR /app/frontend

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Install dependencies first (better caching)
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy frontend source and build
COPY frontend/ ./
RUN pnpm run build

# -----------------------------------------------------------------------------
# Stage 2: Rust Runtime Builder
# -----------------------------------------------------------------------------
FROM ${RUST_IMAGE} AS rust-builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential pkg-config && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app/backend/rust

COPY backend/rust/ ./

RUN cargo build --release --locked && \
    test -x target/release/proxyd && \
    test -f target/release/libstreamcore.so

# -----------------------------------------------------------------------------
# Stage 3: Backend Builder
# -----------------------------------------------------------------------------
FROM ${GOLANG_IMAGE} AS backend-builder

# Build arguments for version info (set by CI)
ARG VERSION=
ARG COMMIT=docker
ARG DATE
ARG GOPROXY
ARG GOSUMDB
ARG RELEASE_REPO

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# Install build dependencies. build-essential is required so cgo includes the real
# Rust FFI loader instead of the non-cgo stub.
RUN apt-get update && \
    apt-get install -y --no-install-recommends git ca-certificates tzdata build-essential pkg-config && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app/backend

# Copy go mod files first (better caching)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source first
COPY backend/ ./

# Copy frontend dist from previous stage (must be after backend copy to avoid being overwritten)
COPY --from=frontend-builder /app/backend/internal/web/dist ./internal/web/dist

# Build the binary (BuildType=release for CI builds, embed frontend)
# Version precedence: build arg VERSION > cmd/server/VERSION
RUN VERSION_VALUE="${VERSION}" && \
    if [ -z "${VERSION_VALUE}" ]; then VERSION_VALUE="$(tr -d '\r\n' < ./cmd/server/VERSION)"; fi && \
    RELEASE_REPO_VALUE="${RELEASE_REPO}" && \
    DATE_VALUE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}" && \
    CGO_ENABLED=1 GOOS=linux go build \
    -tags embed \
    -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=${COMMIT} -X main.Date=${DATE_VALUE} -X main.BuildType=release -X main.ReleaseRepo=${RELEASE_REPO_VALUE}" \
    -trimpath \
    -o /app/sub2api \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 4: PostgreSQL Client (version-matched with docker-compose)
# -----------------------------------------------------------------------------
FROM ${POSTGRES_IMAGE} AS pg-client

# -----------------------------------------------------------------------------
# Stage 5: Final Runtime Image
# -----------------------------------------------------------------------------
FROM ${BASE_IMAGE}
ARG REPO_URL

# Labels
LABEL maintainer="Sub2API Contributors"
LABEL description="Sub2API - AI API Gateway Platform"
LABEL org.opencontainers.image.source="${REPO_URL}"

# Install runtime dependencies.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    gosu \
    libedit2 \
    libgcc-s1 \
    libkrb5-3 \
    libldap-2.5-0 \
    liblz4-1 \
    libpq5 \
    python3 \
    tzdata \
    wget \
    zstd \
    && rm -rf /var/lib/apt/lists/*

# Copy pg_dump and psql from the same postgres image used in docker-compose
# This ensures version consistency between backup tools and the database server
COPY --from=pg-client /usr/lib/postgresql/18/bin/pg_dump /usr/local/bin/pg_dump
COPY --from=pg-client /usr/lib/postgresql/18/bin/psql /usr/local/bin/psql

# Create non-root user
RUN groupadd -g 1000 sub2api && \
    useradd -u 1000 -g sub2api -m -s /bin/sh sub2api

# Set working directory
WORKDIR /app

# Default artifact paths used when Rust integration is enabled via config/env.
ENV RUST_SIDECAR_BINARY_PATH=/app/bin/sub2api-rust-proxyd \
    RUST_FFI_LIBRARY_PATH=/app/lib/libstreamcore.so

RUN mkdir -p /app/data /app/bin /app/lib && \
    chown -R sub2api:sub2api /app/data /app/bin /app/lib

# Copy binary/resources with ownership to avoid extra full-layer chown copy
COPY --from=backend-builder --chown=sub2api:sub2api /app/sub2api /app/sub2api
COPY --from=backend-builder --chown=sub2api:sub2api /app/backend/resources /app/resources
COPY --from=rust-builder --chown=sub2api:sub2api /app/backend/rust/target/release/proxyd /app/bin/sub2api-rust-proxyd
COPY --from=rust-builder --chown=sub2api:sub2api /app/backend/rust/target/release/libstreamcore.so /app/lib/libstreamcore.so

# Copy entrypoint script (fixes volume permissions then drops to sub2api)
COPY deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Expose port (can be overridden by SERVER_PORT env var)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q -T 5 -O /dev/null http://localhost:${SERVER_PORT:-8080}/health || exit 1

# Run the application (entrypoint fixes /app/data ownership then execs as sub2api)
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
