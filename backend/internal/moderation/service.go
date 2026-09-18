package moderation

import "time"

type Store interface {
	CreatePlayerReportSignal(params CreatePlayerReportSignalParams) (ModerationSignalCreated, error)
	ListSubjectModerationProfile(userID string) (ModerationSubjectProfile, error)
	ListModerationSignals(limit int) ([]ModerationSignalSummary, error)
	ListModerationLog(limit int) ([]ModerationAuditLogEntry, error)
	SetPlayerBan(userID, reason, actorUserID string, banned bool) error
	SetPlayerMute(userID, kind, reason, actorUserID string, until time.Time, muted bool) error
	BanPlayerForCheating(userID, reason, actorUserID string) (CheatingBanSummary, error)
	ClearReporterMute(userID string) error
	IssueEloRefundsForCheater(userID string, lookback time.Duration) (EloRefundSummary, error)
	AddSignupIPBan(ipAddress, reason, createdBy string) error
	RemoveSignupIPBan(ipAddress string) error
	ListSignupIPBans(limit int) ([]SignupIPBan, error)
	IsSignupIPBanned(ipAddress string) (bool, error)
	PreviewCommunityPardon(olderThan time.Duration) (CommunityPardonSummary, error)
	PardonBannedPlayers(olderThan time.Duration, actorUserID string) (CommunityPardonSummary, error)
	EvaluateAutoCheatBansForMatch(matchID string) error
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
