#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

ensure_docker

log_step "Running frontend lint/typecheck in Docker"
docker run --rm \
  -v "${repo_root}:/work" \
  --mount type=volume,target=/work/frontend/node_modules \
  --mount type=volume,target=/pnpm/store \
  -w /work/frontend \
  -e PNPM_HOME=/pnpm \
  -e PNPM_STORE_DIR=/pnpm/store \
  node:24-bookworm \
  sh -lc 'corepack enable && pnpm install --frozen-lockfile && pnpm run lint:check && pnpm run typecheck'
