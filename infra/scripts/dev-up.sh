#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

./infra/scripts/compose.sh up -d postgres redis
./backend/scripts/migrate.sh up
./infra/scripts/bootstrap-dev-data.sh
./infra/scripts/compose.sh up -d gameplay-node match-coordinator realtime-gateway api
