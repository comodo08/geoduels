package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"geoduels/pkg/contracts"
	"geoduels/pkg/maintenance"
)

const defaultLobbyChangelog = `
### Changes

- **Mobile fixed**
- **Accurate "Online players"**
- Intuitive reconnects
- Improved stability on bad networks
- Upgraded server hardware

### A personal message

I never imagined being able to play against real people in my own game, where you get matchmaked in under 10 seconds...

It's just surreal. And you guys made it possible.

Thank you everyone! And keep Dueling ⚔️

---

_Posted on March 19, 2026 by sourcelocation_
`

var defaultLobbyChangelogContent = LobbyChangelogContent{
	Eyebrow:  "Latest News",
	Title:    "GeoDuels v1.1",
	Markdown: strings.TrimSpace(defaultLobbyChangelog),
}

func (a *api) adminIdentity(r *http.Request) (Identity, error) {
	identity, err := a.authenticatedIdentity(r)
	if err != nil {
		return Identity{}, err
	}
	if identity.IsBanned || !identity.IsAdmin {
		return Identity{}, errors.New("forbidden")
	}
	return identity, nil
}

func (a *api) moderatorIdentity(r *http.Request) (Identity, error) {
	identity, err := a.authenticatedIdentity(r)
	if err != nil {
		return Identity{}, err
	}
	if identity.IsBanned || (!identity.IsAdmin && !identity.IsModerator) {
		return Identity{}, errors.New("forbidden")
	}
	return identity, nil
}

func (a *api) adminBootstrap(c echo.Context) error {
	r := c.Request()
	claims, identity, err := a.authenticatedAccount(r)
	if err != nil {
		return plainTextError(c, http.StatusUnauthorized, "identity not found")
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	if email == "" {
		return plainTextError(c, http.StatusForbidden, "email required")
	}
	if _, ok := a.adminBootstrapEmails[email]; !ok {
		return plainTextError(c, http.StatusForbidden, "not allowlisted")
	}
	if !identity.IsAdmin {
		if err := a.accounts.SetUserAdmin(identity.Sub, true); err != nil {
			return plainTextError(c, http.StatusInternalServerError, "failed to promote account")
		}
		identity, err = a.accounts.GetIdentity(claims.Sub)
		if err != nil {
			return plainTextError(c, http.StatusUnauthorized, "identity not found")
		}
	}
	payload, err := a.issueAuthSessionPayload(identity, claims.SessionID)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "issue session failed")
	}
	return writeJSON(c, payload)
}

func (a *api) adminPlayers(c echo.Context) error {
	r := c.Request()
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	players, err := a.admin.SearchPlayers(c.QueryParam("query"), 30)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "player search unavailable")
	}
	if !identity.IsAdmin {
		sanitizeAdminPlayerSummariesForModerator(players)
	}
	return writeJSON(c, map[string]any{"players": players})
}

func (a *api) adminPlayerDetail(c echo.Context) error {
	r := c.Request()
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	detail, err := a.admin.GetAdminPlayerDetail(a.resolveEntityID("user", c.Param("id")))
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return plainTextError(c, http.StatusNotFound, "player not found")
		}
		return plainTextError(c, http.StatusInternalServerError, "player detail unavailable")
	}
	if !identity.IsAdmin {
		sanitizeAdminPlayerSummaryForModerator(&detail.Player)
	}
	return writeJSON(c, detail)
}

func (a *api) adminPlayerMatches(c echo.Context) error {
	r := c.Request()
	if _, err := a.moderatorIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	matches, err := a.matchStore.ListPlayerMatchHistory(a.resolveEntityID("user", c.Param("id")), 50)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "match history unavailable")
	}
	return writeJSON(c, map[string]any{"matches": matches})
}

func (a *api) adminMatchChat(c echo.Context) error {
	r := c.Request()
	if _, err := a.moderatorIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	limit := 200
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return plainTextError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = parsed
	}
	matchID := a.resolveEntityID("match", c.Param("id"))
	messages, err := a.chatStore.ListChatMessages("match:"+matchID, limit)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "chat log unavailable")
	}
	return writeJSON(c, map[string]any{"messages": messages})
}

func sanitizeAdminPlayerSummariesForModerator(players []AdminPlayerSummary) {
	for i := range players {
		sanitizeAdminPlayerSummaryForModerator(&players[i])
	}
}

func sanitizeAdminPlayerSummaryForModerator(player *AdminPlayerSummary) {
	if player == nil {
		return
	}
	player.Email = ""
	player.LastIPAddress = ""
	player.Identities = nil
}

func (a *api) moderatorSubject(c echo.Context) error {
	r := c.Request()
	if _, err := a.moderatorIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	profile, err := a.moderation.ListSubjectModerationProfile(a.resolveEntityID("user", c.Param("userId")))
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "moderation subject unavailable")
	}
	sanitizeAdminPlayerSummaryForModerator(&profile.Player)
	return writeJSON(c, profile)
}

func (a *api) moderatorSignals(c echo.Context) error {
	r := c.Request()
	if _, err := a.moderatorIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	signals, err := a.moderation.ListModerationSignals(100)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "moderation signals unavailable")
	}
	return writeJSON(c, map[string]any{"signals": signals})
}

func (a *api) adminBanPlayer(c echo.Context) error {
	r := c.Request()
	admin, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	a.banPlayerForCheating(c, c.Param("id"), admin.Sub)
	return nil
}

func (a *api) moderatorSubjectCheatingBan(c echo.Context) error {
	r := c.Request()
	moderator, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	a.banPlayerForCheating(c, c.Param("userId"), moderator.Sub)
	return nil
}

func (a *api) moderatorSubjectUnban(c echo.Context) error {
	r := c.Request()
	moderator, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if err := a.moderation.SetPlayerBan(a.resolveEntityID("user", c.Param("userId")), req.Reason, moderator.Sub, false); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to unban player")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) moderatorSubjectMute(c echo.Context) error {
	r := c.Request()
	moderator, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Reason        string `json:"reason"`
		DurationHours int    `json:"durationHours"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if req.DurationHours <= 0 {
		req.DurationHours = 7 * 24
	}
	if err := a.moderation.SetPlayerMute(a.resolveEntityID("user", c.Param("userId")), c.Param("kind"), req.Reason, moderator.Sub, time.Now().Add(time.Duration(req.DurationHours)*time.Hour), true); err != nil {
		return plainTextError(c, http.StatusBadRequest, "failed to mute player")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) moderatorSubjectUnmute(c echo.Context) error {
	r := c.Request()
	moderator, err := a.moderatorIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if err := a.moderation.SetPlayerMute(a.resolveEntityID("user", c.Param("userId")), c.Param("kind"), "", moderator.Sub, time.Time{}, false); err != nil {
		return plainTextError(c, http.StatusBadRequest, "failed to unmute player")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) banPlayerForCheating(c echo.Context, rawUserID, actorUserID string) {
	r := c.Request()
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		_ = plainTextError(c, http.StatusBadRequest, "invalid payload")
		return
	}
	summary, err := a.moderation.BanPlayerForCheating(a.resolveEntityID("user", rawUserID), req.Reason, actorUserID)
	if err != nil {
		_ = plainTextError(c, http.StatusInternalServerError, "failed to ban player")
		return
	}
	_ = writeJSON(c, summary)
}

func (a *api) adminUnbanPlayer(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if err := a.moderation.SetPlayerBan(a.resolveEntityID("user", c.Param("id")), "", admin.Sub, false); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to unban player")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminCommunityPardonPreview(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	summary, err := a.moderation.PreviewCommunityPardon(7 * 24 * time.Hour)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to preview community pardon")
	}
	return writeJSON(c, summary)
}

func (a *api) adminCommunityPardon(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := decodeJSONBody(r, &req); err != nil || !req.Confirm {
		return plainTextError(c, http.StatusBadRequest, "explicit confirmation required")
	}
	summary, err := a.moderation.PardonBannedPlayers(7*24*time.Hour, admin.Sub)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to pardon banned players")
	}
	return writeJSON(c, summary)
}

func (a *api) adminClearReporterMute(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if err := a.moderation.ClearReporterMute(a.resolveEntityID("user", c.Param("id"))); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to unmute reporter")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminPromoteModerator(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	userID := a.resolveEntityID("user", c.Param("id"))
	if err := a.accounts.SetUserModerator(userID, true); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to promote moderator")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminDemoteModerator(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if err := a.accounts.SetUserModerator(a.resolveEntityID("user", c.Param("id")), false); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to demote moderator")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminSetMapCreatorTier(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Tier string `json:"tier"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	var tier *int
	switch strings.ToLower(strings.TrimSpace(req.Tier)) {
	case "auto":
	case "base":
		value := 0
		tier = &value
	case "trusted":
		value := 1
		tier = &value
	case "established":
		value := 2
		tier = &value
	default:
		return plainTextError(c, http.StatusBadRequest, "tier must be auto, base, trusted, or established")
	}
	quota, err := a.maps.SetMapCreatorTierOverride(a.resolveEntityID("user", c.Param("id")), tier)
	if errors.Is(err, ErrNoRows) {
		return plainTextError(c, http.StatusNotFound, "404 page not found")
	}
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to update map creator tier")
	}
	return writeJSON(c, quota)
}

func (a *api) adminListRoles(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	roles, err := a.admin.ListUserRoles()
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "roles unavailable")
	}
	return writeJSON(c, map[string]any{"roles": roles})
}

func (a *api) adminGrantRole(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if err := a.admin.GrantUserRole(a.resolveEntityID("user", req.UserID), req.Role, admin.Sub, req.Reason); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to grant role")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminRevokeRole(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if err := a.admin.RevokeUserRole(a.resolveEntityID("user", c.Param("id")), strings.TrimSpace(c.Param("role")), admin.Sub, req.Reason); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to revoke role")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminBadgeDefinitions(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	return writeJSON(c, map[string]any{"badges": a.badges.ListAdminGrantableBadges()})
}

func (a *api) adminGrantBadge(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		Nickname string `json:"nickname"`
		BadgeID  string `json:"badgeId"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		return plainTextError(c, http.StatusBadRequest, "nickname required")
	}
	if strings.TrimSpace(req.BadgeID) == "" {
		return plainTextError(c, http.StatusBadRequest, "badge id required")
	}
	badge, changed, err := a.badges.GrantBadgeToUser(nickname, req.BadgeID, admin.Sub)
	if err != nil {
		switch {
		case errors.Is(err, ErrBadgeUserNotFound):
			return plainTextError(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrBadgeUnavailable), errors.Is(err, ErrBadgeNicknameRequired):
			return plainTextError(c, http.StatusBadRequest, err.Error())
		default:
			return plainTextError(c, http.StatusInternalServerError, "failed to grant badge")
		}
	}
	return writeJSON(c, map[string]any{"badge": badge, "changed": changed})
}

func (a *api) moderatorLog(c echo.Context) error {
	r := c.Request()
	if _, err := a.moderatorIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	entries, err := a.moderation.ListModerationLog(100)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "moderation log unavailable")
	}
	return writeJSON(c, map[string]any{"log": entries})
}

func (a *api) adminListSignupIPBans(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	bans, err := a.moderation.ListSignupIPBans(100)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "ip bans unavailable")
	}
	return writeJSON(c, map[string]any{"bans": bans})
}

func (a *api) adminAddSignupIPBan(c echo.Context) error {
	r := c.Request()
	admin, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		IPAddress string `json:"ipAddress"`
		Reason    string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if err := a.moderation.AddSignupIPBan(strings.TrimSpace(req.IPAddress), req.Reason, admin.Sub); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to ban ip")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminRemoveSignupIPBan(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	ip, err := url.PathUnescape(strings.TrimSpace(c.Param("ip")))
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid ip")
	}
	if err := a.moderation.RemoveSignupIPBan(ip); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to remove ip ban")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminGetMaintenance(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	status, err := maintenance.Read(r.Context(), a.redis)
	if err != nil {
		return plainTextError(c, http.StatusBadGateway, "maintenance unavailable")
	}
	return writeJSON(c, status)
}

func (a *api) adminPutMaintenance(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if a.redis == nil {
		return plainTextError(c, http.StatusBadGateway, "redis unavailable")
	}
	var status maintenance.Status
	if err := decodeJSONBody(r, &status); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	status = status.Normalized()
	body, err := json.Marshal(status)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid maintenance status")
	}
	if err := a.redis.Set(r.Context(), maintenance.RedisKey, body, 0).Err(); err != nil {
		return plainTextError(c, http.StatusBadGateway, "failed to save maintenance")
	}
	return writeJSON(c, status)
}

func (a *api) adminClearMaintenance(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if a.redis == nil {
		return plainTextError(c, http.StatusBadGateway, "redis unavailable")
	}
	if err := a.redis.Del(r.Context(), maintenance.RedisKey).Err(); err != nil {
		return plainTextError(c, http.StatusBadGateway, "failed to clear maintenance")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *api) adminGetModerationSettings(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	settings, err := a.content.GetModerationSettings()
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "moderation settings unavailable")
	}
	return writeJSON(c, settings)
}

func (a *api) adminPutModerationSettings(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req ModerationSettings
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	webhookURL, err := normalizeDiscordWebhookURL(req.DiscordWebhookURL)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	settings := ModerationSettings{DiscordWebhookURL: webhookURL}
	if err := a.content.SetModerationSettings(settings); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to save moderation settings")
	}
	return writeJSON(c, settings)
}

func (a *api) adminGetDiscordIntegrationSettings(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	settings, err := a.content.GetDiscordIntegrationSettings()
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "discord integration settings unavailable")
	}
	return writeJSON(c, settings)
}

func (a *api) adminPutDiscordIntegrationSettings(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var settings DiscordIntegrationSettings
	if err := decodeJSONBody(r, &settings); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	for label, value := range map[string]string{
		"guild id":         settings.GuildID,
		"joins channel id": settings.JoinsChannelID,
		"1k role id":       settings.Elo1000RoleID,
		"1.5k role id":     settings.Elo1500RoleID,
		"2k role id":       settings.Elo2000RoleID,
	} {
		if err := validateOptionalDiscordSnowflake(label, value); err != nil {
			return plainTextError(c, http.StatusBadRequest, err.Error())
		}
	}
	if settings.ReconcileIntervalMinutes < 1 || settings.ReconcileIntervalMinutes > 1440 {
		return plainTextError(c, http.StatusBadRequest, "reconcile interval must be between 1 and 1440 minutes")
	}
	// Managed role history is server-owned so old configured roles can be
	// removed safely after an administrator changes a role ID.
	settings.ManagedRoleIDs = nil
	if err := a.content.SetDiscordIntegrationSettings(settings); err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to save discord integration settings")
	}
	saved, err := a.content.GetDiscordIntegrationSettings()
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "discord integration settings unavailable")
	}
	return writeJSON(c, saved)
}

func validateOptionalDiscordSnowflake(label, raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if len(value) < 17 || len(value) > 20 {
		return fmt.Errorf("%s must be a Discord ID", label)
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return fmt.Errorf("%s must be a Discord ID", label)
		}
	}
	return nil
}

func (a *api) adminGetRankedSeason(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	settings, err := a.seasons.GetRankedSeasonSettings()
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "season settings unavailable")
	}
	return writeJSON(c, settings)
}

func (a *api) adminPutRankedSeasonResetRule(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req struct {
		MonthlyResetDay int `json:"monthlyResetDay"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	settings, err := a.seasons.SetRankedSeasonResetRule(req.MonthlyResetDay)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "reset day") {
			return plainTextError(c, http.StatusBadRequest, err.Error())
		}
		return plainTextError(c, http.StatusInternalServerError, "season settings update failed")
	}
	return writeJSON(c, settings)
}

func normalizeDiscordWebhookURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > 2000 {
		return "", errors.New("discord webhook url is too long")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("discord webhook url must be an https url")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "discord.com" && host != "discordapp.com" && host != "canary.discord.com" && host != "ptb.discord.com" {
		return "", errors.New("discord webhook url must be a Discord webhook")
	}
	if !strings.HasPrefix(parsed.EscapedPath(), "/api/webhooks/") {
		return "", errors.New("discord webhook url must be a Discord webhook")
	}
	return value, nil
}

func (a *api) publicLobbyChangelog(c echo.Context) error {
	content, err := a.content.GetLobbyChangelog(defaultLobbyChangelogContent)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "changelog unavailable")
	}
	return writeJSON(c, content)
}

func (a *api) publicChangelogPosts(c echo.Context) error {
	posts, err := a.content.ListChangelogPosts(false)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "changelog unavailable")
	}
	return writeJSON(c, map[string]any{"posts": posts})
}

func (a *api) publicChangelogPost(c echo.Context) error {
	slug := strings.TrimSpace(c.Param("slug"))
	post, ok, err := a.content.GetChangelogPostBySlug(slug, true)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "changelog unavailable")
	}
	if !ok {
		return plainTextError(c, http.StatusNotFound, "404 page not found")
	}
	return writeJSON(c, post)
}

func (a *api) adminGetChangelog(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	posts, err := a.content.ListChangelogPosts(true)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "changelog unavailable")
	}
	return writeJSON(c, map[string]any{"posts": posts})
}

func normalizeChangelogPostInput(req ChangelogPostInput) (ChangelogPostInput, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Markdown = strings.TrimSpace(req.Markdown)
	req.Slug = slugifyChangelogPost(req.Slug)
	if req.Slug == "" {
		req.Slug = slugifyChangelogPost(req.Title)
	}
	if req.Title == "" {
		return ChangelogPostInput{}, errors.New("title is required")
	}
	if req.Slug == "" {
		return ChangelogPostInput{}, errors.New("slug is required")
	}
	if len(req.Slug) > 120 {
		return ChangelogPostInput{}, errors.New("slug is too long")
	}
	if len(req.Title) > 160 {
		return ChangelogPostInput{}, errors.New("title is too long")
	}
	return req, nil
}

func slugifyChangelogPost(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func (a *api) adminCreateChangelogPost(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	var req ChangelogPostInput
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	input, err := normalizeChangelogPostInput(req)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	post, err := a.content.CreateChangelogPost(input)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to save changelog")
	}
	return writeJSONStatus(c, http.StatusCreated, post)
}

func (a *api) adminUpdateChangelogPost(c echo.Context) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		return plainTextError(c, http.StatusBadRequest, "invalid post id")
	}
	var req ChangelogPostInput
	if err := decodeJSONBody(r, &req); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	input, err := normalizeChangelogPostInput(req)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	post, ok, err := a.content.UpdateChangelogPost(id, input)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to save changelog")
	}
	if !ok {
		return plainTextError(c, http.StatusNotFound, "404 page not found")
	}
	return writeJSON(c, post)
}

func (a *api) adminImportOfficialMap(c echo.Context) error {
	r := c.Request()
	identity, err := a.adminIdentity(r)
	if err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	file, closeFile, err := mapUploadFile(c)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	defer closeFile()
	input := OfficialMapImportInput{
		MapKey:             r.FormValue("mapKey"),
		DisplayName:        r.FormValue("displayName"),
		Description:        r.FormValue("description"),
		Visibility:         r.FormValue("visibility"),
		Difficulty:         r.FormValue("difficulty"),
		ThumbnailKey:       r.FormValue("thumbnailKey"),
		ThumbnailVariant:   atoiDefault(r.FormValue("thumbnailVariant"), 1),
		OfficialRegionType: r.FormValue("officialRegionType"),
		OfficialRegionCode: r.FormValue("officialRegionCode"),
	}
	item, err := a.maps.ImportOfficialMap(identity.Sub, input, file)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	return writeJSON(c, item)
}

func (a *api) adminUploadCurrentMap(c echo.Context) error {
	return a.uploadMap(c, contracts.MapKeyMoving)
}

func (a *api) adminUploadMap(c echo.Context) error {
	mapKey := strings.TrimSpace(c.Param("mapKey"))
	if mapKey != contracts.MapKeyMoving && mapKey != contracts.MapKeyNMPZ {
		return plainTextError(c, http.StatusBadRequest, "unsupported map key")
	}
	return a.uploadMap(c, mapKey)
}

func (a *api) uploadMap(c echo.Context, mapKey string) error {
	r := c.Request()
	if _, err := a.adminIdentity(r); err != nil {
		return plainTextError(c, http.StatusForbidden, "forbidden")
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return plainTextError(c, http.StatusBadRequest, "invalid multipart form")
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, "file is required")
	}
	defer file.Close()
	dataset, err := readUploadedFile(file, header)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, "failed to read file")
	}
	summary, err := a.maps.ReplaceMapLocations(mapKey, mapKey, dataset)
	if err != nil {
		return plainTextError(c, http.StatusBadRequest, err.Error())
	}
	return writeJSON(c, summary)
}

func readUploadedFile(file multipart.File, _ *multipart.FileHeader) ([]byte, error) {
	return io.ReadAll(file)
}

var _ contracts.MapImportSummary
