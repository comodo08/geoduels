#!/usr/bin/env bash
set -euo pipefail

# Single entry point for docker compose in this repository. Compose files live
# in infra/compose but keep the repository root as the project directory so
# relative bind mounts (backend/, maps/datasets/) resolve the same way they did
# when the compose files sat at the root. The project name defaults to the
# checkout directory's basename, matching the pre-relocation behavior so
# existing named volumes (e.g. postgres-data) are not silently switched.
REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_DIR="$REPO_ROOT/infra/compose"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"

env_file_args=()
if [ -f "$COMPOSE_DIR/.env" ]; then
  env_file_args=(--env-file "$COMPOSE_DIR/.env")
fi

exec docker compose \
  --project-name "${COMPOSE_PROJECT_NAME:-$(basename "$REPO_ROOT")}" \
  --project-directory "$REPO_ROOT" \
  "${env_file_args[@]}" \
  -f "$COMPOSE_DIR/$COMPOSE_FILE" \
  "$@"
