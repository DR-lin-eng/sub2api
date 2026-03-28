#!/usr/bin/env bash

set -euo pipefail

docker_tools_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${docker_tools_dir}/../.." && pwd)"

log_step() {
  printf '\n==> %s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

ensure_docker() {
  command -v docker >/dev/null 2>&1 || fail "docker is not installed"
  docker info >/dev/null 2>&1 || fail "docker daemon is not available"
}

go_test_runner_image() {
  printf '%s' "${GO_TEST_RUNNER_IMAGE:-sub2api-go-test-runner:local}"
}

ensure_go_test_runner_image() {
  local image
  image="$(go_test_runner_image)"
  log_step "Building Go test runner image (${image})"
  docker build \
    -f "${docker_tools_dir}/Dockerfile.go-test" \
    -t "${image}" \
    "${docker_tools_dir}"
}

docker_socket_path() {
  if [[ -n "${DOCKER_SOCKET_PATH:-}" ]]; then
    printf '%s' "${DOCKER_SOCKET_PATH}"
    return 0
  fi
  if [[ -n "${DOCKER_HOST:-}" && "${DOCKER_HOST}" == unix://* ]]; then
    printf '%s' "${DOCKER_HOST#unix://}"
    return 0
  fi
  if [[ -S /var/run/docker.sock ]]; then
    printf '%s' "/var/run/docker.sock"
    return 0
  fi
  return 1
}

docker_host_value() {
  local socket_path="$1"
  if [[ -n "${DOCKER_HOST:-}" ]]; then
    printf '%s' "${DOCKER_HOST}"
    return 0
  fi
  printf 'unix://%s' "${socket_path}"
}
