#!/usr/bin/env bash
# Local image build helper with retry support and overridable base images.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BUILD_MODE="${BUILD_MODE:-default}"
IMAGE_TAG="${IMAGE_TAG:-sub2api:latest}"
BUILD_RETRIES="${BUILD_RETRIES:-3}"
BUILD_RETRY_DELAY="${BUILD_RETRY_DELAY:-5}"
DOCKERFILE_PATH="${DOCKERFILE_PATH:-}"

usage() {
    cat <<'EOF'
Usage:
  deploy/build_image.sh [options]

Options:
  --compat               Build with Dockerfile.compat
  --tag <image:tag>      Override output image tag
  --retries <count>      Retry count for transient docker build failures
  --retry-delay <sec>    Delay between retries in seconds
  --dockerfile <path>    Explicit dockerfile path
  --help                 Show this help message

Environment overrides:
  IMAGE_TAG
  BUILD_RETRIES
  BUILD_RETRY_DELAY
  BUILD_MODE
  DOCKERFILE_PATH
  GOPROXY
  GOSUMDB
  NODE_IMAGE
  GOLANG_IMAGE
  POSTGRES_IMAGE
  ALPINE_IMAGE
  RUNTIME_IMAGE

Examples:
  deploy/build_image.sh
  deploy/build_image.sh --compat --tag sub2api:compat
  NODE_IMAGE=docker.m.daocloud.io/library/node:24-bookworm \
    GOLANG_IMAGE=docker.m.daocloud.io/library/golang:1.26.1 \
    deploy/build_image.sh --compat
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --compat)
            BUILD_MODE="compat"
            shift
            ;;
        --tag)
            IMAGE_TAG="${2:?missing value for --tag}"
            shift 2
            ;;
        --retries)
            BUILD_RETRIES="${2:?missing value for --retries}"
            shift 2
            ;;
        --retry-delay)
            BUILD_RETRY_DELAY="${2:?missing value for --retry-delay}"
            shift 2
            ;;
        --dockerfile)
            DOCKERFILE_PATH="${2:?missing value for --dockerfile}"
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [[ -z "${DOCKERFILE_PATH}" ]]; then
    if [[ "${BUILD_MODE}" == "compat" ]]; then
        DOCKERFILE_PATH="${REPO_ROOT}/Dockerfile.compat"
    else
        DOCKERFILE_PATH="${REPO_ROOT}/Dockerfile"
    fi
fi

if ! [[ "${BUILD_RETRIES}" =~ ^[0-9]+$ ]] || ! [[ "${BUILD_RETRY_DELAY}" =~ ^[0-9]+$ ]]; then
    echo "BUILD_RETRIES and BUILD_RETRY_DELAY must be non-negative integers" >&2
    exit 1
fi

build_args=(
    --build-arg "GOPROXY=${GOPROXY:-https://goproxy.cn,direct}"
    --build-arg "GOSUMDB=${GOSUMDB:-sum.golang.google.cn}"
)

append_optional_build_arg() {
    local name="$1"
    local value="${!name:-}"
    if [[ -n "${value}" ]]; then
        build_args+=(--build-arg "${name}=${value}")
    fi
}

append_optional_build_arg NODE_IMAGE
append_optional_build_arg GOLANG_IMAGE
append_optional_build_arg POSTGRES_IMAGE
append_optional_build_arg ALPINE_IMAGE
append_optional_build_arg RUNTIME_IMAGE

attempt=1
while true; do
    echo "[build] mode=${BUILD_MODE} tag=${IMAGE_TAG} dockerfile=${DOCKERFILE_PATH} attempt=${attempt}/${BUILD_RETRIES}"
    if docker build \
        -t "${IMAGE_TAG}" \
        "${build_args[@]}" \
        -f "${DOCKERFILE_PATH}" \
        "${REPO_ROOT}"; then
        echo "[build] success"
        exit 0
    fi

    if (( attempt >= BUILD_RETRIES )); then
        echo "[build] failed after ${attempt} attempt(s)" >&2
        exit 1
    fi

    echo "[build] failed, retrying in ${BUILD_RETRY_DELAY}s..." >&2
    sleep "${BUILD_RETRY_DELAY}"
    attempt=$((attempt + 1))
done
