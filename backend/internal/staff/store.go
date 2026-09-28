package staff

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"geoduels/internal/badges"
	"geoduels/internal/content"
	"geoduels/internal/jobs"
	"geoduels/internal/maps"
	"geoduels/internal/seasons"
	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore persists staff data and delegates other feature storage operations.
type PGStore struct {
	pool    *pgxpool.Pool
	db      *db.Queries
	tx      pgx.Tx
	content content.Store
	seasons seasons.Store
	maps    *maps.PGStore
	badges  badges.Store
	redis   *redis.Client
	jobs    jobs.Enqueuer
}

// NewPGStore builds the staff store.
func NewPGStore(pool *pgxpool.Pool, contentStore content.Store, seasonStore seasons.Store, mapStore *maps.PGStore, badgeStore badges.Store, rdb *redis.Client, enqueuer jobs.Enqueuer) *PGStore {
	return &PGStore{
		pool:    pool,
		db:      db.New(pool),
		content: contentStore,
		seasons: seasonStore,
		maps:    mapStore,
		badges:  badgeStore,
		redis:   rdb,
		jobs:    enqueuer,
	}
}

// q selects the transaction when present and the pool otherwise.
func (a *PGStore) q() *db.Queries {
	if a.tx != nil {
		return db.New(a.tx)
	}
	return a.db
}

func (a *PGStore) requireTx() (pgx.Tx, error) {
	if a.tx == nil {
		return nil, errors.New("staff store: operation requires a transaction")
	}
	return a.tx, nil
}

// WithinTx runs fn against a copy bound to the current transaction.
func (a *PGStore) WithinTx(ctx context.Context, fn func(Store) error) error {
	return storekit.WithTx(ctx, a.pool, func(tx pgx.Tx) error {
		bound := *a
		bound.tx = tx
		return fn(&bound)
	})
}

var _ Store = (*PGStore)(nil)

func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
