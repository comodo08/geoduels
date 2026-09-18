package matches

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// CheatBanEvaluator is the moderation hook run after a match is persisted.
type CheatBanEvaluator interface {
	EvaluateAutoCheatBansForMatch(matchID string) error
}

// PGStore owns PostgreSQL access for this feature.
type PGStore struct {
	pool      *pgxpool.Pool
	db        *db.Queries
	CheatBans CheatBanEvaluator
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool, db: db.New(pool)}
}

func (s *PGStore) evaluateCheatBans(matchID string) error {
	if s == nil || s.CheatBans == nil {
		return nil
	}
	return s.CheatBans.EvaluateAutoCheatBansForMatch(matchID)
}

func (s *PGStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return storekit.WithTx(ctx, s.pool, fn)
}
