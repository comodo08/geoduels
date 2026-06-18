# Development on macOS

## Prerequisites

- Docker Desktop
- Go 1.24+
- Node 20+

## Start the backend stack

```bash
cp .env.example .env
docker compose up -d postgres redis gameplay-node match-coordinator realtime-gateway api
```

This starts the local backend services and infrastructure defined in `docker-compose.yml`.

To rebuild containers before starting them:

```bash
docker compose up -d --build postgres redis gameplay-node match-coordinator realtime-gateway api
```

## Start the web app

Run the Next.js app separately:

```bash
cd apps/web
npm ci
cp .env.local.example .env.local
npm run dev
```

The browser connects directly to the local backend services. Next.js does not
proxy API, coordinator, or realtime traffic in development.

## Endpoints

- Web: `http://localhost:3000`
- API health: `http://localhost:8080/health`
- Queue health: `http://localhost:8090/health`
- Gameplay health: `http://localhost:8091/health`
- Realtime websocket base: `ws://localhost:8092/ws/{node}`

## PostgreSQL Local (macOS)

Start local PostgreSQL container:

```bash
docker compose up -d postgres
```

Run SQL migrations directly with the `golang-migrate` CLI:

```bash
MIGRATIONS_DB_URL=postgres://geoduels:geoduels@127.0.0.1:5432/geoduels?sslmode=disable \
migrate -path db/migrations -database "$MIGRATIONS_DB_URL" up
```

Set backend DB URL in `.env`:

```bash
POSTGRES_URL=postgres://geoduels:geoduels@localhost:5432/geoduels?sslmode=disable
```

Restart backend services after changing `.env`:

```bash
docker compose up -d --force-recreate gameplay-node match-coordinator realtime-gateway api
```

Stop local PostgreSQL:

```bash
docker compose stop postgres
```

## Stop stack

```bash
docker compose down
```

## Running Go tests locally

```bash
go test ./...
```

## Local k3d routing test

For a local multi-node Kubernetes test of websocket routing and `gameplay-node` scaling, use `infra/k3s/overlays/k3d` with a k3d cluster that has 3 agent nodes. The overlay expects PostgreSQL and Redis to stay on the host and be reachable from the cluster through `host.k3d.internal`.
