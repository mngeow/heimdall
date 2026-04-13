#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/new-sqlite-db.sh <name-or-path> [directory]

Examples:
  scripts/new-sqlite-db.sh local-dev
  scripts/new-sqlite-db.sh local-dev tmp/databases
  scripts/new-sqlite-db.sh /tmp/heimdall-dev.db

Notes:
  - If you pass only a name, the script creates tmp/<name>.db
  - If you pass a relative path, it is resolved from the repo root
  - If the path does not end with .db, the suffix is added automatically
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage >&2
  exit 1
fi

name_or_path="$1"
default_dir="tmp"

if [[ -n "${2:-}" ]]; then
  default_dir="$2"
fi

if [[ "$name_or_path" == */* ]]; then
  db_path="$name_or_path"
else
  db_path="${default_dir%/}/$name_or_path"
fi

if [[ "$db_path" != *.db ]]; then
  db_path="${db_path}.db"
fi

if [[ "$db_path" != /* ]]; then
  db_path="$PWD/$db_path"
fi

db_dir="$(dirname "$db_path")"
mkdir -p "$db_dir"

if [[ -e "$db_path" ]]; then
  printf 'database already exists: %s\n' "$db_path" >&2
  exit 1
fi

go run ./cmd/heimdall-initdb --dsn "$db_path"
printf 'created Heimdall SQLite database: %s\n' "$db_path"
