package parties

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	"geoduels/pkg/contracts"
	db "geoduels/pkg/persistence/sqlc/db"
)

// MapResolver resolves the default/party map. Implemented by maps.PGStore.
type MapResolver interface {
	ResolveGameplayMapID(mode contracts.MatchMode, ruleset contracts.GameRuleset, requestedMapID string) (string, error)
}

type PGStore struct {
	pool *pgxpool.Pool
	db   *db.Queries
	maps MapResolver
}

func NewPGStore(pool *pgxpool.Pool, maps MapResolver) *PGStore {
	return &PGStore{pool: pool, db: db.New(pool), maps: maps}
}

func (s *PGStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return storekit.WithTx(ctx, s.pool, fn)
}
