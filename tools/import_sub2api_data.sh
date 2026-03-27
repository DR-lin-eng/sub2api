#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tools/import_sub2api_data.sh \
    --base-url http://127.0.0.1:8080 \
    --admin-key YOUR_ADMIN_KEY \
    --file /path/to/sub2api_import.json \
    --group-id 12 \
    [--group-name "OpenAI Default"] \
    [--group-id 13] \
    [--skip-default-group-bind true|false] \
    [--idempotency-key custom-key]

Description:
  Upload a sub2api export/import JSON file to:
    POST /api/v1/admin/accounts/data

  The script uses multipart/form-data and can bind all imported accounts
  that do not already define their own group_ids to one or more target groups.

Options:
  --base-url                 Server base URL, e.g. http://127.0.0.1:8080
  --admin-key                Admin API key used in x-api-key header
  --file                     Path to sub2api_import.json
  --group-id                 Group ID to bind imported accounts to; repeatable
  --group-ids                Comma-separated group IDs, e.g. 12,13
  --group-name               Group name to resolve automatically; repeatable
  --group-names              Comma-separated group names
  --skip-default-group-bind  Default: true
  --idempotency-key          Optional custom Idempotency-Key header
  -h, --help                 Show this help

Examples:
  tools/import_sub2api_data.sh \
    --base-url http://127.0.0.1:5231 \
    --admin-key 'admin_xxx' \
    --file /Users/lin/Downloads/sub2api_import.json \
    --group-name 'openai-default'

  tools/import_sub2api_data.sh \
    --base-url http://127.0.0.1:5231 \
    --admin-key 'admin_xxx' \
    --file /Users/lin/Downloads/sub2api_import.json \
    --group-id 7

  tools/import_sub2api_data.sh \
    --base-url http://127.0.0.1:5231 \
    --admin-key 'admin_xxx' \
    --file /Users/lin/Downloads/sub2api_import.json \
    --group-ids 7,8 \
    --group-names 'OpenAI Default,Gemini Default'
EOF
}

require_value() {
  local flag="$1"
  local value="${2-}"
  if [[ -z "$value" ]]; then
    echo "Missing value for $flag" >&2
    exit 1
  fi
}

base_url=""
admin_key=""
file_path=""
skip_default_group_bind="true"
idempotency_key=""
declare -a group_ids=()
declare -a group_names=()

append_unique_group_id() {
  local candidate="$1"
  local existing
  for existing in "${group_ids[@]}"; do
    if [[ "$existing" == "$candidate" ]]; then
      return 0
    fi
  done
  group_ids+=("$candidate")
}

require_python3() {
  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 is required to resolve --group-name values automatically" >&2
    exit 1
  fi
}

resolve_group_id_by_name() {
  local group_name="$1"
  local groups_endpoint="${base_url%/}/api/v1/admin/groups"
  local tmp_body
  local http_code
  local resolved_id

  require_python3

  tmp_body="$(mktemp)"
  http_code="$(
    curl \
      --silent \
      --show-error \
      --location \
      --output "$tmp_body" \
      --write-out "%{http_code}" \
      --request GET \
      --header "x-api-key: ${admin_key}" \
      --get \
      --data-urlencode "page=1" \
      --data-urlencode "page_size=1000" \
      --data-urlencode "search=${group_name}" \
      "$groups_endpoint"
  )"

  if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
    echo "Failed to resolve group name: ${group_name}" >&2
    cat "$tmp_body" >&2
    echo >&2
    rm -f "$tmp_body"
    exit 1
  fi

  if ! resolved_id="$(
    python3 - "$tmp_body" "$group_name" <<'PY'
import json
import sys

path = sys.argv[1]
target = sys.argv[2].strip()
target_folded = target.casefold()

with open(path, "r", encoding="utf-8") as fh:
    payload = json.load(fh)

if payload.get("code") != 0:
    print(
        f"Group lookup API returned code={payload.get('code')} message={payload.get('message')}",
        file=sys.stderr,
    )
    sys.exit(2)

data = payload.get("data") or {}
items = data.get("items") or []
exact = []
for item in items:
    name = str(item.get("name") or "").strip()
    if name.casefold() == target_folded:
        exact.append(item)

if not exact:
    print(f"No exact group name match found for: {target}", file=sys.stderr)
    suggestions = [str(item.get("name") or "").strip() for item in items if str(item.get("name") or "").strip()]
    if suggestions:
        print("Nearby results:", file=sys.stderr)
        for suggestion in suggestions[:10]:
            print(f"  - {suggestion}", file=sys.stderr)
    sys.exit(3)

ids = []
for item in exact:
    raw_id = item.get("id")
    try:
        group_id = int(raw_id)
    except Exception:
        continue
    if group_id not in ids:
        ids.append(group_id)

if len(ids) != 1:
    print(f"Group name is ambiguous: {target}", file=sys.stderr)
    for item in exact:
        print(
            f"  - id={item.get('id')} name={item.get('name')}",
            file=sys.stderr,
        )
    sys.exit(4)

print(ids[0])
PY
  )"; then
    rm -f "$tmp_body"
    exit 1
  fi

  rm -f "$tmp_body"
  echo "$resolved_id"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url)
      require_value "$1" "${2-}"
      base_url="$2"
      shift 2
      ;;
    --admin-key)
      require_value "$1" "${2-}"
      admin_key="$2"
      shift 2
      ;;
    --file)
      require_value "$1" "${2-}"
      file_path="$2"
      shift 2
      ;;
    --group-id)
      require_value "$1" "${2-}"
      group_ids+=("$2")
      shift 2
      ;;
    --group-ids)
      require_value "$1" "${2-}"
      IFS=',' read -r -a parsed_ids <<<"$2"
      for id in "${parsed_ids[@]}"; do
        id="${id//[[:space:]]/}"
        [[ -n "$id" ]] && group_ids+=("$id")
      done
      shift 2
      ;;
    --group-name)
      require_value "$1" "${2-}"
      group_names+=("$2")
      shift 2
      ;;
    --group-names)
      require_value "$1" "${2-}"
      IFS=',' read -r -a parsed_names <<<"$2"
      for name in "${parsed_names[@]}"; do
        name="${name#"${name%%[![:space:]]*}"}"
        name="${name%"${name##*[![:space:]]}"}"
        [[ -n "$name" ]] && group_names+=("$name")
      done
      shift 2
      ;;
    --skip-default-group-bind)
      require_value "$1" "${2-}"
      skip_default_group_bind="$2"
      shift 2
      ;;
    --idempotency-key)
      require_value "$1" "${2-}"
      idempotency_key="$2"
      shift 2
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

if [[ -z "$base_url" || -z "$admin_key" || -z "$file_path" ]]; then
  echo "--base-url, --admin-key and --file are required" >&2
  usage >&2
  exit 1
fi

if [[ ! -f "$file_path" ]]; then
  echo "Import file not found: $file_path" >&2
  exit 1
fi

if [[ "$skip_default_group_bind" != "true" && "$skip_default_group_bind" != "false" ]]; then
  echo "--skip-default-group-bind must be true or false" >&2
  exit 1
fi

for id in "${group_ids[@]}"; do
  if [[ ! "$id" =~ ^[1-9][0-9]*$ ]]; then
    echo "Invalid group id: $id" >&2
    exit 1
  fi
done

if [[ ${#group_ids[@]} -eq 0 && ${#group_names[@]} -eq 0 ]]; then
  echo "At least one --group-id or --group-name is required" >&2
  exit 1
fi

for name in "${group_names[@]}"; do
  resolved_group_id="$(resolve_group_id_by_name "$name")"
  echo "Resolved group name '${name}' -> ${resolved_group_id}" >&2
  append_unique_group_id "$resolved_group_id"
done

if [[ -z "$idempotency_key" ]]; then
  idempotency_key="import-$(date +%s)-$$"
fi

endpoint="${base_url%/}/api/v1/admin/accounts/data"
tmp_body="$(mktemp)"
trap 'rm -f "$tmp_body"' EXIT

declare -a curl_args=(
  --silent
  --show-error
  --location
  --output "$tmp_body"
  --write-out "%{http_code}"
  --request POST
  --header "x-api-key: ${admin_key}"
  --header "Idempotency-Key: ${idempotency_key}"
  --form "file=@${file_path};type=application/json"
  --form "skip_default_group_bind=${skip_default_group_bind}"
)

for id in "${group_ids[@]}"; do
  curl_args+=(--form "group_ids=${id}")
done

http_code="$(curl "${curl_args[@]}" "$endpoint")"

echo "HTTP ${http_code}"
cat "$tmp_body"
echo

if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
  exit 1
fi
