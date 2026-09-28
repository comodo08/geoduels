package accounts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/jobs"
	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore persists accounts data using the pool, or the transaction supplied
// by WithinTx. The original store is never mutated when starting a transaction.
type PGStore struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
	jobs jobs.Enqueuer
}

func NewPGStore(pool *pgxpool.Pool, enqueuer jobs.Enqueuer) *PGStore {
	return &PGStore{pool: pool, jobs: enqueuer}
}

// q selects the transaction when present and the pool otherwise.
func (s *PGStore) q() *db.Queries {
	if s.tx != nil {
		return db.New(s.tx)
	}
	return db.New(s.pool)
}

func (s *PGStore) requireTx() (pgx.Tx, error) {
	if s.tx == nil {
		return nil, errors.New("accounts store: operation requires a transaction")
	}
	return s.tx, nil
}

// WithinTx runs fn with a copy of the store bound to one transaction.
func (s *PGStore) WithinTx(ctx context.Context, fn func(Store) error) error {
	return storekit.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		bound := *s
		bound.tx = tx
		return fn(&bound)
	})
}

var _ Store = (*PGStore)(nil)
