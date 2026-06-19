package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) CreateModerationReport(params CreateModerationReportParams) (ModerationReportCreated, error) {
	params.MatchID = strings.TrimSpace(params.MatchID)
	params.ReporterUserID = strings.TrimSpace(params.ReporterUserID)
	params.ReportedUserID = strings.TrimSpace(params.ReportedUserID)
	params.Category = normalizeReportCategory(params.Category)
	params.Reason = strings.TrimSpace(params.Reason)
	if len(params.Reason) > 1000 {
		params.Reason = params.Reason[:1000]
	}
	if params.MatchID == "" || params.ReporterUserID == "" || params.ReportedUserID == "" {
		return ModerationReportCreated{}, errors.New("matchID, reporter, and reported user are required")
	}
	if params.ReporterUserID == params.ReportedUserID {
		return ModerationReportCreated{}, errors.New("self reports are not allowed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ModerationReportCreated{}, err
	}
	defer tx.Rollback(ctx)

	var reporterName, reportedName string
	var reporterCreatedAt time.Time
	var reputation reporterReputation
	var mutedUntil *time.Time
	if err := tx.QueryRow(ctx, `
		select
			coalesce(nullif(reporter_user.display_name, ''), $2),
			coalesce(nullif(reported_user.display_name, ''), $3),
			reporter_user.created_at,
			coalesce(rep.reports_confirmed, 0),
			coalesce(rep.reports_dismissed, 0),
			coalesce(rep.reports_inconclusive, 0),
			coalesce(rep.reports_abusive, 0),
			rep.muted_until
		from match_players reporter
		join users reporter_user on reporter_user.id = reporter.user_id
		join match_players reported on reported.match_id = reporter.match_id
		join users reported_user on reported_user.id = reported.user_id
		left join moderation_reporter_reputation rep on rep.user_id = reporter.user_id
		where reporter.match_id = $1
		  and reporter.user_id = $2
		  and reported.user_id = $3
		  and reporter_user.account_type <> 'guest'
	`, params.MatchID, params.ReporterUserID, params.ReportedUserID).Scan(&reporterName, &reportedName, &reporterCreatedAt, &reputation.Confirmed, &reputation.Dismissed, &reputation.Inconclusive, &reputation.Abusive, &mutedUntil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationReportCreated{}, errors.New("report target not found")
		}
		return ModerationReportCreated{}, err
	}
	if _, err := tx.Exec(ctx, `
		update match_history
		set replay_expires_at = null
		where match_id = $1
		  and (replay_zstd is not null or replay_json is not null)
	`, params.MatchID); err != nil {
		return ModerationReportCreated{}, err
	}
	volume, err := moderationReporterVolume(ctx, tx, params.ReporterUserID)
	if err != nil {
		return ModerationReportCreated{}, err
	}
	reporterWeight := moderationReporterWeight(reporterCreatedAt, reputation, volume)
	if mutedUntil != nil && mutedUntil.After(time.Now()) {
		reporterWeight = 0
	}
	var caseID int64
	if err := tx.QueryRow(ctx, `
		insert into moderation_cases(target_user_id, target_display_name, status, priority, queue, summary)
		values($1, $2, 'new', 'low', 'intake', 'Player reported by match opponent.')
		on conflict (target_user_id) where status in ('new', 'triaged', 'reviewing', 'watching')
		do update set
			target_display_name = excluded.target_display_name,
			latest_activity_at = now(),
			updated_at = now()
		returning id
	`, params.ReportedUserID, reportedName).Scan(&caseID); err != nil {
		return ModerationReportCreated{}, err
	}

	var reportID int64
	if err := tx.QueryRow(ctx, `
		insert into moderation_reports(case_id, match_id, reporter_user_id, reported_user_id, category, reason, reporter_weight)
		values($1, $2, $3, $4, $5, nullif($6, ''), $7)
		on conflict (match_id, reporter_user_id, reported_user_id) do nothing
		returning id
	`, caseID, params.MatchID, params.ReporterUserID, params.ReportedUserID, params.Category, params.Reason, reporterWeight).Scan(&reportID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationReportCreated{CaseID: caseID, Status: "duplicate"}, tx.Commit(ctx)
		}
		return ModerationReportCreated{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_reporter_reputation(user_id, reports_submitted, report_weight, updated_at)
		values($1, 1, $2, now())
		on conflict (user_id) do update set
			reports_submitted = moderation_reporter_reputation.reports_submitted + 1,
			report_weight = excluded.report_weight,
			updated_at = now()
	`, params.ReporterUserID, reporterWeight); err != nil {
		return ModerationReportCreated{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_events(case_id, actor_user_id, event_type, body, metadata)
		values($1, $2, 'report_created', $3, jsonb_build_object('reportId', $4::bigint, 'matchId', $5::text, 'category', $6::text, 'reporterName', $7::text))
	`, caseID, params.ReporterUserID, params.Reason, reportID, params.MatchID, params.Category, reporterName); err != nil {
		return ModerationReportCreated{}, err
	}
	if _, _, err := refreshModerationCaseSummary(ctx, tx, caseID); err != nil {
		return ModerationReportCreated{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ModerationReportCreated{}, err
	}
	return ModerationReportCreated{CaseID: caseID, Status: "created"}, nil
}

func (s *pgStore) CreateDebugModerationReports(params CreateDebugModerationReportsParams) (DebugModerationReportsResult, error) {
	params.ReportedUserID = strings.TrimSpace(params.ReportedUserID)
	params.Category = normalizeReportCategory(params.Category)
	params.Reason = strings.TrimSpace(params.Reason)
	params.CreatedBy = strings.TrimSpace(params.CreatedBy)
	if params.ReportedUserID == "" {
		return DebugModerationReportsResult{}, errors.New("reported user required")
	}
	if params.Count <= 0 {
		params.Count = 3
	}
	if params.Count > 20 {
		params.Count = 20
	}
	if params.Reason == "" {
		params.Reason = "Debug generated report"
	}
	if !strings.HasPrefix(strings.ToLower(params.Reason), "[debug]") {
		params.Reason = "[debug] " + params.Reason
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DebugModerationReportsResult{}, err
	}
	defer tx.Rollback(ctx)

	var reportedName string
	var reportedUserID string
	if err := tx.QueryRow(ctx, `
		select id, coalesce(nullif(display_name, ''), id::text)
		from users
		where coalesce(account_type, 'registered') <> 'guest'
			and (
				id = $1
				or lower(display_name) = lower($1)
				or lower(coalesce(email, '')) = lower($1)
			)
		order by case when id = $1 then 0 else 1 end, created_at asc
		limit 1
	`, params.ReportedUserID).Scan(&reportedUserID, &reportedName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DebugModerationReportsResult{}, errors.New("reported user not found")
		}
		return DebugModerationReportsResult{}, err
	}
	params.ReportedUserID = reportedUserID

	rows, err := tx.Query(ctx, `
		select
			u.id,
			coalesce(nullif(u.display_name, ''), u.id::text),
			u.created_at,
			coalesce(rep.reports_confirmed, 0),
			coalesce(rep.reports_dismissed, 0),
			coalesce(rep.reports_inconclusive, 0),
			coalesce(rep.reports_abusive, 0)
		from users u
		left join moderation_reporter_reputation rep on rep.user_id = u.id
		where u.id <> $1
			and coalesce(u.account_type, 'registered') <> 'guest'
			and coalesce(rep.muted_until, '-infinity'::timestamptz) <= now()
		order by u.created_at asc, u.id asc
		limit $2
	`, params.ReportedUserID, params.Count)
	if err != nil {
		return DebugModerationReportsResult{}, err
	}
	type reporter struct {
		id         string
		name       string
		createdAt  time.Time
		reputation reporterReputation
	}
	reporters := []reporter{}
	for rows.Next() {
		var item reporter
		if err := rows.Scan(&item.id, &item.name, &item.createdAt, &item.reputation.Confirmed, &item.reputation.Dismissed, &item.reputation.Inconclusive, &item.reputation.Abusive); err != nil {
			rows.Close()
			return DebugModerationReportsResult{}, err
		}
		reporters = append(reporters, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DebugModerationReportsResult{}, err
	}
	rows.Close()
	if len(reporters) == 0 {
		return DebugModerationReportsResult{}, errors.New("no existing registered reporter users available")
	}

	var caseID int64
	if err := tx.QueryRow(ctx, `
		insert into moderation_cases(target_user_id, target_display_name, status, priority, queue, summary)
		values($1, $2, 'new', 'low', 'intake', 'Debug generated moderation case.')
		on conflict (target_user_id) where status in ('new', 'triaged', 'reviewing', 'watching')
		do update set
			target_display_name = excluded.target_display_name,
			latest_activity_at = now(),
			updated_at = now()
		returning id
	`, params.ReportedUserID, reportedName).Scan(&caseID); err != nil {
		return DebugModerationReportsResult{}, err
	}

	createdReporterIDs := []string{}
	for i, reporter := range reporters {
		matchID := newDebugMatchID(i + 1)
		if _, err := tx.Exec(ctx, `
			insert into match_history(match_id, mode, started_at, ended_at, ranked, source_kind, round_count)
			values($1, 'duel', now(), now(), false, 'debug', 0)
			on conflict (match_id) do nothing
		`, matchID); err != nil {
			return DebugModerationReportsResult{}, err
		}
		for _, player := range []struct {
			id   string
			name string
		}{
			{params.ReportedUserID, reportedName},
			{reporter.id, reporter.name},
		} {
			if _, err := tx.Exec(ctx, `
				insert into match_players(match_id, user_id, display_name, mmr, hp)
				values($1, $2, $3, $4, 0)
				on conflict (match_id, user_id) do nothing
			`, matchID, player.id, player.name, initialMMR); err != nil {
				return DebugModerationReportsResult{}, err
			}
		}
		volume, err := moderationReporterVolume(ctx, tx, reporter.id)
		if err != nil {
			return DebugModerationReportsResult{}, err
		}
		weight := moderationReporterWeight(reporter.createdAt, reporter.reputation, volume)
		var reportID int64
		if err := tx.QueryRow(ctx, `
			insert into moderation_reports(case_id, match_id, reporter_user_id, reported_user_id, category, reason, reporter_weight)
			values($1, $2, $3, $4, $5, nullif($6, ''), $7)
			returning id
		`, caseID, matchID, reporter.id, params.ReportedUserID, params.Category, params.Reason, weight).Scan(&reportID); err != nil {
			return DebugModerationReportsResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_reporter_reputation(user_id, reports_submitted, report_weight, updated_at)
			values($1, 1, $2, now())
			on conflict (user_id) do update set
				reports_submitted = moderation_reporter_reputation.reports_submitted + 1,
				report_weight = excluded.report_weight,
				updated_at = now()
		`, reporter.id, weight); err != nil {
			return DebugModerationReportsResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into moderation_case_events(case_id, actor_user_id, event_type, body, metadata)
			values($1, nullif($2, '')::uuid, 'debug_report_created', $3, jsonb_build_object('reportId', $4::bigint, 'matchId', $5::text, 'reporterUserId', $6::text, 'category', $7::text))
		`, caseID, params.CreatedBy, params.Reason, reportID, matchID, reporter.id, params.Category); err != nil {
			return DebugModerationReportsResult{}, err
		}
		createdReporterIDs = append(createdReporterIDs, reporter.id)
	}

	if _, err := tx.Exec(ctx, `
		insert into moderation_case_events(case_id, actor_user_id, event_type, body, metadata)
		values($1, nullif($2, '')::uuid, 'debug_reports_created', $3, jsonb_build_object('count', $4::int))
	`, caseID, params.CreatedBy, params.Reason, len(createdReporterIDs)); err != nil {
		return DebugModerationReportsResult{}, err
	}
	if _, _, err := refreshModerationCaseSummary(ctx, tx, caseID); err != nil {
		return DebugModerationReportsResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DebugModerationReportsResult{}, err
	}
	return DebugModerationReportsResult{CaseID: caseID, ReportsCreated: len(createdReporterIDs), ReporterUserIDs: createdReporterIDs}, nil
}

func enqueueNotificationOutbox(ctx context.Context, tx pgx.Tx, notificationType, dedupeKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into notification_outbox(type, dedupe_key, payload_json)
		values($1, $2, $3::jsonb)
		on conflict (dedupe_key) do nothing
	`, notificationType, dedupeKey, string(body))
	return err
}
