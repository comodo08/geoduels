package staff

import (
	"context"
	"errors"
	"strings"
	"time"

	"geoduels/pkg/contracts"
	pkgstaff "geoduels/pkg/staff"
)

func containsRole(roles []string, wanted string) bool {
	for _, role := range roles {
		if role == wanted {
			return true
		}
	}
	return false
}

// ---- Access ----

// BadgeGrant is the result of granting a badge by nickname.
type BadgeGrant struct {
	Badge   contracts.PlayerBadge `json:"badge"`
	Changed bool                  `json:"changed"`
}

func (s *Service) SetRole(ctx context.Context, actor Actor, userID, role, reason string, grant bool) error {
	parsed, err := pkgstaff.Parse(role)
	if err != nil {
		return err
	}
	return runVoid(ctx, s.store, actor, requireCap(pkgstaff.CapManageAccess), func(ctx context.Context, actor Actor, store Store) error {
		return mutateRole(ctx, store, actor, userID, string(parsed), reason, grant)
	})
}

// BootstrapAdmin grants admin unconditionally. The composition root's
// allowlist gates who may reach it.
func (s *Service) BootstrapAdmin(ctx context.Context, userID string) error {
	return runVoid(ctx, s.store, Actor{ID: userID}, func(Actor) error { return nil }, func(ctx context.Context, actor Actor, store Store) error {
		return mutateRole(ctx, store, actor, userID, string(pkgstaff.Admin), "Configured admin bootstrap", true)
	})
}

func mutateRole(ctx context.Context, store Store, actor Actor, userID, role, reason string, grant bool) error {
	if err := store.LockUser(ctx, userID); err != nil {
		return err
	}
	current, err := store.UserRoles(ctx, userID)
	if err != nil {
		return err
	}
	if containsRole(current, role) == grant {
		return nil
	}
	action := ActionRoleGranted
	if grant {
		if err := store.GrantRole(ctx, userID, role, actor.ID, reason); err != nil {
			return err
		}
	} else {
		action = ActionRoleRevoked
		if err := store.RevokeRole(ctx, userID, role); err != nil {
			return err
		}
	}
	if _, err := store.RecordAudit(ctx, AuditEntry{ActorID: actor.ID, SubjectID: userID, Action: action, Reason: reason, Metadata: map[string]any{"role": role}}); err != nil {
		return err
	}
	next, err := store.UserRoles(ctx, userID)
	if err != nil {
		return err
	}
	if len(next) > 0 {
		_, err = store.AwardBadge(ctx, userID, "geoduels-team")
		return err
	}
	return store.RemoveBadge(ctx, userID, "geoduels-team")
}

func (s *Service) GrantBadge(ctx context.Context, actor Actor, nickname, badgeID string) (BadgeGrant, error) {
	return run(ctx, s.store, actor, requireCap(pkgstaff.CapManageAccess), func(ctx context.Context, actor Actor, store Store) (BadgeGrant, error) {
		badge, changed, err := store.GrantBadgeByNickname(ctx, nickname, badgeID, actor.ID)
		if err != nil {
			return BadgeGrant{}, err
		}
		return BadgeGrant{Badge: badge, Changed: changed}, nil
	})
}

func (s *Service) BadgeCatalog(ctx context.Context, actor Actor) ([]contracts.AdminBadgeDefinition, error) {
	if err := s.require(actor, pkgstaff.CapManageAccess); err != nil {
		return nil, err
	}
	return s.store.BadgeCatalog(), nil
}

func (s *Service) ListRoleGrants(ctx context.Context, actor Actor) ([]contracts.UserRoleGrant, error) {
	if err := s.require(actor, pkgstaff.CapManageAccess); err != nil {
		return nil, err
	}
	return s.store.ListRoleGrants(ctx)
}

// ---- Directory / review reads ----

func (s *Service) SearchSubjects(ctx context.Context, actor Actor, query string, limit int) ([]contracts.AdminPlayerSummary, error) {
	if err := s.require(actor, pkgstaff.CapReviewReports, pkgstaff.CapManageAccess); err != nil {
		return nil, err
	}
	return s.store.SearchSubjects(ctx, actor, query, limit)
}

func (s *Service) GetSubject(ctx context.Context, actor Actor, userID string) (SubjectDetail, error) {
	if err := s.require(actor, pkgstaff.CapReviewReports, pkgstaff.CapManageAccess); err != nil {
		return SubjectDetail{}, err
	}
	return s.store.GetSubject(ctx, actor, userID)
}

func (s *Service) SubjectProfile(ctx context.Context, actor Actor, userID string) (contracts.ModerationSubjectProfile, error) {
	if err := s.require(actor, pkgstaff.CapReviewReports, pkgstaff.CapManageAccess); err != nil {
		return contracts.ModerationSubjectProfile{}, err
	}
	detail, err := s.store.GetSubject(ctx, actor, userID)
	if err != nil {
		return contracts.ModerationSubjectProfile{}, err
	}
	signals, err := s.store.ListSignals(ctx, actor, userID, 100)
	if err != nil {
		return contracts.ModerationSubjectProfile{}, err
	}
	log, err := s.store.ListAudit(ctx, actor, userID, 100)
	if err != nil {
		return contracts.ModerationSubjectProfile{}, err
	}
	return contracts.ModerationSubjectProfile{Player: detail.Player, Signals: signals, Log: log}, nil
}

func (s *Service) ListSignals(ctx context.Context, actor Actor, limit int) ([]contracts.ModerationSignalSummary, error) {
	if err := s.require(actor, pkgstaff.CapReviewReports, pkgstaff.CapManageAccess); err != nil {
		return nil, err
	}
	return s.store.ListSignals(ctx, actor, "", limit)
}

func (s *Service) ListAudit(ctx context.Context, actor Actor, limit int) ([]contracts.ModerationAuditLogEntry, error) {
	if err := s.require(actor, pkgstaff.CapReviewReports, pkgstaff.CapManageAccess); err != nil {
		return nil, err
	}
	return s.store.ListAudit(ctx, actor, "", limit)
}

// CreateReport records a player-initiated report. It is not a staff command.
func (s *Service) CreateReport(ctx context.Context, matchID, reporter, reported, category, reason string) (contracts.ModerationSignalCreated, error) {
	report := PlayerReport{MatchID: strings.TrimSpace(matchID), ReporterID: strings.TrimSpace(reporter), SubjectID: strings.TrimSpace(reported), Category: normalizeReportCategory(category), Reason: strings.TrimSpace(reason)}
	if report.MatchID == "" || report.ReporterID == "" || report.SubjectID == "" {
		return contracts.ModerationSignalCreated{}, errors.New("matchID, reporter, and reported user are required")
	}
	if report.ReporterID == report.SubjectID {
		return contracts.ModerationSignalCreated{}, errors.New("self reports are not allowed")
	}
	report.Severity, report.Score = reportSeverity(report.Category), reportScore(report.Category)
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	var out contracts.ModerationSignalCreated
	err := s.store.WithinTx(ctx, func(store Store) error {
		eligibility, err := store.ReportEligibility(ctx, report.MatchID, report.ReporterID, report.SubjectID)
		if err != nil {
			return err
		}
		if !eligibility.TargetParticipated {
			return errors.New("report target not found")
		}
		if eligibility.ReporterMuted {
			return errors.New("reporting is temporarily muted")
		}
		out, err = store.SaveReport(ctx, report)
		return err
	})
	if err != nil {
		return contracts.ModerationSignalCreated{}, err
	}
	return out, nil
}

// ---- Moderation ----

func (s *Service) BanCheater(ctx context.Context, actor Actor, userID, reason string) (CheatingBanSummary, error) {
	return run(ctx, s.store, actor, requireCap(pkgstaff.CapEnforceBans), func(ctx context.Context, actor Actor, store Store) (CheatingBanSummary, error) {
		userID, reason = strings.TrimSpace(userID), strings.TrimSpace(reason)
		if userID == "" {
			return CheatingBanSummary{}, errors.New("userID required")
		}
		if reason == "" {
			reason = "cheating"
		}
		registrationIP, err := store.ApplyCheatingBan(ctx, userID, reason, actor.ID)
		if err != nil {
			return CheatingBanSummary{}, err
		}
		refunds, err := s.refundVictims(ctx, store, userID, reason)
		if err != nil {
			return CheatingBanSummary{}, err
		}
		logID, err := store.RecordAudit(ctx, AuditEntry{ActorID: actor.ID, SubjectID: userID, Action: ActionPermanentBan, Reason: reason,
			Metadata: map[string]any{"refundsIssued": refunds.RefundsIssued, "totalRefunded": refunds.TotalRefunded}})
		if err != nil {
			return CheatingBanSummary{}, err
		}
		if err := store.NotifyCheatingBan(ctx, userID, reason, logID); err != nil {
			return CheatingBanSummary{}, err
		}
		summary := CheatingBanSummary{UserID: userID, Reason: reason, Refunds: refunds}
		if registrationIP != "" {
			related, err := store.HasRelatedCheater(ctx, userID, registrationIP)
			if err != nil {
				return CheatingBanSummary{}, err
			}
			if ShouldBanRegistrationIP(registrationIP, related) {
				if err := store.AddIPBan(ctx, registrationIP, "Automatic signup ban: repeated cheating bans from registration IP", actor.ID); err != nil {
					return CheatingBanSummary{}, err
				}
				summary.IPSignupBanned = true
			}
		}
		return summary, nil
	})
}

func (s *Service) SetBan(ctx context.Context, actor Actor, userID, reason string, banned bool) error {
	return runVoid(ctx, s.store, actor, requireAny(pkgstaff.CapReviewReports, pkgstaff.CapManageAccess), func(ctx context.Context, actor Actor, store Store) error {
		return store.SetBan(ctx, userID, reason, actor.ID, banned)
	})
}

func (s *Service) SetMute(ctx context.Context, actor Actor, userID, kind, reason string, until time.Time, muted bool) error {
	return runVoid(ctx, s.store, actor, requireCap(pkgstaff.CapReviewReports), func(ctx context.Context, actor Actor, store Store) error {
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind != "chat" && kind != "report" {
			return errors.New("unsupported mute kind")
		}
		if muted && until.IsZero() {
			until = s.now().Add(7 * 24 * time.Hour)
		}
		return store.SetMute(ctx, userID, kind, reason, actor.ID, until, muted)
	})
}

func (s *Service) ClearReporterMute(ctx context.Context, actor Actor, userID string) error {
	return runVoid(ctx, s.store, actor, requireAny(pkgstaff.CapReviewReports, pkgstaff.CapManageAccess), func(ctx context.Context, actor Actor, store Store) error {
		return store.ClearReporterMute(ctx, userID)
	})
}

func (s *Service) PreviewPardon(ctx context.Context, actor Actor, olderThan time.Duration) (CommunityPardonSummary, error) {
	if err := s.require(actor, pkgstaff.CapManageAccess); err != nil {
		return CommunityPardonSummary{}, err
	}
	return s.store.PreviewPardon(ctx, pardonCutoff(s.now(), olderThan))
}

func (s *Service) Pardon(ctx context.Context, actor Actor, olderThan time.Duration) (CommunityPardonSummary, error) {
	return run(ctx, s.store, actor, requireCap(pkgstaff.CapManageAccess), func(ctx context.Context, actor Actor, store Store) (CommunityPardonSummary, error) {
		return store.Pardon(ctx, pardonCutoff(s.now(), olderThan), actor.ID)
	})
}

func (s *Service) AddIPBan(ctx context.Context, actor Actor, ipAddress, reason string) error {
	return runVoid(ctx, s.store, actor, requireCap(pkgstaff.CapManageConfig), func(ctx context.Context, actor Actor, store Store) error {
		return store.AddIPBan(ctx, ipAddress, reason, actor.ID)
	})
}

func (s *Service) RemoveIPBan(ctx context.Context, actor Actor, ipAddress string) error {
	return runVoid(ctx, s.store, actor, requireCap(pkgstaff.CapManageConfig), func(ctx context.Context, _ Actor, store Store) error {
		return store.RemoveIPBan(ctx, ipAddress)
	})
}

func (s *Service) ListIPBans(ctx context.Context, actor Actor, limit int) ([]SignupIPBan, error) {
	if err := s.require(actor, pkgstaff.CapManageConfig); err != nil {
		return nil, err
	}
	return s.store.ListIPBans(ctx, limit)
}

// IsSignupIPBanned gates guest/oauth signup on blocked IPs.
func (s *Service) IsSignupIPBanned(ctx context.Context, ipAddress string) (bool, error) {
	return s.store.IsSignupIPBanned(ctx, ipAddress)
}

// ---- Integrity ----

// EvaluateMatch runs the integrity detector for a finished match and records
// its signals. It is dispatched by the staff job worker, not by HTTP.
func (s *Service) EvaluateMatch(ctx context.Context, matchID string) error {
	store := s.store
	if s.risk == nil || !s.risk.Enabled() {
		return nil
	}
	playerIDs, err := store.MatchPlayerIDs(ctx, matchID)
	if err != nil {
		return err
	}
	if len(playerIDs) == 0 {
		return nil
	}
	seasonID, err := store.ActiveSeasonID(ctx)
	if err != nil {
		return err
	}
	req := RiskRequest{MatchID: matchID, FactsVersion: "match-facts/2026-01", GeneratedAt: s.now(), Players: make([]RiskPlayer, 0, len(playerIDs))}
	for _, userID := range playerIDs {
		events, err := store.RecentGuessEvents(ctx, userID)
		if err != nil {
			return err
		}
		rating, rankedGames, err := store.PlayerContext(ctx, userID, seasonID)
		if err != nil {
			return err
		}
		req.Players = append(req.Players, RiskPlayer{UserID: userID, CurrentRating: rating, RankedGames: rankedGames, Events: events})
	}
	resp, err := s.risk.Analyze(ctx, req)
	if err != nil {
		return err
	}
	return s.store.WithinTx(ctx, func(store Store) error {
		for _, signal := range resp.Signals {
			if signal.SubjectUserID == "" {
				continue
			}
			if err := store.RecordSignal(ctx, matchID, normalizeRiskSignal(signal), s.now()); err != nil {
				return err
			}
		}
		return nil
	})
}
