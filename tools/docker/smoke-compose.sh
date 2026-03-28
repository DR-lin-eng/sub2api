#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

ensure_docker

smoke_id="$(date +%s)-$$"
smoke_root="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-docker-smoke.${smoke_id}.XXXXXX")"
env_file="${smoke_root}/smoke.env"
port="${DOCKER_SMOKE_PORT:-18080}"
project_name="sub2api-smoke-${smoke_id}"

cat >"${env_file}" <<EOF
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-sub2api-smoke-pass}
JWT_SECRET=${JWT_SECRET:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}
TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}
BIND_HOST=127.0.0.1
SERVER_PORT=${port}
SUB2API_CONTAINER_NAME=sub2api-smoke-${smoke_id}
POSTGRES_CONTAINER_NAME=sub2api-smoke-postgres-${smoke_id}
REDIS_CONTAINER_NAME=sub2api-smoke-redis-${smoke_id}
SUB2API_DATA_DIR=${smoke_root}/data
POSTGRES_DATA_DIR=${smoke_root}/postgres
REDIS_DATA_DIR=${smoke_root}/redis
EOF

mkdir -p "${smoke_root}/data" "${smoke_root}/postgres" "${smoke_root}/redis"

compose_cmd=(
  docker compose
  --env-file "${env_file}"
  -f "${repo_root}/deploy/docker-compose.dev.yml"
  -p "${project_name}"
)

cleanup() {
  if [[ "${KEEP_DOCKER_SMOKE_STACK:-false}" != "true" ]]; then
    "${compose_cmd[@]}" down --remove-orphans >/dev/null 2>&1 || true
    rm -rf "${smoke_root}"
  fi
}
trap cleanup EXIT

log_step "Starting docker compose smoke stack"
"${compose_cmd[@]}" up --build -d

container_id="$("${compose_cmd[@]}" ps -q sub2api)"
[[ -n "${container_id}" ]] || fail "sub2api container did not start"

log_step "Waiting for sub2api health check"
health_status=""
for _ in $(seq 1 90); do
  health_status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${container_id}" 2>/dev/null || true)"
  case "${health_status}" in
    healthy)
      break
      ;;
    unhealthy|exited|dead)
      "${compose_cmd[@]}" logs --no-color sub2api postgres redis || true
      fail "sub2api container became ${health_status}"
      ;;
  esac
  sleep 2
done

[[ "${health_status}" == "healthy" ]] || {
  "${compose_cmd[@]}" logs --no-color sub2api postgres redis || true
  fail "timed out waiting for sub2api to become healthy"
}

log_step "Checking /health"
if command -v curl >/dev/null 2>&1; then
  curl -fsS "http://127.0.0.1:${port}/health" >/dev/null
else
  wget -q -O /dev/null "http://127.0.0.1:${port}/health"
fi

log_step "Checking startup logs"
logs="$("${compose_cmd[@]}" logs --no-color sub2api)"
if grep -Eiq 'panic:|failed to apply migrations|failed to connect to postgres|redis connection failed|database connection failed' <<<"${logs}"; then
  printf '%s\n' "${logs}" >&2
  fail "startup logs contain fatal initialization errors"
fi

printf 'Smoke check passed on http://127.0.0.1:%s\n' "${port}"
