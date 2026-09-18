package leaderboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

const modeDuel = "duel"

// PGStore owns PostgreSQL access for the leaderboard feature.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) q() *db.Queries { return db.New(s.pool) }

func (s *PGStore) ListLeaderboard(ctx context.Context, mode, seasonID string, limit, offset int) ([]Entry, error) {
	if mode == "" {
		mode = modeDuel
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	if seasonID == "" {
		var err error
		seasonID, err = s.q().GetActiveSeasonID(ctx)
		if err != nil {
			return nil, err
		}
	}
	rows, err := s.q().ListLeaderboard(ctx, db.ListLeaderboardParams{Mode: db.GdMatchMode(mode), SeasonID: seasonID, Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, limit)
	for _, row := range rows {
		entries = append(entries, Entry{Rank: int(row.Rank), UserID: storekit.UUIDVal(row.UserID), DisplayName: row.DisplayName.String, AvatarURL: row.AvatarUrl, MMR: int(row.Mmr), GamesPlayed: int(row.GamesPlayed), Wins: int(row.Wins)})
	}
	return entries, nil
}

func (s *PGStore) GetLeaderboardOverview(ctx context.Context, userID, mode, seasonID string, limit int) (Overview, error) {
	if mode == "" {
		mode = modeDuel
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	entries, err := s.ListLeaderboard(ctx, mode, seasonID, limit, 0)
	if err != nil {
		return Overview{}, err
	}
	if seasonID == "" {
		seasonID, err = s.q().GetActiveSeasonID(ctx)
		if err != nil {
			return Overview{}, err
		}
	}
	totals, err := s.q().GetLeaderboardTotals(ctx, db.GetLeaderboardTotalsParams{SelfUserID: userID, Mode: db.GdMatchMode(mode), SeasonID: seasonID})
	if err != nil {
		return Overview{}, err
	}
	return Overview{Mode: mode, SeasonID: seasonID, SelfRank: int(totals.SelfRank), TotalPlayers: int(totals.TotalPlayers), Entries: entries}, nil
}
