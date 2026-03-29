#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

usage() {
  cat <<'EOF'
Usage:
  tools/docker/test-backend.sh [--unit|--integration] [options] [packages...]

Examples:
  tools/docker/test-backend.sh --unit
  tools/docker/test-backend.sh --unit ./internal/service --run TestGetAccountsLoadBatch
  tools/docker/test-backend.sh --integration ./internal/repository -run TestGatewayCacheSuite

Options:
  --unit                  Run unit tests (default).
  --integration           Run integration tests.
  --tags <value>          Override go test tags (default: unit/integration by mode).
  --run <regex>           Pass -run to go test.
  --timeout <duration>    Pass -timeout to go test, e.g. 60s.
  --count <n>             Pass -count to go test.
  --race                  Enable -race.
  --short                 Enable -short.
  --failfast              Enable -failfast.
  --verbose, -v           Enable -v.
  --help, -h              Show this help.

Environment:
  GO_TEST_RUNNER_IMAGE    Override Docker image used to run tests.
  GOPROXY / GOSUMDB       Override Go module proxy settings inside container.
EOF
}

mode="unit"
tags=""
run_filter=""
packages=()
go_test_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --unit)
      mode="unit"
      ;;
    --integration)
      mode="integration"
      ;;
    --tags)
      [[ $# -ge 2 ]] || fail "--tags requires a value"
      tags="$2"
      shift
      ;;
    --tags=*)
      tags="${1#*=}"
      ;;
    --run)
      [[ $# -ge 2 ]] || fail "--run requires a value"
      run_filter="$2"
      shift
      ;;
    --run=*)
      run_filter="${1#*=}"
      ;;
    --timeout)
      [[ $# -ge 2 ]] || fail "--timeout requires a value"
      go_test_args+=("-timeout=$2")
      shift
      ;;
    --timeout=*)
      go_test_args+=("-timeout=${1#*=}")
      ;;
    --count)
      [[ $# -ge 2 ]] || fail "--count requires a value"
      go_test_args+=("-count=$2")
      shift
      ;;
    --count=*)
      go_test_args+=("-count=${1#*=}")
      ;;
    --race)
      go_test_args+=("-race")
      ;;
    --short)
      go_test_args+=("-short")
      ;;
    --failfast)
      go_test_args+=("-failfast")
      ;;
    --verbose|-v)
      go_test_args+=("-v")
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    --)
      shift
      packages+=("$@")
      break
      ;;
    -*)
      fail "unknown option: $1"
      ;;
    *)
      packages+=("$1")
      ;;
  esac
  shift
done

if [[ -z "${tags}" ]]; then
  tags="${mode}"
fi

if [[ -n "${run_filter}" ]]; then
  go_test_args+=("-run=${run_filter}")
fi

if [[ ${#packages[@]} -eq 0 ]]; then
  packages=(./...)
fi

ensure_docker
ensure_go_test_runner_image

docker_cmd=(
  docker run --rm
  -v "${repo_root}:/work"
  --mount type=volume,src=sub2api-go-build-cache,target=/tmp/go-build
  --mount type=volume,src=sub2api-go-mod-cache,target=/tmp/go-mod
  -w /work/backend
  -e "GOPROXY=${GOPROXY:-https://goproxy.cn,direct}"
  -e "GOSUMDB=${GOSUMDB:-sum.golang.google.cn}"
  -e "GOTOOLCHAIN=local"
  -e "GOCACHE=/tmp/go-build"
  -e "GOMODCACHE=/tmp/go-mod"
)

if [[ "${mode}" == "integration" ]]; then
  docker_socket="$(docker_socket_path || true)"
  [[ -n "${docker_socket}" ]] || fail "docker socket not found; set DOCKER_SOCKET_PATH or DOCKER_HOST=unix://..."
  docker_host="$(docker_host_value "${docker_socket}")"
  docker_cmd+=(
    -v "${docker_socket}:${docker_socket}"
    -e "DOCKER_HOST=${docker_host}"
    -e "TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=${docker_socket}"
  )
fi

docker_cmd+=(
  "$(go_test_runner_image)"
  /usr/local/go/bin/go test
  "-tags=${tags}"
)
if [[ ${#go_test_args[@]} -gt 0 ]]; then
  docker_cmd+=("${go_test_args[@]}")
fi
docker_cmd+=("${packages[@]}")

log_step "Running backend ${mode} tests in Docker"
"${docker_cmd[@]}"
