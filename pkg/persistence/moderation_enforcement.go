package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) IssueEloRefundsForCheater(userID string, lookback time.Duration) (EloRefundSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return EloRefundSummary{}, errors.New("userID required")
	}
	if lookback <= 0 {
		lookback = 24 * time.Hour
	}
	since := time.Now().Add(-lookback)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EloRefundSummary{}, err
	}
	defer tx.Rollback(ctx)
	summary, err := issueCurrentMMRRefundsForCheater(ctx, tx, userID, "cheating_verdict", since)
	if err != nil {
		return EloRefundSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EloRefundSummary{}, err
	}
	return summary, nil
}

func (s *pgStore) BanPlayerForCheating(userID, reason, actorUserID string) (CheatingBanSummary, error) {
	userID = strings.TrimSpace(userID)
	reason = strings.TrimSpace(reason)
	actorUserID = strings.TrimSpace(actorUserID)
	if userID == "" {
		return CheatingBanSummary{}, errors.New("userID required")
	}
	if reason == "" {
		reason = "cheating"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheatingBanSummary{}, err
	}
	defer tx.Rollback(ctx)

	var registrationIP string
	tag, err := tx.Exec(ctx, `
		update users
		set banned_at = coalesce(banned_at, now()),
			ban_reason = $2
		where id = $1
	`, userID, reason)
	if err != nil {
		return CheatingBanSummary{}, err
	}
	if tag.RowsAffected() == 0 {
		return CheatingBanSummary{}, errors.New("user not found")
	}
	if err := banUserOAuthIdentities(ctx, tx, userID, reason, actorUserID); err != nil {
		return CheatingBanSummary{}, err
	}
	if err := tx.QueryRow(ctx, `
		select coalesce(registration_ip_address, '')
		from users
		where id = $1
	`, userID).Scan(&registrationIP); err != nil {
		return CheatingBanSummary{}, err
	}

	summary := CheatingBanSummary{UserID: userID, Reason: reason}
	refunds, err := issueCurrentMMRRefundsForCheater(ctx, tx, userID, reason, time.Time{})
	if err != nil {
		return CheatingBanSummary{}, err
	}
	summary.Refunds = refunds

	caseRows, err := tx.Query(ctx, `
		update moderation_cases
		set status = 'actioned',
			queue = 'archive',
			resolved_at = coalesce(resolved_at, now()),
			resolved_by = nullif($2, ''),
			resolution = $3,
			resolution_code = 'ban_refund',
			resolution_note = $3,
			updated_at = now(),
			latest_activity_at = now()
		where target_user_id = $1
			and status in ('new', 'triaged', 'reviewing', 'watching')
		returning id
	`, userID, actorUserID, reason)
	if err != nil {
		return CheatingBanSummary{}, err
	}
	for caseRows.Next() {
		var caseID int64
		if err := caseRows.Scan(&caseID); err != nil {
			caseRows.Close()
			return CheatingBanSummary{}, err
		}
		summary.ArchivedCaseIDs = append(summary.ArchivedCaseIDs, caseID)
	}
	if err := caseRows.Err(); err != nil {
		caseRows.Close()
		return CheatingBanSummary{}, err
	}
	caseRows.Close()
	for _, caseID := range summary.ArchivedCaseIDs {
		if err := updateReporterReputationForCase(ctx, tx, caseID, "confirmed"); err != nil {
			return CheatingBanSummary{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_actions(case_id, actor_user_id, target_user_id, action_type, reason)
			values($1, nullif($2, ''), $3, 'ban', nullif($4, ''))
		`, caseID, actorUserID, userID, reason); err != nil {
			return CheatingBanSummary{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_case_events(case_id, actor_user_id, event_type, body)
			values($1, nullif($2, ''), 'action_ban', nullif($3, ''))
		`, caseID, actorUserID, reason); err != nil {
			return CheatingBanSummary{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_case_log(case_id, actor_user_id, event_type, reason_code, body)
			values($1, nullif($2, ''), 'action_ban', 'ban_refund', nullif($3, ''))
		`, caseID, actorUserID, reason); err != nil {
			return CheatingBanSummary{}, err
		}
	}

	var sourceCaseID any
	if len(summary.ArchivedCaseIDs) == 1 {
		sourceCaseID = summary.ArchivedCaseIDs[0]
	}
	if _, err := tx.Exec(ctx, `
		insert into enforcement_actions(target_user_id, actor_user_id, source_case_id, action_type, reason_code, reason_note, metadata)
		values($1, nullif($2, ''), $3, 'ban', 'cheating', nullif($4, ''), jsonb_build_object('refundsIssued', $5::int, 'totalRefunded', $6::int, 'caseIds', $7::jsonb))
	`, userID, actorUserID, sourceCaseID, reason, summary.Refunds.RefundsIssued, summary.Refunds.TotalRefunded, mustJSON(summary.ArchivedCaseIDs)); err != nil {
		return CheatingBanSummary{}, err
	}

	if registrationIP != "" {
		var relatedCheater bool
		if err := tx.QueryRow(ctx, `
			select exists(
				select 1
				from users
				where id <> $1
					and registration_ip_address = $2
					and banned_at >= now() - interval '7 days'
					and (
						lower(coalesce(ban_reason, '')) like '%cheat%'
						or lower(coalesce(ban_reason, '')) like 'auto_%'
					)
			)
		`, userID, registrationIP).Scan(&relatedCheater); err != nil {
			return CheatingBanSummary{}, err
		}
		if relatedCheater {
			if _, err := tx.Exec(ctx, `
				insert into ip_signup_bans(ip_address, reason, created_by, created_at, revoked_at)
				values($1, $2, nullif($3, ''), now(), null)
				on conflict (ip_address) do update set
					reason = excluded.reason,
					created_by = excluded.created_by,
					created_at = now(),
					revoked_at = null
			`, registrationIP, "Automatic signup ban: repeated cheating bans from registration IP", actorUserID); err != nil {
				return CheatingBanSummary{}, err
			}
			summary.IPSignupBanned = true
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CheatingBanSummary{}, err
	}
	return summary, nil
}

func issueCurrentMMRRefundsForCheater(ctx context.Context, tx pgx.Tx, cheaterUserID, reason string, since time.Time) (EloRefundSummary, error) {
	var sinceArg any
	if !since.IsZero() {
		sinceArg = since
	}
	seasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return EloRefundSummary{}, err
	}
	rows, err := tx.Query(ctx, `
		with candidate_matches as (
			select
				h.match_id,
				h.ended_at,
				h.winner_user_id,
				h.snapshot_json,
				opponent.user_id as opponent_user_id,
				cheater.mmr as cheater_mmr,
				coalesce(cheater.rating_rd, $3) as cheater_rd
			from match_history h
			join match_players cheater on cheater.match_id = h.match_id and cheater.user_id = $1
			join match_players opponent on opponent.match_id = h.match_id and opponent.user_id <> $1
			left join lobbies l on l.active_match_id = h.match_id
				or l.started_match_id = h.match_id
				or l.last_match_id = h.match_id
			where h.mode = $2
				and h.winner_user_id = $1
				and ($4::timestamptz is null or h.ended_at >= $4)
				and coalesce((h.snapshot_json->>'unranked')::boolean, false) = false
				and l.id is null
		)
		select
			match_id,
			opponent_user_id,
			cheater_mmr,
			cheater_rd,
			case
				when winner_user_id = $1 then nullif(snapshot_json->'ratingPreview'->(opponent_user_id::text)->>'lose', '')::int
				else 0
			end as original_delta
		from candidate_matches
		where snapshot_json->'ratingPreview' ? opponent_user_id
		order by ended_at asc, match_id asc
	`, cheaterUserID, modeDuel, initialRatingRD, sinceArg)
	if err != nil {
		return EloRefundSummary{}, err
	}
	defer rows.Close()
	type refundCandidate struct {
		matchID       string
		opponentID    string
		cheaterMMR    int
		cheaterRD     float64
		originalDelta int
	}
	candidates := []refundCandidate{}
	for rows.Next() {
		var item refundCandidate
		if err := rows.Scan(&item.matchID, &item.opponentID, &item.cheaterMMR, &item.cheaterRD, &item.originalDelta); err != nil {
			return EloRefundSummary{}, err
		}
		if item.originalDelta < 0 {
			candidates = append(candidates, item)
		}
	}
	if err := rows.Err(); err != nil {
		return EloRefundSummary{}, err
	}
	rows.Close()

	var summary EloRefundSummary
	for _, item := range candidates {
		var current RatingState
		if err := tx.QueryRow(ctx, `
			select mmr, rd, updated_at
			from ranks
			where user_id = $1 and mode = $2 and season_id = $3
			for update
			`, item.opponentID, modeDuel, seasonID).Scan(&current.MMR, &current.RD, &current.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return EloRefundSummary{}, err
		}
		now := time.Now()
		victimWin, _ := CalculateDuelRatingUpdates(current, RatingState{MMR: item.cheaterMMR, RD: item.cheaterRD, UpdatedAt: now}, "p1", now)
		refundDelta := victimWin.Delta
		if refundDelta <= 0 {
			continue
		}
		originalLoss := -item.originalDelta
		if refundDelta > originalLoss {
			refundDelta = originalLoss
		}
		before := current.MMR
		after := clampRankedMMR(before + refundDelta)
		refundDelta = after - before
		if refundDelta <= 0 {
			continue
		}
		tag, err := tx.Exec(ctx, `
			insert into elo_refunds(
				user_id, match_id, cheater_user_id, original_delta, refund_delta,
				victim_mmr_before, victim_mmr_after, computed_refund_delta, reason, created_by_reason
			)
			values($1, $2, $3, $4, $5, $6, $7, $5, 'cheating_verdict', $8)
			on conflict (user_id, match_id, cheater_user_id) do nothing
		`, item.opponentID, item.matchID, cheaterUserID, item.originalDelta, refundDelta, before, after, reason)
		if err != nil {
			return EloRefundSummary{}, err
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		var notificationID int64
		payload := map[string]any{
			"refundDelta":   refundDelta,
			"matchId":       item.matchID,
			"cheaterUserId": cheaterUserID,
			"reason":        reason,
			"mmrBefore":     before,
			"mmrAfter":      after,
		}
		if err := upsertUserNotification(ctx, tx, item.opponentID, "mmr_refund", fmt.Sprintf("mmr_refund:%s:%s:%s", item.opponentID, item.matchID, cheaterUserID), payload, &notificationID); err != nil {
			return EloRefundSummary{}, err
		}
		if _, err := tx.Exec(ctx, `
			update elo_refunds
			set notification_id = $4
			where user_id = $1 and match_id = $2 and cheater_user_id = $3
		`, item.opponentID, item.matchID, cheaterUserID, notificationID); err != nil {
			return EloRefundSummary{}, err
		}
		if _, err := tx.Exec(ctx, `
			update ranks
			set mmr = $4,
				updated_at = now()
			where user_id = $1
				and mode = $2
				and season_id = $3
			`, item.opponentID, modeDuel, seasonID, after); err != nil {
			return EloRefundSummary{}, err
		}
		summary.RefundsIssued++
		summary.TotalRefunded += refundDelta
	}
	return summary, nil
}

func (s *pgStore) ListEnforcementActions(limit int) ([]EnforcementActionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select
			e.id,
			e.target_user_id,
			coalesce(nullif(target.display_name, ''), e.target_user_id),
			coalesce(e.actor_user_id, ''),
			coalesce(nullif(actor.display_name, ''), coalesce(e.actor_user_id, '')),
			coalesce(e.source_case_id, 0),
			e.action_type,
			coalesce(e.reason_code, ''),
			coalesce(e.reason_note, ''),
			e.metadata::text,
			e.starts_at,
			e.ends_at,
			e.revoked_at,
			e.created_at
		from enforcement_actions e
		left join users target on target.id = e.target_user_id
		left join users actor on actor.id = e.actor_user_id
		order by e.created_at desc, e.id desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EnforcementActionSummary{}
	for rows.Next() {
		var item EnforcementActionSummary
		var metadata string
		var sourceCaseID int64
		var endsAt, revokedAt *time.Time
		if err := rows.Scan(
			&item.ID,
			&item.TargetUserID,
			&item.TargetName,
			&item.ActorUserID,
			&item.ActorName,
			&sourceCaseID,
			&item.ActionType,
			&item.ReasonCode,
			&item.ReasonNote,
			&metadata,
			&item.StartsAt,
			&endsAt,
			&revokedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.SourceCaseID = sourceCaseID
		item.Metadata = json.RawMessage(metadata)
		if endsAt != nil {
			item.EndsAt = *endsAt
		}
		if revokedAt != nil {
			item.RevokedAt = *revokedAt
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) EvaluateAutoCheatBansForMatch(matchID string) error {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select distinct user_id
		from ranked_guess_events
		where match_id = $1
	`, matchID)
	if err != nil {
		return err
	}
	userIDs := []string{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return err
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, userID := range userIDs {
		reason, shouldBan, err := s.autoCheatBanReason(ctx, userID)
		if err != nil {
			return err
		}
		if shouldBan {
			if err := s.createAutoDetectionCase(ctx, userID, matchID, reason); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *pgStore) createAutoDetectionCase(ctx context.Context, userID, matchID, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var displayName string
	if err := tx.QueryRow(ctx, `
		select coalesce(nullif(display_name, ''), id)
		from users
		where id = $1
	`, userID).Scan(&displayName); err != nil {
		return err
	}
	var caseID int64
	if err := tx.QueryRow(ctx, `
		insert into moderation_cases(
			target_user_id, target_display_name, status, priority, source, summary,
			risk_score, risk_breakdown, confidence
		)
		values(
			$1, $2, 'new', 'urgent', 'auto_detection', 'Automatic cheat detection needs moderator review.',
			6, jsonb_build_object('gameplayRisk', 6, 'ruleId', $3::text, 'detectorVersion', 'fast-guess-v1'), 0.9
		)
		on conflict (target_user_id) where status in ('new', 'triaged', 'reviewing', 'watching')
		do update set
			target_display_name = excluded.target_display_name,
			source = case when moderation_cases.source = 'report' then 'auto_detection' else moderation_cases.source end,
			priority = 'urgent',
			risk_score = greatest(moderation_cases.risk_score, excluded.risk_score),
			risk_breakdown = moderation_cases.risk_breakdown || excluded.risk_breakdown,
			confidence = greatest(moderation_cases.confidence, excluded.confidence),
			latest_activity_at = now(),
			updated_at = now()
		returning id
	`, userID, displayName, reason).Scan(&caseID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_evidence(
			case_id, evidence_type, match_id, subject_user_id, detector_version, rule_id,
			score, weight, payload_json, occurred_at
		)
		values(
			$1, 'fast_guess', $2, $3, 'fast-guess-v1', $4, 6, 6,
			jsonb_build_object('recommendedAction', 'ban_refund', 'reason', $4::text),
			now()
		)
	`, caseID, matchID, userID, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_log(case_id, event_type, reason_code, body, metadata)
		values($1, 'auto_detection_queued', $2, 'Automatic detection queued this case for review.', jsonb_build_object('matchId', $3::text))
	`, caseID, reason, matchID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *pgStore) autoCheatBanReason(ctx context.Context, userID string) (string, bool, error) {
	rows, err := s.pool.Query(ctx, `
		select score, guess_ms, evidence
		from ranked_guess_events
		where user_id = $1
		order by occurred_at desc, id desc
		limit 20
	`, userID)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	type ev struct {
		score    int
		guessMS  int64
		evidence float64
	}
	events := []ev{}
	for rows.Next() {
		var item ev
		if err := rows.Scan(&item.score, &item.guessMS, &item.evidence); err != nil {
			return "", false, err
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return "", false, err
	}
	if len(events) < 10 {
		return "", false, nil
	}
	var evidence10, evidence20 float64
	fast4950In10 := 0
	fast4900In20 := 0
	highEvidence20 := 0
	for i, item := range events {
		if i < 10 {
			evidence10 += item.evidence
			if item.score >= 4950 && item.guessMS <= 10000 {
				fast4950In10++
			}
		}
		evidence20 += item.evidence
		if item.score >= 4900 && item.guessMS <= 10000 {
			fast4900In20++
		}
		if item.evidence >= 8 {
			highEvidence20++
		}
	}
	switch {
	case evidence10 >= 18:
		return "auto_cheat_fast_guess_evidence_10", true, nil
	case len(events) >= 20 && evidence20 >= 28:
		return "auto_cheat_fast_guess_evidence_20", true, nil
	case highEvidence20 >= 3:
		return "auto_cheat_repeated_extreme_fast_guesses", true, nil
	case fast4950In10 >= 5:
		return "auto_cheat_fast_4950_5_of_10", true, nil
	case len(events) >= 20 && fast4900In20 >= 19:
		return "auto_cheat_fast_4900_19_of_20", true, nil
	default:
		return "", false, nil
	}
}

func (s *pgStore) AddSignupIPBan(ipAddress, reason, createdBy string) error {
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return errors.New("ip address required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		insert into ip_signup_bans(ip_address, reason, created_by, created_at, revoked_at)
		values($1, nullif($2, ''), nullif($3, ''), now(), null)
		on conflict (ip_address) do update set
			reason = excluded.reason,
			created_by = excluded.created_by,
			created_at = now(),
			revoked_at = null
	`, ipAddress, strings.TrimSpace(reason), strings.TrimSpace(createdBy))
	return err
}

func (s *pgStore) RemoveSignupIPBan(ipAddress string) error {
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return errors.New("ip address required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update ip_signup_bans
		set revoked_at = coalesce(revoked_at, now())
		where ip_address = $1
	`, ipAddress)
	return err
}

func (s *pgStore) ListSignupIPBans(limit int) ([]SignupIPBan, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select id, ip_address, coalesce(reason, ''), coalesce(created_by, ''), created_at
		from ip_signup_bans
		where revoked_at is null
		order by created_at desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SignupIPBan, 0, limit)
	for rows.Next() {
		var item SignupIPBan
		if err := rows.Scan(&item.ID, &item.IPAddress, &item.Reason, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) IsSignupIPBanned(ipAddress string) (bool, error) {
	ipAddress = strings.TrimSpace(ipAddress)
	if ipAddress == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		select exists(
			select 1 from ip_signup_bans
			where ip_address = $1 and revoked_at is null
		)
	`, ipAddress).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
