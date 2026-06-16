package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) GetRankedSeasonSettings() (RankedSeasonSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return RankedSeasonSettings{}, err
	}
	return RankedSeasonSettings{ActiveSeasonID: seasonID}, nil
}

func (s *pgStore) RolloverRankedSeason(nextSeasonID string) (RankedSeasonRolloverResult, error) {
	nextSeasonID = strings.TrimSpace(nextSeasonID)
	if nextSeasonID == "" {
		return RankedSeasonRolloverResult{}, errors.New("season id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	defer tx.Rollback(ctx)
	previousSeasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if previousSeasonID == nextSeasonID {
		return RankedSeasonRolloverResult{}, errors.New("season is already active")
	}
	badgeTag, err := tx.Exec(ctx, `
		with ranked as (
			select
				r.user_id,
				row_number() over (order by r.mmr desc, r.updated_at asc, r.user_id asc)::int as rank
			from ranks r
			join users u on u.id = r.user_id
			where r.mode = $1
				and r.season_id = $2
				and coalesce(u.account_type, 'registered') <> 'guest'
				and u.banned_at is null
		)
		insert into user_badges(user_id, badge_code, badge_season_id, rank)
		select
			user_id,
			$3,
			$2,
			rank
		from ranked
		where rank between 1 and 100
		on conflict (user_id, badge_code, badge_season_id) do nothing
	`, modeDuel, previousSeasonID, badgeCodeSeasonRank)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	seedTag, err := tx.Exec(ctx, `
		insert into ranks(user_id, mode, season_id, mmr, rd)
		select u.id, $1, $2, $3, $4
		from users u
		where coalesce(u.account_type, 'registered') <> 'guest'
		on conflict (user_id, mode, season_id) do nothing
	`, modeDuel, nextSeasonID, initialMMR, initialRatingRD)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranked_stats(user_id, mode, season_id, games_played, wins)
		select u.id, $1, $2, 0, 0
		from users u
		where coalesce(u.account_type, 'registered') <> 'guest'
		on conflict (user_id, mode, season_id) do nothing
	`, modeDuel, nextSeasonID); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	settings := RankedSeasonSettings{ActiveSeasonID: nextSeasonID}
	payload, err := json.Marshal(settings)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('ranked_season', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload)); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	return RankedSeasonRolloverResult{
		PreviousSeasonID: previousSeasonID,
		ActiveSeasonID:   nextSeasonID,
		BadgesAwarded:    int(badgeTag.RowsAffected()),
		PlayersSeeded:    int(seedTag.RowsAffected()),
	}, nil
}

func (s *pgStore) activeSeasonID(ctx context.Context) (string, error) {
	return activeSeasonIDTx(ctx, s.pool)
}

type seasonQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func activeSeasonIDTx(ctx context.Context, q seasonQuerier) (string, error) {
	var raw string
	err := q.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'ranked_season'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSeasonID, nil
		}
		return "", err
	}
	var settings RankedSeasonSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaultSeasonID, nil
	}
	seasonID := strings.TrimSpace(settings.ActiveSeasonID)
	if seasonID == "" {
		return defaultSeasonID, nil
	}
	return seasonID, nil
}
