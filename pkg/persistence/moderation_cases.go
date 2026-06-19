package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *pgStore) ListModerationCases(status string, limit int) ([]ModerationCaseSummary, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	status = strings.TrimSpace(status)
	statuses := []string{"new", "triaged", "reviewing"}
	extraWhere := ""
	args := []any{statuses, limit}
	switch status {
	case "archived":
		statuses = []string{"actioned", "dismissed", "duplicate"}
		extraWhere = "and queue = 'archive'"
	case "":
		extraWhere = "and queue = 'active' and source <> 'auto_detection' and assigned_to is not null and escalated_at is null"
	case "auto-detection":
		extraWhere = "and queue = 'active' and source = 'auto_detection'"
	case "unclaimed":
		extraWhere = "and queue = 'active' and source <> 'auto_detection' and assigned_to is null and escalated_at is null"
	case "watching":
		statuses = []string{"watching"}
		extraWhere = "and queue = 'active'"
	case "escalated":
		extraWhere = "and queue = 'active' and escalated_at is not null"
	default:
		if strings.HasPrefix(status, "active:") {
			extraWhere = "and queue = 'active' and source <> 'auto_detection' and assigned_to is not null and assigned_to <> $3 and escalated_at is null"
			args = append(args, strings.TrimPrefix(status, "active:"))
		} else if strings.HasPrefix(status, "mine:") {
			extraWhere = "and queue = 'active' and assigned_to = $3"
			args = append(args, strings.TrimPrefix(status, "mine:"))
		} else {
			statuses = []string{status}
		}
	}
	args[0] = statuses
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	query := `
		select
			id, target_user_id, target_display_name, status, priority, score,
			report_count, unique_reporter_count, categories::text,
			coalesce(summary, ''), coalesce(assigned_to::text, ''),
			latest_activity_at, created_at, notification_sent_at,
			coalesce(queue, ''), coalesce(source, ''), risk_score, risk_breakdown::text,
			confidence, claimed_at, claim_expires_at, resolved_at, coalesce(resolved_by::text, ''),
			coalesce(resolution_code, ''), coalesce(resolution_note, '')
		from moderation_cases
		where status = any($1)
		` + extraWhere + `
		order by
			case priority when 'urgent' then 0 when 'high' then 1 when 'medium' then 2 else 3 end,
			latest_activity_at desc
		limit $2
	`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ModerationCaseSummary, 0, limit)
	for rows.Next() {
		item, err := scanModerationCaseSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) GetModerationCase(caseID int64) (ModerationCaseDetail, error) {
	if caseID <= 0 {
		return ModerationCaseDetail{}, errors.New("caseID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select
			id, target_user_id, target_display_name, status, priority, score,
			report_count, unique_reporter_count, categories::text,
			coalesce(summary, ''), coalesce(assigned_to::text, ''),
			latest_activity_at, created_at, notification_sent_at,
			coalesce(queue, ''), coalesce(source, ''), risk_score, risk_breakdown::text,
			confidence, claimed_at, claim_expires_at, resolved_at, coalesce(resolved_by::text, ''),
			coalesce(resolution_code, ''), coalesce(resolution_note, '')
		from moderation_cases
		where id = $1
	`, caseID)
	var detail ModerationCaseDetail
	var err error
	detail.Case, err = scanModerationCaseSummary(row)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	targetPlayer, err := s.getAdminPlayerSummary(ctx, detail.Case.TargetUserID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	detail.TargetPlayer = &targetPlayer
	reports, err := s.listModerationCaseReports(ctx, caseID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	events, err := s.listModerationCaseEvents(ctx, caseID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	actions, err := s.listModerationCaseActions(ctx, caseID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	evidence, err := s.listModerationEvidence(ctx, caseID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	matches, err := s.listModerationCaseMatches(ctx, caseID, detail.Case.TargetUserID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	timeline, err := s.listModerationCaseLog(ctx, caseID)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	detail.Reports = reports
	detail.Events = events
	detail.Actions = actions
	detail.Evidence = evidence
	detail.Matches = matches
	detail.Timeline = timeline
	return detail, nil
}

func (s *pgStore) AddModerationCaseAction(params ModerationCaseActionParams) (ModerationCaseDetail, error) {
	params.ActionType = strings.TrimSpace(params.ActionType)
	params.Status = strings.TrimSpace(params.Status)
	params.ActorUserID = strings.TrimSpace(params.ActorUserID)
	params.Reason = strings.TrimSpace(params.Reason)
	if params.CaseID <= 0 || params.ActionType == "" {
		return ModerationCaseDetail{}, errors.New("caseID and action type required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	defer tx.Rollback(ctx)
	var targetUserID, currentStatus string
	if err := tx.QueryRow(ctx, `select target_user_id, status from moderation_cases where id = $1`, params.CaseID).Scan(&targetUserID, &currentStatus); err != nil {
		return ModerationCaseDetail{}, err
	}
	applyReputation := currentStatus != "actioned" && currentStatus != "dismissed" && currentStatus != "duplicate"
	if params.ActionType == "status" || params.ActionType == "dismiss" {
		status := params.Status
		if status == "" && params.ActionType == "dismiss" {
			status = "dismissed"
		}
		if status == "" {
			return ModerationCaseDetail{}, errors.New("status required")
		}
		_, err = tx.Exec(ctx, `
			update moderation_cases
			set status = $2,
				queue = case
					when $2 in ('actioned', 'dismissed', 'duplicate') then 'archive'
					when $2 in ('reviewing', 'watching') then 'active'
					else queue
				end,
				resolved_at = case when $2 in ('actioned', 'dismissed', 'duplicate') then now() else resolved_at end,
				resolved_by = case when $2 in ('actioned', 'dismissed', 'duplicate') then nullif($3, '')::uuid else resolved_by end,
				resolution = nullif($4, ''),
				resolution_code = case when $2 in ('actioned', 'dismissed', 'duplicate') then $2 else resolution_code end,
				resolution_note = case when $2 in ('actioned', 'dismissed', 'duplicate') then nullif($4, '') else resolution_note end,
				updated_at = now(),
				latest_activity_at = now()
			where id = $1
		`, params.CaseID, status, params.ActorUserID, params.Reason)
		if err != nil {
			return ModerationCaseDetail{}, err
		}
		switch status {
		case "actioned":
			if applyReputation {
				if err := updateReporterReputationForCase(ctx, tx, params.CaseID, "confirmed"); err != nil {
					return ModerationCaseDetail{}, err
				}
			}
		case "dismissed":
			if applyReputation {
				if err := updateReporterReputationForCase(ctx, tx, params.CaseID, "dismissed"); err != nil {
					return ModerationCaseDetail{}, err
				}
			}
		}
	}
	if params.ActionType == "mark_inconclusive" {
		if applyReputation {
			if err := updateReporterReputationForCase(ctx, tx, params.CaseID, "inconclusive"); err != nil {
				return ModerationCaseDetail{}, err
			}
		}
		_, err = tx.Exec(ctx, `
			update moderation_cases
			set status = 'dismissed',
				queue = 'archive',
				resolved_at = now(),
				resolved_by = nullif($2, '')::uuid,
				resolution = nullif($3, ''),
				resolution_code = 'inconclusive',
				resolution_note = nullif($3, ''),
				updated_at = now(),
				latest_activity_at = now()
			where id = $1
		`, params.CaseID, params.ActorUserID, params.Reason)
		if err != nil {
			return ModerationCaseDetail{}, err
		}
	}
	if params.ActionType == "abusive_reports" {
		if applyReputation {
			if err := updateReporterReputationForCase(ctx, tx, params.CaseID, "abusive"); err != nil {
				return ModerationCaseDetail{}, err
			}
		}
		_, err = tx.Exec(ctx, `
			update moderation_cases
			set status = 'dismissed',
				queue = 'archive',
				resolved_at = now(),
				resolved_by = nullif($2, '')::uuid,
				resolution = nullif($3, ''),
				resolution_code = 'abusive_reports',
				resolution_note = nullif($3, ''),
				updated_at = now(),
				latest_activity_at = now()
			where id = $1
		`, params.CaseID, params.ActorUserID, params.Reason)
		if err != nil {
			return ModerationCaseDetail{}, err
		}
		if applyReputation {
			if _, err := tx.Exec(ctx, `
			update moderation_reporter_reputation rep
			set muted_until = greatest(coalesce(muted_until, now()), now() + interval '7 days'),
				updated_at = now()
			from (
				select distinct reporter_user_id
				from moderation_reports
				where case_id = $1
			) reporters
			where rep.user_id = reporters.reporter_user_id
		`, params.CaseID); err != nil {
				return ModerationCaseDetail{}, err
			}
		}
	}
	if params.ActionType == "assign" {
		_, err = tx.Exec(ctx, `
			update moderation_cases
			set assigned_to = nullif($2, '')::uuid,
				status = 'reviewing',
				queue = 'active',
				updated_at = now(),
				latest_activity_at = now()
			where id = $1
		`, params.CaseID, params.AssignedTo)
		if err != nil {
			return ModerationCaseDetail{}, err
		}
	}
	if params.ActionType == "escalate" {
		_, err = tx.Exec(ctx, `
			update moderation_cases
			set priority = 'urgent',
				queue = 'active',
				escalated_at = coalesce(escalated_at, now()),
				status = case when status = 'new' then 'triaged' else status end,
				updated_at = now(),
				latest_activity_at = now()
			where id = $1
		`, params.CaseID)
		if err != nil {
			return ModerationCaseDetail{}, err
		}
	}
	if params.ActionType == "report_mute" {
		muteUserID := strings.TrimSpace(params.MuteUserID)
		if muteUserID == "" {
			muteUserID = targetUserID
		}
		if params.MuteUntil.IsZero() {
			params.MuteUntil = time.Now().Add(7 * 24 * time.Hour)
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_reporter_reputation(user_id, muted_until, report_weight, updated_at)
			values($1, $2, 0, now())
			on conflict (user_id) do update set
				muted_until = excluded.muted_until,
				report_weight = 0,
				updated_at = now()
		`, muteUserID, params.MuteUntil); err != nil {
			return ModerationCaseDetail{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_actions(case_id, actor_user_id, target_user_id, action_type, reason)
		values($1, nullif($2, '')::uuid, $3, $4, nullif($5, ''))
	`, params.CaseID, params.ActorUserID, targetUserID, params.ActionType, params.Reason); err != nil {
		return ModerationCaseDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_events(case_id, actor_user_id, event_type, body)
		values($1, nullif($2, '')::uuid, $3, nullif($4, ''))
	`, params.CaseID, params.ActorUserID, "action_"+params.ActionType, params.Reason); err != nil {
		return ModerationCaseDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_log(case_id, actor_user_id, event_type, body, metadata)
		values($1, nullif($2, '')::uuid, $3, nullif($4, ''), jsonb_build_object('actionType', $5::text, 'status', $6::text))
	`, params.CaseID, params.ActorUserID, "action_"+params.ActionType, params.Reason, params.ActionType, params.Status); err != nil {
		return ModerationCaseDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ModerationCaseDetail{}, err
	}
	return s.GetModerationCase(params.CaseID)
}

func (s *pgStore) RecomputeModerationProjections(limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, `select pg_try_advisory_lock($1)`, moderationProjectionAdvisoryKey).Scan(&locked); err != nil {
		return 0, err
	}
	if !locked {
		return 0, nil
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `select pg_advisory_unlock($1)`, moderationProjectionAdvisoryKey)
	}()
	rows, err := s.pool.Query(ctx, `
		select id
		from moderation_cases
		where status in ('new', 'triaged', 'reviewing', 'watching')
		order by latest_activity_at desc, id asc
		limit $1
	`, limit)
	if err != nil {
		return 0, err
	}
	caseIDs := []int64{}
	for rows.Next() {
		var caseID int64
		if err := rows.Scan(&caseID); err != nil {
			rows.Close()
			return 0, err
		}
		caseIDs = append(caseIDs, caseID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	recomputed := 0
	for _, caseID := range caseIDs {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return recomputed, err
		}
		summary, notify, err := refreshModerationCaseSummary(ctx, tx, caseID)
		if err == nil && notify {
			payload := ModerationCaseNotificationPayload{
				CaseID:               summary.ID,
				TargetUserID:         summary.TargetUserID,
				TargetDisplayName:    summary.TargetDisplayName,
				Priority:             summary.Priority,
				Score:                summary.Score,
				ReporterScore:        summary.ReporterScore,
				RecentReportPressure: summary.RecentReportPressure,
				GameplayEvidence:     summary.GameplayEvidence,
				ReportCount:          summary.ReportCount,
				UniqueReporterCount:  summary.UniqueReporterCount,
				Categories:           summary.Categories,
				LatestActivityAt:     summary.LatestActivityAt,
			}
			err = enqueueNotificationOutbox(ctx, tx, "moderation_case_threshold", fmt.Sprintf("moderation_case:%d:threshold", caseID), payload)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return recomputed, err
		}
		if err := tx.Commit(ctx); err != nil {
			return recomputed, err
		}
		recomputed++
	}
	return recomputed, nil
}

func (s *pgStore) listModerationCaseReports(ctx context.Context, caseID int64) ([]ModerationReportSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select
			r.id, r.case_id, r.match_id, r.reporter_user_id,
			coalesce(nullif(reporter.display_name, ''), r.reporter_user_id::text),
			r.reported_user_id,
			coalesce(nullif(reported.display_name, ''), r.reported_user_id::text),
			r.category, coalesce(r.reason, ''), r.reporter_weight, r.created_at
		from moderation_reports r
		left join users reporter on reporter.id = r.reporter_user_id
		left join users reported on reported.id = r.reported_user_id
		where r.case_id = $1
		order by r.created_at desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ModerationReportSummary, 0)
	for rows.Next() {
		var item ModerationReportSummary
		if err := rows.Scan(&item.ID, &item.CaseID, &item.MatchID, &item.ReporterUserID, &item.ReporterName, &item.ReportedUserID, &item.ReportedName, &item.Category, &item.Reason, &item.ReporterWeight, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) listModerationCaseMatches(ctx context.Context, caseID int64, targetUserID string) ([]ModerationMatchSummary, error) {
	rows, err := s.pool.Query(ctx, `
		with case_matches as (
			select match_id from moderation_reports where case_id = $1
			union
			select match_id from moderation_evidence where case_id = $1 and match_id is not null
		)
		select
			cm.match_id,
			coalesce(h.mode, ''),
			h.started_at,
			h.ended_at,
			coalesce(h.winner_user_id::text, ''),
			coalesce(h.round_count, 0),
			coalesce(p.user_id::text, ''),
			coalesce(nullif(p.display_name, ''), p.user_id::text, ''),
			coalesce(p.total_score, 0),
			coalesce(p.hp, 0)
		from case_matches cm
		left join match_history h on h.match_id = cm.match_id
		left join match_players p on p.match_id = cm.match_id
		order by h.ended_at desc nulls last, cm.match_id desc,
			case when p.user_id = $2 then 0 else 1 end, p.user_id
	`, caseID, targetUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ModerationMatchSummary{}
	matchIndexes := map[string]int{}
	for rows.Next() {
		var item ModerationMatchSummary
		var player ModerationMatchPlayerSummary
		if err := rows.Scan(
			&item.MatchID,
			&item.Mode,
			&item.StartedAt,
			&item.EndedAt,
			&item.WinnerUserID,
			&item.RoundCount,
			&player.UserID,
			&player.DisplayName,
			&player.TotalScore,
			&player.FinalHP,
		); err != nil {
			return nil, err
		}
		index, exists := matchIndexes[item.MatchID]
		if !exists {
			item.Players = []ModerationMatchPlayerSummary{}
			out = append(out, item)
			index = len(out) - 1
			matchIndexes[item.MatchID] = index
		}
		if player.UserID != "" {
			out[index].Players = append(out[index].Players, player)
		}
	}
	return out, rows.Err()
}

func (s *pgStore) listModerationCaseEvents(ctx context.Context, caseID int64) ([]ModerationCaseEvent, error) {
	rows, err := s.pool.Query(ctx, `
		select id, case_id, coalesce(actor_user_id::text, ''), event_type, coalesce(body, ''), created_at
		from moderation_case_events
		where case_id = $1
		order by created_at desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModerationCaseEvent{}
	for rows.Next() {
		var item ModerationCaseEvent
		if err := rows.Scan(&item.ID, &item.CaseID, &item.ActorUserID, &item.EventType, &item.Body, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) listModerationCaseActions(ctx context.Context, caseID int64) ([]ModerationActionSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select id, case_id, coalesce(actor_user_id::text, ''), target_user_id, action_type, coalesce(reason, ''), created_at
		from moderation_actions
		where case_id = $1
		order by created_at desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModerationActionSummary{}
	for rows.Next() {
		var item ModerationActionSummary
		if err := rows.Scan(&item.ID, &item.CaseID, &item.ActorUserID, &item.TargetUserID, &item.ActionType, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) listModerationEvidence(ctx context.Context, caseID int64) ([]ModerationEvidenceSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select
			id, case_id, evidence_type, coalesce(match_id::text, ''), coalesce(round_id, ''),
			coalesce(subject_user_id::text, ''), coalesce(detector_version, ''), coalesce(rule_id, ''),
			score, weight, payload_json::text, occurred_at, created_at
		from moderation_evidence
		where case_id = $1
		order by score desc, created_at desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModerationEvidenceSummary{}
	for rows.Next() {
		var item ModerationEvidenceSummary
		var payload string
		var occurredAt *time.Time
		if err := rows.Scan(&item.ID, &item.CaseID, &item.EvidenceType, &item.MatchID, &item.RoundID, &item.SubjectUserID, &item.DetectorVersion, &item.RuleID, &item.Score, &item.Weight, &payload, &occurredAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Payload = json.RawMessage(payload)
		if occurredAt != nil {
			item.OccurredAt = *occurredAt
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) listModerationCaseLog(ctx context.Context, caseID int64) ([]ModerationCaseLogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		select
			id, case_id, coalesce(actor_user_id::text, ''), event_type,
			coalesce(reason_code, ''), coalesce(body, ''), metadata::text, created_at
		from moderation_case_log
		where case_id = $1
		order by created_at desc, id desc
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModerationCaseLogEntry{}
	for rows.Next() {
		var item ModerationCaseLogEntry
		var metadata string
		if err := rows.Scan(&item.ID, &item.CaseID, &item.ActorUserID, &item.EventType, &item.ReasonCode, &item.Body, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = json.RawMessage(metadata)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) ClaimModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error) {
	return s.assignModerationCase(caseID, actorUserID, actorUserID)
}

func (s *pgStore) ReleaseModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error) {
	return s.assignModerationCase(caseID, actorUserID, "")
}

func (s *pgStore) assignModerationCase(caseID int64, actorUserID, assignedTo string) (ModerationCaseDetail, error) {
	if caseID <= 0 {
		return ModerationCaseDetail{}, errors.New("caseID required")
	}
	actorUserID = strings.TrimSpace(actorUserID)
	assignedTo = strings.TrimSpace(assignedTo)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ModerationCaseDetail{}, err
	}
	defer tx.Rollback(ctx)
	eventType := "case_released"
	var claimedAt, claimExpiresAt any
	statusExpr := "status"
	if assignedTo != "" {
		eventType = "case_claimed"
		claimedAt = time.Now()
		claimExpiresAt = time.Now().Add(30 * time.Minute)
		statusExpr = "'reviewing'"
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		update moderation_cases
		set assigned_to = nullif($2, '')::uuid,
			claimed_at = $3,
			claim_expires_at = $4,
			status = %s,
			latest_activity_at = now(),
			updated_at = now()
		where id = $1
			and queue = 'active'
	`, statusExpr), caseID, assignedTo, claimedAt, claimExpiresAt); err != nil {
		return ModerationCaseDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_log(case_id, actor_user_id, event_type, metadata)
		values($1, nullif($2, '')::uuid, $3, jsonb_build_object('assignedTo', $4::text))
	`, caseID, actorUserID, eventType, assignedTo); err != nil {
		return ModerationCaseDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ModerationCaseDetail{}, err
	}
	return s.GetModerationCase(caseID)
}

type moderationCaseScanner interface {
	Scan(dest ...any) error
}

func scanModerationCaseSummary(row moderationCaseScanner) (ModerationCaseSummary, error) {
	var item ModerationCaseSummary
	var categoriesRaw string
	var riskRaw string
	var notificationSentAt *time.Time
	var claimedAt, claimExpiresAt, resolvedAt *time.Time
	if err := row.Scan(
		&item.ID,
		&item.TargetUserID,
		&item.TargetDisplayName,
		&item.Status,
		&item.Priority,
		&item.Score,
		&item.ReportCount,
		&item.UniqueReporterCount,
		&categoriesRaw,
		&item.Summary,
		&item.AssignedTo,
		&item.LatestActivityAt,
		&item.CreatedAt,
		&notificationSentAt,
		&item.Queue,
		&item.Source,
		&item.RiskScore,
		&riskRaw,
		&item.Confidence,
		&claimedAt,
		&claimExpiresAt,
		&resolvedAt,
		&item.ResolvedBy,
		&item.ResolutionCode,
		&item.ResolutionNote,
	); err != nil {
		return ModerationCaseSummary{}, err
	}
	if err := json.Unmarshal([]byte(categoriesRaw), &item.Categories); err != nil {
		item.Categories = map[string]int{}
	}
	if item.Categories == nil {
		item.Categories = map[string]int{}
	}
	if notificationSentAt != nil {
		item.NotificationSentAt = *notificationSentAt
	}
	if item.RiskScore == 0 && item.Score != 0 {
		item.RiskScore = item.Score
	}
	if err := json.Unmarshal([]byte(riskRaw), &item.RiskBreakdown); err != nil || item.RiskBreakdown == nil {
		item.RiskBreakdown = map[string]any{}
	}
	if claimedAt != nil {
		item.ClaimedAt = *claimedAt
	}
	if claimExpiresAt != nil {
		item.ClaimExpiresAt = *claimExpiresAt
	}
	if resolvedAt != nil {
		item.ResolvedAt = *resolvedAt
	}
	return item, nil
}
