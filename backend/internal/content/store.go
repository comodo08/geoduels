package content

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/jobs"
	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore owns PostgreSQL access for this feature.
type PGStore struct {
	pool *pgxpool.Pool
	db   *db.Queries
	jobs jobs.Enqueuer
}

func NewPGStore(pool *pgxpool.Pool, enqueuer jobs.Enqueuer) *PGStore {
	return &PGStore{pool: pool, db: db.New(pool), jobs: enqueuer}
}

func (s *PGStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return storekit.WithTx(ctx, s.pool, fn)
}
