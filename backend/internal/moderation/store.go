package moderation

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/admin"
	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore owns PostgreSQL access for this feature.
type PGStore struct {
	pool    *pgxpool.Pool
	db      *db.Queries
	players *admin.PGStore
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool, db: db.New(pool), players: admin.NewPGStore(pool)}
}

func (s *PGStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return storekit.WithTx(ctx, s.pool, fn)
}
