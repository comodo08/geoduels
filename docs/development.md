# Development notes

For local setup, see [Running GeoDuels yourself](../README.md#running-geoduels-yourself).
Commands below run from the repository root unless shown otherwise.

## Backend

Generated sqlc output is ignored. Generate it before building or testing a clean
checkout and after changing SQL queries or migrations; do not edit generated files.

```sh
cd backend
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
go test ./...
go vet ./...
```

## Frontend

```sh
npm --prefix web run lint:architecture:strict
npm --prefix web test
(cd web && npx tsc --noEmit)
npm --prefix web run build
```

## Local infrastructure

Apply database migrations with `./backend/scripts/migrate.sh up`.
After changing Compose environment variables, recreate containers:

```sh
docker compose -f backend/dev.yaml up -d --force-recreate
```
