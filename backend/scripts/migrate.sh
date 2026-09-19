#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <migrate-command> [args...]"
  echo "example: MIGRATIONS_DB_URL='postgres://user:pass@127.0.0.1:5544/geoduels?sslmode=disable' $0 up"
}

if [ $# -lt 1 ]; then
  usage
  exit 1
fi

REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_MIGRATIONS_DB_URL="postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable"
MIGRATIONS_DB_URL="${MIGRATIONS_DB_URL:-$DEFAULT_MIGRATIONS_DB_URL}"
MIGRATION_PATH="$REPO_ROOT/db/migrations"
# dev.yaml pins the compose project name to geoduels; the default URL reaches
# its postgres over the compose network by service name.
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-geoduels}"
COMPOSE_NETWORK="${COMPOSE_NETWORK:-${COMPOSE_PROJECT_NAME}_default}"

docker_run_args=(--rm)
if [ "$MIGRATIONS_DB_URL" = "$DEFAULT_MIGRATIONS_DB_URL" ]; then
  # Reach the Compose postgres service by name so a host Postgres bound to
  # 127.0.0.1:5432 cannot intercept the default local URL.
  DB_URL_FOR_CONTAINER="postgres://geoduels:geoduels@postgres:5432/geoduels?sslmode=disable"
  docker_run_args+=(--network "$COMPOSE_NETWORK")
else
  DB_URL_FOR_CONTAINER="$MIGRATIONS_DB_URL"
  DB_URL_FOR_CONTAINER="${DB_URL_FOR_CONTAINER//@127.0.0.1/@host.docker.internal}"
  DB_URL_FOR_CONTAINER="${DB_URL_FOR_CONTAINER//@localhost/@host.docker.internal}"
  docker_run_args+=(--add-host=host.docker.internal:host-gateway)
fi

# Version 2000 is the v2 baseline: a blank database (no schema_migrations row)
# applies 002000_v2_schema.up.sql and later files. Databases already on versions
# 1–1999 must finish the v2.0.1 legacy path before this tree will touch them.
# Use a disposable psql container so this helper does not require psql on the
# host and does not print or otherwise expose the database URL.
has_schema_migrations="$(docker run "${docker_run_args[@]}" postgres:16-alpine \
  psql "$DB_URL_FOR_CONTAINER" -X -Atqc "select to_regclass('public.schema_migrations') is not null" 2>/dev/null || true)"

if [ "$has_schema_migrations" = "t" ]; then
  current_version="$(docker run "${docker_run_args[@]}" postgres:16-alpine \
    psql "$DB_URL_FOR_CONTAINER" -X -Atqc "select coalesce(max(version), 0) from public.schema_migrations" 2>/dev/null || true)"
elif [ "$has_schema_migrations" = "f" ]; then
  current_version=0
else
  current_version=""
fi

if ! [[ "$current_version" =~ ^[0-9]+$ ]]; then
  echo "could not read public.schema_migrations from the database" >&2
  exit 1
fi

if [ "$current_version" -gt 0 ] && [ "$current_version" -lt 2000 ]; then
  echo "database must be migrated to GeoDuels v2 schema version 2000 before applying this release" >&2
  echo "check out the v2.0.1 tag and run ./scripts/migrate.sh up to complete the v2 migration first" >&2
  exit 2
fi

exec docker run "${docker_run_args[@]}" \
  -v "$MIGRATION_PATH:/migrations:ro" \
  migrate/migrate:v4.18.3 \
  -path=/migrations \
  -database "$DB_URL_FOR_CONTAINER" \
  "$@"
