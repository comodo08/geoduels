package matches

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// MatchAnalyzeEnqueuer inserts an integrity-analysis job in the caller's tx.
type MatchAnalyzeEnqueuer func(ctx context.Context, tx pgx.Tx, matchID string) error

// PGStore owns PostgreSQL access for this feature.
type PGStore struct {
	pool    *pgxpool.Pool
	db      *db.Queries
	analyze MatchAnalyzeEnqueuer
}

func NewPGStore(pool *pgxpool.Pool, analyze MatchAnalyzeEnqueuer) *PGStore {
	return &PGStore{pool: pool, db: db.New(pool), analyze: analyze}
}

func (s *PGStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return storekit.WithTx(ctx, s.pool, fn)
}
