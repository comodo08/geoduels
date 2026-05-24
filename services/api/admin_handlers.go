package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"

	"geoduels/pkg/contracts"
	"geoduels/pkg/maintenance"
	"geoduels/pkg/persistence"
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

var defaultLobbyChangelogContent = persistence.LobbyChangelogContent{
	Eyebrow:  "Latest News",
	Title:    "GeoDuels v1.1",
	Markdown: strings.TrimSpace(defaultLobbyChangelog),
}

func (a *api) adminIdentity(r *http.Request) (persistence.Identity, error) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		return persistence.Identity{}, err
	}
	identity, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		return persistence.Identity{}, err
	}
	if !identity.IsAdmin {
		return persistence.Identity{}, errors.New("forbidden")
	}
	return identity, nil
}

func (a *api) moderatorIdentity(r *http.Request) (persistence.Identity, error) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		return persistence.Identity{}, err
	}
	identity, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		return persistence.Identity{}, err
	}
	if !identity.IsAdmin && !identity.IsModerator {
		return persistence.Identity{}, errors.New("forbidden")
	}
	return identity, nil
}

func (a *api) adminBootstrap(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	identity, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		http.Error(w, "identity not found", http.StatusUnauthorized)
		return
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	if email == "" {
		http.Error(w, "email required", http.StatusForbidden)
		return
	}
	if _, ok := a.adminBootstrapEmails[email]; !ok {
		http.Error(w, "not allowlisted", http.StatusForbidden)
		return
	}
	if !identity.IsAdmin {
		if err := a.store.SetUserAdmin(identity.Sub, true); err != nil {
			http.Error(w, "failed to promote account", http.StatusInternalServerError)
			return
		}
		identity, err = a.store.GetIdentity(claims.Sub)
		if err != nil {
			http.Error(w, "identity not found", http.StatusUnauthorized)
			return
		}
	}
	payload, err := a.issueAuthSessionPayload(identity, claims.SessionID)
	if err != nil {
		http.Error(w, "issue session failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (a *api) adminPlayers(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	players, err := a.store.SearchPlayers(r.URL.Query().Get("query"), 30)
	if err != nil {
		http.Error(w, "player search unavailable", http.StatusInternalServerError)
		return
	}
	if !identity.IsAdmin {
		sanitizeAdminPlayerSummariesForModerator(players)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"players": players})
}

func (a *api) adminPlayerDetail(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	detail, err := a.store.GetAdminPlayerDetail(strings.TrimSpace(mux.Vars(r)["id"]))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "player not found", http.StatusNotFound)
			return
		}
		http.Error(w, "player detail unavailable", http.StatusInternalServerError)
		return
	}
	if !identity.IsAdmin {
		sanitizeAdminPlayerSummaryForModerator(&detail.Player)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (a *api) adminModerationCases(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if view != "" {
		status = view
	}
	switch status {
	case "active":
		status = "active:" + identity.Sub
	case "archive":
		status = "archived"
	case "mine":
		status = "mine:" + identity.Sub
	}
	cases, err := a.store.ListModerationCases(status, 50)
	if err != nil {
		http.Error(w, "moderation cases unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"cases": cases,
	})
}

func (a *api) adminPlayerMatches(w http.ResponseWriter, r *http.Request) {
	if _, err := a.moderatorIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	matches, err := a.store.ListPlayerMatchHistory(strings.TrimSpace(mux.Vars(r)["id"]), 50)
	if err != nil {
		http.Error(w, "match history unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"matches": matches})
}

func (a *api) adminMatchChat(w http.ResponseWriter, r *http.Request) {
	if _, err := a.moderatorIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	matchID := strings.TrimSpace(mux.Vars(r)["id"])
	messages, err := a.store.ListChatMessages("match:"+matchID, limit)
	if err != nil {
		http.Error(w, "chat log unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": messages})
}

func (a *api) adminModerationCase(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	caseID, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)["id"]), 10, 64)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}
	detail, err := a.store.GetModerationCase(caseID)
	if err != nil {
		http.Error(w, "moderation case unavailable", http.StatusInternalServerError)
		return
	}
	if !identity.IsAdmin {
		sanitizeModerationCaseDetailForModerator(&detail)
	}
	_ = json.NewEncoder(w).Encode(detail)
}

func (a *api) adminModerationCaseClaim(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	caseID, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)["id"]), 10, 64)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}
	detail, err := a.store.ClaimModerationCase(caseID, identity.Sub)
	if err != nil {
		http.Error(w, "failed to claim moderation case", http.StatusInternalServerError)
		return
	}
	if !identity.IsAdmin {
		sanitizeModerationCaseDetailForModerator(&detail)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (a *api) adminModerationCaseRelease(w http.ResponseWriter, r *http.Request) {
	identity, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	caseID, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)["id"]), 10, 64)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}
	detail, err := a.store.ReleaseModerationCase(caseID, identity.Sub)
	if err != nil {
		http.Error(w, "failed to release moderation case", http.StatusInternalServerError)
		return
	}
	if !identity.IsAdmin {
		sanitizeModerationCaseDetailForModerator(&detail)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (a *api) adminModerationCaseAction(w http.ResponseWriter, r *http.Request) {
	admin, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	caseID, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)["id"]), 10, 64)
	if err != nil {
		http.Error(w, "invalid case id", http.StatusBadRequest)
		return
	}
	var req struct {
		ActionType string `json:"actionType"`
		Reason     string `json:"reason"`
		Status     string `json:"status"`
		AssignedTo string `json:"assignedTo"`
		MuteUserID string `json:"muteUserId"`
		MuteUntil  string `json:"muteUntil"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	var muteUntil time.Time
	if strings.TrimSpace(req.MuteUntil) != "" {
		muteUntil, err = time.Parse(time.RFC3339, strings.TrimSpace(req.MuteUntil))
		if err != nil {
			http.Error(w, "invalid muteUntil", http.StatusBadRequest)
			return
		}
	}
	detail, err := a.store.AddModerationCaseAction(persistence.ModerationCaseActionParams{
		CaseID:      caseID,
		ActorUserID: admin.Sub,
		ActionType:  req.ActionType,
		Reason:      req.Reason,
		Status:      req.Status,
		AssignedTo:  req.AssignedTo,
		MuteUserID:  req.MuteUserID,
		MuteUntil:   muteUntil,
	})
	if err != nil {
		http.Error(w, "failed to update moderation case", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func sanitizeModerationCaseDetailForModerator(detail *persistence.ModerationCaseDetail) {
	if detail == nil || detail.TargetPlayer == nil {
		return
	}
	detail.TargetPlayer.Email = ""
	detail.TargetPlayer.LastIPAddress = ""
	detail.TargetPlayer.Identities = nil
}

func sanitizeAdminPlayerSummariesForModerator(players []persistence.AdminPlayerSummary) {
	for i := range players {
		sanitizeAdminPlayerSummaryForModerator(&players[i])
	}
}

func sanitizeAdminPlayerSummaryForModerator(player *persistence.AdminPlayerSummary) {
	if player == nil {
		return
	}
	player.Email = ""
	player.LastIPAddress = ""
	player.Identities = nil
}

func (a *api) adminDebugTestReports(w http.ResponseWriter, r *http.Request) {
	admin, err := a.adminIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		ReportedUserID string `json:"reportedUserId"`
		Count          int    `json:"count"`
		Category       string `json:"category"`
		Reason         string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	result, err := a.store.CreateDebugModerationReports(persistence.CreateDebugModerationReportsParams{
		ReportedUserID: req.ReportedUserID,
		Count:          req.Count,
		Category:       req.Category,
		Reason:         req.Reason,
		CreatedBy:      admin.Sub,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (a *api) adminBanPlayer(w http.ResponseWriter, r *http.Request) {
	admin, err := a.moderatorIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	summary, err := a.store.BanPlayerForCheating(strings.TrimSpace(mux.Vars(r)["id"]), req.Reason, admin.Sub)
	if err != nil {
		http.Error(w, "failed to ban player", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}

func (a *api) adminUnbanPlayer(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := a.store.SetPlayerBan(strings.TrimSpace(mux.Vars(r)["id"]), "", false); err != nil {
		http.Error(w, "failed to unban player", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminClearReporterMute(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := a.store.ClearReporterMute(strings.TrimSpace(mux.Vars(r)["id"])); err != nil {
		http.Error(w, "failed to unmute reporter", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminPromoteModerator(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	userID := strings.TrimSpace(mux.Vars(r)["id"])
	if err := a.store.SetUserModerator(userID, true); err != nil {
		http.Error(w, "failed to promote moderator", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminDemoteModerator(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := a.store.SetUserModerator(strings.TrimSpace(mux.Vars(r)["id"]), false); err != nil {
		http.Error(w, "failed to demote moderator", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminListRoles(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	roles, err := a.store.ListUserRoles()
	if err != nil {
		http.Error(w, "roles unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"roles": roles})
}

func (a *api) adminGrantRole(w http.ResponseWriter, r *http.Request) {
	admin, err := a.adminIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := a.store.GrantUserRole(req.UserID, req.Role, admin.Sub, req.Reason); err != nil {
		http.Error(w, "failed to grant role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminRevokeRole(w http.ResponseWriter, r *http.Request) {
	admin, err := a.adminIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := a.store.RevokeUserRole(strings.TrimSpace(mux.Vars(r)["id"]), strings.TrimSpace(mux.Vars(r)["role"]), admin.Sub, req.Reason); err != nil {
		http.Error(w, "failed to revoke role", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminEnforcementActions(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actions, err := a.store.ListEnforcementActions(100)
	if err != nil {
		http.Error(w, "enforcement actions unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"actions": actions})
}

func (a *api) adminListSignupIPBans(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	bans, err := a.store.ListSignupIPBans(100)
	if err != nil {
		http.Error(w, "ip bans unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"bans": bans})
}

func (a *api) adminAddSignupIPBan(w http.ResponseWriter, r *http.Request) {
	admin, err := a.adminIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		IPAddress string `json:"ipAddress"`
		Reason    string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := a.store.AddSignupIPBan(strings.TrimSpace(req.IPAddress), req.Reason, admin.Sub); err != nil {
		http.Error(w, "failed to ban ip", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminRemoveSignupIPBan(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ip, err := url.PathUnescape(strings.TrimSpace(mux.Vars(r)["ip"]))
	if err != nil {
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}
	if err := a.store.RemoveSignupIPBan(ip); err != nil {
		http.Error(w, "failed to remove ip ban", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminGetMaintenance(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	status, err := maintenance.Read(r.Context(), a.redis)
	if err != nil {
		http.Error(w, "maintenance unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (a *api) adminPutMaintenance(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if a.redis == nil {
		http.Error(w, "redis unavailable", http.StatusBadGateway)
		return
	}
	var status maintenance.Status
	if err := decodeJSONBody(r, &status); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	status = status.Normalized()
	body, err := json.Marshal(status)
	if err != nil {
		http.Error(w, "invalid maintenance status", http.StatusBadRequest)
		return
	}
	if err := a.redis.Set(r.Context(), maintenance.RedisKey, body, 0).Err(); err != nil {
		http.Error(w, "failed to save maintenance", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (a *api) adminClearMaintenance(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if a.redis == nil {
		http.Error(w, "redis unavailable", http.StatusBadGateway)
		return
	}
	if err := a.redis.Del(r.Context(), maintenance.RedisKey).Err(); err != nil {
		http.Error(w, "failed to clear maintenance", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminGetModerationSettings(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := a.store.GetModerationSettings()
	if err != nil {
		http.Error(w, "moderation settings unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (a *api) adminPutModerationSettings(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req persistence.ModerationSettings
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	webhookURL, err := normalizeDiscordWebhookURL(req.DiscordWebhookURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	settings := persistence.ModerationSettings{DiscordWebhookURL: webhookURL}
	if err := a.store.SetModerationSettings(settings); err != nil {
		http.Error(w, "failed to save moderation settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (a *api) adminGetRankedSeason(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := a.store.GetRankedSeasonSettings()
	if err != nil {
		http.Error(w, "season settings unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (a *api) adminRolloverRankedSeason(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		NextSeasonID string `json:"nextSeasonId"`
	}
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	result, err := a.store.RolloverRankedSeason(req.NextSeasonID)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "season") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "season rollover failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
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

func (a *api) publicLobbyChangelog(w http.ResponseWriter, r *http.Request) {
	content, err := a.store.GetLobbyChangelog(defaultLobbyChangelogContent)
	if err != nil {
		http.Error(w, "changelog unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(content)
}

func (a *api) adminGetChangelog(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	a.publicLobbyChangelog(w, r)
}

func (a *api) adminPutChangelog(w http.ResponseWriter, r *http.Request) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Eyebrow  string `json:"eyebrow"`
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := a.store.SetLobbyChangelog(persistence.LobbyChangelogContent{
		Eyebrow:  req.Eyebrow,
		Title:    req.Title,
		Markdown: req.Markdown,
	}); err != nil {
		http.Error(w, "failed to save changelog", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminUploadCurrentMap(w http.ResponseWriter, r *http.Request) {
	a.uploadMap(w, r, contracts.MapKeyMoving)
}

func (a *api) adminUploadMap(w http.ResponseWriter, r *http.Request) {
	mapKey := strings.TrimSpace(mux.Vars(r)["mapKey"])
	if mapKey != contracts.MapKeyMoving && mapKey != contracts.MapKeyNMPZ {
		http.Error(w, "unsupported map key", http.StatusBadRequest)
		return
	}
	a.uploadMap(w, r, mapKey)
}

func (a *api) uploadMap(w http.ResponseWriter, r *http.Request, mapKey string) {
	if _, err := a.adminIdentity(r); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	dataset, err := readUploadedFile(file, header)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusBadRequest)
		return
	}
	summary, err := a.store.ActivateMapRevision(mapKey, mapKey, dataset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}

func readUploadedFile(file multipart.File, _ *multipart.FileHeader) ([]byte, error) {
	return io.ReadAll(file)
}

var _ contracts.MapRevisionSummary
