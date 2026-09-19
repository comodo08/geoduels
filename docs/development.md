# Development notes

Commands run from the repository root unless a block changes directory. Keep this file for tool usage and non-obvious prerequisites; scripts/configuration own flag lists and defaults.

## Build and verification

sqlc output is ignored, so a clean checkout cannot build Go until generation runs. Regenerate after changing queries or migrations; do not edit generated files.

```sh
cd backend
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
go test ./...
go vet ./...
```

`go test ./...` runs the retained test suite: snapshot privacy, Go/TypeScript rating parity, gameplay match invariants, geographic scoring, and gameplay-ticket policy. The rating parity golden file under `tests/shared/` is produced by the Go suite; regenerate it with `GEODUELS_UPDATE_RATING_GOLDEN=1 go test ./internal/rating` after an intentional rating change and update the TypeScript side to match.

```sh
npm --prefix web run lint:architecture:strict
npm --prefix web test
(cd web && npx tsc --noEmit)
npm --prefix web run build
```

The architecture checker owns exact native-element allowances, geometry exceptions, and size budgets. Document new exceptions beside that contract.

## Container image builds

`docker-bake.hcl` is the image definition. The `default` group builds seven
images from the service Dockerfiles: `api`, `discord-worker`,
`match-coordinator`, `moderation-worker`, `realtime-gateway`, `gameplay-node`
(`backend/services/<name>/Dockerfile`) and `web` (`web/Dockerfile`). Tags are
`${REGISTRY}/geoduels-<name>:${TAG}` (default registry
`ghcr.io/sourcelocation`, tag `latest`). Bake variables: `REGISTRY`, `TAG`,
`PLATFORMS` (default `linux/arm64`), `APP_VERSION` (defaults to `TAG`),
`GIT_SHA` (default `dev`), and `CACHE_SCOPE` (empty locally; CI sets it for
GitHub Actions cache).

Each Go Dockerfile runs sqlc generate in the image and cross-compiles from the
builder’s native platform (`FROM --platform=$BUILDPLATFORM`). The web image
bakes only `NEXT_PUBLIC_APP_VERSION` and `NEXT_PUBLIC_GIT_SHA`. Public origin
and API/queue/realtime URLs are server runtime environment variables, serialized
into page props by `_app.getInitialProps`, not Bake args.

CI (`images` job) bakes `default` for `linux/amd64` and `linux/arm64` without
push. Version tags (`release-prod`) bake and push `backend` and `web` for
`linux/amd64` only, tagged `<release>-amd64`.

```sh
# local images (one platform; --load cannot load a multi-platform manifest)
docker buildx bake --load
docker buildx bake api --load
docker buildx bake backend --load

# push (authenticate to REGISTRY first)
TAG=beta PLATFORMS=linux/arm64 GIT_SHA="$(git rev-parse --short HEAD)" docker buildx bake --push
```


## Local infrastructure and migrations

- The gameplay node imports missing required playable maps from the bundled sample dataset on startup when `DEV_MAP_DATASET` is set (`maps.PGStore.EnsurePlayableMaps`); it never replaces existing maps and does not initialize the database schema. See [Running GeoDuels yourself](../README.md#running-geoduels-yourself).
- Compose environment changes require container recreation (`docker compose -f backend/dev.yaml up -d --force-recreate`), not just restarting the web app.
- For private detector integration, run sibling `../geoduels-risk-engine` and configure `RISK_ENGINE_URL=http://host.docker.internal:8096` plus `RISK_ENGINE_TOKEN` in Compose. Moderation can run without it.
- `./backend/scripts/migrate.sh up` uses a pinned Docker migration tool and `MIGRATIONS_DB_URL`. Blank databases apply from version 2000. It refuses existing schemas on versions 1–1999; complete those upgrades using the `v2.0.1` tag. Never bypass the guard against a production database.


For local multi-node routing checks:

```sh
k3d cluster create geoduels --servers 1 --agents 3 --port "80:80@loadbalancer"
kubectl create namespace geoduels
# Fill a local copy of k3s/overlays/k3d/secrets.env.example first.
kubectl -n geoduels create secret generic geoduels-secrets --from-env-file=/path/to/local-secrets.env
kubectl apply -k k3s/overlays/k3d
```

Before applying, migrate host PostgreSQL, make PostgreSQL/Redis reachable via `host.k3d.internal`, and import matching images with `k3d image import -c geoduels ...` or provide registry access. The overlay references `ghcr-creds`; remove its pull-secret patches in a local copy for fully local images. Include optional workers' images or remove those workloads in that copy. PgBouncer needs the direct upstream `PGBOUNCER_POSTGRES_*` values from the secret template. Validate manifest changes with `kustomize build k3s/overlays/k3d`.

## Releases

Production overlays, runtime configuration, secrets, and Flux state live in `../geoduels-prod`. A version tag builds images and opens an ops PR; merging that PR permits Flux rollout. It does **not** execute database migrations. Apply required forward migrations before dependent images and check compatibility before rolling back an image. Historical pre-v2 maintenance belongs to the `v2.0.1` instructions, not the current release.

After rollout, check readiness, browser bootstrap/refresh, queue assignment, websocket reconnect, completion, and saved history. A healthy HTTP process alone does not verify a playable match. Coordinator standby may reject queue work while alive: failover verification must confirm exactly one coordinator accepts queue work after lease expiry.

## Map tools

Generate country datasets from a working Vali template, then validate panoramas and dry-run import:

```sh
node maps/scripts/generate-vali-country-batch.mjs --template maps/config/country.json --vali-bin /path/to/vali --source-root /path/to/vali/downloaded/countries --output-root maps/datasets/generated/countries
# GOOGLE_MAPS_API_KEY must be available in the environment.
node maps/scripts/validate-vali-streetview.mjs --input maps/datasets/generated/countries/FR/france.json --output maps/datasets/generated/countries/FR/france.clean.json
node maps/scripts/import-country-maps.mjs --manifest maps/datasets/generated/countries/country-maps.manifest.json --report maps/datasets/generated/countries/import-report.json
```

The generator is sequential and checkpointed. The validator uses Street View metadata, refreshes unavailable panorama IDs from coordinates, and keeps an append-only checkpoint; it needs a key enabled for Street View Static API. **Update manifest paths to cleaned files before import**; validation does not switch them for you.

Import is dry-run by default. For an intended production import, supply `GEODUELS_ADMIN_ACCESS_TOKEN` and add `--api-base https://geoduels.io --import --confirm-production`. Resolve blocked entries or explicitly use `--skip-errors`. Deterministic official map keys make reimport a replacement of current locations, not a new map. Inspect the response report and launch a private match against imported maps.

Thumbnail sources belong under `web/assets/source-map-thumbnails`: countries use ISO alpha-2 filenames (`US.jpg`, `BR_001.jpg`), continents use slugs (`africa.jpg`), generic variants use `variant-1` through `variant-5`. Sources are ignored; generated WebPs and **both** backend/frontend catalogs are committed for independent builds.

```sh
npm --prefix web run maps:thumbnails
npm --prefix web run maps:thumbnails:check
```

Removing a source does not remove its picker entry: delete the generated WebP under `web/public/map-thumbnails` and regenerate catalogs. The check command verifies local sources; a checkout without those ignored originals cannot prove their provenance.
