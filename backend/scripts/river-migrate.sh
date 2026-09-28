#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DB_URL="${MIGRATIONS_DB_URL:-postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable}"
export MIGRATIONS_DB_URL
cd "$REPO_ROOT"
exec go run ./cmd/river-migrate "$@"
