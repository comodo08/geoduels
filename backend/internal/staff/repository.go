package staff

import (
	"context"
	"io"
	"time"

	"geoduels/internal/content"
	"geoduels/internal/maps"
	"geoduels/internal/seasons"
	"geoduels/pkg/contracts"
	"geoduels/pkg/maintenance"
)

// Store persists staff operations. WithinTx supplies a store whose writes
// commit together, or roll back when the callback returns an error.
type Store interface {
	WithinTx(ctx context.Context, fn func(Store) error) error
	RecordAudit(ctx context.Context, entry AuditEntry) (int64, error)
	AwardBadge(ctx context.Context, userID, badgeID string) (bool, error)
	RemoveBadge(ctx context.Context, userID, badgeID string) error
	BadgeCatalog() []contracts.AdminBadgeDefinition
	GrantBadgeByNickname(ctx context.Context, nickname, badgeID, actorID string) (contracts.PlayerBadge, bool, error)
	SearchSubjects(ctx context.Context, viewer Actor, query string, limit int) ([]contracts.AdminPlayerSummary, error)
	GetSubject(ctx context.Context, viewer Actor, userID string) (SubjectDetail, error)
	ListSignals(ctx context.Context, viewer Actor, subjectID string, limit int) ([]contracts.ModerationSignalSummary, error)
	ListAudit(ctx context.Context, viewer Actor, subjectID string, limit int) ([]contracts.ModerationAuditLogEntry, error)
	ListRoleGrants(ctx context.Context) ([]contracts.UserRoleGrant, error)
	GetMaintenance(ctx context.Context) (maintenance.Status, error)
	SetMaintenance(ctx context.Context, status maintenance.Status) error
	ClearMaintenance(ctx context.Context) error
	GetModerationSettings() (content.ModerationSettings, error)
	SetModerationSettings(content.ModerationSettings) error
	GetDiscordIntegrationSettings() (content.DiscordIntegrationSettings, error)
	SetDiscordIntegrationSettings(content.DiscordIntegrationSettings) error
	GetLobbyChangelog(defaultContent content.LobbyChangelogContent) (content.LobbyChangelogContent, error)
	ListChangelogPosts(includeUnpublished bool) ([]content.ChangelogPost, error)
	GetChangelogPostBySlug(slug string, publishedOnly bool) (content.ChangelogPost, bool, error)
	CreateChangelogPost(input content.ChangelogPostInput) (content.ChangelogPost, error)
	UpdateChangelogPost(id int64, input content.ChangelogPostInput) (content.ChangelogPost, bool, error)
	GetRankedSeasonSettings() (seasons.RankedSeasonSettings, error)
	SetRankedSeasonResetRule(monthlyResetDay int) (seasons.RankedSeasonSettings, error)
	SetMapCreatorTierOverride(userID string, tier *int) (contracts.MapUploadQuota, error)
	ImportOfficialMap(adminUserID string, input maps.OfficialMapImportInput, source io.Reader) (contracts.CustomMap, error)
	ReplaceMapLocations(mapKey, displayName string, dataset []byte) (contracts.MapImportSummary, error)
	NominateMap(ctx context.Context, actor, mapID string, start time.Time) error
	SetNominationLike(ctx context.Context, actor string, id int64, liked bool, start time.Time) error
	ListNominations(ctx context.Context, actor string, page int, cycle CurationCycle) (CurationPage, error)
	LockCurationCycle(ctx context.Context) (CurationCycle, error)
	CurationWinner(ctx context.Context, start time.Time) (CurationWinner, bool, error)
	TrendingCurationMap(ctx context.Context) (CurationWinner, bool, error)
	SaveCurationAward(ctx context.Context, start, now time.Time, winner CurationWinner, source string) error
	AdvanceCurationCycle(ctx context.Context, cycle CurationCycle, awardedAt time.Time) error
	ActiveSeasonID(ctx context.Context) (string, error)
	MatchPlayerIDs(ctx context.Context, matchID string) ([]string, error)
	RecentGuessEvents(ctx context.Context, userID string) ([]RiskGuessEvent, error)
	PlayerContext(ctx context.Context, userID, seasonID string) (rating int, rankedGames int, err error)
	RecordSignal(ctx context.Context, matchID string, signal RiskSignal, occurredAt time.Time) error
	ApplyCheatingBan(ctx context.Context, userID, reason, actorID string) (string, error)
	HasRelatedCheater(ctx context.Context, userID, registrationIP string) (bool, error)
	NotifyCheatingBan(ctx context.Context, userID, reason string, logID int64) error
	RefundCandidates(ctx context.Context, cheaterID string) ([]RefundCandidate, error)
	LockRefundRating(ctx context.Context, userID, seasonID string) (RatingState, bool, error)
	SaveRefund(ctx context.Context, award RefundAward) (bool, error)
	SetBan(ctx context.Context, userID, reason, actorID string, banned bool) error
	SetMute(ctx context.Context, userID, kind, reason, actorID string, until time.Time, muted bool) error
	ClearReporterMute(ctx context.Context, userID string) error
	Pardon(ctx context.Context, cutoff time.Time, actorID string) (CommunityPardonSummary, error)
	PreviewPardon(ctx context.Context, cutoff time.Time) (CommunityPardonSummary, error)
	AddIPBan(ctx context.Context, ipAddress, reason, actorID string) error
	RemoveIPBan(ctx context.Context, ipAddress string) error
	ListIPBans(ctx context.Context, limit int) ([]SignupIPBan, error)
	IsSignupIPBanned(ctx context.Context, ipAddress string) (bool, error)
	ReportEligibility(ctx context.Context, matchID, reporterID, subjectID string) (ReportEligibility, error)
	SaveReport(ctx context.Context, report PlayerReport) (contracts.ModerationSignalCreated, error)
	LockUser(ctx context.Context, userID string) error
	UserRoles(ctx context.Context, userID string) ([]string, error)
	GrantRole(ctx context.Context, userID, role, actorID, reason string) error
	RevokeRole(ctx context.Context, userID, role string) error
}
