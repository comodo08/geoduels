package persistence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func normalizeReportCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "cheating", "profile", "harassment", "boosting", "other":
		return strings.ToLower(strings.TrimSpace(category))
	default:
		return "cheating"
	}
}

type reporterReputation struct {
	Confirmed    int
	Dismissed    int
	Inconclusive int
	Abusive      int
}

type reporterVolume struct {
	Last24h int
	Last7d  int
}

func moderationReporterWeight(createdAt time.Time, reputation reporterReputation, volume reporterVolume) float64 {
	age := time.Since(createdAt)
	weight := 1.0
	if age < 24*time.Hour {
		weight = 0.25
	} else if age < 7*24*time.Hour {
		weight = 0.5
	}
	weight *= reporterReputationMultiplier(reputation)
	weight *= reporterVolumeMultiplier(volume)
	if weight < 0.05 {
		return 0.05
	}
	if weight > 1.5 {
		return 1.5
	}
	return weight
}

func reporterReputationMultiplier(reputation reporterReputation) float64 {
	if reputation.Abusive >= 3 {
		return 0.1
	}
	if reputation.Abusive > 0 {
		return 0.25
	}
	reviewed := reputation.Confirmed + reputation.Dismissed
	if reviewed < 5 {
		return 1
	}
	accuracy := float64(reputation.Confirmed) / float64(reviewed)
	switch {
	case reputation.Confirmed >= 10 && accuracy >= 0.85:
		return 1.25
	case accuracy >= 0.65:
		return 1
	case accuracy >= 0.4:
		return 0.5
	default:
		return 0.15
	}
}

func reporterVolumeMultiplier(volume reporterVolume) float64 {
	daily := smoothReportVolumeDecay(float64(volume.Last24h), 6, 1.4)
	weeklyAverage := float64(volume.Last7d) / 7
	weekly := smoothReportVolumeDecay(weeklyAverage, 6, 1.4)
	if daily < weekly {
		return daily
	}
	return weekly
}

func smoothReportVolumeDecay(count, softLimit, power float64) float64 {
	if count <= 0 || softLimit <= 0 || power <= 0 {
		return 1
	}
	return 1 / (1 + math.Pow(count/softLimit, power))
}

func moderationReporterVolume(ctx context.Context, tx pgx.Tx, reporterUserID string) (reporterVolume, error) {
	var volume reporterVolume
	err := tx.QueryRow(ctx, `
		select
			count(*) filter (where created_at >= now() - interval '24 hours')::int,
			count(*) filter (where created_at >= now() - interval '7 days')::int
		from moderation_reports
		where reporter_user_id = $1
	`, reporterUserID).Scan(&volume.Last24h, &volume.Last7d)
	return volume, err
}

func moderationReporterWeightSQL() string {
	return `
		least(1.5, greatest(0.05,
			case
				when u.created_at > now() - interval '24 hours' then 0.25
				when u.created_at > now() - interval '7 days' then 0.5
				else 1.0
			end *
			case
				when rep.reports_abusive >= 3 then 0.1
				when rep.reports_abusive > 0 then 0.25
				when (rep.reports_confirmed + rep.reports_dismissed) < 5 then 1.0
				when rep.reports_confirmed >= 10 and (rep.reports_confirmed::double precision / greatest(1, rep.reports_confirmed + rep.reports_dismissed)) >= 0.85 then 1.25
				when (rep.reports_confirmed::double precision / greatest(1, rep.reports_confirmed + rep.reports_dismissed)) >= 0.65 then 1.0
				when (rep.reports_confirmed::double precision / greatest(1, rep.reports_confirmed + rep.reports_dismissed)) >= 0.4 then 0.5
				else 0.15
			end
		))
	`
}

func updateReporterReputationForCase(ctx context.Context, tx pgx.Tx, caseID int64, outcome string) error {
	column := ""
	switch outcome {
	case "confirmed":
		column = "reports_confirmed"
	case "dismissed":
		column = "reports_dismissed"
	case "inconclusive":
		column = "reports_inconclusive"
	case "abusive":
		column = "reports_abusive"
	default:
		return errors.New("unknown reporter reputation outcome")
	}
	query := fmt.Sprintf(`
		insert into moderation_reporter_reputation(user_id, %s, updated_at)
		select reporter_user_id, count(*)::int, now()
		from moderation_reports
		where case_id = $1
		group by reporter_user_id
		on conflict (user_id) do update set
			%s = moderation_reporter_reputation.%s + excluded.%s,
			updated_at = now()
	`, column, column, column, column)
	if _, err := tx.Exec(ctx, query, caseID); err != nil {
		return err
	}
	reweight := fmt.Sprintf(`
		update moderation_reporter_reputation rep
		set report_weight = %s,
			updated_at = now()
		from users u
		where u.id = rep.user_id
			and rep.user_id in (
				select distinct reporter_user_id
				from moderation_reports
				where case_id = $1
			)
	`, moderationReporterWeightSQL())
	_, err := tx.Exec(ctx, reweight, caseID)
	return err
}

func moderationPriority(score float64) string {
	if score >= 6 {
		return "urgent"
	}
	if score >= 3 {
		return "high"
	}
	if score >= 1.5 {
		return "medium"
	}
	return "low"
}

func refreshModerationCaseSummary(ctx context.Context, tx pgx.Tx, caseID int64) (ModerationCaseSummary, bool, error) {
	var reporterScore float64
	var reportCount, uniqueReporterCount int
	var categoriesRaw string
	var notificationSentAt *time.Time
	var targetUserID string
	if err := tx.QueryRow(ctx, `
		with category_counts as (
			select coalesce(jsonb_object_agg(category, count), '{}'::jsonb) as categories
			from (
				select category, count(*)::int as count
				from moderation_reports
				where case_id = $1
				group by category
			) c
		),
		reporter_score_by_match as (
			select least(1.25, sum(reporter_weight)) as match_score
			from moderation_reports
			where case_id = $1
			group by match_id
		),
		report_stats as (
			select
				count(*)::int as report_count,
				count(distinct reporter_user_id)::int as unique_reporter_count,
				coalesce(max(reported_user_id), '') as target_user_id
			from moderation_reports
			where case_id = $1
		)
		select
			coalesce((select sum(match_score) from reporter_score_by_match), 0),
			report_stats.report_count,
			report_stats.unique_reporter_count,
			coalesce((select categories from category_counts), '{}'::jsonb)::text,
			report_stats.target_user_id
		from report_stats
	`, caseID).Scan(&reporterScore, &reportCount, &uniqueReporterCount, &categoriesRaw, &targetUserID); err != nil {
		return ModerationCaseSummary{}, false, err
	}
	recentReportPressure, err := moderationRecentReportPressure(ctx, tx, targetUserID)
	if err != nil {
		return ModerationCaseSummary{}, false, err
	}
	gameplayEvidence, err := moderationGameplayEvidence(ctx, tx, targetUserID)
	if err != nil {
		return ModerationCaseSummary{}, false, err
	}
	score := reporterScore + recentReportPressure + gameplayEvidence
	priority := moderationPriority(score)
	if err := tx.QueryRow(ctx, `select notification_sent_at from moderation_cases where id = $1`, caseID).Scan(&notificationSentAt); err != nil {
		return ModerationCaseSummary{}, false, err
	}
	shouldNotify := (priority == "high" || priority == "urgent") && notificationSentAt == nil
	row := tx.QueryRow(ctx, `
		update moderation_cases
		set score = $2,
			risk_score = $2,
			report_count = $3,
			unique_reporter_count = $4,
			categories = $5::jsonb,
			priority = $6,
			risk_breakdown = jsonb_build_object(
				'reportRisk', $8::double precision,
				'trendRisk', $9::double precision,
				'gameplayRisk', $10::double precision
			),
			confidence = least(1.0, greatest(0.0, ($4::double precision / 5.0))),
			queue = case
				when status in ('actioned', 'dismissed', 'duplicate') then 'archive'
				when source = 'auto_detection'
					or assigned_to is not null
					or status in ('reviewing', 'watching')
					or escalated_at is not null
					or $2 >= $11 then 'active'
				else 'intake'
			end,
			notification_sent_at = case when $7 then now() else notification_sent_at end,
			latest_activity_at = now(),
			updated_at = now()
		where id = $1
		returning
			id, target_user_id, target_display_name, status, priority, score,
			report_count, unique_reporter_count, categories::text,
			coalesce(summary, ''), coalesce(assigned_to, ''),
			latest_activity_at, created_at, notification_sent_at,
			coalesce(queue, ''), coalesce(source, ''), risk_score, risk_breakdown::text,
			confidence, claimed_at, claim_expires_at, resolved_at, coalesce(resolved_by, ''),
			coalesce(resolution_code, ''), coalesce(resolution_note, '')
	`, caseID, score, reportCount, uniqueReporterCount, categoriesRaw, priority, shouldNotify, reporterScore, recentReportPressure, gameplayEvidence, moderationActiveRiskThreshold)
	summary, err := scanModerationCaseSummary(row)
	if err != nil {
		return ModerationCaseSummary{}, false, err
	}
	summary.ReporterScore = reporterScore
	summary.RecentReportPressure = recentReportPressure
	summary.GameplayEvidence = gameplayEvidence
	return summary, shouldNotify, nil
}

func moderationRecentReportPressure(ctx context.Context, tx pgx.Tx, targetUserID string) (float64, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return 0, nil
	}
	var recentGames, reportedGames int
	if err := tx.QueryRow(ctx, `
		with recent_matches as (
			select mp.match_id
			from match_players mp
			join match_history h on h.match_id = mp.match_id
			where mp.user_id = $1
			order by h.ended_at desc, h.match_id desc
			limit 10
		)
		select
			count(*)::int,
			count(distinct r.match_id)::int
		from recent_matches rm
		left join moderation_reports r
			on r.match_id = rm.match_id
			and r.reported_user_id = $1
	`, targetUserID).Scan(&recentGames, &reportedGames); err != nil {
		return 0, err
	}
	if recentGames <= 0 || reportedGames <= 0 {
		return 0, nil
	}
	ratio := float64(reportedGames) / float64(recentGames)
	confidence := float64(recentGames) / 5.0
	if confidence > 1 {
		confidence = 1
	}
	return 3.5 * math.Pow(ratio, 1.6) * confidence, nil
}

func moderationGameplayEvidence(ctx context.Context, tx pgx.Tx, targetUserID string) (float64, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return 0, nil
	}
	var evidence10, evidence20 float64
	if err := tx.QueryRow(ctx, `
		with recent as (
			select evidence, row_number() over (order by occurred_at desc, id desc) as rn
			from ranked_guess_events
			where user_id = $1
			order by occurred_at desc, id desc
			limit 20
		)
		select
			coalesce(sum(evidence) filter (where rn <= 10), 0),
			coalesce(sum(evidence), 0)
		from recent
	`, targetUserID).Scan(&evidence10, &evidence20); err != nil {
		return 0, err
	}
	pressure := math.Max(evidence10/6.0, evidence20/10.0)
	if pressure > 6 {
		return 6, nil
	}
	if pressure < 0 {
		return 0, nil
	}
	return pressure, nil
}
