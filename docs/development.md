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

### Service and store boundaries

Handlers decode requests and format responses. Services apply policies and
coordinate operations; stores load facts and persist changes. Pure policies
accept ordinary values. Simple service methods may delegate directly to a store.

Accounts, parties, social, and staff services receive their `Store` interface
through `NewService(store)`. Staff also receives its external risk detector.
Production supplies `PGStore`; tests supply an in-memory store. Dependencies are
fields on the service, without separate dependency or port bundles.

For related writes, use `store.WithinTx(ctx, func(tx Store) error { ... })`.
Use the supplied `tx` for every operation that belongs to that transaction.
The PostgreSQL store makes a transaction-bound copy; it never changes the shared
store's transaction field. Returning an error rolls back the database writes;
commit errors must propagate to the caller. Do not start a separate transaction
inside the callback. External HTTP calls happen outside database transactions;
notification jobs are inserted in the same transaction as their source changes.

Store methods may group related SQL operations, such as accepting a friend
request and inserting the friendship. Database constraints and locking remain
responsible for concurrent-write safety. Fake-based tests exercise the real
service and policies, but do not validate SQL or PostgreSQL isolation behavior.
Tests involving atomic operations use a fake that stages changes until success
and discards them on failure.

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
