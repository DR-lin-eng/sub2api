#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tools/import_admin_accounts.sh \
    --base-url http://127.0.0.1:8080 \
    --admin-key YOUR_ADMIN_KEY \
    --file /path/to/sub2api_import.json \
    --group-id 12 \
    [--group-id 34] \
    [--keep-default-group-bind]

Description:
  Import a sub2api account export JSON file through the admin import API and
  bind imported accounts to one or more existing group IDs.

Options:
  --base-url                 Sub2API server base URL, for example http://127.0.0.1:8080
  --admin-key                Admin API key used in x-api-key header
  --file                     Path to exported JSON file
  --group-id                 Existing group ID to bind imported accounts to; may be repeated
  --keep-default-group-bind  Do not force skip_default_group_bind=true
  -h, --help                 Show this help message

Notes:
  - If an account item inside the JSON already has its own group_ids, those win.
  - The request-level group_ids are used for items that do not contain group_ids.
  - This script uses multipart/form-data and sends an Idempotency-Key automatically.
EOF
}

require_arg() {
  local value="$1"
  local name="$2"
  if [[ -z "$value" ]]; then
    echo "Missing required argument: $name" >&2
    usage >&2
    exit 1
  fi
}

BASE_URL=""
ADMIN_KEY=""
IMPORT_FILE=""
SKIP_DEFAULT_GROUP_BIND="true"
GROUP_IDS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url)
      BASE_URL="${2:-}"
      shift 2
      ;;
    --admin-key)
      ADMIN_KEY="${2:-}"
      shift 2
      ;;
    --file)
      IMPORT_FILE="${2:-}"
      shift 2
      ;;
    --group-id)
      GROUP_IDS+=("${2:-}")
      shift 2
      ;;
    --keep-default-group-bind)
      SKIP_DEFAULT_GROUP_BIND="false"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_arg "$BASE_URL" "--base-url"
require_arg "$ADMIN_KEY" "--admin-key"
require_arg "$IMPORT_FILE" "--file"

if [[ ! -f "$IMPORT_FILE" ]]; then
  echo "Import file not found: $IMPORT_FILE" >&2
  exit 1
fi

if [[ ${#GROUP_IDS[@]} -eq 0 ]]; then
  echo "At least one --group-id is required." >&2
  exit 1
fi

for id in "${GROUP_IDS[@]}"; do
  if [[ ! "$id" =~ ^[1-9][0-9]*$ ]]; then
    echo "Invalid --group-id value: $id" >&2
    exit 1
  fi
done

BASE_URL="${BASE_URL%/}"
IMPORT_ENDPOINT="${BASE_URL}/api/v1/admin/accounts/data"

if command -v uuidgen >/dev/null 2>&1; then
  IDEMPOTENCY_KEY="import-$(uuidgen | tr '[:upper:]' '[:lower:]')"
else
  IDEMPOTENCY_KEY="import-$(date +%s)-$$"
fi

curl_args=(
  --silent
  --show-error
  --fail-with-body
  --request POST
  --url "$IMPORT_ENDPOINT"
  --header "x-api-key: $ADMIN_KEY"
  --header "Idempotency-Key: $IDEMPOTENCY_KEY"
  --form "file=@${IMPORT_FILE};type=application/json"
  --form "skip_default_group_bind=${SKIP_DEFAULT_GROUP_BIND}"
)

for id in "${GROUP_IDS[@]}"; do
  curl_args+=(--form "group_ids=${id}")
done

echo "Importing file: $IMPORT_FILE" >&2
echo "Target groups: ${GROUP_IDS[*]}" >&2
echo "Endpoint: $IMPORT_ENDPOINT" >&2

curl "${curl_args[@]}"
echo
