#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "usage: $0 <migrate-command> [args...]"
  echo "example: MIGRATIONS_DB_URL='postgres://user:pass@127.0.0.1:5544/geoduels?sslmode=disable' $0 up"
  exit 1
fi

DEFAULT_MIGRATIONS_DB_URL="postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable"
MIGRATIONS_DB_URL="${MIGRATIONS_DB_URL:-$DEFAULT_MIGRATIONS_DB_URL}"

DB_URL_FOR_CONTAINER="$MIGRATIONS_DB_URL"
DB_URL_FOR_CONTAINER="${DB_URL_FOR_CONTAINER//@127.0.0.1/@host.docker.internal}"
DB_URL_FOR_CONTAINER="${DB_URL_FOR_CONTAINER//@localhost/@host.docker.internal}"

# Git Bash/MSYS2 converts POSIX-looking paths inside docker -v arguments, which
# breaks the migrations bind mount on Windows. Use a Windows-style source path
# (via cygpath when available) and disable MSYS path conversion for docker.
MIGRATIONS_SRC="$(pwd)/db/migrations"
if command -v cygpath >/dev/null 2>&1; then
  MIGRATIONS_SRC="$(cygpath -w "$MIGRATIONS_SRC")"
fi

export MSYS_NO_PATHCONV=1

exec docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -v "$MIGRATIONS_SRC:/migrations:ro" \
  migrate/migrate:v4.18.3 \
  -path=/migrations \
  -database "$DB_URL_FOR_CONTAINER" \
  "$@"
