package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"

	"geoduels/pkg/contracts"
)

func (s *pgStore) PrepareMatchPlan(ctx context.Context, found *contracts.MatchFound) error {
	if found == nil || strings.TrimSpace(found.MatchID) == "" {
		return errors.New("match required")
	}
	if len(found.PlannedRounds) > 0 && found.ResolvedMap.RevisionID != "" {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `select round_index, lat, lng, coalesce(country,''), pano_id, heading, pitch, map_id, map_revision_id from match_round_plans where match_id=$1 order by round_index`, found.MatchID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p contracts.PlannedRound
		var mapID, rev string
		if err := rows.Scan(&p.RoundIndex, &p.Location.Lat, &p.Location.Lng, &p.Location.Country, &p.Location.PanoID, &p.Location.Heading, &p.Location.Pitch, &mapID, &rev); err != nil {
			rows.Close()
			return err
		}
		found.PlannedRounds = append(found.PlannedRounds, p)
		found.ResolvedMap.MapID, found.ResolvedMap.RevisionID = mapID, rev
	}
	rows.Close()
	if len(found.PlannedRounds) > 0 {
		_ = tx.QueryRow(ctx, `select display_name from maps where map_key=$1`, found.ResolvedMap.MapID).Scan(&found.ResolvedMap.DisplayName)
		found.Config.MapID = found.ResolvedMap.MapID
		found.Config.MapName = found.ResolvedMap.DisplayName
		return tx.Commit(ctx)
	}
	cfg := contracts.NormalizeMatchConfig(found.Config)
	mapID := cfg.MapID
	if found.Mode == contracts.ModeDuel && !found.Unranked && strings.TrimSpace(found.SourcePartyID) == "" {
		resolved, err := s.ResolveGameplayMapID(found.Mode, cfg.Ruleset, "")
		if err != nil {
			return err
		}
		mapID = resolved
	}
	var owner, visibility, status, revisionID, displayName string
	var count int
	err = tx.QueryRow(ctx, `select coalesce(owner_user_id,''), visibility, status, coalesce(active_revision_id,''), display_name, location_count from maps where map_key=$1 and archived_at is null for share`, mapID).Scan(&owner, &visibility, &status, &revisionID, &displayName, &count)
	if err != nil {
		return fmt.Errorf("selected map unavailable: %w", err)
	}
	if status != "ready" || revisionID == "" {
		return errors.New("selected map is not ready")
	}
	if owner != "" && owner != found.MapAccessUserID && visibility != "unlisted" {
		return errors.New("selected map is not accessible")
	}
	requiredRounds := plannedRoundCount
	if found.Mode == contracts.ModeFreeForAll || found.Mode == contracts.ModeSingleplayer {
		requiredRounds = minMapLocations
	}
	if count < requiredRounds {
		return errors.New("selected map has too few locations")
	}
	pivot := deterministicPivot(found.MatchID, revisionID)
	selected, err := selectPlanRows(ctx, tx, revisionID, pivot, requiredRounds)
	if err != nil {
		return err
	}
	if len(selected) < requiredRounds {
		return errors.New("selected map has too few locations")
	}
	for i, row := range selected {
		if _, err := tx.Exec(ctx, `insert into match_round_plans(match_id,round_index,map_id,map_revision_id,location_id,lat,lng,country,pano_id,heading,pitch) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) on conflict(match_id,round_index) do nothing`, found.MatchID, i, mapID, revisionID, row.ID, row.Lat, row.Lng, row.Country, row.PanoID, row.Heading, row.Pitch); err != nil {
			return err
		}
		found.PlannedRounds = append(found.PlannedRounds, contracts.PlannedRound{RoundIndex: i, Location: row.LocationPoint})
	}
	found.ResolvedMap = contracts.ResolvedMap{MapID: mapID, RevisionID: revisionID, DisplayName: displayName}
	found.Config.MapID = mapID
	found.Config.MapName = displayName
	found.Config.MapKey = ""
	if err := incrementMapPlayStats(ctx, tx, mapID, found.Players); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type plannedLocation struct {
	ID int64
	contracts.LocationPoint
}

func selectPlanRows(ctx context.Context, tx pgx.Tx, revisionID string, pivot float64, limit int) ([]plannedLocation, error) {
	query := func(op string, n int) ([]plannedLocation, error) {
		rows, err := tx.Query(ctx, `select id,lat,lng,coalesce(country,''),pano_id,heading,pitch from locations where map_revision_id=$1 and rand_key `+op+` $2 order by rand_key asc limit $3`, revisionID, pivot, n)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []plannedLocation{}
		for rows.Next() {
			var p plannedLocation
			if err := rows.Scan(&p.ID, &p.Lat, &p.Lng, &p.Country, &p.PanoID, &p.Heading, &p.Pitch); err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, rows.Err()
	}
	out, err := query(">=", limit)
	if err != nil {
		return nil, err
	}
	if len(out) < limit {
		rest, err := query("<", limit-len(out))
		if err != nil {
			return nil, err
		}
		out = append(out, rest...)
	}
	return out, nil
}
func deterministicPivot(matchID, revisionID string) float64 {
	sum := sha256.Sum256([]byte(matchID + ":" + revisionID))
	return float64(binary.BigEndian.Uint64(sum[:8])) / float64(math.MaxUint64)
}

func incrementMapPlayStats(ctx context.Context, tx pgx.Tx, mapID string, players []string) error {
	if _, err := tx.Exec(ctx, `update maps set play_count=play_count+1, updated_at=now() where map_key=$1`, mapID); err != nil {
		return err
	}
	for _, userID := range players {
		if strings.TrimSpace(userID) == "" {
			continue
		}
		if err := markMapDailyUser(ctx, tx, mapID, userID, "played"); err != nil {
			return err
		}
	}
	return refreshMapTrendingScore(ctx, tx, mapID)
}
