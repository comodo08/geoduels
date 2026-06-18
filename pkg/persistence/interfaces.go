package persistence

import (
	"time"

	"geoduels/pkg/contracts"
)

type AccountRepository interface {
	UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (Identity, error)
	ProviderIdentityExists(provider, providerUserID string) (bool, error)
	GoogleIdentityExists(googleSub string) (bool, error)
	IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error)
	UnlinkProviderIdentity(userID, provider string) (Identity, error)
	CreateGuestIdentity() (Identity, error)
	GetIdentity(sub string) (Identity, error)
	SetNickname(sub, displayName string) error
	SuggestNickname(sub, displayName string) (string, error)
	DeleteAccount(userID string) error
	DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error)
}

type SessionRepository interface {
	CreateAuthSession(userID, refreshTokenHash string, expiresAt time.Time, params AuthSessionParams) (RefreshTokenRecord, error)
	GetAuthSessionByRefreshToken(hash string) (RefreshTokenRecord, bool, error)
	RotateAuthSession(sessionID, currentHash, nextHash string, expiresAt time.Time, usedAt time.Time) (RefreshTokenRecord, bool, error)
	RevokeAuthSession(sessionID string) error
	RevokeAuthSessionsForUser(userID string) error
}

type ProfileRepository interface {
	UpsertUser(userID, email, displayName string) error
	GetProfile(userID string) (Profile, error)
	UpdateSelectedBadge(userID, badgeID string) (Profile, error)
}

type BadgeRepository interface {
	SyncLoginBadges(userID string) error
	AwardDiscordServerMemberByDiscordID(discordUserID string) (bool, error)
	ClaimPendingDiscordSync(now time.Time) (DiscordSyncOutboxItem, bool, error)
	MarkDiscordSyncProcessed(id int64) error
	MarkDiscordSyncFailed(id int64, nextAttemptAt time.Time, lastError string) error
	GetDiscordLinkedUser(discordUserID string) (DiscordLinkedUser, bool, error)
	CreateDonationRef(userID string) (string, error)
	AwardSupporterByDonationRef(ref string) (bool, error)
}

type LeaderboardRepository interface {
	ListLeaderboard(mode, seasonID string, limit, offset int) ([]LeaderboardEntry, error)
	GetLeaderboardOverview(userID, mode, seasonID string, limit int) (LeaderboardOverview, error)
}

type MatchRepository interface {
	RecordMatchResult(snap contracts.MatchSnapshot) error
	RecordFinalMatchSnapshot(matchID string, snapshot []byte) error
	GetFinalMatchSnapshot(matchID string) ([]byte, bool, error)
	ListPlayerMatchHistory(userID string, limit int) ([]MatchHistorySummary, error)
}

type ModerationRepository interface {
	CreateModerationReport(params CreateModerationReportParams) (ModerationReportCreated, error)
	CreateDebugModerationReports(params CreateDebugModerationReportsParams) (DebugModerationReportsResult, error)
	RecomputeModerationProjections(limit int) (int, error)
	ListModerationCases(status string, limit int) ([]ModerationCaseSummary, error)
	GetModerationCase(caseID int64) (ModerationCaseDetail, error)
	AddModerationCaseAction(params ModerationCaseActionParams) (ModerationCaseDetail, error)
	ClaimModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error)
	ReleaseModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error)
	SetPlayerBan(userID, reason string, banned bool) error
	BanPlayerForCheating(userID, reason, actorUserID string) (CheatingBanSummary, error)
	ClearReporterMute(userID string) error
	IssueEloRefundsForCheater(userID string, lookback time.Duration) (EloRefundSummary, error)
	ListEnforcementActions(limit int) ([]EnforcementActionSummary, error)
	AddSignupIPBan(ipAddress, reason, createdBy string) error
	RemoveSignupIPBan(ipAddress string) error
	ListSignupIPBans(limit int) ([]SignupIPBan, error)
	IsSignupIPBanned(ipAddress string) (bool, error)
}

type AdminRepository interface {
	SetUserAdmin(userID string, isAdmin bool) error
	SetUserModerator(userID string, isModerator bool) error
	SearchPlayers(query string, limit int) ([]AdminPlayerSummary, error)
	GetAdminPlayerDetail(userID string) (AdminPlayerDetail, error)
	ListUserRoles() ([]UserRoleGrant, error)
	GrantUserRole(userID, role, grantedBy, reason string) error
	RevokeUserRole(userID, role, revokedBy, reason string) error
}

type ContentRepository interface {
	GetLobbyChangelog(defaultContent LobbyChangelogContent) (LobbyChangelogContent, error)
	SetLobbyChangelog(content LobbyChangelogContent) error
	ListChangelogPosts(includeUnpublished bool) ([]ChangelogPost, error)
	GetChangelogPostBySlug(slug string, publishedOnly bool) (ChangelogPost, bool, error)
	CreateChangelogPost(input ChangelogPostInput) (ChangelogPost, error)
	UpdateChangelogPost(id int64, input ChangelogPostInput) (ChangelogPost, bool, error)
	GetModerationSettings() (ModerationSettings, error)
	SetModerationSettings(settings ModerationSettings) error
}

type SeasonRepository interface {
	GetRankedSeasonSettings() (RankedSeasonSettings, error)
	SetRankedSeasonResetRule(monthlyResetDay int) (RankedSeasonSettings, error)
	RunDueRankedSeasonReset(now time.Time) (RankedSeasonResetResult, bool, error)
	ActivateMapRevision(mapKey, displayName string, dataset []byte) (MapRevisionSummary, error)
}

type GameplayMapRepository interface {
	GetGameplayMapSettings() (contracts.GameplayMapSettings, error)
	ResolveGameplayMapID(mode contracts.MatchMode, ruleset contracts.GameRuleset, requestedMapID string) (string, error)
}

type NotificationRepository interface {
	ListUserNotifications(userID string, limit int) ([]UserNotification, error)
	MarkUserNotificationRead(userID string, notificationID int64) error
	ClaimPendingNotification(notificationType string, now time.Time) (NotificationOutboxItem, bool, error)
	MarkNotificationSent(id int64) error
	MarkNotificationFailed(id int64, nextAttemptAt time.Time, lastError string) error
}

type RuntimeRepository interface {
	GetRuntimeMatch(matchID string) (RuntimeMatch, bool, error)
	RecordRuntimeMatch(matchID, state string, ownerEpoch int64, terminal bool) error
	UpsertMatchSession(params MatchSessionUpsert) error
	CompleteMatchSession(matchID string) error
	MatchSessionSourceParty(matchID string) (string, string, bool, error)
	ExpireStaleRuntimeMatches(prefix string, olderThan time.Duration) error
}

type ChatRepository interface {
	RecordChatMessage(conversationID, scopeKind, scopeID string, message ChatMessage) error
	ListChatMessages(conversationID string, limit int) ([]ChatMessage, error)
}

type PartyRepository interface {
	ExpireOpenParties() error
	ListOpenPartyIDs() ([]string, error)
	CloseInactiveOpenParties(lobbyIDs []string, inactiveFor time.Duration) (int64, error)
	CreateParty(ownerUserID string, mode contracts.MatchMode, mapScope string, ttl time.Duration) (contracts.PartySnapshot, error)
	SetPartyMode(lobbyID string, mode contracts.MatchMode) error
	GetPartyByID(lobbyID string) (contracts.PartySnapshot, bool, error)
	GetPartyByInviteCode(inviteCode string) (contracts.PartySnapshot, bool, error)
	GetPartyByMatchID(matchID string) (contracts.PartySnapshot, bool, error)
	JoinParty(lobbyID, userID string) (contracts.PartySnapshot, error)
	LeaveParty(lobbyID, userID string) (contracts.PartySnapshot, error)
	SetPartyMemberTeam(lobbyID, userID, teamID string) (contracts.PartySnapshot, error)
	KickPartyMember(lobbyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error)
	TransferPartyOwner(lobbyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error)
	MarkPartyInMatch(lobbyID, matchID string) (contracts.PartySnapshot, error)
	ReopenEndedParties() (int64, error)
}

type PartyConfigRepository interface {
	SetPartyConfig(lobbyID string, config contracts.MatchConfig) (contracts.PartySnapshot, error)
}

type Store interface {
	AccountRepository
	SessionRepository
	ProfileRepository
	BadgeRepository
	LeaderboardRepository
	MatchRepository
	ModerationRepository
	AdminRepository
	ContentRepository
	SeasonRepository
	GameplayMapRepository
	NotificationRepository
	RuntimeRepository
	ChatRepository
	PartyRepository
	Close()
}
