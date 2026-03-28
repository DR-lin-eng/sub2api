#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${script_dir}/test-backend-unit.sh"
"${script_dir}/test-backend-integration.sh"
"${script_dir}/test-frontend.sh"
"${script_dir}/smoke-compose.sh"
