package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"geoduels/pkg/contracts"
)

func (s *pgStore) RecordMatchResult(snap contracts.MatchSnapshot) error {
	if len(snap.Players) != 2 {
		return nil
	}
	ids := make([]string, 0, 2)
	for id := range snap.Players {
		ids = append(ids, id)
	}
	p1 := snap.Players[ids[0]]
	p2 := snap.Players[ids[1]]
	winner := ""
	if p1.HP > p2.HP {
		winner = p1.UserID
	} else if p2.HP > p1.HP {
		winner = p2.UserID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	seasonID := strings.TrimSpace(snap.SeasonID)
	if seasonID == "" {
		seasonID, err = activeSeasonIDTx(ctx, tx)
		if err != nil {
			return err
		}
	}

	ensure := func(p contracts.PlayerState) error {
		if p.UserID == "" {
			return errors.New("player user id missing")
		}
		name := p.DisplayName
		if name == "" {
			name = p.UserID
		}
		if _, err := tx.Exec(ctx, `
			insert into users (id, email, display_name, avatar_url, account_type)
			values ($1, $2, $3, null, 'guest')
			on conflict (id) do nothing
		`, p.UserID, nil, name); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into ranks (user_id, mode, mmr, season_id)
			values ($1, $2, $4, $3)
			on conflict (user_id, mode, season_id) do nothing
			`, p.UserID, modeDuel, seasonID, initialMMR); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into user_stats (user_id, games_played, wins)
			values ($1, 0, 0)
			on conflict (user_id) do nothing
		`, p.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into ranked_stats (user_id, mode, season_id, games_played, wins)
			values ($1, $2, $3, 0, 0)
			on conflict (user_id, mode, season_id) do nothing
		`, p.UserID, modeDuel, seasonID); err != nil {
			return err
		}
		return nil
	}
	if err := ensure(p1); err != nil {
		return err
	}
	if err := ensure(p2); err != nil {
		return err
	}

	var (
		p1Rating, p2Rating RatingState
		p1Guest, p2Guest   bool
	)
	if err := tx.QueryRow(ctx, `
		select account_type = 'guest'
		from users
		where id = $1
	`, p1.UserID).Scan(&p1Guest); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		select account_type = 'guest'
		from users
		where id = $1
	`, p2.UserID).Scan(&p2Guest); err != nil {
		return err
	}
	matchWinner := ""
	if winner == p1.UserID {
		matchWinner = "p1"
	} else if winner == p2.UserID {
		matchWinner = "p2"
	}
	privatePartyMatch, err := s.matchBelongsToParty(ctx, tx, snap.MatchID)
	if err != nil {
		return err
	}
	ratedMatch := !snap.Unranked && !privatePartyMatch && (!p1Guest || !p2Guest)
	now := time.Now()
	p1Update := RatingUpdate{MMR: p1.MMR, RD: clampRatingRD(p1.RatingRD)}
	p2Update := RatingUpdate{MMR: p2.MMR, RD: clampRatingRD(p2.RatingRD)}
	if ratedMatch {
		if err := tx.QueryRow(ctx, `
			select mmr, rd, updated_at
			from ranks
			where user_id=$1 and mode=$2 and season_id=$3
			for update
		`, p1.UserID, modeDuel, seasonID).Scan(&p1Rating.MMR, &p1Rating.RD, &p1Rating.UpdatedAt); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			select mmr, rd, updated_at
			from ranks
			where user_id=$1 and mode=$2 and season_id=$3
			for update
		`, p2.UserID, modeDuel, seasonID).Scan(&p2Rating.MMR, &p2Rating.RD, &p2Rating.UpdatedAt); err != nil {
			return err
		}
		p1Update, p2Update = CalculateDuelRatingUpdates(p1Rating, p2Rating, matchWinner, now)
	}
	if ratedMatch && !p1Guest {
		if _, err := tx.Exec(ctx, `
			update ranks set mmr=$2, rd=$5, updated_at=$6
			where user_id=$1 and mode=$3 and season_id=$4
		`, p1.UserID, p1Update.MMR, modeDuel, seasonID, p1Update.RD, now); err != nil {
			return err
		}
	}
	if ratedMatch && !p2Guest {
		if _, err := tx.Exec(ctx, `
			update ranks set mmr=$2, rd=$5, updated_at=$6
			where user_id=$1 and mode=$3 and season_id=$4
		`, p2.UserID, p2Update.MMR, modeDuel, seasonID, p2Update.RD, now); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		update user_stats
		set games_played = games_played + 1,
			wins = wins + case when user_id = $2 then 1 else 0 end,
			updated_at = now()
		where user_id = $1
	`, p1.UserID, winner); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		update user_stats
		set games_played = games_played + 1,
			wins = wins + case when user_id = $2 then 1 else 0 end,
			updated_at = now()
		where user_id = $1
	`, p2.UserID, winner); err != nil {
		return err
	}
	if ratedMatch && !p1Guest {
		if _, err := tx.Exec(ctx, `
			update ranked_stats
			set games_played = games_played + 1,
				wins = wins + case when user_id = $2 then 1 else 0 end,
				updated_at = now()
			where user_id = $1 and mode = $3 and season_id = $4
		`, p1.UserID, winner, modeDuel, seasonID); err != nil {
			return err
		}
	}
	if ratedMatch && !p2Guest {
		if _, err := tx.Exec(ctx, `
			update ranked_stats
			set games_played = games_played + 1,
				wins = wins + case when user_id = $2 then 1 else 0 end,
				updated_at = now()
			where user_id = $1 and mode = $3 and season_id = $4
		`, p2.UserID, winner, modeDuel, seasonID); err != nil {
			return err
		}
	}
	if ratedMatch {
		if _, err := tx.Exec(ctx, `
			update match_players
				set
					rating_before = $2,
					rating_after = $3,
					final_ranked_delta = $3::integer - $2::integer
				where match_id = $1 and user_id = $4 and $5 = false
		`, snap.MatchID, p1Rating.MMR, p1Update.MMR, p1.UserID, p1Guest); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			update match_players
				set
					rating_before = $2,
					rating_after = $3,
					final_ranked_delta = $3::integer - $2::integer
				where match_id = $1 and user_id = $4 and $5 = false
		`, snap.MatchID, p2Rating.MMR, p2Update.MMR, p2.UserID, p2Guest); err != nil {
			return err
		}
	}
	if ratedMatch && !p1Guest {
		if err := awardEloBadgesTx(ctx, tx, p1.UserID, p1Update.MMR); err != nil {
			return err
		}
	}
	if ratedMatch && !p2Guest {
		if err := awardEloBadgesTx(ctx, tx, p2.UserID, p2Update.MMR); err != nil {
			return err
		}
	}
	if ratedMatch {
		fast5000 := rankedSpeedrunnerUsers(snap)
		if fast5000[p1.UserID] && !p1Guest {
			if _, err := awardBadgeTx(ctx, tx, p1.UserID, "speedrunner"); err != nil {
				return err
			}
		}
		if fast5000[p2.UserID] && !p2Guest {
			if _, err := awardBadgeTx(ctx, tx, p2.UserID, "speedrunner"); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func rankedSpeedrunnerUsers(snap contracts.MatchSnapshot) map[string]bool {
	out := map[string]bool{}
	for _, round := range snap.RoundResults {
		if round == nil {
			continue
		}
		for userID, result := range round.Players {
			if result.Score >= 5000 && result.GuessMS > 0 && result.GuessMS < 30000 {
				out[userID] = true
			}
		}
	}
	return out
}

func (s *pgStore) matchBelongsToParty(ctx context.Context, tx pgx.Tx, matchID string) (bool, error) {
	if matchID == "" {
		return false, nil
	}
	var exists bool
	if err := tx.QueryRow(ctx, `
		select exists (
			select 1
			from match_sessions
			where match_id = $1
			  and source_kind = 'party'
		) or exists (
			select 1
			from parties
			where active_match_id = $1
			   or started_match_id = $1
			   or last_match_id = $1
		)
	`, matchID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *pgStore) RecordFinalMatchSnapshot(matchID string, snapshot []byte) error {
	if matchID == "" {
		return errors.New("matchID required")
	}
	var snap contracts.MatchSnapshot
	if err := json.Unmarshal(snapshot, &snap); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	replay, err := finalReplaySnapshotJSON(snap)
	if err != nil {
		return err
	}
	if err := recordMatchHistory(ctx, tx, matchID, snap, replay); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if snap.Mode == contracts.ModeDuel && !snap.Unranked {
		_ = s.EvaluateAutoCheatBansForMatch(matchID)
	}
	return nil
}

func recordMatchHistory(ctx context.Context, tx pgx.Tx, matchID string, snap contracts.MatchSnapshot, replaySnapshot []byte) error {
	if matchID == "" {
		matchID = snap.MatchID
	}
	if matchID == "" || len(snap.Players) == 0 {
		return nil
	}
	startedAt := snapshotStartedAt(snap)
	endedAt := time.Now()
	winner := snapshotWinner(snap)
	privatePartyMatch := false
	if err := tx.QueryRow(ctx, `
		select exists (
			select 1
			from match_sessions
			where match_id = $1
			  and source_kind = 'party'
		) or exists (
			select 1
			from parties
			where active_match_id = $1
			   or started_match_id = $1
			   or last_match_id = $1
		)
	`, matchID).Scan(&privatePartyMatch); err != nil {
		return err
	}
	ranked := !snap.Unranked && !privatePartyMatch
	sourceKind := "queue"
	var sourcePartyID any
	if privatePartyMatch {
		sourceKind = "party"
		var partyID string
		if err := tx.QueryRow(ctx, `
			select coalesce(source_party_id, '')
			from match_sessions
			where match_id = $1
		`, matchID).Scan(&partyID); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		} else if strings.TrimSpace(partyID) != "" {
			sourcePartyID = partyID
		}
		if sourcePartyID == nil {
			if err := tx.QueryRow(ctx, `
				select id
				from parties
				where active_match_id = $1
				   or started_match_id = $1
				   or last_match_id = $1
				limit 1
			`, matchID).Scan(&partyID); err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
			} else {
				sourcePartyID = partyID
			}
		}
	}
	ruleset := string(contracts.NormalizeRuleset(snap.Config.Ruleset))
	mapID := snap.Config.MapID
	var mapRevisionID string
	_ = tx.QueryRow(ctx, `select map_id, map_revision_id from match_round_plans where match_id=$1 order by round_index limit 1`, matchID).Scan(&mapID, &mapRevisionID)
	compressedReplay, replayHash, err := compressReplay(replaySnapshot)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into match_history(
			match_id, mode, started_at, ended_at, winner_user_id,
			ranked, source_kind, source_party_id, ruleset, map_key, map_revision_id,
			replay_zstd, replay_codec, replay_schema_version, replay_uncompressed_bytes,
			replay_sha256, replay_expires_at, round_count
		)
		values($1, $2, $3, $4, nullif($5, ''), $6, $7, $8, nullif($9, ''),
		       nullif($10, ''), nullif($11, ''), $12, $13, $14, $15, $16,
		       $4 + make_interval(days => $17), $18)
		on conflict (match_id) do update set
			mode = excluded.mode,
			started_at = excluded.started_at,
			ended_at = excluded.ended_at,
			winner_user_id = excluded.winner_user_id,
			ranked = excluded.ranked,
			source_kind = excluded.source_kind,
			source_party_id = excluded.source_party_id,
			ruleset = excluded.ruleset,
			map_key = excluded.map_key,
			map_revision_id = excluded.map_revision_id,
			replay_json = null,
			replay_zstd = excluded.replay_zstd,
			replay_codec = excluded.replay_codec,
			replay_schema_version = excluded.replay_schema_version,
			replay_uncompressed_bytes = excluded.replay_uncompressed_bytes,
			replay_sha256 = excluded.replay_sha256,
			replay_expires_at = excluded.replay_expires_at,
			round_count = excluded.round_count
	`, matchID, string(snap.Mode), startedAt, endedAt, winner,
		ranked, sourceKind, sourcePartyID, ruleset, mapID, mapRevisionID,
		compressedReplay, replayCodecZstd, replaySchemaVersion, len(replaySnapshot),
		replayHash[:], replayRetentionDays, len(snap.RoundResults)); err != nil {
		return err
	}
	for userID, player := range snap.Players {
		displayName := player.DisplayName
		if displayName == "" {
			displayName = userID
		}
		totalScore := 0
		for _, round := range snap.RoundResults {
			if round != nil {
				totalScore += round.Players[userID].Score
			}
		}
		if _, err := tx.Exec(ctx, `
			insert into match_players(match_id, user_id, display_name, mmr, hp, rating_rd, total_score)
			values($1, $2, $3, $4, $5, $6, $7)
			on conflict (match_id, user_id) do update set
				display_name = excluded.display_name,
				mmr = excluded.mmr,
				hp = excluded.hp,
				rating_rd = excluded.rating_rd,
				total_score = excluded.total_score
		`, matchID, userID, displayName, player.MMR, player.HP, clampRatingRD(player.RatingRD), totalScore); err != nil {
			return err
		}
	}
	for _, round := range snap.RoundResults {
		if round == nil {
			continue
		}
		for userID, result := range round.Players {
			if snap.Mode == contracts.ModeDuel && !snap.Unranked && !privatePartyMatch && result.GuessMS > 0 {
				occurredAt := endedAt
				if result.GuessUnixMS > 0 {
					occurredAt = time.UnixMilli(result.GuessUnixMS)
				}
				if _, err := tx.Exec(ctx, `
					insert into ranked_guess_events(
						user_id, match_id, round_number, score, guess_ms, evidence, occurred_at
					)
					values($1, $2, $3, $4, $5, $6, $7)
					on conflict (match_id, round_number, user_id) do update set
						score = excluded.score,
						guess_ms = excluded.guess_ms,
						evidence = excluded.evidence,
						occurred_at = excluded.occurred_at
				`, userID, matchID, round.RoundNumber, result.Score, min(result.GuessMS, int64(2147483647)), guessEvidence(result.Score, result.GuessMS), occurredAt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func finalReplaySnapshotJSON(snap contracts.MatchSnapshot) ([]byte, error) {
	snap.CurrentRound = nil
	snap.RoundMSLeft = 0
	snap.PhaseEndsAt = 0
	snap.PhaseStartedAt = 0
	snap.EventSequence = 0
	snap.ServerUnixMS = 0
	snap.GraceWindowSec = 0
	for id, player := range snap.Players {
		player.Finalized = false
		player.LastGuessLat = 0
		player.LastGuessLng = 0
		player.HasGuess = false
		player.Disconnected = false
		player.DisconnectDue = 0
		snap.Players[id] = player
	}
	body, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func guessEvidence(score int, guessMS int64) float64 {
	if score < 4900 || guessMS <= 0 || guessMS > 15000 {
		return 0
	}
	seconds := float64(guessMS) / 1000.0
	scoreExcess := float64(score-4900) / 100.0
	if scoreExcess < 0 {
		scoreExcess = 0
	}
	speed := (15.0 - seconds) / 12.0
	if speed < 0 {
		speed = 0
	}
	if speed > 1.25 {
		speed = 1.25
	}
	evidence := 0.75 + scoreExcess*4.0
	evidence *= speed
	if score >= 5000 && guessMS <= 3000 {
		evidence += 5
	} else if score >= 4990 && guessMS <= 5000 {
		evidence += 3
	} else if score >= 4950 && guessMS <= 5000 {
		evidence += 1.5
	}
	return evidence
}

func snapshotStartedAt(snap contracts.MatchSnapshot) time.Time {
	if len(snap.RoundResults) > 0 {
		first := snap.RoundResults[0]
		if first != nil {
			for _, player := range first.Players {
				if player.GuessUnixMS > 0 {
					t := time.UnixMilli(player.GuessUnixMS - player.GuessMS)
					if !t.IsZero() {
						return t
					}
				}
			}
		}
	}
	if snap.PhaseStartedAt > 0 {
		return time.UnixMilli(snap.PhaseStartedAt)
	}
	return time.Now()
}

func snapshotWinner(snap contracts.MatchSnapshot) string {
	if snap.Mode == contracts.ModeSingleplayer {
		return ""
	}
	winner := ""
	winnerHP := -1
	tie := false
	for userID, player := range snap.Players {
		if player.HP > winnerHP {
			winner = userID
			winnerHP = player.HP
			tie = false
		} else if player.HP == winnerHP {
			tie = true
		}
	}
	if tie {
		return ""
	}
	return winner
}
