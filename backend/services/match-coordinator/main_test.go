package main

import (
	"context"
	"encoding/json"
	"geoduels/internal/accounts"
	"geoduels/internal/admin"
	"geoduels/internal/badges"
	"geoduels/internal/chat"
	"geoduels/internal/content"
	"geoduels/internal/leaderboard"
	"geoduels/internal/matches"
	"geoduels/internal/moderation"
	"geoduels/internal/profiles"
	"geoduels/internal/seasons"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/auth"
	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
	"geoduels/pkg/matchstore"
	"geoduels/pkg/observability"
)

type recoverTestStore struct {
	runtimeMatches map[string]matches.RuntimeMatch
	profiles       map[string]profiles.Profile
	parties        map[string]contracts.PartySnapshot
	accountTypes   map[string]string
}

func (s *recoverTestStore) ListAdminGrantableBadges() []badges.AdminBadgeDefinition {
	panic("unexpected call")
}

func (s *recoverTestStore) GrantBadgeToUser(nickname, badgeID, actorUserID string) (contracts.PlayerBadge, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UpsertIdentity(sub, email, googleName, avatarURL string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (accounts.Identity, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (accounts.Identity, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (accounts.Identity, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GoogleIdentityExists(googleSub string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ProviderIdentityExists(provider, providerUserID string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UnlinkProviderIdentity(userID, provider string) (accounts.Identity, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) CreateGuestIdentity() (accounts.Identity, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetIdentity(sub string) (accounts.Identity, error) {
	accountType := "registered"
	if s.accountTypes != nil && s.accountTypes[sub] != "" {
		accountType = s.accountTypes[sub]
	}
	return accounts.Identity{
		Sub:              sub,
		NicknameRequired: false,
		AccountType:      accountType,
	}, nil
}

func (s *recoverTestStore) SetNickname(sub, displayName string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) SuggestNickname(sub, displayName string) (string, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetUserAdmin(userID string, isAdmin bool) error {
	panic("unexpected call")
}

func (s *recoverTestStore) SetUserModerator(userID string, isModerator bool) error {
	panic("unexpected call")
}

func (s *recoverTestStore) SearchPlayers(query string, limit int) ([]admin.AdminPlayerSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetPlayerBan(userID, reason, actorUserID string, banned bool) error {
	panic("unexpected call")
}

func (s *recoverTestStore) SetPlayerMute(userID, kind, reason, actorUserID string, until time.Time, muted bool) error {
	panic("unexpected call")
}

func (s *recoverTestStore) BanPlayerForCheating(userID, reason, actorUserID string) (moderation.CheatingBanSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) PreviewCommunityPardon(olderThan time.Duration) (moderation.CommunityPardonSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) PardonBannedPlayers(olderThan time.Duration, actorUserID string) (moderation.CommunityPardonSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ClearReporterMute(userID string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetLobbyChangelog(defaultContent content.LobbyChangelogContent) (content.LobbyChangelogContent, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetLobbyChangelog(content content.LobbyChangelogContent) error {
	panic("unexpected call")
}

func (s *recoverTestStore) ListChangelogPosts(includeUnpublished bool) ([]content.ChangelogPost, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetChangelogPostBySlug(slug string, publishedOnly bool) (content.ChangelogPost, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) CreateChangelogPost(input content.ChangelogPostInput) (content.ChangelogPost, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UpdateChangelogPost(id int64, input content.ChangelogPostInput) (content.ChangelogPost, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetModerationSettings() (content.ModerationSettings, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetModerationSettings(settings content.ModerationSettings) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetDiscordIntegrationSettings() (content.DiscordIntegrationSettings, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetDiscordIntegrationSettings(settings content.DiscordIntegrationSettings) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetRankedSeasonSettings() (seasons.RankedSeasonSettings, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetRankedSeasonResetRule(monthlyResetDay int) (seasons.RankedSeasonSettings, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) RunDueRankedSeasonReset(now time.Time) (seasons.RankedSeasonResetResult, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ReplaceMapLocations(mapKey, displayName string, dataset []byte) (contracts.MapImportSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetGameplayMapSettings() (contracts.GameplayMapSettings, error) {
	return contracts.GameplayMapSettings{
		MovingMapID: contracts.MapKeyMoving,
		NoMoveMapID: contracts.MapKeyNMPZ,
		NMPZMapID:   contracts.MapKeyNMPZ,
	}, nil
}

func (s *recoverTestStore) ResolveGameplayMapID(mode contracts.MatchMode, ruleset contracts.GameRuleset, requestedMapID string) (string, error) {
	if requestedMapID != "" {
		return requestedMapID, nil
	}
	switch contracts.NormalizeRuleset(ruleset) {
	case contracts.RulesetNoMove, contracts.RulesetNMPZ:
		return contracts.MapKeyNMPZ, nil
	default:
		return contracts.MapKeyMoving, nil
	}
}

func (s *recoverTestStore) CreateAuthSession(userID, refreshTokenHash string, expiresAt time.Time, params contracts.AuthSessionParams) (contracts.RefreshTokenRecord, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetAuthSessionByRefreshToken(hash string) (contracts.RefreshTokenRecord, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) RotateAuthSession(sessionID, currentHash, nextHash string, expiresAt time.Time, usedAt time.Time) (contracts.RefreshTokenRecord, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) RevokeAuthSession(sessionID string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) RevokeAuthSessionsForUser(userID string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) DeleteAccount(userID string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UpsertUser(userID, email, displayName string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetProfile(userID string) (profiles.Profile, error) {
	if profile, ok := s.profiles[userID]; ok {
		return profile, nil
	}
	return profiles.Profile{UserID: userID, DisplayName: userID, MMR: 1000}, nil
}

func (s *recoverTestStore) GetPublicPlayerProfileByNickname(nickname string) (profiles.PublicPlayerProfile, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) UpdateSelectedBadge(userID, badgeID string) (profiles.Profile, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SyncLoginBadges(userID string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) AwardDiscordServerMemberByDiscordID(discordUserID string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ClaimPendingDiscordSync(now time.Time) (badges.DiscordSyncOutboxItem, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) MarkDiscordSyncProcessed(id int64) error {
	panic("unexpected call")
}

func (s *recoverTestStore) MarkDiscordSyncFailed(id int64, nextAttemptAt time.Time, lastError string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetDiscordLinkedUser(discordUserID string) (badges.DiscordLinkedUser, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) CreateDonationRef(userID string) (string, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) AwardSupporterByDonationRef(ref string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListLeaderboard(context.Context, string, string, int, int) ([]leaderboard.Entry, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetLeaderboardOverview(context.Context, string, string, string, int) (leaderboard.Overview, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) FinalizeMatch(snap contracts.MatchSnapshot, ownerEpoch int64) (contracts.MatchSnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) RenewMatchSessionLeases(nodeID string, ownerEpoch int64, matchIDs []string, ttl time.Duration) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetFinalMatchSnapshot(matchID string) ([]byte, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListPlayerMatchHistory(userID string, limit int) ([]matches.MatchHistorySummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListPlayerMatchHistoryPage(userID string, limit int, beforeEndedAt time.Time, beforeMatchID string, rankedOnly bool) (matches.MatchHistoryPage, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) PlayerParticipatedInMatch(userID, matchID string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetAdminPlayerDetail(userID string) (admin.AdminPlayerDetail, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) CreatePlayerReportSignal(params moderation.CreatePlayerReportSignalParams) (moderation.ModerationSignalCreated, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListSubjectModerationProfile(userID string) (moderation.ModerationSubjectProfile, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListModerationSignals(limit int) ([]moderation.ModerationSignalSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListModerationLog(limit int) ([]moderation.ModerationAuditLogEntry, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListUserRoles() ([]admin.UserRoleGrant, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GrantUserRole(userID, role, grantedBy, reason string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) RevokeUserRole(userID, role, revokedBy, reason string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) IssueEloRefundsForCheater(userID string, lookback time.Duration) (moderation.EloRefundSummary, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListUserNotifications(userID string, limit int) ([]contracts.UserNotification, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListNotificationInbox(string, int, int64) ([]contracts.UserNotification, error) {
	panic("unexpected call")
}
func (s *recoverTestStore) MarkAllUserNotificationsRead(string) error { panic("unexpected call") }

func (s *recoverTestStore) MarkUserNotificationRead(userID string, notificationID int64) error {
	panic("unexpected call")
}

func (s *recoverTestStore) ClaimPendingNotification(notificationType string, now time.Time) (contracts.NotificationOutboxItem, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) MarkNotificationSent(id int64) error {
	panic("unexpected call")
}

func (s *recoverTestStore) MarkNotificationFailed(id int64, nextAttemptAt time.Time, lastError string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) AddSignupIPBan(ipAddress, reason, createdBy string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) RemoveSignupIPBan(ipAddress string) error {
	panic("unexpected call")
}

func (s *recoverTestStore) ListSignupIPBans(limit int) ([]moderation.SignupIPBan, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) IsSignupIPBanned(ipAddress string) (bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) GetRuntimeMatch(_ context.Context, matchID string) (matches.RuntimeMatch, bool, error) {
	rec, ok := s.runtimeMatches[matchID]
	return rec, ok, nil
}

func (s *recoverTestStore) RecordRuntimeMatch(_ context.Context, matchID, state string, ownerEpoch int64, terminal bool) error {
	if s.runtimeMatches == nil {
		s.runtimeMatches = map[string]matches.RuntimeMatch{}
	}
	rec := s.runtimeMatches[matchID]
	rec.MatchID = matchID
	rec.State = state
	rec.OwnerEpoch = ownerEpoch
	if rec.StartedAt.IsZero() {
		rec.StartedAt = time.Now()
	}
	if terminal {
		rec.EndedAt = time.Now()
	}
	s.runtimeMatches[matchID] = rec
	return nil
}

func (s *recoverTestStore) RecordChatMessage(conversationID, scopeKind, scopeID string, message chat.ChatMessage) error {
	panic("unexpected call")
}

func (s *recoverTestStore) ListChatMessages(conversationID string, limit int) ([]chat.ChatMessage, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ListChatMessagesForUser(conversationID, userID string, limit int) ([]chat.ChatMessage, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) ActivePartyChatTeam(partyID, userID string) (string, string, bool, error) {
	return "", "", false, nil
}

func (s *recoverTestStore) ChatTeamForMatch(matchID, userID string) (string, bool, error) {
	return "", false, nil
}

func (s *recoverTestStore) GetActiveChatRestriction(userID string) (chat.ChatRestriction, bool, error) {
	return chat.ChatRestriction{}, false, nil
}

func (s *recoverTestStore) ExpireStaleRuntimeMatches(_ context.Context, prefix string, olderThan time.Duration) error {
	return nil
}

func (s *recoverTestStore) ExpireOpenParties() error {
	return nil
}

func (s *recoverTestStore) UpsertMatchSession(_ context.Context, params matches.MatchSessionUpsert) error {
	return nil
}

func (s *recoverTestStore) MatchSessionReturnTarget(_ context.Context, matchID string) (*contracts.MatchReturnTarget, bool, error) {
	return nil, false, nil
}

func (s *recoverTestStore) SetPartyConfig(id string, cfg contracts.MatchConfig) (contracts.PartySnapshot, error) {
	if s.parties != nil {
		if snap, ok := s.parties[id]; ok {
			snap.Config = cfg
			s.parties[id] = snap
			return snap, nil
		}
	}
	return contracts.PartySnapshot{}, nil
}

func (s *recoverTestStore) ListOpenPartyIDs() ([]string, error) {
	return nil, nil
}

func (s *recoverTestStore) CloseInactiveOpenParties(partyIDs []string, inactiveFor time.Duration) (int64, error) {
	return 0, nil
}

func (s *recoverTestStore) CreateParty(ownerUserID string, mode contracts.MatchMode, mapScope string, ttl time.Duration) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetPartyMode(partyID string, mode contracts.MatchMode) error {
	panic("unexpected call")
}

func (s *recoverTestStore) GetPartyByID(partyID string) (contracts.PartySnapshot, bool, error) {
	if s.parties == nil {
		panic("unexpected call")
	}
	snap, ok := s.parties[partyID]
	return snap, ok, nil
}

func (s *recoverTestStore) GetPartyByInviteCode(inviteCode string) (contracts.PartySnapshot, bool, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) MatchSessionSourceParty(_ context.Context, matchID string) (string, string, bool, error) {
	if s.parties == nil {
		return "", "", false, nil
	}
	for _, snap := range s.parties {
		if snap.ActiveMatchID == matchID || snap.LastMatchID == matchID || snap.StartedMatchID == matchID {
			return snap.ID, snap.InviteCode, true, nil
		}
	}
	return "", "", false, nil
}

func (s *recoverTestStore) GetPartyByMatchID(matchID string) (contracts.PartySnapshot, bool, error) {
	if s.parties == nil {
		return contracts.PartySnapshot{}, false, nil
	}
	for _, snap := range s.parties {
		if snap.ActiveMatchID == matchID || snap.LastMatchID == matchID || snap.StartedMatchID == matchID {
			return snap, true, nil
		}
	}
	return contracts.PartySnapshot{}, false, nil
}

func (s *recoverTestStore) JoinParty(partyID, userID string) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) LeaveParty(partyID, userID string) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) SetPartyMemberTeam(partyID, userID, teamID string) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) KickPartyMember(partyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) TransferPartyOwner(partyID, ownerUserID, targetUserID string) (contracts.PartySnapshot, error) {
	panic("unexpected call")
}

func (s *recoverTestStore) MarkPartyInMatch(partyID, matchID string) (contracts.PartySnapshot, error) {
	if s.parties == nil {
		panic("unexpected call")
	}
	snap := s.parties[partyID]
	snap.State = contracts.PartyInMatch
	snap.ActiveMatchID = matchID
	snap.StartedMatchID = matchID
	s.parties[partyID] = snap
	return snap, nil
}

func (s *recoverTestStore) ReopenEndedParties() (int64, error) {
	if s.parties == nil || s.runtimeMatches == nil {
		return 0, nil
	}
	var reopened int64
	for id, snap := range s.parties {
		matchID := snap.ActiveMatchID
		if matchID == "" {
			matchID = snap.StartedMatchID
		}
		rec, ok := s.runtimeMatches[matchID]
		if matchID == "" || !ok || rec.State != string(contracts.MatchEnded) {
			continue
		}
		snap.State = contracts.PartyOpen
		snap.LastMatchID = matchID
		snap.ActiveMatchID = ""
		snap.StartedMatchID = ""
		s.parties[id] = snap
		reopened++
	}
	return reopened, nil
}

func (s *recoverTestStore) Close() {}

type recoverTestMatchStore struct{}

func (s *recoverTestMatchStore) Join(pool matchstore.QueuePool, ruleset contracts.GameRuleset, req contracts.QueueJoinRequest) (contracts.QueueJoinResponse, *contracts.MatchFound, error) {
	panic("unexpected call")
}

func (s *recoverTestMatchStore) Heartbeat(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (string, error) {
	panic("unexpected call")
}

func (s *recoverTestMatchStore) Leave(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) error {
	panic("unexpected call")
}

func (s *recoverTestMatchStore) LeaveAllRulesets(pool matchstore.QueuePool, userID string) error {
	panic("unexpected call")
}

func (s *recoverTestMatchStore) Poll(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (*contracts.MatchFound, error) {
	panic("unexpected call")
}

func (s *recoverTestMatchStore) IsQueued(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (bool, error) {
	return false, nil
}

func (s *recoverTestMatchStore) RunMatchmaking(pool matchstore.QueuePool, ruleset contracts.GameRuleset, limit int) (int, error) {
	return 0, nil
}

type queueTestMatchStore struct{}

func (s *queueTestMatchStore) Join(pool matchstore.QueuePool, ruleset contracts.GameRuleset, req contracts.QueueJoinRequest) (contracts.QueueJoinResponse, *contracts.MatchFound, error) {
	return contracts.QueueJoinResponse{TicketID: "t-1", Status: "queued"}, nil, nil
}

func (s *queueTestMatchStore) Heartbeat(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (string, error) {
	panic("unexpected call")
}

func (s *queueTestMatchStore) Leave(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) error {
	return nil
}

func (s *queueTestMatchStore) LeaveAllRulesets(pool matchstore.QueuePool, userID string) error {
	return nil
}

func (s *queueTestMatchStore) Poll(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (*contracts.MatchFound, error) {
	return nil, context.Canceled
}

func (s *queueTestMatchStore) IsQueued(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (bool, error) {
	return false, nil
}

func (s *queueTestMatchStore) RunMatchmaking(pool matchstore.QueuePool, ruleset contracts.GameRuleset, limit int) (int, error) {
	return 0, nil
}

type staleQueuePollStore struct {
	match  *contracts.MatchFound
	cancel context.CancelFunc
	polled bool
}

func (s *staleQueuePollStore) Join(pool matchstore.QueuePool, ruleset contracts.GameRuleset, req contracts.QueueJoinRequest) (contracts.QueueJoinResponse, *contracts.MatchFound, error) {
	return contracts.QueueJoinResponse{TicketID: "t-1", Status: "queued"}, nil, nil
}

func (s *staleQueuePollStore) Heartbeat(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (string, error) {
	panic("unexpected call")
}

func (s *staleQueuePollStore) Leave(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) error {
	return nil
}

func (s *staleQueuePollStore) LeaveAllRulesets(pool matchstore.QueuePool, userID string) error {
	return nil
}

func (s *staleQueuePollStore) Poll(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (*contracts.MatchFound, error) {
	if !s.polled {
		s.polled = true
		return s.match, nil
	}
	return nil, nil
}

func (s *staleQueuePollStore) IsQueued(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (bool, error) {
	return false, nil
}

func (s *staleQueuePollStore) RunMatchmaking(pool matchstore.QueuePool, ruleset contracts.GameRuleset, limit int) (int, error) {
	return 0, nil
}

type heartbeatTestStore struct {
	status string
	pool   matchstore.QueuePool
}

func (s *heartbeatTestStore) Join(pool matchstore.QueuePool, ruleset contracts.GameRuleset, req contracts.QueueJoinRequest) (contracts.QueueJoinResponse, *contracts.MatchFound, error) {
	panic("unexpected call")
}

func (s *heartbeatTestStore) Heartbeat(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (string, error) {
	s.pool = pool
	return s.status, nil
}

func (s *heartbeatTestStore) Leave(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) error {
	panic("unexpected call")
}

func (s *heartbeatTestStore) LeaveAllRulesets(pool matchstore.QueuePool, userID string) error {
	panic("unexpected call")
}

func (s *heartbeatTestStore) Poll(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (*contracts.MatchFound, error) {
	panic("unexpected call")
}

func (s *heartbeatTestStore) IsQueued(pool matchstore.QueuePool, rulesets []contracts.GameRuleset, userID string) (bool, error) {
	panic("unexpected call")
}

func (s *heartbeatTestStore) RunMatchmaking(pool matchstore.QueuePool, ruleset contracts.GameRuleset, limit int) (int, error) {
	return 0, nil
}

func attachRecoverStore(q *matchCoordinator, store *recoverTestStore) {
	q.accounts = store
	q.profiles = store
	q.matches = store
	q.parties = store
	q.chat = store
}

func readyCoordinator(q *matchCoordinator) *matchCoordinator {
	if s, ok := q.matches.(*recoverTestStore); ok {
		attachRecoverStore(q, s)
	}
	return q
}

var _ matchstore.Store = (*recoverTestMatchStore)(nil)
var _ matchstore.Store = (*queueTestMatchStore)(nil)
var _ matchstore.Store = (*staleQueuePollStore)(nil)

func TestPartyPatchIncludesChangedMembersAndMode(t *testing.T) {
	prev := contracts.PartySnapshot{
		ID:          "lob-1",
		OwnerUserID: "u1",
		State:       contracts.PartyOpen,
		Mode:        contracts.ModeDuel,
		Members: []contracts.PartyMember{
			{UserID: "u1", DisplayName: "One", TeamID: "a", Connected: true},
			{UserID: "u2", DisplayName: "Two", TeamID: "b", Connected: true},
		},
	}
	next := prev
	next.Mode = contracts.ModeTeamDuel
	next.Members = []contracts.PartyMember{
		{UserID: "u1", DisplayName: "One", TeamID: "b", Connected: true},
		{UserID: "u3", DisplayName: "Three", TeamID: "a", Connected: false},
	}
	patch := partyPatch(prev, next, 2)
	if patch.Mode == nil || *patch.Mode != contracts.ModeTeamDuel {
		t.Fatalf("expected mode patch, got %+v", patch.Mode)
	}
	if len(patch.UpsertMembers) != 2 {
		t.Fatalf("expected changed and new members, got %+v", patch.UpsertMembers)
	}
	if len(patch.RemoveMemberIDs) != 1 || patch.RemoveMemberIDs[0] != "u2" {
		t.Fatalf("expected u2 removal, got %+v", patch.RemoveMemberIDs)
	}
}

func TestPartyPatchIncludesPresenceChange(t *testing.T) {
	prev := contracts.PartySnapshot{
		ID:          "lob-1",
		OwnerUserID: "u1",
		State:       contracts.PartyOpen,
		Mode:        contracts.ModeDuel,
		Members: []contracts.PartyMember{
			{UserID: "u1", DisplayName: "One", Connected: true},
			{UserID: "u2", DisplayName: "Two", Connected: false},
		},
	}
	next := prev
	next.Members = []contracts.PartyMember{
		{UserID: "u1", DisplayName: "One", Connected: true},
		{UserID: "u2", DisplayName: "Two", Connected: true},
	}
	patch := partyPatch(prev, next, 2)
	if len(patch.UpsertMembers) != 1 {
		t.Fatalf("expected one presence upsert, got %+v", patch.UpsertMembers)
	}
	if patch.UpsertMembers[0].UserID != "u2" || !patch.UpsertMembers[0].Connected {
		t.Fatalf("expected u2 connected patch, got %+v", patch.UpsertMembers[0])
	}
}

func TestPartyPatchIncludesActiveMatchRosterChange(t *testing.T) {
	prev := contracts.PartySnapshot{
		Members: []contracts.PartyMember{{UserID: "u1"}, {UserID: "u2"}},
	}
	next := prev
	next.Members = []contracts.PartyMember{
		{UserID: "u1", InActiveMatch: true},
		{UserID: "u2"},
	}
	patch := partyPatch(prev, next, 2)
	if len(patch.UpsertMembers) != 1 || patch.UpsertMembers[0].UserID != "u1" || !patch.UpsertMembers[0].InActiveMatch {
		t.Fatalf("expected active roster upsert, got %+v", patch.UpsertMembers)
	}
}

func TestApplyPartyPresenceComputesStatuses(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := readyCoordinator(&matchCoordinator{redis: rdb})
	now := time.Now().UnixMilli()
	if err := rdb.HSet(context.Background(), partyPresenceKey("lob-1"), map[string]any{
		"u1":        now,
		"u2":        now - 30_000,
		"u3|conn-1": now,
	}).Err(); err != nil {
		t.Fatalf("set presence: %v", err)
	}

	snap := contracts.PartySnapshot{
		ID: "lob-1",
		Members: []contracts.PartyMember{
			{UserID: "u1", DisplayName: "One"},
			{UserID: "u2", DisplayName: "Two"},
			{UserID: "u3", DisplayName: "Three"},
		},
	}
	q.applyPartyPresence(&snap)

	if !snap.Members[0].Connected || snap.Members[0].PresenceStatus != contracts.PartyPresenceOnline {
		t.Fatalf("u1 presence = %+v", snap.Members[0])
	}
	if snap.Members[1].Connected || snap.Members[1].PresenceStatus != contracts.PartyPresenceAway {
		t.Fatalf("u2 presence = %+v", snap.Members[1])
	}
	if snap.Members[2].Connected || snap.Members[2].PresenceStatus != contracts.PartyPresenceOffline {
		t.Fatalf("u3 presence = %+v", snap.Members[2])
	}
}

func TestTouchPartyPresencePublishesOnlyOnVisibleStatusChange(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := readyCoordinator(&matchCoordinator{redis: rdb})
	if !q.touchPartyPresence("lob-1", "u1", "conn-1") {
		t.Fatal("first touch should publish because the user becomes online")
	}
	if q.touchPartyPresence("lob-1", "u1", "conn-2") {
		t.Fatal("second online touch should not publish")
	}
	if err := rdb.HSet(context.Background(), partyPresenceKey("lob-1"), "u1", time.Now().Add(-30*time.Second).UnixMilli()).Err(); err != nil {
		t.Fatalf("age presence: %v", err)
	}
	if !q.touchPartyPresence("lob-1", "u1", "conn-3") {
		t.Fatal("away to online touch should publish")
	}
}

var _ matchstore.Store = (*heartbeatTestStore)(nil)

func queueWSURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http")
}

// dispatch runs req through an Echo router with the routes registered the same
// way main.go registers them, replacing direct handler calls.
func dispatch(_ *matchCoordinator, register func(*echo.Echo), rec *httptest.ResponseRecorder, req *http.Request) {
	e := echo.New()
	e.HideBanner = true
	register(e)
	e.ServeHTTP(rec, req)
}

func TestParseQueueVariantsSupportsRankedQueuesAndMigratesLegacyRulesets(t *testing.T) {
	got := parseQueueVariants(
		"moving_hidden,no_move_hidden,nmpz_hidden,moving,no_move,nmpz",
		"",
	)
	want := []matchstore.QueueVariant{matchstore.QueueNoMoveHidden, matchstore.QueueMoving}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("queues = %#v, want %#v", got, want)
	}

	legacy := parseQueueVariants("", "moving,nmpz")
	wantLegacy := []matchstore.QueueVariant{matchstore.QueueMoving}
	if !reflect.DeepEqual(legacy, wantLegacy) {
		t.Fatalf("legacy queues = %#v, want %#v", legacy, wantLegacy)
	}
}

func testParty(id, owner string, members ...string) contracts.PartySnapshot {
	out := contracts.PartySnapshot{
		ID:          id,
		InviteCode:  "ABC123",
		OwnerUserID: owner,
		State:       contracts.PartyOpen,
		Mode:        contracts.ModeDuel,
		MapScope:    "world",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	for _, userID := range members {
		out.Members = append(out.Members, contracts.PartyMember{
			UserID:      userID,
			DisplayName: "Player " + strings.TrimPrefix(userID, "u"),
			Role:        "member",
			Ready:       true,
			JoinedAt:    time.Now(),
		})
	}
	return out
}

func readQueueEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, body, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode ws payload: %v", err)
	}
	return payload
}

func TestQueueIgnoresEndedAssignment(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	state := coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	assignment := coordinator.Assignment{
		MatchID:     "m-ended",
		NodeID:      "game-1",
		PublicRoute: "game-1",
		Players:     []string{"u1", "u2"},
	}
	if err := state.SaveAssignment(context.Background(), assignment); err != nil {
		t.Fatalf("save assignment: %v", err)
	}
	if err := state.RegisterNode(context.Background(), coordinator.NodeRecord{
		NodeID:      "game-1",
		PublicRoute: "game-1",
		InternalURL: "http://gameplay-node:8091",
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	q := readyCoordinator(&matchCoordinator{
		store: &queueTestMatchStore{},
		state: state,
		matches: &recoverTestStore{
			runtimeMatches: map[string]matches.RuntimeMatch{"m-ended": {MatchID: "m-ended", State: string(contracts.MatchEnded)}},
			profiles:       map[string]profiles.Profile{"u1": {UserID: "u1", DisplayName: "u1", MMR: 1000}},
		},
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	queueEcho := echo.New()
	queueEcho.HideBanner = true
	queueEcho.GET("/queue", q.queue)
	srv := httptest.NewServer(queueEcho)
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial(queueWSURL(srv.URL)+"/queue", http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if err != nil {
		t.Fatalf("dial queue ws: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	event := readQueueEvent(t, conn)
	if event["type"] != "queue_status" {
		t.Fatalf("unexpected event type: %#v", event["type"])
	}
	if _, ok, err := state.GetAssignmentByUser(context.Background(), "u1"); err != nil {
		t.Fatalf("get assignment after queue: %v", err)
	} else if ok {
		t.Fatalf("assignment was not cleared")
	}
}

func TestQueueAllowsDuelWhenSingleplayerIsActive(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	state := coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	gameplay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(gameplay.Close)
	assignment := coordinator.Assignment{
		MatchID:     "solo-123",
		Mode:        contracts.ModeSingleplayer,
		NodeID:      "game-1",
		PublicRoute: "game-1",
		Players:     []string{"u1"},
	}
	if err := state.SaveAssignment(context.Background(), assignment); err != nil {
		t.Fatalf("save assignment: %v", err)
	}
	if err := state.RegisterNode(context.Background(), coordinator.NodeRecord{
		NodeID:      "game-1",
		PublicRoute: "game-1",
		InternalURL: gameplay.URL,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	q := readyCoordinator(&matchCoordinator{
		store: &queueTestMatchStore{},
		state: state,
		matches: &recoverTestStore{
			profiles: map[string]profiles.Profile{"u1": {UserID: "u1", DisplayName: "u1", MMR: 1000}},
		},
		httpClient: &http.Client{Timeout: time.Second},
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	queueEcho := echo.New()
	queueEcho.HideBanner = true
	queueEcho.GET("/queue", q.queue)
	srv := httptest.NewServer(queueEcho)
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial(queueWSURL(srv.URL)+"/queue", http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if err != nil {
		t.Fatalf("dial queue ws: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	event := readQueueEvent(t, conn)
	if event["type"] != "queue_status" {
		t.Fatalf("unexpected event type: %#v", event["type"])
	}
	payload, _ := event["payload"].(map[string]any)
	if payload["status"] != "queued" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestQueueRejectsGuestAccount(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := readyCoordinator(&matchCoordinator{
		store: &recoverTestMatchStore{},
		state: coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second),
		matches: &recoverTestStore{
			accountTypes: map[string]string{"guest-1": "guest"},
		},
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "guest-1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	queueEcho := echo.New()
	queueEcho.HideBanner = true
	queueEcho.GET("/queue", q.queue)
	srv := httptest.NewServer(queueEcho)
	t.Cleanup(srv.Close)

	conn, resp, err := websocket.DefaultDialer.Dial(queueWSURL(srv.URL)+"/queue", http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("guest queue dial succeeded, want forbidden")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
	}
}

func TestStartPartyAllowsDuelWhenSingleplayerIsActive(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	state := coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	terminateCalled := false
	gameplay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/terminate") {
			terminateCalled = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(gameplay.Close)
	if err := state.RegisterNode(context.Background(), coordinator.NodeRecord{
		NodeID:      "game-1",
		PublicRoute: "game-1",
		InternalURL: gameplay.URL,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}
	if err := state.SaveAssignment(context.Background(), coordinator.Assignment{
		MatchID:     "solo-123",
		Mode:        contracts.ModeSingleplayer,
		NodeID:      "game-1",
		PublicRoute: "game-1",
		Players:     []string{"u1"},
	}); err != nil {
		t.Fatalf("save assignment: %v", err)
	}

	store := &recoverTestStore{
		profiles: map[string]profiles.Profile{
			"u1": {UserID: "u1", DisplayName: "Player One", MMR: 1000},
			"u2": {UserID: "u2", DisplayName: "Player Two", MMR: 1000},
		},
		parties: map[string]contracts.PartySnapshot{
			"lob-1": testParty("lob-1", "u1", "u1", "u2"),
		},
	}
	q := readyCoordinator(&matchCoordinator{
		state:      state,
		matches:    store,
		httpClient: gameplay.Client(),
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})
	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/parties/lob-1/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	dispatch(q, func(e *echo.Echo) { e.POST("/parties/:id/start", q.startParty) }, rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if rec := store.runtimeMatches["solo-123"]; rec.State != string(contracts.MatchEnded) {
		t.Fatalf("singleplayer runtime state = %#v", rec)
	}
	if !terminateCalled {
		t.Fatalf("singleplayer match was not terminated")
	}
	if snap := store.parties["lob-1"]; snap.State != contracts.PartyInMatch || snap.ActiveMatchID == "" {
		t.Fatalf("party was not moved into match: %#v", snap)
	}
}

func TestStartPartyRequiresPlayersInParty(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := readyCoordinator(&matchCoordinator{
		state: coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second),
		matches: &recoverTestStore{
			profiles: map[string]profiles.Profile{
				"u1": {UserID: "u1", DisplayName: "Player One", MMR: 1000},
				"u2": {UserID: "u2", DisplayName: "Player Two", MMR: 1000},
			},
			parties: map[string]contracts.PartySnapshot{
				"lob-1": testParty("lob-1", "u1", "u1", "u2"),
			},
		},
		redis:      rdb,
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})
	q.touchPartyPresence("lob-1", "u1", "conn-1")

	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/parties/lob-1/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	dispatch(q, func(e *echo.Echo) { e.POST("/parties/:id/start", q.startParty) }, rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); !strings.Contains(body, "all players must be in the party") || !strings.Contains(body, "Player 2") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestStartPartyActiveDuelConflictNamesPlayerAndMatch(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	state := coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	gameplay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(gameplay.Close)
	if err := state.RegisterNode(context.Background(), coordinator.NodeRecord{
		NodeID:      "game-1",
		PublicRoute: "game-1",
		InternalURL: gameplay.URL,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}
	if err := state.SaveAssignment(context.Background(), coordinator.Assignment{
		MatchID:     "m-existing",
		Mode:        contracts.ModeDuel,
		NodeID:      "game-1",
		PublicRoute: "game-1",
		Players:     []string{"u2", "u3"},
	}); err != nil {
		t.Fatalf("save assignment: %v", err)
	}

	q := readyCoordinator(&matchCoordinator{
		state: state,
		matches: &recoverTestStore{
			profiles: map[string]profiles.Profile{
				"u1": {UserID: "u1", DisplayName: "Player One", MMR: 1000},
				"u2": {UserID: "u2", DisplayName: "Player Two", MMR: 1000},
			},
			parties: map[string]contracts.PartySnapshot{
				"lob-1": testParty("lob-1", "u1", "u1", "u2"),
			},
		},
		httpClient: gameplay.Client(),
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})
	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/parties/lob-1/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	dispatch(q, func(e *echo.Echo) { e.POST("/parties/:id/start", q.startParty) }, rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{"Player 2", "u2", "duel", "m-existing"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body %q does not contain %q", body, want)
		}
	}
}

func TestQueueClearsEndedQueuedMatch(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	state := coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second)
	match := contracts.MatchFound{
		MatchID: "m-ended",
		Players: []string{"u1", "u2"},
		Profiles: map[string]contracts.PlayerProfile{
			"u1": {UserID: "u1", DisplayName: "u1", MMR: 1000},
			"u2": {UserID: "u2", DisplayName: "u2", MMR: 1000},
		},
		MapScope: "world",
	}
	rawMatch, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("marshal match: %v", err)
	}
	if err := rdb.Set(context.Background(), "queue:registered:ticket:u1", `{"userId":"u1"}`, 30*time.Second).Err(); err != nil {
		t.Fatalf("set ticket: %v", err)
	}
	if err := rdb.ZAdd(context.Background(), "queue:registered:pool", redis.Z{Score: 1000, Member: "u1"}).Err(); err != nil {
		t.Fatalf("add pool: %v", err)
	}
	if err := rdb.Set(context.Background(), "queue:registered:match:u1", rawMatch, 2*time.Minute).Err(); err != nil {
		t.Fatalf("set queue match u1: %v", err)
	}
	if err := rdb.Set(context.Background(), "queue:registered:match:u2", rawMatch, 2*time.Minute).Err(); err != nil {
		t.Fatalf("set queue match u2: %v", err)
	}

	q := readyCoordinator(&matchCoordinator{
		store:      &staleQueuePollStore{match: &match},
		state:      state,
		matches:    &recoverTestStore{runtimeMatches: map[string]matches.RuntimeMatch{"m-ended": {MatchID: "m-ended", State: string(contracts.MatchEnded)}}},
		redis:      rdb,
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	queueEcho := echo.New()
	queueEcho.HideBanner = true
	queueEcho.GET("/queue", q.queue)
	srv := httptest.NewServer(queueEcho)
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial(queueWSURL(srv.URL)+"/queue", http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	if err != nil {
		t.Fatalf("dial queue ws: %v", err)
	}
	event := readQueueEvent(t, conn)
	if event["type"] != "queue_status" {
		t.Fatalf("unexpected event type: %#v", event["type"])
	}
	_ = conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		val1, err1 := mr.Get("queue:registered:match:u1")
		val2, err2 := mr.Get("queue:registered:match:u2")
		if (err1 != nil || val1 == "") && (err2 != nil || val2 == "") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if val, err := mr.Get("queue:registered:match:u1"); err == nil && val != "" {
		t.Fatalf("queue:registered:match:u1 was not cleared")
	}
	if val, err := mr.Get("queue:registered:match:u2"); err == nil && val != "" {
		t.Fatalf("queue:registered:match:u2 was not cleared")
	}
}

func TestHeartbeatReturnsQueueStatus(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	heartbeatStore := &heartbeatTestStore{status: matchstore.QueuePresenceMissing}
	q := readyCoordinator(&matchCoordinator{
		store: heartbeatStore,
		state: coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second),
		matches: &recoverTestStore{
			profiles: map[string]profiles.Profile{"u1": {UserID: "u1", DisplayName: "u1", MMR: 1000}},
		},
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "u1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/queue/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	dispatch(q, func(e *echo.Echo) { e.POST("/queue/heartbeat", q.heartbeat) }, rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != matchstore.QueuePresenceMissing {
		t.Fatalf("status = %q", payload["status"])
	}
	if heartbeatStore.pool != matchstore.QueuePoolRegistered {
		t.Fatalf("heartbeat pool = %q, want %q", heartbeatStore.pool, matchstore.QueuePoolRegistered)
	}
}

func TestHeartbeatRejectsGuestAccount(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := readyCoordinator(&matchCoordinator{
		store: &heartbeatTestStore{status: matchstore.QueuePresenceMissing},
		state: coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second),
		matches: &recoverTestStore{
			accountTypes: map[string]string{"guest-1": "guest"},
		},
		appSecret:  []byte("0123456789abcdef0123456789abcdef"),
		ticketAuth: []byte("abcdef0123456789abcdef0123456789"),
		internal:   "secret",
		metrics:    observability.NewAPIMetrics(),
	})

	token, err := auth.IssueAppAccessToken(q.appSecret, "guest-1", "sess-1", 15*time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/queue/heartbeat", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	dispatch(q, func(e *echo.Echo) { e.POST("/queue/heartbeat", q.heartbeat) }, rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}
