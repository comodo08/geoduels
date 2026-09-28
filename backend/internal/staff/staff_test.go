package staff

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"geoduels/pkg/contracts"
	pkgstaff "geoduels/pkg/staff"
)

// Unused Store methods deliberately panic via the embedded nil interface.
// Tests supply state for the persistence operations exercised by real services.
type fakeStore struct {
	Store
	calls, locks, cheats                     int
	beginErr, commitErr, notifyErr, badgeErr error
	inTx                                     bool
	roles                                    map[string][]string
	audit                                    []AuditEntry
	awarded, removed                         []string
	signals                                  []RiskSignal
	risk                                     *fakeRisk
	registrationIP                           string
	relatedCheater                           bool
	ipBans                                   []string
	candidates                               []RefundCandidate
	ratings                                  map[string]RatingState
	refunds                                  map[string]RefundAward
	notifications                            int
	cycle                                    CurationCycle
	winner, trending                         CurationWinner
	awardSources                             []string
	cycleAwards                              []time.Time
	reportEligibility                        ReportEligibility
	reports                                  []PlayerReport
}

func newFakeStore(beginErr error) *fakeStore {
	return &fakeStore{beginErr: beginErr, roles: map[string][]string{"u1": {}}, risk: &fakeRisk{}, ratings: map[string]RatingState{}, refunds: map[string]RefundAward{}}
}

func (f *fakeStore) WithinTx(ctx context.Context, fn func(Store) error) error {
	f.calls++
	if f.beginErr != nil {
		return f.beginErr
	}
	tx := *f
	tx.inTx = true
	tx.roles = maps.Clone(f.roles)
	for id, roles := range tx.roles {
		tx.roles[id] = slices.Clone(roles)
	}
	tx.audit = slices.Clone(f.audit)
	tx.awarded, tx.removed = slices.Clone(f.awarded), slices.Clone(f.removed)
	tx.signals, tx.ipBans = slices.Clone(f.signals), slices.Clone(f.ipBans)
	tx.ratings, tx.refunds = maps.Clone(f.ratings), maps.Clone(f.refunds)
	tx.awardSources, tx.cycleAwards = slices.Clone(f.awardSources), slices.Clone(f.cycleAwards)
	tx.reports = slices.Clone(f.reports)
	if err := fn(&tx); err != nil {
		return err
	}
	if f.commitErr != nil {
		return f.commitErr
	}
	tx.inTx = false
	*f = tx
	return nil
}

func (f *fakeStore) LockUser(_ context.Context, id string) error {
	f.locks++
	if _, ok := f.roles[id]; !ok {
		return ErrNotFound
	}
	return nil
}
func (f *fakeStore) UserRoles(_ context.Context, id string) ([]string, error) {
	return slices.Clone(f.roles[id]), nil
}
func (f *fakeStore) GrantRole(_ context.Context, id, role, _, _ string) error {
	f.roles[id] = append(f.roles[id], role)
	return nil
}
func (f *fakeStore) RevokeRole(_ context.Context, id, role string) error {
	f.roles[id] = slices.DeleteFunc(f.roles[id], func(r string) bool { return r == role })
	return nil
}
func (f *fakeStore) RecordAudit(_ context.Context, entry AuditEntry) (int64, error) {
	f.audit = append(f.audit, entry)
	return int64(len(f.audit)), nil
}
func (f *fakeStore) AwardBadge(_ context.Context, id, badge string) (bool, error) {
	if f.badgeErr != nil {
		return false, f.badgeErr
	}
	f.awarded = append(f.awarded, id+":"+badge)
	return true, nil
}
func (f *fakeStore) RemoveBadge(_ context.Context, id, _ string) error {
	f.removed = append(f.removed, id)
	return nil
}
func (f *fakeStore) ApplyCheatingBan(context.Context, string, string, string) (string, error) {
	f.cheats++
	return f.registrationIP, nil
}
func (f *fakeStore) HasRelatedCheater(context.Context, string, string) (bool, error) {
	return f.relatedCheater, nil
}
func (f *fakeStore) AddIPBan(_ context.Context, ip, _, _ string) error {
	f.ipBans = append(f.ipBans, ip)
	return nil
}
func (f *fakeStore) NotifyCheatingBan(context.Context, string, string, int64) error {
	if f.notifyErr != nil {
		return f.notifyErr
	}
	f.notifications++
	return nil
}
func (f *fakeStore) RefundCandidates(context.Context, string) ([]RefundCandidate, error) {
	return f.candidates, nil
}
func (f *fakeStore) LockRefundRating(_ context.Context, id, _ string) (RatingState, bool, error) {
	r, ok := f.ratings[id]
	return r, ok, nil
}
func (f *fakeStore) SaveRefund(_ context.Context, award RefundAward) (bool, error) {
	key := award.UserID + ":" + award.MatchID + ":" + award.CheaterID
	if _, exists := f.refunds[key]; exists {
		return false, nil
	}
	f.refunds[key] = award
	state := f.ratings[award.UserID]
	state.MMR = award.After
	f.ratings[award.UserID] = state
	return true, nil
}
func (f *fakeStore) ActiveSeasonID(context.Context) (string, error) { return "s1", nil }
func (f *fakeStore) MatchPlayerIDs(context.Context, string) ([]string, error) {
	return []string{"u1"}, nil
}
func (f *fakeStore) RecentGuessEvents(context.Context, string) ([]RiskGuessEvent, error) {
	return nil, nil
}
func (f *fakeStore) PlayerContext(context.Context, string, string) (int, int, error) {
	return 1000, 10, nil
}
func (f *fakeStore) RecordSignal(_ context.Context, _ string, signal RiskSignal, _ time.Time) error {
	f.signals = append(f.signals, signal)
	return nil
}
func (f *fakeStore) LockCurationCycle(context.Context) (CurationCycle, error) { return f.cycle, nil }
func (f *fakeStore) CurationWinner(context.Context, time.Time) (CurationWinner, bool, error) {
	return f.winner, f.winner.MapID != "", nil
}
func (f *fakeStore) TrendingCurationMap(context.Context) (CurationWinner, bool, error) {
	return f.trending, f.trending.MapID != "", nil
}
func (f *fakeStore) SaveCurationAward(_ context.Context, start, _ time.Time, _ CurationWinner, source string) error {
	f.cycleAwards = append(f.cycleAwards, start)
	f.awardSources = append(f.awardSources, source)
	return nil
}
func (f *fakeStore) AdvanceCurationCycle(_ context.Context, cycle CurationCycle, _ time.Time) error {
	f.cycle = cycle
	return nil
}
func (f *fakeStore) ReportEligibility(context.Context, string, string, string) (ReportEligibility, error) {
	return f.reportEligibility, nil
}
func (f *fakeStore) SaveReport(_ context.Context, report PlayerReport) (contracts.ModerationSignalCreated, error) {
	f.reports = append(f.reports, report)
	return contracts.ModerationSignalCreated{SignalID: int64(len(f.reports)), Status: "created"}, nil
}

type fakeRisk struct {
	enabled bool
	resp    RiskResponse
	calls   int
}

func (f *fakeRisk) Enabled() bool { return f.enabled }
func (f *fakeRisk) Analyze(context.Context, RiskRequest) (RiskResponse, error) {
	f.calls++
	return f.resp, nil
}

func adminActor() Actor { return Actor{ID: "admin-1", Roles: pkgstaff.Roles{pkgstaff.Admin}} }
func judgeActor() Actor { return Actor{ID: "judge-1", Roles: pkgstaff.Roles{pkgstaff.Judge}} }
func modActor() Actor   { return Actor{ID: "mod-1", Roles: pkgstaff.Roles{pkgstaff.Moderator}} }

func TestRoleCapabilitiesAreIndependent(t *testing.T) {
	if !adminActor().Can(pkgstaff.CapManageAccess) {
		t.Fatal("admin must manage access")
	}
	if adminActor().Can(pkgstaff.CapReviewReports) {
		t.Fatal("admin must not review reports without the judge role")
	}
	if judgeActor().Can(pkgstaff.CapManageAccess) {
		t.Fatal("judge must not manage access")
	}
	if !judgeActor().Can(pkgstaff.CapReviewReports) {
		t.Fatal("judge must review reports")
	}
	if !modActor().Can(pkgstaff.CapCurate) || modActor().Can(pkgstaff.CapReviewReports) {
		t.Fatal("moderator capabilities misconfigured")
	}
}

func TestUnauthorizedCommandHasNoSideEffects(t *testing.T) {
	f := newFakeStore(nil)
	err := NewService(f, f.risk).SetRole(context.Background(), judgeActor(), "u1", "admin", "escalate", true)
	if !errors.Is(err, pkgstaff.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if f.calls != 0 {
		t.Fatal("unauthorized command must not open a transaction")
	}
	if f.locks != 0 {
		t.Fatal("unauthorized command must not touch persistence")
	}
}

func TestSetRoleAuditsAndSyncsBadge(t *testing.T) {
	f := newFakeStore(nil)
	svc := NewService(f, f.risk)
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "judge", "trusted", true); err != nil {
		t.Fatalf("grant failed: %v", err)
	}
	if len(f.audit) != 1 || f.audit[0].Action != ActionRoleGranted {
		t.Fatalf("expected one role_granted audit entry, got %+v", f.audit)
	}
	if len(f.awarded) != 1 {
		t.Fatalf("expected team badge award, got %+v", f.awarded)
	}
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "judge", "again", true); err != nil {
		t.Fatalf("re-grant failed: %v", err)
	}
	if len(f.audit) != 1 {
		t.Fatalf("idempotent grant must not audit, got %+v", f.audit)
	}
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "judge", "", false); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	if len(f.removed) != 1 {
		t.Fatalf("expected team badge removal, got %+v", f.removed)
	}
}

func TestRevokeKeepsBadgeWhileRolesRemain(t *testing.T) {
	f := newFakeStore(nil)
	svc := NewService(f, f.risk)
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "judge", "", true); err != nil {
		t.Fatalf("grant judge: %v", err)
	}
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "admin", "", true); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	f.removed = nil
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "judge", "", false); err != nil {
		t.Fatalf("revoke judge: %v", err)
	}
	if len(f.removed) != 0 {
		t.Fatalf("badge must remain while other staff roles exist, got %+v", f.removed)
	}
	if err := svc.SetRole(context.Background(), adminActor(), "u1", "admin", "", false); err != nil {
		t.Fatalf("revoke admin: %v", err)
	}
	if len(f.removed) != 1 {
		t.Fatalf("removing the last role must remove the badge, got %+v", f.removed)
	}
}

func TestTransactionFailurePropagates(t *testing.T) {
	f := newFakeStore(errors.New("begin failed"))
	if _, err := NewService(f, f.risk).BanCheater(context.Background(), judgeActor(), "u1", "cheating"); err == nil {
		t.Fatal("expected transaction error")
	}
	if f.cheats != 0 {
		t.Fatal("enforcement must not run when the transaction cannot be opened")
	}
}

func TestEvaluateMatchRecordsSignalsOnlyWhenEnabled(t *testing.T) {
	f := newFakeStore(nil)
	if err := NewService(f, f.risk).EvaluateMatch(context.Background(), "m1"); err != nil {
		t.Fatalf("disabled evaluation must succeed: %v", err)
	}
	if len(f.signals) != 0 || f.calls != 0 {
		t.Fatal("disabled detector must not record signals")
	}

	f.risk.enabled = true
	f.risk.resp = RiskResponse{Signals: []RiskSignal{{SubjectUserID: "u1", Severity: "high"}}}
	if err := NewService(f, f.risk).EvaluateMatch(context.Background(), "m1"); err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	if len(f.signals) != 1 {
		t.Fatalf("expected one recorded signal, got %d", len(f.signals))
	}
}

func TestRoleChangeRollsBackOnBadgeOrCommitFailure(t *testing.T) {
	for _, stage := range []string{"badge", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newFakeStore(nil)
			failure := errors.New(stage + " failure")
			if stage == "badge" {
				f.badgeErr = failure
			} else {
				f.commitErr = failure
			}
			err := NewService(f, nil).SetRole(context.Background(), adminActor(), "u1", "judge", "trusted", true)
			if !errors.Is(err, failure) {
				t.Fatalf("expected failure, got %v", err)
			}
			if len(f.roles["u1"]) != 0 || len(f.audit) != 0 || len(f.awarded) != 0 {
				t.Fatal("failed role grant leaked changes")
			}
		})
	}
}

func TestBanCheaterRefundsLossesOnceAndBlocksRelatedIP(t *testing.T) {
	f := newFakeStore(nil)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	f.registrationIP, f.relatedCheater = "192.0.2.1", true
	f.ratings["victim"] = RatingState{MMR: 1000, RD: 60, UpdatedAt: now}
	f.candidates = []RefundCandidate{
		{MatchID: "loss", UserID: "victim", CheaterMMR: 1000, CheaterRD: 60, OriginalDelta: -5},
		{MatchID: "win", UserID: "victim", CheaterMMR: 1000, CheaterRD: 60, OriginalDelta: 10},
	}
	svc := NewService(f, nil)
	svc.clock = func() time.Time { return now }
	result, err := svc.BanCheater(context.Background(), judgeActor(), "u1", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Refunds.RefundsIssued != 1 || result.Refunds.TotalRefunded <= 0 || result.Refunds.TotalRefunded > 5 {
		t.Fatalf("invalid refunds: %+v", result)
	}
	if !result.IPSignupBanned || len(f.ipBans) != 1 || result.Reason != "cheating" || f.notifications != 1 {
		t.Fatalf("incomplete ban: %+v", result)
	}
	rating := f.ratings["victim"].MMR
	result, err = svc.BanCheater(context.Background(), judgeActor(), "u1", "again")
	if err != nil {
		t.Fatal(err)
	}
	if result.Refunds.RefundsIssued != 0 || f.ratings["victim"].MMR != rating {
		t.Fatal("repeat ban refunded the same loss twice")
	}
}

func TestBanFailureRollsBackRefundAndAudit(t *testing.T) {
	f := newFakeStore(nil)
	f.notifyErr = errors.New("notification failed")
	f.ratings["victim"] = RatingState{MMR: 1000, RD: 60, UpdatedAt: time.Now()}
	f.candidates = []RefundCandidate{{MatchID: "loss", UserID: "victim", CheaterMMR: 1000, CheaterRD: 60, OriginalDelta: -5}}
	svc := NewService(f, nil)
	result, err := svc.BanCheater(context.Background(), judgeActor(), "u1", "cheating")
	if !errors.Is(err, f.notifyErr) || result != (CheatingBanSummary{}) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if f.cheats != 0 || len(f.audit) != 0 || len(f.refunds) != 0 || f.ratings["victim"].MMR != 1000 {
		t.Fatal("failed ban leaked changes")
	}
}

func TestCurationClosesOneCycleAcrossDowntime(t *testing.T) {
	for _, source := range []string{"nomination", "trending", "none"} {
		t.Run(source, func(t *testing.T) {
			f := newFakeStore(nil)
			start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
			f.cycle = CurationCycle{StartsAt: start, ClosesAt: start.Add(CurationWindow)}
			winner := CurationWinner{MapID: "map1", CreatorID: "creator"}
			if source == "nomination" {
				f.winner = winner
				f.trending = CurationWinner{MapID: "other"}
			}
			if source == "trending" {
				f.trending = winner
			}
			now := start.Add(3*CurationWindow + time.Hour)
			svc := NewService(f, nil)
			svc.clock = func() time.Time { return now }
			if err := svc.RunCurationSweep(context.Background()); err != nil {
				t.Fatal(err)
			}
			wantAwards := 1
			if source == "none" {
				wantAwards = 0
			}
			if len(f.cycleAwards) != wantAwards || len(f.awarded) != wantAwards {
				t.Fatalf("awards=%v badges=%v", f.cycleAwards, f.awarded)
			}
			if wantAwards == 1 && (f.awardSources[0] != source || !f.cycleAwards[0].Equal(start)) {
				t.Fatal("wrong winner source or cycle")
			}
			if f.cycle.StartsAt.After(now) || !f.cycle.ClosesAt.After(now) {
				t.Fatalf("invalid next cycle: %+v", f.cycle)
			}
			if err := svc.RunCurationSweep(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(f.cycleAwards) != wantAwards {
				t.Fatal("sweep repeated award")
			}
		})
	}
}

func TestCurationBadgeFailureLeavesCycleUnchanged(t *testing.T) {
	f := newFakeStore(nil)
	f.cycle = CurationCycle{StartsAt: time.Now().Add(-2 * CurationWindow), ClosesAt: time.Now().Add(-CurationWindow)}
	original := f.cycle
	f.winner = CurationWinner{MapID: "map1", CreatorID: "creator"}
	f.badgeErr = errors.New("badge failed")
	if err := NewService(f, nil).RunCurationSweep(context.Background()); !errors.Is(err, f.badgeErr) {
		t.Fatal(err)
	}
	if f.cycle != original || len(f.cycleAwards) != 0 {
		t.Fatal("failed award advanced the cycle")
	}
}

func TestReportPolicyRejectsUnavailableReports(t *testing.T) {
	for _, tc := range []struct {
		name, target string
		facts        ReportEligibility
		allowed      bool
	}{
		{"self", "reporter", ReportEligibility{TargetParticipated: true}, false},
		{"absent target", "target", ReportEligibility{}, false},
		{"muted", "target", ReportEligibility{TargetParticipated: true, ReporterMuted: true}, false},
		{"allowed", "target", ReportEligibility{TargetParticipated: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore(nil)
			f.reportEligibility = tc.facts
			_, err := NewService(f, nil).CreateReport(context.Background(), "match", "reporter", tc.target, " HARASSMENT ", "reason")
			if (err == nil) != tc.allowed {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.allowed && len(f.reports) != 0 {
				t.Fatal("rejected report persisted")
			}
			if tc.allowed && (len(f.reports) != 1 || f.reports[0].Category != "harassment" || f.reports[0].Score != 1.5 || f.reports[0].Severity != "medium") {
				t.Fatalf("wrong report: %+v", f.reports)
			}
		})
	}
}
