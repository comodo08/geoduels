package persistence

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/pkg/contracts"
)

const (
	modeDuel                        = "duel"
	defaultSeasonID                 = "s2"
	moderationProjectionAdvisoryKey = int64(0x67646d6f646572)
	moderationActiveRiskThreshold   = 1.5
	IdentityProviderGoogle          = "google"
	IdentityProviderDiscord         = "discord"
	badgeCodeDiscordMember          = int16(1)
	badgeCodeGeoDuelsTeam           = int16(2)
	badgeCodeDiscordServerMember    = int16(3)
	badgeCodeSupporter              = int16(4)
	badgeCodeSpeedrunner            = int16(5)
	badgeCodeElo1000                = int16(6)
	badgeCodeElo1500                = int16(7)
	badgeCodeElo2000                = int16(8)
	badgeCodeSeasonRank             = int16(10)
)

type Profile struct {
	UserID            string                  `json:"userId"`
	DisplayName       string                  `json:"displayName"`
	AvatarURL         string                  `json:"avatarUrl,omitempty"`
	MMR               int                     `json:"mmr"`
	RatingRD          float64                 `json:"ratingRd,omitempty"`
	SeasonID          string                  `json:"seasonId,omitempty"`
	GamesPlayed       int                     `json:"gamesPlayed"`
	Wins              int                     `json:"wins"`
	RankedGamesPlayed int                     `json:"rankedGamesPlayed"`
	RankedWins        int                     `json:"rankedWins"`
	IsGuest           bool                    `json:"isGuest"`
	IsAdmin           bool                    `json:"isAdmin"`
	IsModerator       bool                    `json:"isModerator"`
	IsBanned          bool                    `json:"isBanned"`
	BanReason         string                  `json:"banReason,omitempty"`
	Badges            []contracts.PlayerBadge `json:"badges,omitempty"`
	SelectedBadge     *contracts.PlayerBadge  `json:"selectedBadge,omitempty"`
}

type LeaderboardEntry struct {
	Rank        int    `json:"rank"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	MMR         int    `json:"mmr"`
	GamesPlayed int    `json:"gamesPlayed"`
	Wins        int    `json:"wins"`
}

type LeaderboardOverview struct {
	Mode         string             `json:"mode"`
	SeasonID     string             `json:"season"`
	SelfRank     int                `json:"selfRank"`
	TotalPlayers int                `json:"totalPlayers"`
	Entries      []LeaderboardEntry `json:"entries"`
}

type Identity struct {
	Sub                   string
	Email                 string
	GoogleName            string
	ProviderName          string
	AvatarURL             string
	Onboarded             bool
	DisplayName           string
	AccountType           string
	LinkedProviders       []string
	AuthMigrationRequired bool
	RecoveryAvailable     bool
	IsAdmin               bool
	IsModerator           bool
	IsBanned              bool
	BanReason             string
}

type AdminPlayerSummary = contracts.AdminPlayerSummary

type ModerationCaseSummary = contracts.ModerationCaseSummary
type ModerationReportSummary = contracts.ModerationReportSummary
type ModerationCaseEvent = contracts.ModerationCaseEvent
type ModerationActionSummary = contracts.ModerationActionSummary
type ModerationEvidenceSummary = contracts.ModerationEvidenceSummary
type ModerationCaseLogEntry = contracts.ModerationCaseLogEntry
type ModerationMatchSummary = contracts.ModerationMatchSummary
type ModerationMatchPlayerSummary = contracts.ModerationMatchPlayerSummary
type ModerationCaseDetail = contracts.ModerationCaseDetail
type ModerationReportCreated = contracts.ModerationReportCreated
type ModerationCaseNotificationPayload = contracts.ModerationCaseNotificationPayload
type EnforcementActionSummary = contracts.EnforcementActionSummary
type UserRoleGrant = contracts.UserRoleGrant

type MapRevisionSummary = contracts.MapRevisionSummary

type MatchHistorySummary struct {
	MatchID      string    `json:"matchId"`
	Mode         string    `json:"mode"`
	StartedAt    time.Time `json:"startedAt"`
	EndedAt      time.Time `json:"endedAt"`
	WinnerUserID string    `json:"winnerUserId,omitempty"`
}

type CreateModerationReportParams struct {
	MatchID        string
	ReporterUserID string
	ReportedUserID string
	Category       string
	Reason         string
}

type ModerationCaseActionParams struct {
	CaseID      int64
	ActorUserID string
	ActionType  string
	Reason      string
	Status      string
	AssignedTo  string
	MuteUserID  string
	MuteUntil   time.Time
}

type CreateDebugModerationReportsParams struct {
	ReportedUserID string
	Count          int
	Category       string
	Reason         string
	CreatedBy      string
}

type DebugModerationReportsResult struct {
	CaseID          int64    `json:"caseId"`
	ReportsCreated  int      `json:"reportsCreated"`
	ReporterUserIDs []string `json:"reporterUserIds"`
}

type NotificationOutboxItem struct {
	ID          int64
	Type        string
	PayloadJSON []byte
	Attempts    int
}

type EloRefundSummary struct {
	RefundsIssued int `json:"refundsIssued"`
	TotalRefunded int `json:"totalRefunded"`
}

type CheatingBanSummary struct {
	UserID          string           `json:"userId"`
	Reason          string           `json:"reason,omitempty"`
	Refunds         EloRefundSummary `json:"refunds"`
	IPSignupBanned  bool             `json:"ipSignupBanned"`
	ArchivedCaseIDs []int64          `json:"archivedCaseIds,omitempty"`
}

type AdminPlayerStats struct {
	TotalMatches     int `json:"totalMatches"`
	RankedMatches    int `json:"rankedMatches"`
	DuelMatches      int `json:"duelMatches"`
	SingleplayerRuns int `json:"singleplayerRuns"`
	Wins             int `json:"wins"`
	Losses           int `json:"losses"`
}

type AdminPlayerEloPoint struct {
	Date   time.Time `json:"date"`
	MMR    int       `json:"mmr"`
	Delta  int       `json:"delta"`
	Played int       `json:"played"`
}

type AdminPlayerDetail struct {
	Player     AdminPlayerSummary    `json:"player"`
	Stats      AdminPlayerStats      `json:"stats"`
	EloHistory []AdminPlayerEloPoint `json:"eloHistory"`
	Matches    []MatchHistorySummary `json:"matches"`
}

type UserNotification struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

type SignupIPBan struct {
	ID        int64     `json:"id"`
	IPAddress string    `json:"ipAddress"`
	Reason    string    `json:"reason,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type LobbyChangelogContent struct {
	Eyebrow   string    `json:"eyebrow"`
	Title     string    `json:"title"`
	Markdown  string    `json:"markdown"`
	Slug      string    `json:"slug,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type ChangelogPost struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	Markdown  string    `json:"markdown"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChangelogPostInput struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Markdown  string `json:"markdown"`
	Published bool   `json:"published"`
}

type ModerationSettings struct {
	DiscordWebhookURL string `json:"discordWebhookUrl"`
}

type RankedSeasonSettings struct {
	ActiveSeasonID string `json:"activeSeasonId"`
}

type RankedSeasonRolloverResult struct {
	PreviousSeasonID string `json:"previousSeasonId"`
	ActiveSeasonID   string `json:"activeSeasonId"`
	BadgesAwarded    int    `json:"badgesAwarded"`
	PlayersSeeded    int    `json:"playersSeeded"`
}

type RefreshTokenRecord struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	LastUsedAt       time.Time
	RevokedAt        *time.Time
	UserAgent        string
	IPAddress        string
}

type AuthSessionParams struct {
	UserAgent string
	IPAddress string
}

type RuntimeMatch struct {
	MatchID    string
	State      string
	OwnerEpoch int64
	StartedAt  time.Time
	EndedAt    time.Time
}

type ChatMessage = contracts.ChatMessage

type Store interface {
	UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (Identity, error)
	ProviderIdentityExists(provider, providerUserID string) (bool, error)
	GoogleIdentityExists(googleSub string) (bool, error)
	IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error)
	UnlinkProviderIdentity(userID, provider string) (Identity, error)
	CreateGuestIdentity() (Identity, error)
	GetIdentity(sub string) (Identity, error)
	CompleteOnboarding(sub, email, displayName string) error
	UpdateDisplayName(sub, displayName string) error
	SetUserAdmin(userID string, isAdmin bool) error
	SetUserModerator(userID string, isModerator bool) error
	SearchPlayers(query string, limit int) ([]AdminPlayerSummary, error)
	GetAdminPlayerDetail(userID string) (AdminPlayerDetail, error)
	SetPlayerBan(userID, reason string, banned bool) error
	BanPlayerForCheating(userID, reason, actorUserID string) (CheatingBanSummary, error)
	ClearReporterMute(userID string) error
	GetLobbyChangelog(defaultContent LobbyChangelogContent) (LobbyChangelogContent, error)
	SetLobbyChangelog(content LobbyChangelogContent) error
	ListChangelogPosts(includeUnpublished bool) ([]ChangelogPost, error)
	GetChangelogPostBySlug(slug string, publishedOnly bool) (ChangelogPost, bool, error)
	CreateChangelogPost(input ChangelogPostInput) (ChangelogPost, error)
	UpdateChangelogPost(id int64, input ChangelogPostInput) (ChangelogPost, bool, error)
	GetModerationSettings() (ModerationSettings, error)
	SetModerationSettings(settings ModerationSettings) error
	GetRankedSeasonSettings() (RankedSeasonSettings, error)
	RolloverRankedSeason(nextSeasonID string) (RankedSeasonRolloverResult, error)
	ActivateMapRevision(mapKey, displayName string, dataset []byte) (MapRevisionSummary, error)
	CreateAuthSession(userID, refreshTokenHash string, expiresAt time.Time, params AuthSessionParams) (RefreshTokenRecord, error)
	GetAuthSessionByRefreshToken(hash string) (RefreshTokenRecord, bool, error)
	RotateAuthSession(sessionID, currentHash, nextHash string, expiresAt time.Time, usedAt time.Time) (RefreshTokenRecord, bool, error)
	RevokeAuthSession(sessionID string) error
	RevokeAuthSessionsForUser(userID string) error
	DeleteAccount(userID string) error
	DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error)
	UpsertUser(userID, email, displayName string) error
	GetProfile(userID string) (Profile, error)
	UpdateSelectedBadge(userID, badgeID string) (Profile, error)
	SyncLoginBadges(userID string) error
	AwardDiscordServerMemberByDiscordID(discordUserID string) (bool, error)
	CreateDonationRef(userID string) (string, error)
	AwardSupporterByDonationRef(ref string) (bool, error)
	ListLeaderboard(mode, seasonID string, limit, offset int) ([]LeaderboardEntry, error)
	GetLeaderboardOverview(userID, mode, seasonID string, limit int) (LeaderboardOverview, error)
	RecordMatchResult(snap contracts.MatchSnapshot) error
	RecordFinalMatchSnapshot(matchID string, snapshot []byte) error
	GetFinalMatchSnapshot(matchID string) ([]byte, bool, error)
	ListPlayerMatchHistory(userID string, limit int) ([]MatchHistorySummary, error)
	CreateModerationReport(params CreateModerationReportParams) (ModerationReportCreated, error)
	CreateDebugModerationReports(params CreateDebugModerationReportsParams) (DebugModerationReportsResult, error)
	RecomputeModerationProjections(limit int) (int, error)
	ListModerationCases(status string, limit int) ([]ModerationCaseSummary, error)
	GetModerationCase(caseID int64) (ModerationCaseDetail, error)
	AddModerationCaseAction(params ModerationCaseActionParams) (ModerationCaseDetail, error)
	ClaimModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error)
	ReleaseModerationCase(caseID int64, actorUserID string) (ModerationCaseDetail, error)
	ListEnforcementActions(limit int) ([]EnforcementActionSummary, error)
	ListUserRoles() ([]UserRoleGrant, error)
	GrantUserRole(userID, role, grantedBy, reason string) error
	RevokeUserRole(userID, role, revokedBy, reason string) error
	IssueEloRefundsForCheater(userID string, lookback time.Duration) (EloRefundSummary, error)
	ListUserNotifications(userID string, limit int) ([]UserNotification, error)
	MarkUserNotificationRead(userID string, notificationID int64) error
	ClaimPendingNotification(notificationType string, now time.Time) (NotificationOutboxItem, bool, error)
	MarkNotificationSent(id int64) error
	MarkNotificationFailed(id int64, nextAttemptAt time.Time, lastError string) error
	AddSignupIPBan(ipAddress, reason, createdBy string) error
	RemoveSignupIPBan(ipAddress string) error
	ListSignupIPBans(limit int) ([]SignupIPBan, error)
	IsSignupIPBanned(ipAddress string) (bool, error)
	GetRuntimeMatch(matchID string) (RuntimeMatch, bool, error)
	RecordRuntimeMatch(matchID, state string, ownerEpoch int64, terminal bool) error
	RecordChatMessage(conversationID, scopeKind, scopeID string, message ChatMessage) error
	ListChatMessages(conversationID string, limit int) ([]ChatMessage, error)
	ExpireStaleRuntimeMatches(prefix string, olderThan time.Duration) error
	ExpireOpenLobbies() error
	ListOpenLobbyIDs() ([]string, error)
	CloseInactiveOpenLobbies(lobbyIDs []string, inactiveFor time.Duration) (int64, error)
	CreateLobby(ownerUserID string, mode contracts.MatchMode, mapScope string, ttl time.Duration) (contracts.LobbySnapshot, error)
	SetLobbyMode(lobbyID string, mode contracts.MatchMode) error
	GetLobbyByID(lobbyID string) (contracts.LobbySnapshot, bool, error)
	GetLobbyByInviteCode(inviteCode string) (contracts.LobbySnapshot, bool, error)
	GetLobbyByMatchID(matchID string) (contracts.LobbySnapshot, bool, error)
	JoinLobby(lobbyID, userID string) (contracts.LobbySnapshot, error)
	LeaveLobby(lobbyID, userID string) (contracts.LobbySnapshot, error)
	SetLobbyMemberTeam(lobbyID, userID, teamID string) (contracts.LobbySnapshot, error)
	KickLobbyMember(lobbyID, ownerUserID, targetUserID string) (contracts.LobbySnapshot, error)
	TransferLobbyOwner(lobbyID, ownerUserID, targetUserID string) (contracts.LobbySnapshot, error)
	MarkLobbyInMatch(lobbyID, matchID string) (contracts.LobbySnapshot, error)
	ReopenEndedLobbies() (int64, error)
	Close()
}

func NewFromEnv() (Store, error) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		return nil, errors.New("POSTGRES_URL is required")
	}
	url = normalizeDBURLForContainer(url)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	if maxConns := getenvInt("POSTGRES_MAX_CONNS", 0); maxConns > 0 {
		cfg.MaxConns = int32(maxConns)
	}
	if strings.EqualFold(os.Getenv("POSTGRES_PGBOUNCER"), "true") {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &pgStore{pool: pool}, nil
}

func getenvInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

type pgStore struct {
	pool *pgxpool.Pool
}

func providerOnboardedAt(linkedGuest bool) any {
	if linkedGuest {
		return time.Now()
	}
	return nil
}

func googleOnboardedAt(linkedGuest bool) any {
	return providerOnboardedAt(linkedGuest)
}

func chooseProviderIdentityUser(existingProviderUserID, existingEmailUserID, existingEmailAccountType, linkUserID, linkAccountType string) (string, bool) {
	if existingProviderUserID != "" {
		return existingProviderUserID, false
	}
	if existingEmailUserID != "" {
		return existingEmailUserID, existingEmailAccountType == "guest"
	}
	if linkUserID != "" && linkAccountType != "" {
		return linkUserID, linkAccountType == "guest"
	}
	return newUserID(), false
}

func chooseGoogleIdentityUser(existingGoogleUserID, existingEmailUserID, existingEmailAccountType, linkUserID, linkAccountType string) (string, bool) {
	return chooseProviderIdentityUser(existingGoogleUserID, existingEmailUserID, existingEmailAccountType, linkUserID, linkAccountType)
}

func providerUsesAccountEmail(provider string) bool {
	return provider == IdentityProviderGoogle || provider == IdentityProviderDiscord
}

func isSyntheticOAuthEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	return strings.HasSuffix(email, "@oauth.invalid") || strings.HasSuffix(email, ".oauth.invalid")
}

func providerAccountEmail(provider, email string) any {
	email = strings.TrimSpace(email)
	if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
		return email
	}
	return nil
}

func (s *pgStore) UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (Identity, error) {
	return s.UpsertProviderIdentity(IdentityProviderGoogle, googleSub, email, googleName, avatarURL, linkUserID)
}

func (s *pgStore) UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return Identity{}, errors.New("provider required")
	}
	if providerUserID == "" {
		return Identity{}, errors.New("provider subject required")
	}
	if email == "" {
		email = providerUserID + "@oauth.invalid"
	}
	if providerName == "" {
		providerName = providerUserID
	}
	if banned, _, err := s.IsProviderIdentityBanned(provider, providerUserID); err != nil {
		return Identity{}, err
	} else if banned {
		return Identity{}, errors.New("provider identity banned")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Identity{}, err
	}
	defer tx.Rollback(ctx)
	seasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return Identity{}, err
	}

	var existingProviderUserID string
	row := tx.QueryRow(ctx, `
		select user_id
		from user_identities
		where provider = $1 and provider_user_id = $2
	`, provider, providerUserID)
	if err := row.Scan(&existingProviderUserID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, err
	}
	var existingEmailUserID string
	var existingEmailAccountType string
	if providerUsesAccountEmail(provider) && existingProviderUserID == "" && email != "" && !isSyntheticOAuthEmail(email) {
		row = tx.QueryRow(ctx, `
			select id, account_type
			from users
			where lower(email) = lower($1)
			order by case when account_type = 'registered' then 0 else 1 end, created_at asc
			limit 1
		`, email)
		if err := row.Scan(&existingEmailUserID, &existingEmailAccountType); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, err
		}
	}
	var linkAccountType string
	if existingProviderUserID == "" && existingEmailUserID == "" && linkUserID != "" {
		row = tx.QueryRow(ctx, `
			select account_type
			from users
			where id = $1
		`, linkUserID)
		if err := row.Scan(&linkAccountType); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, err
		}
	}
	userID, linkedGuest := chooseProviderIdentityUser(existingProviderUserID, existingEmailUserID, existingEmailAccountType, linkUserID, linkAccountType)
	onboardedAt := providerOnboardedAt(linkedGuest)
	userEmail := providerAccountEmail(provider, email)

	if _, err := tx.Exec(ctx, `
		insert into users (id, email, display_name, avatar_url, onboarded_at, account_type)
		values ($1, $2, $3, $4, $5, 'registered')
		on conflict (id) do update set
			email = coalesce(excluded.email, users.email),
			display_name = case
				when users.account_type = 'guest' then excluded.display_name
				when users.onboarded_at is not null and nullif(users.display_name, '') is not null then users.display_name
				else excluded.display_name
			end,
			avatar_url = excluded.avatar_url,
			onboarded_at = coalesce(users.onboarded_at, excluded.onboarded_at),
			account_type = 'registered'
	`, userID, userEmail, providerName, nullable(avatarURL), onboardedAt); err != nil {
		return Identity{}, err
	}
	if existingProviderUserID != "" {
		if _, err := tx.Exec(ctx, `
			insert into user_identities(user_id, provider, provider_user_id, email, provider_name, avatar_url, last_seen_at)
			values($1, $2, $3, $4, $5, $6, now())
			on conflict (provider, provider_user_id) do update set
				user_id = excluded.user_id,
				email = excluded.email,
				provider_name = excluded.provider_name,
				avatar_url = case
					when excluded.avatar_url is null then user_identities.avatar_url
					when excluded.avatar_url = '' then user_identities.avatar_url
					else excluded.avatar_url
				end,
				last_seen_at = now()
		`, userID, provider, providerUserID, email, providerName, nullable(avatarURL)); err != nil {
			return Identity{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			insert into user_identities(user_id, provider, provider_user_id, email, provider_name, avatar_url, last_seen_at)
			values($1, $2, $3, $4, $5, $6, now())
			on conflict (user_id, provider) do update set
				provider_user_id = excluded.provider_user_id,
				email = excluded.email,
				provider_name = excluded.provider_name,
				avatar_url = case
					when excluded.avatar_url is null then user_identities.avatar_url
					when excluded.avatar_url = '' then user_identities.avatar_url
					else excluded.avatar_url
				end,
				last_seen_at = now()
		`, userID, provider, providerUserID, email, providerName, nullable(avatarURL)); err != nil {
			return Identity{}, err
		}
	}
	if err := recordUserIdentityHistory(ctx, tx, userID, provider, providerUserID, email, providerName, avatarURL); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranks (user_id, mode, mmr, season_id)
		values ($1, $2, $4, $3)
		on conflict (user_id, mode, season_id) do nothing
	`, userID, modeDuel, seasonID, initialMMR); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into user_stats (user_id, games_played, wins)
		values ($1, 0, 0)
		on conflict (user_id) do nothing
	`, userID); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranked_stats (user_id, mode, season_id, games_played, wins)
		values ($1, $2, $3, 0, 0)
		on conflict (user_id, mode, season_id) do nothing
	`, userID, modeDuel, seasonID); err != nil {
		return Identity{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Identity{}, err
	}
	return s.GetIdentity(userID)
}

func (s *pgStore) LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	linkUserID = strings.TrimSpace(linkUserID)
	if provider == "" {
		return Identity{}, errors.New("provider required")
	}
	if providerUserID == "" {
		return Identity{}, errors.New("provider subject required")
	}
	if linkUserID == "" {
		return Identity{}, errors.New("link user required")
	}
	if email == "" {
		email = providerUserID + "@oauth.invalid"
	}
	if providerName == "" {
		providerName = providerUserID
	}
	if banned, _, err := s.IsProviderIdentityBanned(provider, providerUserID); err != nil {
		return Identity{}, err
	} else if banned {
		return Identity{}, errors.New("provider identity banned")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Identity{}, err
	}
	defer tx.Rollback(ctx)
	seasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return Identity{}, err
	}

	var linkAccountType string
	if err := tx.QueryRow(ctx, `
		select account_type
		from users
		where id = $1
	`, linkUserID).Scan(&linkAccountType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, errors.New("link user not found")
		}
		return Identity{}, err
	}

	var existingProviderUserID string
	err = tx.QueryRow(ctx, `
		select user_id
		from user_identities
		where provider = $1 and provider_user_id = $2
	`, provider, providerUserID).Scan(&existingProviderUserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, err
	}
	if existingProviderUserID != "" && existingProviderUserID != linkUserID {
		return Identity{}, errors.New("provider identity already linked")
	}
	if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
		var existingEmailUserID string
		err = tx.QueryRow(ctx, `
			select id
			from users
			where lower(email) = lower($1)
			limit 1
		`, email).Scan(&existingEmailUserID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, err
		}
		if existingEmailUserID != "" && existingEmailUserID != linkUserID {
			return Identity{}, errors.New("provider identity already linked")
		}
	}

	onboardedAt := providerOnboardedAt(linkAccountType == "guest")
	userEmail := providerAccountEmail(provider, email)
	if _, err := tx.Exec(ctx, `
		insert into users (id, email, display_name, avatar_url, onboarded_at, account_type)
		values ($1, $2, $3, $4, $5, 'registered')
		on conflict (id) do update set
			email = coalesce(excluded.email, users.email),
			display_name = case
				when users.account_type = 'guest' then excluded.display_name
				when users.onboarded_at is not null and nullif(users.display_name, '') is not null then users.display_name
				else excluded.display_name
			end,
			avatar_url = excluded.avatar_url,
			onboarded_at = coalesce(users.onboarded_at, excluded.onboarded_at),
			account_type = 'registered'
	`, linkUserID, userEmail, providerName, nullable(avatarURL), onboardedAt); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into user_identities(user_id, provider, provider_user_id, email, provider_name, avatar_url, last_seen_at)
		values($1, $2, $3, $4, $5, $6, now())
		on conflict (user_id, provider) do update set
			provider_user_id = excluded.provider_user_id,
			email = excluded.email,
			provider_name = excluded.provider_name,
			avatar_url = case
				when excluded.avatar_url is null then user_identities.avatar_url
				when excluded.avatar_url = '' then user_identities.avatar_url
				else excluded.avatar_url
			end,
			last_seen_at = now()
	`, linkUserID, provider, providerUserID, email, providerName, nullable(avatarURL)); err != nil {
		return Identity{}, err
	}
	if err := recordUserIdentityHistory(ctx, tx, linkUserID, provider, providerUserID, email, providerName, avatarURL); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranks (user_id, mode, mmr, season_id)
		values ($1, $2, $4, $3)
		on conflict (user_id, mode, season_id) do nothing
	`, linkUserID, modeDuel, seasonID, initialMMR); err != nil {
		return Identity{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranked_stats (user_id, mode, season_id, games_played, wins)
		values ($1, $2, $3, 0, 0)
		on conflict (user_id, mode, season_id) do nothing
	`, linkUserID, modeDuel, seasonID); err != nil {
		return Identity{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Identity{}, err
	}
	return s.GetIdentity(linkUserID)
}

func (s *pgStore) GoogleIdentityExists(googleSub string) (bool, error) {
	return s.ProviderIdentityExists(IdentityProviderGoogle, googleSub)
}

func (s *pgStore) ProviderIdentityExists(provider, providerUserID string) (bool, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" || strings.TrimSpace(providerUserID) == "" {
		return false, errors.New("provider and subject required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		select exists(
			select 1 from user_identities
			where provider = $1 and provider_user_id = $2
		)
	`, provider, providerUserID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *pgStore) IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	if provider == "" || providerUserID == "" {
		return false, "", errors.New("provider and subject required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var reason string
	err := s.pool.QueryRow(ctx, `
		select coalesce(reason, '')
		from oauth_identity_bans
		where provider = $1
		  and provider_user_id = $2
		  and revoked_at is null
		limit 1
	`, provider, providerUserID).Scan(&reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, reason, nil
}

func recordUserIdentityHistory(ctx context.Context, tx pgx.Tx, userID, provider, providerUserID, email, providerName, avatarURL string) error {
	_, err := tx.Exec(ctx, `
		insert into user_identity_history(user_id, provider, provider_user_id, email, provider_name, avatar_url, first_seen_at, last_seen_at, deleted_at)
		values($1, $2, $3, $4, $5, $6, now(), now(), null)
		on conflict (user_id, provider, provider_user_id) do update set
			email = excluded.email,
			provider_name = excluded.provider_name,
			avatar_url = case
				when excluded.avatar_url is null then user_identity_history.avatar_url
				when excluded.avatar_url = '' then user_identity_history.avatar_url
				else excluded.avatar_url
			end,
			last_seen_at = now(),
			deleted_at = null
	`, userID, provider, providerUserID, email, providerName, nullable(avatarURL))
	return err
}

func (s *pgStore) UnlinkProviderIdentity(userID, provider string) (Identity, error) {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(strings.ToLower(provider))
	if userID == "" || provider == "" {
		return Identity{}, errors.New("user and provider required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Identity{}, err
	}
	defer tx.Rollback(ctx)

	var providerCount int
	if err := tx.QueryRow(ctx, `
		select count(*)
		from user_identities
		where user_id = $1
	`, userID).Scan(&providerCount); err != nil {
		return Identity{}, err
	}
	if providerCount <= 1 {
		return Identity{}, errors.New("cannot unlink the last sign-in method")
	}
	tag, err := tx.Exec(ctx, `
		delete from user_identities
		where user_id = $1 and provider = $2
	`, userID, provider)
	if err != nil {
		return Identity{}, err
	}
	if tag.RowsAffected() == 0 {
		return Identity{}, errors.New("provider is not linked")
	}
	if _, err := tx.Exec(ctx, `
		update user_identity_history
		set deleted_at = coalesce(deleted_at, now())
		where user_id = $1
		  and provider = $2
		  and deleted_at is null
	`, userID, provider); err != nil {
		return Identity{}, err
	}
	if provider == IdentityProviderGoogle {
		if _, err := tx.Exec(ctx, `
			update users
			set email = null
			where id = $1
			  and not exists (
				select 1 from user_identities
				where user_id = $1 and provider = 'google'
			  )
		`, userID); err != nil {
			return Identity{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Identity{}, err
	}
	return s.GetIdentity(userID)
}

func (s *pgStore) CreateGuestIdentity() (Identity, error) {
	userID := newUserID()
	if err := s.UpsertUser(userID, "", "Guest"); err != nil {
		return Identity{}, err
	}
	return s.GetIdentity(userID)
}

func (s *pgStore) GetIdentity(sub string) (Identity, error) {
	if sub == "" {
		return Identity{}, errors.New("subject required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select
			u.id,
			coalesce(u.email, ui.email, ''),
			coalesce(ui.provider_name, ''),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(u.onboarded_at is not null, false) as onboarded,
				coalesce(nullif(u.display_name, ''), ui.provider_name, u.id),
				u.account_type,
				coalesce(u.is_admin, false),
				coalesce(u.is_moderator, false),
				coalesce(u.banned_at is not null, false),
				coalesce(u.ban_reason, '')
		from users u
		left join lateral (
			select email, provider_name, avatar_url
			from user_identities
			where user_id = u.id
			  and provider in ('discord', 'google')
			order by case provider when 'discord' then 0 when 'google' then 1 else 2 end, created_at asc
			limit 1
		) ui on true
		where u.id = $1
	`, sub)
	var out Identity
	if err := row.Scan(
		&out.Sub,
		&out.Email,
		&out.GoogleName,
		&out.AvatarURL,
		&out.Onboarded,
		&out.DisplayName,
		&out.AccountType,
		&out.IsAdmin,
		&out.IsModerator,
		&out.IsBanned,
		&out.BanReason,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, errors.New("identity not found")
		}
		return Identity{}, err
	}
	out.ProviderName = out.GoogleName
	out.LinkedProviders, _ = s.userProviders(ctx, sub)
	out.AuthMigrationRequired = false
	out.RecoveryAvailable = false
	return out, nil
}

func (s *pgStore) userProviders(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select provider
		from user_identities
		where user_id = $1
		order by case provider when 'discord' then 0 when 'google' then 1 else 2 end, provider
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []string
	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func (s *pgStore) CompleteOnboarding(sub, email, displayName string) error {
	if sub == "" {
		return errors.New("subject required")
	}
	if displayName == "" {
		return errors.New("display name required")
	}
	var nullableEmail any
	if email != "" {
		nullableEmail = email
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `
		update users
		set email = case
				when email is not null then email
				when $2::text is null then email
				when exists (
					select 1
					from users existing
					where lower(existing.email) = lower($2)
					  and existing.id <> users.id
				) then email
				else $2
			end,
			display_name = $3,
			onboarded_at = coalesce(onboarded_at, now())
		where id = $1
	`, sub, nullableEmail, displayName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *pgStore) UpdateDisplayName(sub, displayName string) error {
	if sub == "" {
		return errors.New("subject required")
	}
	if displayName == "" {
		return errors.New("display name required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `
		update users
		set display_name = $2
		where id = $1
		  and coalesce(account_type, 'registered') <> 'guest'
	`, sub, displayName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *pgStore) SetUserAdmin(userID string, isAdmin bool) error {
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		update users
		set is_admin = $2,
			is_moderator = case when $2 then true else is_moderator end
		where id = $1
	`, userID, isAdmin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if isAdmin {
		if _, err := tx.Exec(ctx, `
			insert into user_roles(user_id, role, granted_at, reason)
			values($1, 'admin', now(), 'legacy admin toggle')
			on conflict (user_id, role) where revoked_at is null do nothing
		`, userID); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			update user_roles
			set revoked_at = coalesce(revoked_at, now()), reason = coalesce(nullif(reason, ''), 'legacy admin toggle')
			where user_id = $1 and role = 'admin' and revoked_at is null
		`, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *pgStore) SetUserModerator(userID string, isModerator bool) error {
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		update users
		set is_moderator = $2
		where id = $1
	`, userID, isModerator)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if isModerator {
		if _, err := tx.Exec(ctx, `
			insert into user_roles(user_id, role, granted_at, reason)
			values($1, 'moderator', now(), 'legacy moderator toggle')
			on conflict (user_id, role) where revoked_at is null do nothing
		`, userID); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			update user_roles
			set revoked_at = coalesce(revoked_at, now()), reason = coalesce(nullif(reason, ''), 'legacy moderator toggle')
			where user_id = $1 and role = 'moderator' and revoked_at is null
		`, userID); err != nil {
			return err
		}
		if err := removeGeoDuelsTeamBadgeTx(ctx, tx, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *pgStore) ListUserRoles() ([]UserRoleGrant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select
			ur.user_id,
			coalesce(nullif(u.display_name, ''), u.id),
			coalesce(u.email, ''),
			ur.role,
			coalesce(ur.granted_by, ''),
			ur.granted_at,
			ur.revoked_at,
			coalesce(ur.reason, '')
		from user_roles ur
		left join users u on u.id = ur.user_id
		where ur.revoked_at is null
		order by
			case ur.role when 'admin' then 0 when 'moderator' then 1 else 2 end,
			ur.granted_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UserRoleGrant{}
	for rows.Next() {
		var item UserRoleGrant
		var revokedAt *time.Time
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.Email, &item.Role, &item.GrantedBy, &item.GrantedAt, &revokedAt, &item.Reason); err != nil {
			return nil, err
		}
		if revokedAt != nil {
			item.RevokedAt = *revokedAt
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func normalizeAdminRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "admin", "moderator":
		return role, nil
	default:
		return "", errors.New("unsupported role")
	}
}

func (s *pgStore) GrantUserRole(userID, role, grantedBy, reason string) error {
	userID = strings.TrimSpace(userID)
	role, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		update users
		set is_admin = case when $2 = 'admin' then true else is_admin end,
			is_moderator = case when $2 in ('admin', 'moderator') then true else is_moderator end
		where id = $1
	`, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if _, err := tx.Exec(ctx, `
		insert into user_roles(user_id, role, granted_by, granted_at, reason)
		values($1, $2, nullif($3, ''), now(), nullif($4, ''))
		on conflict (user_id, role) where revoked_at is null do update set
			granted_by = excluded.granted_by,
			reason = excluded.reason
	`, userID, role, strings.TrimSpace(grantedBy), strings.TrimSpace(reason)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *pgStore) RevokeUserRole(userID, role, revokedBy, reason string) error {
	userID = strings.TrimSpace(userID)
	role, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		update user_roles
		set revoked_at = coalesce(revoked_at, now()),
			reason = coalesce(nullif($3, ''), reason)
		where user_id = $1 and role = $2 and revoked_at is null
	`, userID, role, strings.TrimSpace(reason)); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		update users
		set is_admin = case when $2 = 'admin' then false else is_admin end,
			is_moderator = case
				when not exists (
					select 1 from user_roles
					where user_id = $1 and role in ('admin', 'moderator') and revoked_at is null
				) then false
				else is_moderator
			end
		where id = $1
	`, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	var hasTeamRole bool
	if err := tx.QueryRow(ctx, `
		select exists (
			select 1
			from user_roles
			where user_id = $1 and role in ('admin', 'moderator') and revoked_at is null
		)
	`, userID).Scan(&hasTeamRole); err != nil {
		return err
	}
	if !hasTeamRole {
		if err := removeGeoDuelsTeamBadgeTx(ctx, tx, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *pgStore) SearchPlayers(query string, limit int) ([]AdminPlayerSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	pattern := "%"
	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		pattern = "%" + strings.ToLower(trimmed) + "%"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select
			u.id,
			coalesce(u.email, ''),
			coalesce(nullif(u.display_name, ''), ui.provider_name, u.id),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(r.mmr, $3),
				coalesce(us.games_played, 0),
				coalesce(us.wins, 0),
				coalesce(rs.games_played, 0),
				coalesce(u.account_type = 'guest', false),
				coalesce(u.is_admin, false),
				coalesce(u.is_moderator, false),
				coalesce(u.banned_at is not null, false),
				coalesce(u.ban_reason, ''),
			u.banned_at,
			coalesce(latest_session.ip_address, ''),
			rep.muted_until
		from users u
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = u.id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		left join lateral (
			select ip_address
			from auth_sessions
			where user_id = u.id and coalesce(ip_address, '') <> ''
			order by last_used_at desc, created_at desc
			limit 1
		) latest_session on true
		left join ranks r on r.user_id = u.id and r.mode = $1 and r.season_id = $2
		left join user_stats us on us.user_id = u.id
		left join ranked_stats rs on rs.user_id = u.id and rs.mode = $1 and rs.season_id = $2
		left join moderation_reporter_reputation rep on rep.user_id = u.id
		where $4 = '%%'
		   or lower(u.id) like $4
		   or lower(coalesce(u.email, '')) like $4
		   or lower(coalesce(u.display_name, ui.provider_name, '')) like $4
		   or exists (
			select 1
			from user_identity_history ih
			where ih.user_id = u.id
			  and (
				lower(ih.provider) like $4
				or lower(ih.provider_user_id) like $4
				or lower(coalesce(ih.email, '')) like $4
				or lower(coalesce(ih.provider_name, '')) like $4
			  )
		   )
		order by u.created_at desc, u.id desc
		limit $5
	`, modeDuel, seasonID, initialMMR, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AdminPlayerSummary, 0, limit)
	for rows.Next() {
		var item AdminPlayerSummary
		var bannedAt *time.Time
		var reportMutedUntil *time.Time
		if err := rows.Scan(
			&item.UserID,
			&item.Email,
			&item.DisplayName,
			&item.AvatarURL,
			&item.MMR,
			&item.GamesPlayed,
			&item.Wins,
			&item.RankedGamesPlayed,
			&item.IsGuest,
			&item.IsAdmin,
			&item.IsModerator,
			&item.IsBanned,
			&item.BanReason,
			&bannedAt,
			&item.LastIPAddress,
			&reportMutedUntil,
		); err != nil {
			return nil, err
		}
		if bannedAt != nil {
			item.BannedAt = *bannedAt
		}
		if reportMutedUntil != nil {
			item.ReportMutedUntil = *reportMutedUntil
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.populateAdminPlayerIdentities(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *pgStore) getAdminPlayerSummary(ctx context.Context, userID string) (AdminPlayerSummary, error) {
	var item AdminPlayerSummary
	var bannedAt *time.Time
	var reportMutedUntil *time.Time
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		select
			u.id,
			coalesce(u.email, ''),
			coalesce(nullif(u.display_name, ''), ui.provider_name, u.id),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(r.mmr, $4),
			coalesce(us.games_played, 0),
			coalesce(us.wins, 0),
			coalesce(rs.games_played, 0),
			coalesce(u.account_type = 'guest', false),
			coalesce(u.is_admin, false),
			coalesce(u.is_moderator, false),
			coalesce(u.banned_at is not null, false),
			coalesce(u.ban_reason, ''),
			u.banned_at,
			coalesce(latest_session.ip_address, ''),
			rep.muted_until
		from users u
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = u.id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		left join lateral (
			select ip_address
			from auth_sessions
			where user_id = u.id and coalesce(ip_address, '') <> ''
			order by last_used_at desc, created_at desc
			limit 1
		) latest_session on true
		left join ranks r on r.user_id = u.id and r.mode = $2 and r.season_id = $3
		left join user_stats us on us.user_id = u.id
		left join ranked_stats rs on rs.user_id = u.id and rs.mode = $2 and rs.season_id = $3
		left join moderation_reporter_reputation rep on rep.user_id = u.id
		where u.id = $1
	`, userID, modeDuel, seasonID, initialMMR).Scan(
		&item.UserID,
		&item.Email,
		&item.DisplayName,
		&item.AvatarURL,
		&item.MMR,
		&item.GamesPlayed,
		&item.Wins,
		&item.RankedGamesPlayed,
		&item.IsGuest,
		&item.IsAdmin,
		&item.IsModerator,
		&item.IsBanned,
		&item.BanReason,
		&bannedAt,
		&item.LastIPAddress,
		&reportMutedUntil,
	)
	if err != nil {
		return AdminPlayerSummary{}, err
	}
	if bannedAt != nil {
		item.BannedAt = *bannedAt
	}
	if reportMutedUntil != nil {
		item.ReportMutedUntil = *reportMutedUntil
	}
	items := []AdminPlayerSummary{item}
	if err := s.populateAdminPlayerIdentities(ctx, items); err != nil {
		return AdminPlayerSummary{}, err
	}
	return items[0], nil
}

func (s *pgStore) GetAdminPlayerDetail(userID string) (AdminPlayerDetail, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AdminPlayerDetail{}, errors.New("userID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	player, err := s.getAdminPlayerSummary(ctx, userID)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	stats, err := s.adminPlayerStats(ctx, userID)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	eloHistory, err := s.adminPlayerEloHistory(ctx, userID, 7)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	matches, err := s.ListPlayerMatchHistory(userID, 25)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	return AdminPlayerDetail{
		Player:     player,
		Stats:      stats,
		EloHistory: eloHistory,
		Matches:    matches,
	}, nil
}

func (s *pgStore) adminPlayerStats(ctx context.Context, userID string) (AdminPlayerStats, error) {
	var stats AdminPlayerStats
	err := s.pool.QueryRow(ctx, `
		select
			count(*)::int,
			count(*) filter (where h.ranked)::int,
			count(*) filter (where h.mode = $2)::int,
			count(*) filter (where h.mode = 'singleplayer')::int,
			count(*) filter (where h.winner_user_id = $1)::int,
			count(*) filter (where h.mode = $2 and nullif(h.winner_user_id, '') is not null and h.winner_user_id <> $1)::int
		from match_history h
		join match_players p on p.match_id = h.match_id
		where p.user_id = $1
	`, userID, modeDuel).Scan(
		&stats.TotalMatches,
		&stats.RankedMatches,
		&stats.DuelMatches,
		&stats.SingleplayerRuns,
		&stats.Wins,
		&stats.Losses,
	)
	return stats, err
}

func (s *pgStore) adminPlayerEloHistory(ctx context.Context, userID string, days int) ([]AdminPlayerEloPoint, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)
	rows, err := s.pool.Query(ctx, `
		with ranked_matches as (
			select
				h.ended_at,
				coalesce(
					p.rating_after,
					p.rating_before + case
						when h.winner_user_id = $1 then nullif(h.replay_json->'ratingPreview'->($1::text)->>'win', '')::int
						when nullif(h.winner_user_id, '') is null then nullif(h.replay_json->'ratingPreview'->($1::text)->>'draw', '')::int
						else nullif(h.replay_json->'ratingPreview'->($1::text)->>'lose', '')::int
					end,
					p.mmr + case
						when h.winner_user_id = $1 then nullif(h.replay_json->'ratingPreview'->($1::text)->>'win', '')::int
						when nullif(h.winner_user_id, '') is null then nullif(h.replay_json->'ratingPreview'->($1::text)->>'draw', '')::int
						else nullif(h.replay_json->'ratingPreview'->($1::text)->>'lose', '')::int
					end,
					p.rating_before,
					p.mmr
				)::int as rating_after,
				coalesce(
					p.final_ranked_delta,
					case
						when h.winner_user_id = $1 then nullif(h.replay_json->'ratingPreview'->($1::text)->>'win', '')::int
						when nullif(h.winner_user_id, '') is null then nullif(h.replay_json->'ratingPreview'->($1::text)->>'draw', '')::int
						else nullif(h.replay_json->'ratingPreview'->($1::text)->>'lose', '')::int
					end,
					0
				)::int as delta
			from match_history h
			join match_players p on p.match_id = h.match_id
			where p.user_id = $1
			  and h.mode = $2
			  and h.ranked
			  and h.ended_at >= $3
		),
		latest_per_day as (
			select distinct on (date_trunc('day', ended_at))
				date_trunc('day', ended_at) as day,
				rating_after,
				sum(delta) over (partition by date_trunc('day', ended_at))::int as delta,
				count(*) over (partition by date_trunc('day', ended_at))::int as played,
				ended_at
			from ranked_matches
			order by date_trunc('day', ended_at), ended_at desc
		)
		select day, rating_after, delta, played
		from latest_per_day
		order by day asc
	`, userID, modeDuel, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminPlayerEloPoint{}
	for rows.Next() {
		var point AdminPlayerEloPoint
		if err := rows.Scan(&point.Date, &point.MMR, &point.Delta, &point.Played); err != nil {
			return nil, err
		}
		out = append(out, point)
	}
	return out, rows.Err()
}

func (s *pgStore) populateAdminPlayerIdentities(ctx context.Context, players []AdminPlayerSummary) error {
	if len(players) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(players))
	byUserID := make(map[string]int, len(players))
	for i := range players {
		userIDs = append(userIDs, players[i].UserID)
		byUserID[players[i].UserID] = i
	}
	rows, err := s.pool.Query(ctx, `
		select
			user_id,
			provider,
			provider_user_id,
			coalesce(email, ''),
			coalesce(provider_name, ''),
			last_seen_at,
			deleted_at
		from user_identity_history
		where user_id = any($1)
		order by user_id, provider, deleted_at nulls first, last_seen_at desc
	`, userIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var identity contracts.AdminUserIdentity
		var deletedAt *time.Time
		if err := rows.Scan(
			&userID,
			&identity.Provider,
			&identity.ProviderUserID,
			&identity.Email,
			&identity.ProviderName,
			&identity.LastSeenAt,
			&deletedAt,
		); err != nil {
			return err
		}
		if deletedAt != nil {
			identity.DeletedAt = *deletedAt
		}
		if idx, ok := byUserID[userID]; ok {
			players[idx].Identities = append(players[idx].Identities, identity)
		}
	}
	return rows.Err()
}

func (s *pgStore) SetPlayerBan(userID, reason string, banned bool) error {
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var bannedAt any
	var banReason any
	if banned {
		bannedAt = time.Now()
		if strings.TrimSpace(reason) != "" {
			banReason = strings.TrimSpace(reason)
		}
	}
	tag, err := tx.Exec(ctx, `
		update users
		set banned_at = $2,
			ban_reason = $3
		where id = $1
	`, userID, bannedAt, banReason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if banned {
		if err := banUserOAuthIdentities(ctx, tx, userID, strings.TrimSpace(reason), ""); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into enforcement_actions(target_user_id, action_type, reason_code, reason_note)
			values($1, 'ban', 'manual', nullif($2, ''))
		`, userID, strings.TrimSpace(reason)); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			update oauth_identity_bans
			set revoked_at = coalesce(revoked_at, now())
			where banned_user_id = $1
			  and revoked_at is null
		`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into enforcement_actions(target_user_id, action_type, reason_code, reason_note)
			values($1, 'unban', 'manual', nullif($2, ''))
		`, userID, strings.TrimSpace(reason)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func banUserOAuthIdentities(ctx context.Context, tx pgx.Tx, userID, reason, actorUserID string) error {
	reason = strings.TrimSpace(reason)
	actorUserID = strings.TrimSpace(actorUserID)
	_, err := tx.Exec(ctx, `
		insert into oauth_identity_bans(provider, provider_user_id, banned_user_id, reason, created_by, created_at, revoked_at)
		select provider, provider_user_id, $1, nullif($2, ''), nullif($3, ''), now(), null
		from (
			select provider, provider_user_id
			from user_identity_history
			where user_id = $1
			union
			select provider, provider_user_id
			from user_identities
			where user_id = $1
		) identities
		on conflict (provider, provider_user_id) do update set
			banned_user_id = excluded.banned_user_id,
			reason = excluded.reason,
			created_by = excluded.created_by,
			created_at = now(),
			revoked_at = null
	`, userID, reason, actorUserID)
	return err
}

func (s *pgStore) ClearReporterMute(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `
		update moderation_reporter_reputation
		set muted_until = null,
			report_weight = greatest(report_weight, 0.05),
			updated_at = now()
		where user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, `
			insert into moderation_reporter_reputation(user_id, muted_until, report_weight, updated_at)
			values($1, null, 1, now())
		`, userID)
	}
	return err
}

func (s *pgStore) GetLobbyChangelog(defaultContent LobbyChangelogContent) (LobbyChangelogContent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	posts, err := s.ListChangelogPosts(false)
	if err == nil && len(posts) > 0 {
		post := posts[0]
		preview := strings.TrimSpace(post.Summary)
		if preview == "" {
			preview = post.Markdown
		}
		return LobbyChangelogContent{
			Eyebrow:   "Latest News",
			Title:     post.Title,
			Markdown:  preview,
			Slug:      post.Slug,
			UpdatedAt: post.UpdatedAt,
		}, nil
	}
	var raw string
	err = s.pool.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'lobby_changelog'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultContent, nil
		}
		return LobbyChangelogContent{}, err
	}
	var content LobbyChangelogContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return defaultContent, nil
	}
	if strings.TrimSpace(content.Eyebrow) == "" {
		content.Eyebrow = defaultContent.Eyebrow
	}
	if strings.TrimSpace(content.Title) == "" {
		content.Title = defaultContent.Title
	}
	if strings.TrimSpace(content.Markdown) == "" {
		content.Markdown = defaultContent.Markdown
	}
	return content, nil
}

func (s *pgStore) SetLobbyChangelog(content LobbyChangelogContent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	payload, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('lobby_changelog', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload))
	return err
}

func (s *pgStore) ListChangelogPosts(includeUnpublished bool) ([]ChangelogPost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	query := `
		select id, slug, title, summary, markdown, published, created_at, updated_at
		from changelog_posts
		where ($1::boolean or published = true)
		order by updated_at desc, id desc
	`
	rows, err := s.pool.Query(ctx, query, includeUnpublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []ChangelogPost
	for rows.Next() {
		var post ChangelogPost
		if err := rows.Scan(
			&post.ID,
			&post.Slug,
			&post.Title,
			&post.Summary,
			&post.Markdown,
			&post.Published,
			&post.CreatedAt,
			&post.UpdatedAt,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *pgStore) GetChangelogPostBySlug(slug string, publishedOnly bool) (ChangelogPost, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		select id, slug, title, summary, markdown, published, created_at, updated_at
		from changelog_posts
		where slug = $1 and ($2::boolean = false or published = true)
	`, slug, publishedOnly).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChangelogPost{}, false, nil
		}
		return ChangelogPost{}, false, err
	}
	return post, true, nil
}

func (s *pgStore) CreateChangelogPost(input ChangelogPostInput) (ChangelogPost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		insert into changelog_posts(slug, title, summary, markdown, published, updated_at)
		values($1, $2, $3, $4, $5, now())
		returning id, slug, title, summary, markdown, published, created_at, updated_at
	`, input.Slug, input.Title, input.Summary, input.Markdown, input.Published).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	return post, err
}

func (s *pgStore) UpdateChangelogPost(id int64, input ChangelogPostInput) (ChangelogPost, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		update changelog_posts
		set slug = $2,
			title = $3,
			summary = $4,
			markdown = $5,
			published = $6,
			updated_at = now()
		where id = $1
		returning id, slug, title, summary, markdown, published, created_at, updated_at
	`, id, input.Slug, input.Title, input.Summary, input.Markdown, input.Published).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChangelogPost{}, false, nil
		}
		return ChangelogPost{}, false, err
	}
	return post, true, nil
}

func (s *pgStore) GetModerationSettings() (ModerationSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var raw string
	err := s.pool.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'moderation_settings'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationSettings{}, nil
		}
		return ModerationSettings{}, err
	}
	var settings ModerationSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return ModerationSettings{}, nil
	}
	settings.DiscordWebhookURL = strings.TrimSpace(settings.DiscordWebhookURL)
	return settings, nil
}

func (s *pgStore) SetModerationSettings(settings ModerationSettings) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	settings.DiscordWebhookURL = strings.TrimSpace(settings.DiscordWebhookURL)
	payload, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('moderation_settings', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload))
	return err
}

func (s *pgStore) GetRankedSeasonSettings() (RankedSeasonSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return RankedSeasonSettings{}, err
	}
	return RankedSeasonSettings{ActiveSeasonID: seasonID}, nil
}

func (s *pgStore) RolloverRankedSeason(nextSeasonID string) (RankedSeasonRolloverResult, error) {
	nextSeasonID = strings.TrimSpace(nextSeasonID)
	if nextSeasonID == "" {
		return RankedSeasonRolloverResult{}, errors.New("season id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	defer tx.Rollback(ctx)
	previousSeasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if previousSeasonID == nextSeasonID {
		return RankedSeasonRolloverResult{}, errors.New("season is already active")
	}
	badgeTag, err := tx.Exec(ctx, `
		with ranked as (
			select
				r.user_id,
				row_number() over (order by r.mmr desc, r.updated_at asc, r.user_id asc)::int as rank
			from ranks r
			join users u on u.id = r.user_id
			where r.mode = $1
				and r.season_id = $2
				and coalesce(u.account_type, 'registered') <> 'guest'
				and u.banned_at is null
		)
		insert into user_badges(user_id, badge_code, badge_season_id, rank)
		select
			user_id,
			$3,
			$2,
			rank
		from ranked
		where rank between 1 and 100
		on conflict (user_id, badge_code, badge_season_id) do nothing
	`, modeDuel, previousSeasonID, badgeCodeSeasonRank)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	seedTag, err := tx.Exec(ctx, `
		insert into ranks(user_id, mode, season_id, mmr, rd)
		select u.id, $1, $2, $3, $4
		from users u
		where coalesce(u.account_type, 'registered') <> 'guest'
		on conflict (user_id, mode, season_id) do nothing
	`, modeDuel, nextSeasonID, initialMMR, initialRatingRD)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranked_stats(user_id, mode, season_id, games_played, wins)
		select u.id, $1, $2, 0, 0
		from users u
		where coalesce(u.account_type, 'registered') <> 'guest'
		on conflict (user_id, mode, season_id) do nothing
	`, modeDuel, nextSeasonID); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	settings := RankedSeasonSettings{ActiveSeasonID: nextSeasonID}
	payload, err := json.Marshal(settings)
	if err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('ranked_season', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload)); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RankedSeasonRolloverResult{}, err
	}
	return RankedSeasonRolloverResult{
		PreviousSeasonID: previousSeasonID,
		ActiveSeasonID:   nextSeasonID,
		BadgesAwarded:    int(badgeTag.RowsAffected()),
		PlayersSeeded:    int(seedTag.RowsAffected()),
	}, nil
}

func (s *pgStore) activeSeasonID(ctx context.Context) (string, error) {
	return activeSeasonIDTx(ctx, s.pool)
}

type seasonQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func activeSeasonIDTx(ctx context.Context, q seasonQuerier) (string, error) {
	var raw string
	err := q.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'ranked_season'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSeasonID, nil
		}
		return "", err
	}
	var settings RankedSeasonSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaultSeasonID, nil
	}
	seasonID := strings.TrimSpace(settings.ActiveSeasonID)
	if seasonID == "" {
		return defaultSeasonID, nil
	}
	return seasonID, nil
}

func (s *pgStore) ActivateMapRevision(mapKey, displayName string, dataset []byte) (MapRevisionSummary, error) {
	if strings.TrimSpace(mapKey) == "" {
		return MapRevisionSummary{}, errors.New("map key required")
	}
	rows, err := parseMapRows(dataset)
	if err != nil {
		return MapRevisionSummary{}, err
	}
	if len(rows) == 0 {
		return MapRevisionSummary{}, errors.New("no valid rows")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = mapKey
	}
	sum := sha1.Sum(dataset)
	contentHash := hex.EncodeToString(sum[:])
	revisionID := mapKey + "-" + contentHash[:12]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MapRevisionSummary{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		insert into maps(map_key, display_name)
		values($1, $2)
		on conflict (map_key) do update set
			display_name = excluded.display_name
	`, mapKey, displayName); err != nil {
		return MapRevisionSummary{}, err
	}

	inserted := true
	var existing string
	err = tx.QueryRow(ctx, `select id from map_revisions where map_key = $1 and content_hash = $2 limit 1`, mapKey, contentHash).Scan(&existing)
	if err == nil {
		revisionID = existing
		inserted = false
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return MapRevisionSummary{}, err
	} else {
		if _, err := tx.Exec(ctx, `
			insert into map_revisions(id, map_key, content_hash, status, row_count)
			values($1, $2, $3, 'validated', 0)
		`, revisionID, mapKey, contentHash); err != nil {
			return MapRevisionSummary{}, err
		}
	}

	if inserted {
		block := make([][]any, 0, len(rows))
		for _, r := range rows {
			block = append(block, []any{revisionID, r.Lat, r.Lng, r.Country, r.PanoID, r.Heading, r.Pitch, r.RandKey})
		}
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"locations"},
			[]string{"map_revision_id", "lat", "lng", "country", "pano_id", "heading", "pitch", "rand_key"},
			pgx.CopyFromRows(block),
		); err != nil {
			return MapRevisionSummary{}, err
		}
	}

	if _, err := tx.Exec(ctx, `update map_revisions set row_count = $2, status = 'active' where id = $1`, revisionID, len(rows)); err != nil {
		return MapRevisionSummary{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into map_aliases(map_key, active_revision_id, updated_at)
		values($1, $2, now())
		on conflict (map_key) do update set
			rollback_revision_id = map_aliases.active_revision_id,
			active_revision_id = excluded.active_revision_id,
			updated_at = now()
	`, mapKey, revisionID); err != nil {
		return MapRevisionSummary{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MapRevisionSummary{}, err
	}
	return MapRevisionSummary{
		MapKey:      mapKey,
		RevisionID:  revisionID,
		RowCount:    len(rows),
		Inserted:    inserted,
		DisplayName: displayName,
	}, nil
}

func (s *pgStore) CreateAuthSession(userID, refreshTokenHash string, expiresAt time.Time, params AuthSessionParams) (RefreshTokenRecord, error) {
	if userID == "" || refreshTokenHash == "" {
		return RefreshTokenRecord{}, errors.New("userID and refresh token hash required")
	}
	sessionID := newUserID()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RefreshTokenRecord{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		insert into auth_sessions(
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			created_at,
			last_used_at,
			user_agent,
			ip_address
		)
		values($1, $2, $3, $4, now(), now(), $5, $6)
		returning
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			created_at,
			last_used_at,
			revoked_at,
			coalesce(user_agent, ''),
			coalesce(ip_address, '')
	`, sessionID, userID, refreshTokenHash, expiresAt, nullable(params.UserAgent), nullable(params.IPAddress))
	var rec RefreshTokenRecord
	if err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.RefreshTokenHash,
		&rec.ExpiresAt,
		&rec.CreatedAt,
		&rec.LastUsedAt,
		&rec.RevokedAt,
		&rec.UserAgent,
		&rec.IPAddress,
	); err != nil {
		return RefreshTokenRecord{}, err
	}
	if strings.TrimSpace(params.IPAddress) != "" {
		if _, err := tx.Exec(ctx, `
			update users
			set registration_ip_address = coalesce(nullif(trim(registration_ip_address), ''), $2)
			where id = $1
		`, userID, strings.TrimSpace(params.IPAddress)); err != nil {
			return RefreshTokenRecord{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return RefreshTokenRecord{}, err
	}
	return rec, nil
}

func (s *pgStore) GetAuthSessionByRefreshToken(hash string) (RefreshTokenRecord, bool, error) {
	if hash == "" {
		return RefreshTokenRecord{}, false, errors.New("hash required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			created_at,
			last_used_at,
			revoked_at,
			coalesce(user_agent, ''),
			coalesce(ip_address, '')
		from auth_sessions
		where refresh_token_hash = $1
	`, hash)
	var rec RefreshTokenRecord
	if err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.RefreshTokenHash,
		&rec.ExpiresAt,
		&rec.CreatedAt,
		&rec.LastUsedAt,
		&rec.RevokedAt,
		&rec.UserAgent,
		&rec.IPAddress,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshTokenRecord{}, false, nil
		}
		return RefreshTokenRecord{}, false, err
	}
	return rec, true, nil
}

func (s *pgStore) RotateAuthSession(sessionID, currentHash, nextHash string, expiresAt time.Time, usedAt time.Time) (RefreshTokenRecord, bool, error) {
	if sessionID == "" || currentHash == "" || nextHash == "" {
		return RefreshTokenRecord{}, false, errors.New("session id and token hashes required")
	}
	if usedAt.IsZero() {
		usedAt = time.Now()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		update auth_sessions
		set refresh_token_hash = $3,
			expires_at = $4,
			last_used_at = $5
		where id = $1
		  and refresh_token_hash = $2
		  and revoked_at is null
		returning
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			created_at,
			last_used_at,
			revoked_at,
			coalesce(user_agent, ''),
			coalesce(ip_address, '')
	`, sessionID, currentHash, nextHash, expiresAt, usedAt)
	var rec RefreshTokenRecord
	if err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.RefreshTokenHash,
		&rec.ExpiresAt,
		&rec.CreatedAt,
		&rec.LastUsedAt,
		&rec.RevokedAt,
		&rec.UserAgent,
		&rec.IPAddress,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshTokenRecord{}, false, nil
		}
		return RefreshTokenRecord{}, false, err
	}
	return rec, true, nil
}

func (s *pgStore) RevokeAuthSession(sessionID string) error {
	if sessionID == "" {
		return errors.New("session id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update auth_sessions
		set revoked_at = coalesce(revoked_at, now())
		where id = $1
	`, sessionID)
	return err
}

func (s *pgStore) RevokeAuthSessionsForUser(userID string) error {
	if userID == "" {
		return errors.New("userID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update auth_sessions
		set revoked_at = coalesce(revoked_at, now())
		where user_id = $1 and revoked_at is null
	`, userID)
	return err
}

func (s *pgStore) DeleteAccount(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("userID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var isBanned bool
	var banReason string
	if err := tx.QueryRow(ctx, `
		select banned_at is not null, coalesce(ban_reason, '')
		from users
		where id = $1
	`, userID).Scan(&isBanned, &banReason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}
	if isBanned {
		if strings.TrimSpace(banReason) == "" {
			banReason = "account deleted while banned"
		}
		if err := banUserOAuthIdentities(ctx, tx, userID, banReason, ""); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		update auth_sessions
		set revoked_at = coalesce(revoked_at, now())
		where user_id = $1 and revoked_at is null
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into user_identity_history(user_id, provider, provider_user_id, email, provider_name, avatar_url, first_seen_at, last_seen_at, deleted_at)
		select user_id, provider, provider_user_id, email, provider_name, avatar_url, created_at, last_seen_at, now()
		from user_identities
		where user_id = $1
		on conflict (user_id, provider, provider_user_id) do update set
			email = excluded.email,
			provider_name = excluded.provider_name,
			avatar_url = case
				when excluded.avatar_url is null then user_identity_history.avatar_url
				when excluded.avatar_url = '' then user_identity_history.avatar_url
				else excluded.avatar_url
			end,
			last_seen_at = greatest(user_identity_history.last_seen_at, excluded.last_seen_at),
			deleted_at = coalesce(user_identity_history.deleted_at, excluded.deleted_at)
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		delete from user_identities
		where user_id = $1
	`, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		update users
		set email = null,
			display_name = 'Deleted player',
			avatar_url = null,
			onboarded_at = null,
			account_type = 'guest',
			is_admin = false,
			is_moderator = false,
			deleted_at = coalesce(deleted_at, now())
		where id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return tx.Commit(ctx)
}

func (s *pgStore) DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error) {
	if ttl <= 0 || limit <= 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		with batch as materialized (
			select id
			from users
			where account_type = 'guest'
			  and deleted_at is null
			  and created_at < now() - ($1::double precision * interval '1 second')
			order by created_at asc
			limit $2
		),
		del_ranked as (
			delete from ranked_stats
			using batch
			where ranked_stats.user_id = batch.id
		),
		del_ranks as (
			delete from ranks
			using batch
			where ranks.user_id = batch.id
		),
		del_stats as (
			delete from user_stats
			using batch
			where user_stats.user_id = batch.id
		)
		delete from users
		using batch
		where users.id = batch.id
		returning users.id
	`, ttl.Seconds(), limit)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		deleted++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *pgStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *pgStore) UpsertUser(userID, email, displayName string) error {
	if userID == "" {
		return errors.New("user id required")
	}
	if displayName == "" {
		displayName = userID
	}
	var nullableEmail any
	if email != "" {
		nullableEmail = email
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	seasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		insert into users (id, email, display_name, avatar_url, onboarded_at, account_type)
		values ($1, $2, $3, null, now(), 'guest')
		on conflict (id) do update set
			email = excluded.email,
			display_name = excluded.display_name,
			onboarded_at = coalesce(users.onboarded_at, excluded.onboarded_at)
	`, userID, nullableEmail, displayName); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranks (user_id, mode, mmr, season_id)
		values ($1, $2, $4, $3)
		on conflict (user_id, mode, season_id) do nothing
	`, userID, modeDuel, seasonID, initialMMR); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into user_stats (user_id, games_played, wins)
		values ($1, 0, 0)
		on conflict (user_id) do nothing
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into ranked_stats (user_id, mode, season_id, games_played, wins)
		values ($1, $2, $3, 0, 0)
		on conflict (user_id, mode, season_id) do nothing
	`, userID, modeDuel, seasonID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *pgStore) GetProfile(userID string) (Profile, error) {
	p := Profile{UserID: userID, DisplayName: userID, MMR: initialMMR, RatingRD: initialRatingRD}
	if userID == "" {
		return p, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	seasonID, err := s.activeSeasonID(ctx)
	if err != nil {
		return p, err
	}
	p.SeasonID = seasonID
	row := s.pool.QueryRow(ctx, `
		select
			coalesce(nullif(u.display_name, seed.user_id), ui.provider_name, $1) as display_name,
			coalesce(u.avatar_url, ui.avatar_url, '') as avatar_url,
			coalesce(r.mmr, $4) as mmr,
			coalesce(r.rd, $5) as rating_rd,
			coalesce(us.games_played, 0) as games_played,
			coalesce(us.wins, 0) as wins,
			coalesce(rs.games_played, 0) as ranked_games_played,
				coalesce(rs.wins, 0) as ranked_wins,
				coalesce(u.account_type = 'guest', false) as is_guest,
				coalesce(u.is_admin, false) as is_admin,
				coalesce(u.is_moderator, false) as is_moderator,
				coalesce(u.banned_at is not null, false) as is_banned,
				coalesce(u.ban_reason, '') as ban_reason,
				coalesce(u.selected_badge_code, 0) as selected_badge_code,
				coalesce(u.selected_badge_season_id, '') as selected_badge_season_id
		from (select $1 as user_id) seed
		left join users u on u.id = seed.user_id
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = seed.user_id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		left join ranks r on r.user_id = seed.user_id and r.mode = $2 and r.season_id = $3
		left join user_stats us on us.user_id = seed.user_id
		left join ranked_stats rs on rs.user_id = seed.user_id and rs.mode = $2 and rs.season_id = $3
	`, userID, modeDuel, seasonID, initialMMR, initialRatingRD)
	var selectedBadgeCode int16
	var selectedBadgeSeasonID string
	if err := row.Scan(
		&p.DisplayName,
		&p.AvatarURL,
		&p.MMR,
		&p.RatingRD,
		&p.GamesPlayed,
		&p.Wins,
		&p.RankedGamesPlayed,
		&p.RankedWins,
		&p.IsGuest,
		&p.IsAdmin,
		&p.IsModerator,
		&p.IsBanned,
		&p.BanReason,
		&selectedBadgeCode,
		&selectedBadgeSeasonID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p, nil
		}
		return p, err
	}
	badges, selected, err := s.profileBadges(ctx, userID, badgeIDFromParts(selectedBadgeCode, selectedBadgeSeasonID))
	if err != nil {
		return p, err
	}
	p.Badges = badges
	p.SelectedBadge = selected
	return p, nil
}

func (s *pgStore) UpdateSelectedBadge(userID, badgeID string) (Profile, error) {
	userID = strings.TrimSpace(userID)
	badgeID = strings.TrimSpace(badgeID)
	if userID == "" {
		return Profile{}, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	badges, _, err := s.profileBadges(ctx, userID, badgeID)
	if err != nil {
		return Profile{}, err
	}
	if badgeID != "" {
		owned := false
		for _, badge := range badges {
			if badge.ID == badgeID && badge.Owned {
				owned = true
				break
			}
		}
		if !owned {
			return Profile{}, errors.New("badge unavailable")
		}
	}
	ref, ok := badgeRefFromID(badgeID)
	if badgeID != "" && !ok {
		return Profile{}, errors.New("badge unavailable")
	}
	if _, err := s.pool.Exec(ctx, `
		update users
		set selected_badge_code = nullif($2, 0),
			selected_badge_season_id = $3
		where id = $1
	`, userID, ref.Code, ref.SeasonID); err != nil {
		return Profile{}, err
	}
	return s.GetProfile(userID)
}

func (s *pgStore) profileBadges(ctx context.Context, userID, selectedBadgeID string) ([]contracts.PlayerBadge, *contracts.PlayerBadge, error) {
	badges := []contracts.PlayerBadge{}
	rows, err := s.pool.Query(ctx, `
		select ub.badge_code, coalesce(ub.badge_season_id, ''), coalesce(ub.rank, 0)
		from user_badges ub
		where ub.user_id = $1
		order by ub.awarded_at desc, ub.badge_code asc, ub.badge_season_id asc
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	owned := map[string]bool{}
	for rows.Next() {
		var code int16
		var seasonID string
		var rank int
		if err := rows.Scan(&code, &seasonID, &rank); err != nil {
			return nil, nil, err
		}
		badge, ok := badgeFromParts(code, seasonID, rank, true)
		if !ok {
			continue
		}
		owned[badge.ID] = true
		badges = append(badges, badge)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for _, badge := range badgeTemplates() {
		if !owned[badge.ID] {
			badges = append(badges, badge)
		}
	}
	var selected *contracts.PlayerBadge
	for i := range badges {
		if badges[i].ID == selectedBadgeID && badges[i].Owned {
			selected = &badges[i]
			break
		}
	}
	if selected == nil {
		for i := range badges {
			if badges[i].Owned {
				selected = &badges[i]
				break
			}
		}
	}
	return badges, selected, nil
}

type badgeDefinition struct {
	ID           string
	Code         int16
	Kind         string
	Label        string
	Description  string
	ImageURL     string
	Rarity       string
	Unobtainable bool
}

var badgeDefinitions = []badgeDefinition{
	{
		ID:           "discord-member",
		Code:         badgeCodeDiscordMember,
		Kind:         "community",
		Label:        "Discord Member",
		Description:  "Retired badge previously awarded for linking Discord to your GeoDuels account.",
		ImageURL:     "/medals/discord-medal.v1.png",
		Rarity:       "common",
		Unobtainable: true,
	},
	{
		ID:          "geoduels-team",
		Code:        badgeCodeGeoDuelsTeam,
		Kind:        "special",
		Label:       "GeoDuels Team",
		Description: "An exclusive medal for GeoDuels moderators and team members.",
		ImageURL:    "/medals/team-badge.v1.png",
		Rarity:      "special",
	},
	{
		ID:          "discord-server-member",
		Code:        badgeCodeDiscordServerMember,
		Kind:        "community",
		Label:       "Discord Server Member",
		Description: "Awarded for joining the official GeoDuels Discord server.",
		ImageURL:    "/medals/discord-new-badge.v1.png",
		Rarity:      "common",
	},
	{
		ID:          "supporter",
		Code:        badgeCodeSupporter,
		Kind:        "supporter",
		Label:       "Supporter",
		Description: "Awarded for supporting GeoDuels.",
		ImageURL:    "/medals/supporter-badge.v2.png",
		Rarity:      "rare",
	},
	{
		ID:          "speedrunner",
		Code:        badgeCodeSpeedrunner,
		Kind:        "achievement",
		Label:       "Speedrunner",
		Description: "Awarded for scoring 5000 points in under 30 seconds in ranked.",
		ImageURL:    "/medals/speedrunner-badge.v2.png",
		Rarity:      "epic",
	},
	{
		ID:          "elo-1000",
		Code:        badgeCodeElo1000,
		Kind:        "ranked",
		Label:       "1000 Elo",
		Description: "Awarded for reaching 1000 Elo.",
		ImageURL:    "/medals/1k-medal.v1.png",
		Rarity:      "common",
	},
	{
		ID:          "elo-1500",
		Code:        badgeCodeElo1500,
		Kind:        "ranked",
		Label:       "1500 Elo",
		Description: "Awarded for reaching 1500 Elo.",
		ImageURL:    "/medals/1.5k-medal.v1.png",
		Rarity:      "rare",
	},
	{
		ID:          "elo-2000",
		Code:        badgeCodeElo2000,
		Kind:        "ranked",
		Label:       "2000 Elo",
		Description: "Awarded for reaching 2000 Elo.",
		ImageURL:    "/medals/2k-medal.v1.png",
		Rarity:      "legendary",
	},
}

func badgeDefinitionByID(id string) (badgeDefinition, bool) {
	id = strings.TrimSpace(id)
	for _, def := range badgeDefinitions {
		if def.ID == id {
			return def, true
		}
	}
	return badgeDefinition{}, false
}

func badgeDefinitionByCode(code int16) (badgeDefinition, bool) {
	for _, def := range badgeDefinitions {
		if def.Code == code {
			return def, true
		}
	}
	return badgeDefinition{}, false
}

func badgeTemplates() []contracts.PlayerBadge {
	out := []contracts.PlayerBadge{}
	for _, def := range badgeDefinitions {
		if def.Unobtainable {
			continue
		}
		out = append(out, badgeFromDefinition(def, false))
	}
	out = append(out, seasonRankBadgeTemplate("s2"), seasonRankBadgeTemplate("s2.5"))
	return out
}

func seasonRankBadgeTemplate(seasonID string) contracts.PlayerBadge {
	displaySeason := seasonBadgeDisplayName(seasonID)
	return contracts.PlayerBadge{
		ID:          seasonRankBadgeID(seasonID),
		Kind:        "season_rank",
		Label:       displaySeason + " Top 100",
		Description: "Awarded to players who finish in the top 100 when " + displaySeason + " ends.",
		ImageURL:    "/medals/platinum-medal.v1.png",
		Rarity:      "legendary",
		SeasonID:    seasonID,
		Owned:       false,
	}
}

func seasonRankBadgeID(seasonID string) string {
	return "season-" + strings.TrimSpace(seasonID) + "-top-100"
}

type badgeRef struct {
	Code     int16
	SeasonID string
}

func badgeRefFromID(id string) (badgeRef, bool) {
	id = strings.TrimSpace(id)
	switch id {
	case "":
		return badgeRef{}, true
	default:
		if def, ok := badgeDefinitionByID(id); ok {
			return badgeRef{Code: def.Code}, true
		}
		if strings.HasPrefix(id, "season-") && strings.HasSuffix(id, "-top-100") {
			seasonID := strings.TrimSuffix(strings.TrimPrefix(id, "season-"), "-top-100")
			if strings.TrimSpace(seasonID) == "" {
				return badgeRef{}, false
			}
			return badgeRef{Code: badgeCodeSeasonRank, SeasonID: seasonID}, true
		}
		return badgeRef{}, false
	}
}

func badgeIDFromParts(code int16, seasonID string) string {
	if code == badgeCodeSeasonRank {
		if strings.TrimSpace(seasonID) != "" {
			return seasonRankBadgeID(seasonID)
		}
		return ""
	}
	if def, ok := badgeDefinitionByCode(code); ok {
		return def.ID
	}
	return ""
}

func badgeFromDefinition(def badgeDefinition, owned bool) contracts.PlayerBadge {
	return contracts.PlayerBadge{
		ID:           def.ID,
		Kind:         def.Kind,
		Label:        def.Label,
		Description:  def.Description,
		ImageURL:     def.ImageURL,
		Rarity:       def.Rarity,
		Owned:        owned,
		Unobtainable: def.Unobtainable,
	}
}

func badgeFromParts(code int16, seasonID string, rank int, owned bool) (contracts.PlayerBadge, bool) {
	if code == badgeCodeSeasonRank {
		badge := seasonRankBadgeTemplate(seasonID)
		badge.Rank = rank
		badge.Owned = owned
		if owned && rank > 0 {
			displaySeason := seasonBadgeDisplayName(seasonID)
			badge.Label = displaySeason + " #" + fmt.Sprint(rank)
			badge.Description = "Finished #" + fmt.Sprint(rank) + " in " + displaySeason + "."
		}
		return badge, strings.TrimSpace(seasonID) != ""
	}
	if def, ok := badgeDefinitionByCode(code); ok {
		return badgeFromDefinition(def, owned), true
	}
	return contracts.PlayerBadge{}, false
}

func seasonBadgeDisplayName(seasonID string) string {
	switch strings.TrimSpace(seasonID) {
	case "s2":
		return "Season 1"
	case "s2.5":
		return "Season 2"
	default:
		value := strings.TrimPrefix(strings.TrimSpace(seasonID), "s")
		if value == "" {
			return "Season"
		}
		return "Season " + strings.ToUpper(value)
	}
}

func awardBadgeTx(ctx context.Context, tx pgx.Tx, userID, badgeID string) (bool, error) {
	ref, ok := badgeRefFromID(badgeID)
	if !ok || ref.Code == 0 || ref.Code == badgeCodeSeasonRank {
		return false, errors.New("badge unavailable")
	}
	tag, err := tx.Exec(ctx, `
		insert into user_badges(user_id, badge_code)
		values(
			$1,
			$2
		)
		on conflict (user_id, badge_code, badge_season_id) do nothing
	`, userID, ref.Code)
	if err != nil {
		return false, err
	}
	awarded := tag.RowsAffected() > 0
	if !awarded {
		return false, nil
	}
	badge, ok := badgeFromParts(ref.Code, ref.SeasonID, 0, true)
	if !ok {
		return false, nil
	}
	var notificationID int64
	if err := upsertUserNotification(ctx, tx, userID, "badge_unlocked", "badge_unlocked:"+userID+":"+badge.ID, map[string]any{
		"badge": badge,
	}, &notificationID); err != nil {
		return false, err
	}
	return true, nil
}

func removeGeoDuelsTeamBadgeTx(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, `
		update users
		set selected_badge_code = null,
			selected_badge_season_id = ''
		where id = $1
			and selected_badge_code = $2
	`, userID, badgeCodeGeoDuelsTeam); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		delete from user_badges
		where user_id = $1
			and badge_code = $2
	`, userID, badgeCodeGeoDuelsTeam)
	return err
}

func (s *pgStore) SyncLoginBadges(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	seasonID, err := activeSeasonIDTx(ctx, tx)
	if err != nil {
		return err
	}
	var isGuest, hasTeamRole bool
	var mmr int
	if err := tx.QueryRow(ctx, `
		select
			coalesce(u.account_type = 'guest', false),
			coalesce(u.is_admin, false)
				or coalesce(u.is_moderator, false)
				or exists (
					select 1
					from user_roles ur
					where ur.user_id = u.id
					  and ur.role in ('admin', 'moderator')
					  and ur.revoked_at is null
				),
			coalesce(r.mmr, $3)
		from users u
		left join ranks r on r.user_id = u.id and r.mode = $2 and r.season_id = $4
		where u.id = $1
	`, userID, modeDuel, initialMMR, seasonID).Scan(&isGuest, &hasTeamRole, &mmr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}
	if hasTeamRole {
		if _, err := awardBadgeTx(ctx, tx, userID, "geoduels-team"); err != nil {
			return err
		}
	}
	if !isGuest {
		if err := awardEloBadgesTx(ctx, tx, userID, mmr); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func awardEloBadgesTx(ctx context.Context, tx pgx.Tx, userID string, mmr int) error {
	thresholds := []struct {
		mmr     int
		badgeID string
	}{
		{1000, "elo-1000"},
		{1500, "elo-1500"},
		{2000, "elo-2000"},
	}
	for _, threshold := range thresholds {
		if mmr >= threshold.mmr {
			if _, err := awardBadgeTx(ctx, tx, userID, threshold.badgeID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *pgStore) AwardDiscordServerMemberByDiscordID(discordUserID string) (bool, error) {
	discordUserID = strings.TrimSpace(discordUserID)
	if discordUserID == "" {
		return false, errors.New("discord user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `
		select user_id
		from user_identities
		where provider = $1 and provider_user_id = $2
		limit 1
	`, IdentityProviderDiscord, discordUserID).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	awarded, err := awardBadgeTx(ctx, tx, userID, "discord-server-member")
	if err != nil {
		return false, err
	}
	return awarded, tx.Commit(ctx)
}

func (s *pgStore) CreateDonationRef(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", errors.New("user id required")
	}
	ref := "don_" + strings.TrimPrefix(newUserID(), "u_")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := s.pool.Exec(ctx, `
		insert into support_donation_refs(ref, user_id)
		values($1, $2)
	`, ref, userID); err != nil {
		return "", err
	}
	return ref, nil
}

func (s *pgStore) AwardSupporterByDonationRef(ref string) (bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false, errors.New("donation ref required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `
		update support_donation_refs
		set completed_at = coalesce(completed_at, now())
		where ref = $1
		returning user_id
	`, ref).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	awarded, err := awardBadgeTx(ctx, tx, userID, "supporter")
	if err != nil {
		return false, err
	}
	return awarded, tx.Commit(ctx)
}

func (s *pgStore) ListLeaderboard(mode, seasonID string, limit, offset int) ([]LeaderboardEntry, error) {
	if mode == "" {
		mode = modeDuel
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if seasonID == "" {
		var err error
		seasonID, err = s.activeSeasonID(ctx)
		if err != nil {
			return nil, err
		}
	}

	rows, err := s.pool.Query(ctx, `
		select
			row_number() over (
				order by r.mmr desc, r.updated_at asc, r.user_id asc
			) as rank,
			r.user_id,
			coalesce(nullif(u.display_name, r.user_id), ui.provider_name, r.user_id) as display_name,
			coalesce(u.avatar_url, ui.avatar_url, '') as avatar_url,
			r.mmr,
			coalesce(rs.games_played, 0) as games_played,
			coalesce(rs.wins, 0) as wins
		from ranks r
		left join users u on u.id = r.user_id
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = r.user_id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		left join ranked_stats rs on rs.user_id = r.user_id and rs.mode = r.mode and rs.season_id = r.season_id
		where r.mode = $1
			and r.season_id = $2
			and coalesce(u.account_type, 'registered') <> 'guest'
			and u.banned_at is null
		order by r.mmr desc, r.updated_at asc, r.user_id asc
		limit $3 offset $4
	`, mode, seasonID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]LeaderboardEntry, 0, limit)
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(
			&entry.Rank,
			&entry.UserID,
			&entry.DisplayName,
			&entry.AvatarURL,
			&entry.MMR,
			&entry.GamesPlayed,
			&entry.Wins,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *pgStore) GetLeaderboardOverview(userID, mode, seasonID string, limit int) (LeaderboardOverview, error) {
	if mode == "" {
		mode = modeDuel
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	entries, err := s.ListLeaderboard(mode, seasonID, limit, 0)
	if err != nil {
		return LeaderboardOverview{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if seasonID == "" {
		seasonID, err = s.activeSeasonID(ctx)
		if err != nil {
			return LeaderboardOverview{}, err
		}
	}

	var selfRank, totalPlayers int
	if err := s.pool.QueryRow(ctx, `
		with ranked as (
			select
				r.user_id,
				row_number() over (
					order by r.mmr desc, r.updated_at asc, r.user_id asc
				) as rank,
				count(*) over () as total_players
				from ranks r
				left join users u on u.id = r.user_id
				where r.mode = $1
					and r.season_id = $2
					and coalesce(u.account_type, 'registered') <> 'guest'
					and u.banned_at is null
			)
		select
			coalesce(max(rank) filter (where user_id = $3), 0) as self_rank,
			coalesce(max(total_players), 0) as total_players
		from ranked
	`, mode, seasonID, userID).Scan(&selfRank, &totalPlayers); err != nil {
		return LeaderboardOverview{}, err
	}

	return LeaderboardOverview{
		Mode:         mode,
		SeasonID:     seasonID,
		SelfRank:     selfRank,
		TotalPlayers: totalPlayers,
		Entries:      entries,
	}, nil
}

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
			insert into users (id, email, display_name, avatar_url, onboarded_at, account_type)
			values ($1, $2, $3, null, now(), 'guest')
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
	privateLobbyMatch, err := s.matchBelongsToLobby(ctx, tx, snap.MatchID)
	if err != nil {
		return err
	}
	ratedMatch := !snap.Unranked && !privateLobbyMatch && (!p1Guest || !p2Guest)
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

func (s *pgStore) matchBelongsToLobby(ctx context.Context, tx pgx.Tx, matchID string) (bool, error) {
	if matchID == "" {
		return false, nil
	}
	var exists bool
	if err := tx.QueryRow(ctx, `
		select exists (
			select 1
			from lobbies
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

func recordMatchHistory(ctx context.Context, tx pgx.Tx, matchID string, snap contracts.MatchSnapshot, replaySnapshot string) error {
	if matchID == "" {
		matchID = snap.MatchID
	}
	if matchID == "" || len(snap.Players) == 0 {
		return nil
	}
	startedAt := snapshotStartedAt(snap)
	endedAt := time.Now()
	winner := snapshotWinner(snap)
	privateLobbyMatch := false
	if snap.Mode == contracts.ModeDuel {
		if err := tx.QueryRow(ctx, `
			select exists (
				select 1
				from lobbies
				where active_match_id = $1
				   or started_match_id = $1
				   or last_match_id = $1
			)
		`, matchID).Scan(&privateLobbyMatch); err != nil {
			return err
		}
	}
	ranked := !snap.Unranked && !privateLobbyMatch
	sourceKind := "queue"
	var sourceLobbyID any
	if privateLobbyMatch {
		sourceKind = "lobby"
		var lobbyID string
		if err := tx.QueryRow(ctx, `
			select id
			from lobbies
			where active_match_id = $1
			   or started_match_id = $1
			   or last_match_id = $1
			limit 1
		`, matchID).Scan(&lobbyID); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		} else {
			sourceLobbyID = lobbyID
		}
	}
	ruleset := string(contracts.NormalizeRuleset(snap.Config.Ruleset))
	if _, err := tx.Exec(ctx, `
		insert into match_history(
			match_id, mode, state, started_at, ended_at, winner_user_id, snapshot_json,
			ranked, source_kind, source_lobby_id, ruleset, replay_json
		)
		values($1, $2, $3, $4, $5, nullif($6, ''), $7::jsonb, $8, $9, $10, nullif($11, ''), $12::jsonb)
		on conflict (match_id) do update set
			mode = excluded.mode,
			state = excluded.state,
			started_at = excluded.started_at,
			ended_at = excluded.ended_at,
			winner_user_id = excluded.winner_user_id,
			snapshot_json = excluded.snapshot_json,
			ranked = excluded.ranked,
			source_kind = excluded.source_kind,
			source_lobby_id = excluded.source_lobby_id,
			ruleset = excluded.ruleset,
			replay_json = excluded.replay_json
	`, matchID, string(snap.Mode), string(snap.State), startedAt, endedAt, winner, replaySnapshot, ranked, sourceKind, sourceLobbyID, ruleset, replaySnapshot); err != nil {
		return err
	}
	for userID, player := range snap.Players {
		displayName := player.DisplayName
		if displayName == "" {
			displayName = userID
		}
		if _, err := tx.Exec(ctx, `
			insert into match_players(match_id, user_id, display_name, mmr, hp, rating_rd, ranked_games_played)
			values($1, $2, $3, $4, $5, $6, $7)
			on conflict (match_id, user_id) do update set
				display_name = excluded.display_name,
				mmr = excluded.mmr,
				hp = excluded.hp,
				rating_rd = excluded.rating_rd,
				ranked_games_played = excluded.ranked_games_played
		`, matchID, userID, displayName, player.MMR, player.HP, clampRatingRD(player.RatingRD), player.RankedGamesPlayed); err != nil {
			return err
		}
	}
	for _, round := range snap.RoundResults {
		if round == nil {
			continue
		}
		for userID, result := range round.Players {
			guessUnixMS := nullableInt64(result.GuessUnixMS)
			guessMS := nullableInt64(result.GuessMS)
			guessedAt := any(endedAt)
			if result.GuessUnixMS > 0 {
				guessedAt = time.UnixMilli(result.GuessUnixMS)
			}
			if _, err := tx.Exec(ctx, `
				insert into match_round_guesses(
					match_id, round_id, round_number, user_id, lat, lng, actual_lat, actual_lng,
					distance_km, score, guess_unix_ms, guess_ms, ruleset, ranked, source_kind, guessed_at
				)
				values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, nullif($13, ''), $14, $15, $16)
				on conflict (match_id, round_id, user_id) do update set
					lat = excluded.lat,
					lng = excluded.lng,
					actual_lat = excluded.actual_lat,
					actual_lng = excluded.actual_lng,
					distance_km = excluded.distance_km,
					score = excluded.score,
					guess_unix_ms = excluded.guess_unix_ms,
					guess_ms = excluded.guess_ms,
					ruleset = excluded.ruleset,
					ranked = excluded.ranked,
					source_kind = excluded.source_kind,
					guessed_at = excluded.guessed_at
			`, matchID, round.RoundID, round.RoundNumber, userID, result.Lat, result.Lng, round.ActualLocation.Lat, round.ActualLocation.Lng, result.DistanceKm, result.Score, guessUnixMS, guessMS, ruleset, ranked, sourceKind, guessedAt); err != nil {
				return err
			}
			if snap.Mode == contracts.ModeDuel && !snap.Unranked && !privateLobbyMatch && result.GuessMS > 0 {
				occurredAt := endedAt
				if result.GuessUnixMS > 0 {
					occurredAt = time.UnixMilli(result.GuessUnixMS)
				}
				if _, err := tx.Exec(ctx, `
					insert into ranked_guess_events(
						user_id, match_id, round_id, round_number, ruleset, score, guess_ms, evidence, occurred_at
					)
					values($1, $2, $3, $4, $5, $6, $7, $8, $9)
					on conflict (match_id, round_id, user_id) do update set
						score = excluded.score,
						guess_ms = excluded.guess_ms,
						evidence = excluded.evidence,
						occurred_at = excluded.occurred_at
				`, userID, matchID, round.RoundID, round.RoundNumber, string(contracts.NormalizeRuleset(snap.Config.Ruleset)), result.Score, result.GuessMS, guessEvidence(result.Score, result.GuessMS), occurredAt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func finalReplaySnapshotJSON(snap contracts.MatchSnapshot) (string, error) {
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
		return "", err
	}
	return string(body), nil
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

func nullableInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func (s *pgStore) GetFinalMatchSnapshot(matchID string) ([]byte, bool, error) {
	if matchID == "" {
		return nil, false, errors.New("matchID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select coalesce(replay_json, snapshot_json)::text
		from match_history
		where match_id = $1
		limit 1
	`, matchID)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return []byte(raw), true, nil
}

func (s *pgStore) ListPlayerMatchHistory(userID string, limit int) ([]MatchHistorySummary, error) {
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select h.match_id, h.mode, h.started_at, h.ended_at, coalesce(h.winner_user_id, '')
		from match_history h
		join match_players p on p.match_id = h.match_id
		where p.user_id = $1
		order by h.ended_at desc, h.match_id desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MatchHistorySummary, 0, limit)
	for rows.Next() {
		var item MatchHistorySummary
		if err := rows.Scan(&item.MatchID, &item.Mode, &item.StartedAt, &item.EndedAt, &item.WinnerUserID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

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
		select id, coalesce(nullif(display_name, ''), id)
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
			coalesce(nullif(u.display_name, ''), u.id),
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
		snapshot := map[string]any{
			"matchId": matchID,
			"mode":    "duel",
			"state":   "ended",
			"debug":   true,
			"players": map[string]any{
				params.ReportedUserID: map[string]any{"userId": params.ReportedUserID, "displayName": reportedName},
				reporter.id:           map[string]any{"userId": reporter.id, "displayName": reporter.name},
			},
		}
		rawSnapshot, err := json.Marshal(snapshot)
		if err != nil {
			return DebugModerationReportsResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			insert into match_history(match_id, mode, state, started_at, ended_at, snapshot_json)
			values($1, 'duel', 'ended', now(), now(), $2::jsonb)
			on conflict (match_id) do nothing
		`, matchID, string(rawSnapshot)); err != nil {
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
			values($1, nullif($2, ''), 'debug_report_created', $3, jsonb_build_object('reportId', $4::bigint, 'matchId', $5::text, 'reporterUserId', $6::text, 'category', $7::text))
		`, caseID, params.CreatedBy, params.Reason, reportID, matchID, reporter.id, params.Category); err != nil {
			return DebugModerationReportsResult{}, err
		}
		createdReporterIDs = append(createdReporterIDs, reporter.id)
	}

	if _, err := tx.Exec(ctx, `
		insert into moderation_case_events(case_id, actor_user_id, event_type, body, metadata)
		values($1, nullif($2, ''), 'debug_reports_created', $3, jsonb_build_object('count', $4::int))
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
			coalesce(summary, ''), coalesce(assigned_to, ''),
			latest_activity_at, created_at, notification_sent_at,
			coalesce(queue, ''), coalesce(source, ''), risk_score, risk_breakdown::text,
			confidence, claimed_at, claim_expires_at, resolved_at, coalesce(resolved_by, ''),
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
			coalesce(summary, ''), coalesce(assigned_to, ''),
			latest_activity_at, created_at, notification_sent_at,
			coalesce(queue, ''), coalesce(source, ''), risk_score, risk_breakdown::text,
			confidence, claimed_at, claim_expires_at, resolved_at, coalesce(resolved_by, ''),
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
				resolved_by = case when $2 in ('actioned', 'dismissed', 'duplicate') then nullif($3, '') else resolved_by end,
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
				resolved_by = nullif($2, ''),
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
				resolved_by = nullif($2, ''),
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
			set assigned_to = nullif($2, ''),
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
		values($1, nullif($2, ''), $3, $4, nullif($5, ''))
	`, params.CaseID, params.ActorUserID, targetUserID, params.ActionType, params.Reason); err != nil {
		return ModerationCaseDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_events(case_id, actor_user_id, event_type, body)
		values($1, nullif($2, ''), $3, nullif($4, ''))
	`, params.CaseID, params.ActorUserID, "action_"+params.ActionType, params.Reason); err != nil {
		return ModerationCaseDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into moderation_case_log(case_id, actor_user_id, event_type, body, metadata)
		values($1, nullif($2, ''), $3, nullif($4, ''), jsonb_build_object('actionType', $5::text, 'status', $6::text))
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
			coalesce(nullif(reporter.display_name, ''), r.reporter_user_id),
			r.reported_user_id,
			coalesce(nullif(reported.display_name, ''), r.reported_user_id),
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
			select match_id from moderation_reports where case_id = $1 and match_id <> ''
			union
			select match_id from moderation_evidence where case_id = $1 and match_id is not null and match_id <> ''
		)
		select
			cm.match_id,
			coalesce(h.mode, ''),
			h.started_at,
			h.ended_at,
			coalesce(h.winner_user_id, ''),
			coalesce(jsonb_array_length(coalesce(h.snapshot_json->'roundResults', '[]'::jsonb)), 0),
			coalesce(p.user_id, ''),
			coalesce(nullif(p.display_name, ''), p.user_id, ''),
			coalesce(nullif(h.snapshot_json #>> array['players', p.user_id, 'totalScore'], '')::int, 0),
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
		select id, case_id, coalesce(actor_user_id, ''), event_type, coalesce(body, ''), created_at
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
		select id, case_id, coalesce(actor_user_id, ''), target_user_id, action_type, coalesce(reason, ''), created_at
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
			id, case_id, evidence_type, coalesce(match_id, ''), coalesce(round_id, ''),
			coalesce(subject_user_id, ''), coalesce(detector_version, ''), coalesce(rule_id, ''),
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
			id, case_id, coalesce(actor_user_id, ''), event_type,
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
		set assigned_to = nullif($2, ''),
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
		values($1, nullif($2, ''), $3, jsonb_build_object('assignedTo', $4::text))
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

func (s *pgStore) ClaimPendingNotification(notificationType string, now time.Time) (NotificationOutboxItem, bool, error) {
	notificationType = strings.TrimSpace(notificationType)
	if notificationType == "" {
		return NotificationOutboxItem{}, false, errors.New("notification type required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NotificationOutboxItem{}, false, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		with candidate as (
			select id
			from notification_outbox
			where type = $1
				and sent_at is null
				and next_attempt_at <= $2
			order by next_attempt_at asc, id asc
			limit 1
			for update skip locked
		)
		update notification_outbox n
		set attempts = n.attempts + 1,
			next_attempt_at = $3,
			last_error = null
		from candidate
		where n.id = candidate.id
		returning n.id, n.type, n.payload_json::text, n.attempts
	`, notificationType, now, now.Add(5*time.Minute))
	var item NotificationOutboxItem
	var raw string
	if err := row.Scan(&item.ID, &item.Type, &raw, &item.Attempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NotificationOutboxItem{}, false, nil
		}
		return NotificationOutboxItem{}, false, err
	}
	item.PayloadJSON = []byte(raw)
	if err := tx.Commit(ctx); err != nil {
		return NotificationOutboxItem{}, false, err
	}
	return item, true, nil
}

func (s *pgStore) MarkNotificationSent(id int64) error {
	if id <= 0 {
		return errors.New("notification id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update notification_outbox
		set sent_at = now(),
			last_error = null
		where id = $1
	`, id)
	return err
}

func (s *pgStore) MarkNotificationFailed(id int64, nextAttemptAt time.Time, lastError string) error {
	if id <= 0 {
		return errors.New("notification id required")
	}
	lastError = strings.TrimSpace(lastError)
	if len(lastError) > 1000 {
		lastError = lastError[:1000]
	}
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now().Add(time.Minute)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update notification_outbox
		set next_attempt_at = $2,
			last_error = nullif($3, '')
		where id = $1
			and sent_at is null
	`, id, nextAttemptAt, lastError)
	return err
}

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

func upsertUserNotification(ctx context.Context, tx pgx.Tx, userID, notificationType, dedupeKey string, payload any, id *int64) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return tx.QueryRow(ctx, `
		with inserted as (
			insert into user_notifications(user_id, type, dedupe_key, payload_json)
			values($1, $2, $3, $4::jsonb)
			on conflict (dedupe_key) do update set payload_json = excluded.payload_json
			returning id
		)
		select id from inserted
	`, userID, notificationType, dedupeKey, string(body)).Scan(id)
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(body)
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

func (s *pgStore) ListUserNotifications(userID string, limit int) ([]UserNotification, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select id, type, payload_json::text, created_at
		from user_notifications
		where user_id = $1
			and read_at is null
		order by created_at desc, id desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UserNotification{}
	for rows.Next() {
		var item UserNotification
		var raw string
		if err := rows.Scan(&item.ID, &item.Type, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Payload = json.RawMessage(raw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) MarkUserNotificationRead(userID string, notificationID int64) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || notificationID <= 0 {
		return errors.New("userID and notificationID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update user_notifications
		set read_at = coalesce(read_at, now())
		where id = $1 and user_id = $2
	`, notificationID, userID)
	return err
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

func (s *pgStore) GetRuntimeMatch(matchID string) (RuntimeMatch, bool, error) {
	if matchID == "" {
		return RuntimeMatch{}, false, errors.New("matchID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select
			id,
			state,
			owner_epoch,
			started_at,
			coalesce(ended_at, '0001-01-01 00:00:00+00'::timestamptz)
		from runtime_matches
		where id = $1
	`, matchID)
	var out RuntimeMatch
	if err := row.Scan(&out.MatchID, &out.State, &out.OwnerEpoch, &out.StartedAt, &out.EndedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuntimeMatch{}, false, nil
		}
		return RuntimeMatch{}, false, err
	}
	return out, true, nil
}

func (s *pgStore) RecordRuntimeMatch(matchID, state string, ownerEpoch int64, terminal bool) error {
	if matchID == "" {
		return errors.New("matchID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if terminal {
		_, err := s.pool.Exec(ctx, `
			insert into runtime_matches(id, state, owner_epoch, started_at, ended_at)
			values($1,$2,$3,now(),now())
			on conflict (id) do update set
				state = excluded.state,
				owner_epoch = excluded.owner_epoch,
				ended_at = now()
		`, matchID, state, ownerEpoch)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		insert into runtime_matches(id, state, owner_epoch, started_at)
		values($1,$2,$3,now())
		on conflict (id) do update set
			state = excluded.state,
			owner_epoch = excluded.owner_epoch
	`, matchID, state, ownerEpoch)
	return err
}

func (s *pgStore) RecordChatMessage(conversationID, scopeKind, scopeID string, message ChatMessage) error {
	conversationID = strings.TrimSpace(conversationID)
	scopeKind = strings.TrimSpace(scopeKind)
	scopeID = strings.TrimSpace(scopeID)
	if conversationID == "" || scopeKind == "" || scopeID == "" {
		return errors.New("conversation scope required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	createdAt := message.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return s.recordChatMessage(ctx, conversationID, scopeKind, scopeID, message, createdAt)
}

func (s *pgStore) recordChatMessage(ctx context.Context, conversationID, scopeKind, scopeID string, message ChatMessage, createdAt time.Time) error {
	body := nullable(message.Body)
	emote := nullable(string(message.Emote))
	_, err := s.pool.Exec(ctx, `
		insert into chat_conversations (id, scope_kind, scope_id)
		values ($1, $2, $3)
		on conflict (id) do nothing
	`, conversationID, scopeKind, scopeID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into chat_messages (
			id, conversation_id, match_id, sender_user_id, sender_display_name, kind, body, emote, created_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		on conflict (id) do nothing
	`, message.ID, conversationID, nullable(message.MatchID), message.SenderUserID, message.SenderDisplayName, string(message.Kind), body, emote, createdAt)
	return err
}

func (s *pgStore) ListChatMessages(conversationID string, limit int) ([]ChatMessage, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select id, conversation_id, coalesce(match_id, ''), sender_user_id, sender_display_name, kind, coalesce(body, ''), coalesce(emote, ''), created_at
		from chat_messages
		where conversation_id = $1
		order by created_at asc
		limit $2
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []ChatMessage{}
	for rows.Next() {
		var message ChatMessage
		var kind string
		var emote string
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.MatchID, &message.SenderUserID, &message.SenderDisplayName, &kind, &message.Body, &emote, &message.CreatedAt); err != nil {
			return nil, err
		}
		message.Kind = contracts.ChatMessageKind(kind)
		message.Emote = contracts.ChatEmote(emote)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *pgStore) ExpireStaleRuntimeMatches(prefix string, olderThan time.Duration) error {
	if strings.TrimSpace(prefix) == "" || olderThan <= 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		update runtime_matches
		set state = $1,
			ended_at = now()
		where state = $2
		  and id like $3
		  and started_at < now() - $4::interval
		  and ended_at is null
	`, string(contracts.MatchEnded), string(contracts.MatchLive), prefix+"%", olderThan.String())
	return err
}

type mapRow struct {
	Lat     float64
	Lng     float64
	Country string
	PanoID  *string
	Heading *float64
	Pitch   *float64
	RandKey float64
}

func parseMapRows(b []byte) ([]mapRow, error) {
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]mapRow, 0, len(raw))
	for _, it := range raw {
		lat, ok1 := asFloat(it["lat"])
		lng, ok2 := asFloat(it["lng"])
		if !ok1 || !ok2 {
			continue
		}
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		row := mapRow{Lat: lat, Lng: lng, RandKey: stableRand(lat, lng)}
		if country, ok := it["country"].(string); ok {
			row.Country = country
		}
		if panoID, ok := it["panoId"].(string); ok && panoID != "" {
			row.PanoID = &panoID
		}
		if heading, ok := asFloat(it["heading"]); ok {
			row.Heading = &heading
		}
		if pitch, ok := asFloat(it["pitch"]); ok {
			row.Pitch = &pitch
		}
		out = append(out, row)
	}
	return out, nil
}

func stableRand(lat, lng float64) float64 {
	h := sha1.Sum([]byte(fmt.Sprintf("%.8f:%.8f", lat, lng)))
	v := int(h[0])<<16 | int(h[1])<<8 | int(h[2])
	return float64(v) / float64(1<<24)
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func newUserID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "u_" + hex.EncodeToString(b)
}

func newDebugMatchID(index int) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("debug-report-%s-%02d", hex.EncodeToString(b), index)
}

func normalizeDBURLForContainer(dsn string) string {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return dsn
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" {
		port := u.Port()
		if port == "" {
			port = "5432"
		}
		u.Host = "host.docker.internal:" + port
		return u.String()
	}
	return dsn
}
