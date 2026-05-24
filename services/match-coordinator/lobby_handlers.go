package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
	"geoduels/pkg/lobbyevents"
	"geoduels/pkg/matchlaunch"
	"geoduels/pkg/observability"
	"geoduels/pkg/sessionpolicy"
)

var lobbyUpgrader = websocket.Upgrader{CheckOrigin: wsOriginAllowed}

const (
	defaultLobbyTTL        = 2 * time.Hour
	lobbyPresenceOnlineTTL = 15 * time.Second
	lobbyPresenceAwayTTL   = 60 * time.Second
	lobbyPresenceTTL       = 90 * time.Second
)

func (q *matchCoordinator) createLobby(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	var req contracts.LobbyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = contracts.ModeDuel
	}
	if !contracts.IsPrivatePartyMode(mode) {
		http.Error(w, "unsupported lobby mode", http.StatusBadRequest)
		return
	}
	snap, err := q.persist.CreateLobby(userID, mode, req.MapScope, defaultLobbyTTL)
	if err != nil {
		observability.Log("error", "create lobby failed", map[string]any{"userId": userID, "mode": string(mode), "error": err.Error()})
		http.Error(w, "lobby unavailable", http.StatusInternalServerError)
		return
	}
	if req.Config.Ruleset != "" || req.Config.RoundTimerMode != "" || req.Config.RoundTimeLimitMS > 0 || req.Config.PressureTimeLimitMS > 0 {
		cfg, err := q.lobbySettings.Save(r.Context(), snap.ID, req.Config)
		if err != nil {
			observability.Log("error", "create lobby settings save failed", map[string]any{"userId": userID, "lobbyId": snap.ID, "error": err.Error()})
			http.Error(w, "lobby unavailable", http.StatusInternalServerError)
			return
		}
		snap.Config = cfg
	} else {
		snap = q.lobbySettings.Apply(r.Context(), snap)
	}
	if q.touchLobbyPresence(snap.ID, userID, "") {
		q.publishLobbyChanged(r.Context(), snap.ID)
	}
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) getLobby(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(mux.Vars(r)["code"])
	snap, ok, err := q.persist.GetLobbyByInviteCode(code)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) joinLobby(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	code := strings.TrimSpace(mux.Vars(r)["code"])
	snap, found, err := q.persist.GetLobbyByInviteCode(code)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}
	snap, err = q.persist.JoinLobby(snap.ID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.touchLobbyPresence(snap.ID, userID, "")
	q.applyLobbyPresence(&snap)
	q.publishLobbyChanged(r.Context(), snap.ID)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) leaveLobby(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	snap, err := q.persist.LeaveLobby(id, userID)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadRequest)
		return
	}
	q.publishLobbyChanged(r.Context(), snap.ID)
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) kickLobbyMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	var req contracts.LobbyMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	snap, err := q.persist.KickLobbyMember(id, userID, strings.TrimSpace(req.UserID))
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadRequest)
		return
	}
	q.publishLobbyChanged(r.Context(), snap.ID)
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) transferLobbyOwner(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	var req contracts.LobbyMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	snap, err := q.persist.TransferLobbyOwner(id, userID, strings.TrimSpace(req.UserID))
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadRequest)
		return
	}
	q.publishLobbyChanged(r.Context(), snap.ID)
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) updateLobbyTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	var req contracts.LobbyTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	snap, err := q.persist.SetLobbyMemberTeam(id, userID, strings.TrimSpace(req.TeamID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q.publishLobbyChanged(r.Context(), snap.ID)
	snap = q.lobbySettings.Apply(r.Context(), snap)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) updateLobbySettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := q.requirePlayableUser(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	snap, found, err := q.persist.GetLobbyByID(id)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}
	if snap.OwnerUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if snap.State != contracts.LobbyOpen {
		http.Error(w, "lobby settings are locked", http.StatusConflict)
		return
	}
	var req struct {
		Mode   contracts.MatchMode   `json:"mode"`
		Config contracts.MatchConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.Mode != "" && !contracts.IsPrivatePartyMode(req.Mode) {
		http.Error(w, "unsupported lobby mode", http.StatusBadRequest)
		return
	}
	if req.Mode != "" && req.Mode != snap.Mode {
		if err := q.persist.SetLobbyMode(snap.ID, req.Mode); err != nil {
			http.Error(w, "lobby settings unavailable", http.StatusBadGateway)
			return
		}
		snap.Mode = req.Mode
	}
	cfg, err := q.lobbySettings.Save(r.Context(), snap.ID, req.Config)
	if err != nil {
		http.Error(w, "lobby settings unavailable", http.StatusBadGateway)
		return
	}
	snap.Config = cfg
	q.publishLobbyChanged(r.Context(), snap.ID)
	q.applyLobbyPresence(&snap)
	q.writeJSON(w, snap)
}

func (q *matchCoordinator) lobbyPresence(w http.ResponseWriter, r *http.Request) {
	claims, err := q.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	lobbyID := strings.TrimSpace(mux.Vars(r)["id"])
	snap, ok, err := q.persist.GetLobbyByID(lobbyID)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadGateway)
		return
	}
	if !ok || !lobbyHasMember(snap, claims.Sub) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if q.touchLobbyPresence(lobbyID, claims.Sub, "") {
		q.publishLobbyChanged(r.Context(), lobbyID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (q *matchCoordinator) lobbyWS(w http.ResponseWriter, r *http.Request) {
	claims, err := q.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	lobbyID := strings.TrimSpace(mux.Vars(r)["id"])
	snap, ok, err := q.persist.GetLobbyByID(lobbyID)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadGateway)
		return
	}
	if !ok || !lobbyHasMember(snap, claims.Sub) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	snap = q.lobbySettings.Apply(r.Context(), snap)
	conn, err := lobbyUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	connID := strconvTimeID()
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
		if q.touchLobbyPresence(lobbyID, claims.Sub, connID) {
			q.publishLobbyChanged(r.Context(), lobbyID)
		}
		return conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	})
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	var lobbyEvents <-chan *redis.Message
	if q.redis != nil {
		pubsub := q.redis.Subscribe(ctx, lobbyevents.Channel(lobbyID))
		defer pubsub.Close()
		if _, err := pubsub.Receive(ctx); err != nil {
			observability.Log("warn", "lobby event subscribe failed", map[string]any{"lobbyId": lobbyID, "error": err.Error()})
		} else {
			lobbyEvents = pubsub.Channel()
		}
	}

	var writeMu sync.Mutex
	if q.touchLobbyPresence(lobbyID, claims.Sub, connID) {
		q.publishLobbyChanged(r.Context(), lobbyID)
	}
	if latest, ok, err := q.persist.GetLobbyByID(lobbyID); err != nil || !ok {
		q.writeQueueMessage(conn, &writeMu, "lobby_error", map[string]string{"message": "Lobby unavailable"})
		return
	} else {
		snap = q.lobbySettings.Apply(ctx, latest)
		if !lobbyHasMember(snap, claims.Sub) {
			q.writeQueueMessage(conn, &writeMu, "lobby_error", map[string]string{"message": "You left this lobby"})
			return
		}
	}
	q.applyLobbyPresence(&snap)
	q.writeLobbySnapshot(conn, &writeMu, snap)
	lastLobby := snap
	lastLobbyFingerprint := lobbyFingerprint(snap)
	revision := int64(1)

	refreshLobby := func() bool {
		next, ok, err := q.persist.GetLobbyByID(lobbyID)
		if err != nil || !ok {
			q.writeQueueMessage(conn, &writeMu, "lobby_error", map[string]string{"message": "Lobby unavailable"})
			return false
		}
		next = q.lobbySettings.Apply(ctx, next)
		q.applyLobbyPresence(&next)
		if !lobbyHasMember(next, claims.Sub) {
			q.writeQueueMessage(conn, &writeMu, "lobby_error", map[string]string{"message": "You left this lobby"})
			return false
		}
		nextFingerprint := lobbyFingerprint(next)
		if nextFingerprint != lastLobbyFingerprint {
			revision++
			q.writeLobbyPatch(conn, &writeMu, lobbyPatch(lastLobby, next, revision))
			lastLobby = next
			lastLobbyFingerprint = nextFingerprint
		}
		activeMatchID := next.ActiveMatchID
		if activeMatchID == "" {
			activeMatchID = next.StartedMatchID
		}
		if (next.State == contracts.LobbyInMatch || next.State == contracts.LobbyStarted) && activeMatchID != "" {
			if assigned, ok, err := q.state.GetAssignmentByMatch(ctx, activeMatchID); err == nil && ok {
				if payload, ok, err := q.launcher().AssignedPayload(claims.Sub, assigned); err == nil && ok {
					q.writeQueueMessage(conn, &writeMu, "match_assigned", payload)
					return false
				}
			}
		}
		return next.State == contracts.LobbyOpen
	}

	presenceTicker := time.NewTicker(10 * time.Second)
	defer presenceTicker.Stop()
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-lobbyEvents:
			if !ok {
				return
			}
			if event == nil || event.Payload != lobbyevents.KindChanged {
				continue
			}
			if !refreshLobby() {
				return
			}
		case <-presenceTicker.C:
			if q.touchLobbyPresence(lobbyID, claims.Sub, connID) {
				q.publishLobbyChanged(r.Context(), lobbyID)
			}
		case <-pingTicker.C:
			if !q.writeQueuePing(conn, &writeMu) {
				return
			}
		}
	}
}

func (q *matchCoordinator) requirePlayableUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	appClaims, err := q.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return "", false
	}
	identity, err := q.persist.GetIdentity(appClaims.Sub)
	if err != nil {
		http.Error(w, "identity not found", http.StatusUnauthorized)
		return "", false
	}
	if identity.IsBanned {
		http.Error(w, "account is banned", http.StatusForbidden)
		return "", false
	}
	if !identity.Onboarded {
		http.Error(w, "onboarding incomplete", http.StatusForbidden)
		return "", false
	}
	if identity.AuthMigrationRequired {
		http.Error(w, "connect discord to continue", http.StatusForbidden)
		return "", false
	}
	return appClaims.Sub, true
}

func (q *matchCoordinator) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (q *matchCoordinator) startLobby(w http.ResponseWriter, r *http.Request) {
	claims, err := q.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	lobbyID := strings.TrimSpace(mux.Vars(r)["id"])
	snap, ok, err := q.persist.GetLobbyByID(lobbyID)
	if err != nil {
		http.Error(w, "lobby unavailable", http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}
	snap = q.lobbySettings.Apply(r.Context(), snap)
	if snap.OwnerUserID != claims.Sub {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if q.redis != nil {
		q.applyLobbyPresence(&snap)
		if err := requireLobbyPresence(snap); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
	found, err := q.lobbyMatchFound(snap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	for _, userID := range found.Players {
		if assigned, ok, err := q.state.GetAssignmentByUser(r.Context(), userID); err == nil && ok {
			mode := sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID)
			switch q.launcher().ValidateAssignment(r.Context(), assigned) {
			case matchlaunch.AssignmentValid, matchlaunch.AssignmentPending:
				if contracts.IsPrivatePartyMode(mode) {
					http.Error(w, activeLobbyMatchConflict(userID, assigned, found.Profiles[userID]), http.StatusConflict)
					return
				}
				q.clearSupersededAssignment(context.Background(), assigned)
			case matchlaunch.AssignmentAbandoned, matchlaunch.AssignmentInvalid:
				_ = q.state.ClearAssignment(context.Background(), assigned)
			}
		}
	}
	assigned, err := q.launcher().EnsureAssignment(r.Context(), found)
	if err != nil {
		http.Error(w, "lobby start failed", http.StatusBadGateway)
		return
	}
	snap, err = q.persist.MarkLobbyInMatch(lobbyID, found.MatchID)
	if err != nil {
		http.Error(w, "lobby start failed", http.StatusConflict)
		return
	}
	q.publishLobbyChanged(r.Context(), snap.ID)
	payload, ok, err := q.launcher().AssignedPayload(claims.Sub, assigned)
	if err != nil || !ok {
		http.Error(w, "unable to issue gameplay ticket", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(contracts.LobbyStartResponse{Assignment: payload})
}

func (q *matchCoordinator) clearSupersededAssignment(ctx context.Context, assigned coordinator.Assignment) {
	if sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID) == contracts.ModeSingleplayer {
		q.terminateSupersededMatch(ctx, assigned)
	}
	_ = q.state.ClearAssignment(ctx, assigned)
	if q.persist != nil && sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID) == contracts.ModeSingleplayer {
		_ = q.persist.RecordRuntimeMatch(assigned.MatchID, string(contracts.MatchEnded), assigned.NodeEpoch, true)
	}
}

func (q *matchCoordinator) terminateSupersededMatch(ctx context.Context, assigned coordinator.Assignment) {
	node, ok, err := q.state.GetNodeByRoute(ctx, assigned.PublicRoute)
	if err != nil || !ok || strings.TrimSpace(node.InternalURL) == "" {
		return
	}
	userID := ""
	if len(assigned.Players) > 0 {
		userID = assigned.Players[0]
	}
	body, _ := json.Marshal(map[string]string{"userId": userID})
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodPost,
		strings.TrimRight(node.InternalURL, "/")+"/internal/matches/"+url.PathEscape(assigned.MatchID)+"/terminate",
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Coordinator-Secret", q.internal)
	client := q.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		observability.Log("warn", "superseded lobby match terminate failed", map[string]any{"matchId": assigned.MatchID, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		observability.Log("warn", "superseded lobby match terminate rejected", map[string]any{"matchId": assigned.MatchID, "status": resp.StatusCode})
	}
}

func activeLobbyMatchConflict(userID string, assigned coordinator.Assignment, profile contracts.PlayerProfile) string {
	name := strings.TrimSpace(profile.DisplayName)
	if name == "" {
		name = userID
	}
	mode := sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID)
	return "player " + name + " (" + userID + ") already has an active " + string(mode) + " match " + assigned.MatchID
}

func (q *matchCoordinator) lobbyMatchFound(snap contracts.LobbySnapshot) (contracts.MatchFound, error) {
	if snap.State != contracts.LobbyOpen {
		return contracts.MatchFound{}, errors.New("lobby is not open")
	}
	active := make([]contracts.LobbyMember, 0, len(snap.Members))
	for _, member := range snap.Members {
		if strings.TrimSpace(member.UserID) != "" {
			active = append(active, member)
		}
	}
	if len(active) < 2 || len(active) > 8 {
		return contracts.MatchFound{}, errors.New("lobby requires 2 to 8 players")
	}
	switch snap.Mode {
	case contracts.ModeDuel:
		if len(active) != 2 {
			return contracts.MatchFound{}, errors.New("duel lobby requires exactly two players")
		}
	case contracts.ModeTeamDuel:
		teamCounts := map[string]int{}
		for _, member := range active {
			teamCounts[normalizeLobbyTeam(member.TeamID)]++
		}
		if teamCounts["a"] == 0 || teamCounts["b"] == 0 {
			return contracts.MatchFound{}, errors.New("team duel requires players on both teams")
		}
	case contracts.ModeFreeForAll:
	default:
		return contracts.MatchFound{}, errors.New("unsupported lobby mode")
	}
	match := contracts.MatchFound{
		MatchID:               "m-" + strconvTimeID(),
		Mode:                  snap.Mode,
		Unranked:              true,
		Players:               []string{},
		Profiles:              map[string]contracts.PlayerProfile{},
		Teams:                 map[string]string{},
		Config:                contracts.NormalizeMatchConfig(snap.Config),
		MapScope:              defaultLobbyMapScope(snap.MapScope),
		SourceLobbyID:         snap.ID,
		SourceLobbyInviteCode: snap.InviteCode,
	}
	for _, member := range active {
		match.Players = append(match.Players, member.UserID)
		if snap.Mode == contracts.ModeTeamDuel {
			match.Teams[member.UserID] = normalizeLobbyTeam(member.TeamID)
		}
		match.Profiles[member.UserID] = contracts.PlayerProfile{
			UserID:        member.UserID,
			DisplayName:   member.DisplayName,
			AvatarURL:     member.AvatarURL,
			IsGuest:       member.IsGuest,
			IsAdmin:       member.IsAdmin,
			SelectedBadge: member.SelectedBadge,
		}
		if profile, err := q.persist.GetProfile(member.UserID); err == nil {
			player := match.Profiles[member.UserID]
			player.MMR = profile.MMR
			player.RatingRD = profile.RatingRD
			player.RankedGamesPlayed = profile.RankedGamesPlayed
			player.SelectedBadge = profile.SelectedBadge
			match.Profiles[member.UserID] = player
		}
	}
	return match, nil
}

func (q *matchCoordinator) writeLobbySnapshot(conn *websocket.Conn, writeMu *sync.Mutex, snap contracts.LobbySnapshot) bool {
	q.applyLobbyPresence(&snap)
	return q.writeQueueMessage(conn, writeMu, "lobby_snapshot", snap)
}

func (q *matchCoordinator) writeLobbyPatch(conn *websocket.Conn, writeMu *sync.Mutex, patch contracts.LobbyPatch) bool {
	return q.writeQueueMessage(conn, writeMu, "lobby_patch", patch)
}

func (q *matchCoordinator) touchLobbyPresence(lobbyID, userID, connID string) bool {
	if q.redis == nil || strings.TrimSpace(lobbyID) == "" || strings.TrimSpace(userID) == "" {
		return false
	}
	key := lobbyPresenceKey(lobbyID)
	now := time.Now().UnixMilli()
	field := lobbyPresenceField(userID, connID)
	previous, _ := q.redis.HGet(context.Background(), key, field).Result()
	var addedCmd *redis.IntCmd
	_, err := q.redis.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		addedCmd = pipe.HSet(context.Background(), key, field, now)
		pipe.Expire(context.Background(), key, lobbyPresenceTTL)
		return nil
	})
	if err != nil {
		observability.Log("warn", "lobby presence touch failed", map[string]any{"lobbyId": lobbyID, "userId": userID, "error": err.Error()})
		return false
	}
	if addedCmd != nil && addedCmd.Val() > 0 {
		return true
	}
	prevMS, err := strconv.ParseInt(previous, 10, 64)
	if err != nil {
		return true
	}
	return lobbyPresenceStatus(now, prevMS) != contracts.LobbyPresenceOnline
}

func (q *matchCoordinator) clearLobbyPresence(lobbyID, userID, connID string) {
	if q.redis == nil || strings.TrimSpace(lobbyID) == "" || strings.TrimSpace(userID) == "" {
		return
	}
	removed, err := q.redis.HDel(context.Background(), lobbyPresenceKey(lobbyID), lobbyPresenceField(userID, connID)).Result()
	if err != nil {
		observability.Log("warn", "lobby presence clear failed", map[string]any{"lobbyId": lobbyID, "userId": userID, "error": err.Error()})
		return
	}
	if removed > 0 {
		q.publishLobbyChanged(context.Background(), lobbyID)
	}
}

func (q *matchCoordinator) publishLobbyChanged(ctx context.Context, lobbyID string) {
	if q.redis == nil || strings.TrimSpace(lobbyID) == "" {
		return
	}
	_ = q.redis.Publish(ctx, lobbyevents.Channel(lobbyID), lobbyevents.KindChanged).Err()
}

func (q *matchCoordinator) applyLobbyPresence(snap *contracts.LobbySnapshot) {
	if snap == nil || q.redis == nil {
		return
	}
	fields := make([]string, 0, len(snap.Members))
	for _, member := range snap.Members {
		userID := strings.TrimSpace(member.UserID)
		if userID != "" {
			fields = append(fields, lobbyPresenceField(userID, ""))
		}
	}
	if len(fields) == 0 {
		return
	}
	values, err := q.redis.HMGet(context.Background(), lobbyPresenceKey(snap.ID), fields...).Result()
	if err != nil {
		return
	}
	now := time.Now().UnixMilli()
	seen := map[string]int64{}
	for i, raw := range values {
		rawString, ok := raw.(string)
		if !ok {
			continue
		}
		ms, err := strconv.ParseInt(rawString, 10, 64)
		if err != nil {
			continue
		}
		userID := lobbyPresenceUserID(fields[i])
		if userID != "" {
			seen[userID] = ms
		}
	}
	for i := range snap.Members {
		lastSeen := seen[snap.Members[i].UserID]
		switch lobbyPresenceStatus(now, lastSeen) {
		case contracts.LobbyPresenceOnline:
			snap.Members[i].Connected = true
			snap.Members[i].PresenceStatus = contracts.LobbyPresenceOnline
		case contracts.LobbyPresenceAway:
			snap.Members[i].Connected = false
			snap.Members[i].PresenceStatus = contracts.LobbyPresenceAway
		default:
			snap.Members[i].Connected = false
			snap.Members[i].PresenceStatus = contracts.LobbyPresenceOffline
		}
	}
}

func lobbyPresenceStatus(now, lastSeen int64) contracts.LobbyPresenceStatus {
	if lastSeen <= 0 {
		return contracts.LobbyPresenceOffline
	}
	age := now - lastSeen
	switch {
	case age <= lobbyPresenceOnlineTTL.Milliseconds():
		return contracts.LobbyPresenceOnline
	case age <= lobbyPresenceAwayTTL.Milliseconds():
		return contracts.LobbyPresenceAway
	default:
		return contracts.LobbyPresenceOffline
	}
}

func requireLobbyPresence(snap contracts.LobbySnapshot) error {
	missing := make([]string, 0, len(snap.Members))
	for _, member := range snap.Members {
		if strings.TrimSpace(member.UserID) == "" {
			continue
		}
		if !member.Connected {
			name := strings.TrimSpace(member.DisplayName)
			if name == "" {
				name = member.UserID
			}
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return errors.New("all players must be in the lobby to start: " + strings.Join(missing, ", "))
	}
	return nil
}

func lobbyFingerprint(snap contracts.LobbySnapshot) string {
	b, _ := json.Marshal(snap)
	return string(b)
}

func lobbyPatch(prev, next contracts.LobbySnapshot, revision int64) contracts.LobbyPatch {
	patch := contracts.LobbyPatch{Revision: revision}
	if prev.State != next.State {
		v := next.State
		patch.State = &v
	}
	if prev.OwnerUserID != next.OwnerUserID {
		v := next.OwnerUserID
		patch.OwnerUserID = &v
	}
	if prev.Mode != next.Mode {
		v := next.Mode
		patch.Mode = &v
	}
	if prev.Config != next.Config {
		v := next.Config
		patch.Config = &v
	}
	if prev.ActiveMatchID != next.ActiveMatchID {
		v := next.ActiveMatchID
		patch.ActiveMatchID = &v
	}
	if prev.LastMatchID != next.LastMatchID {
		v := next.LastMatchID
		patch.LastMatchID = &v
	}
	if prev.StartedMatchID != next.StartedMatchID {
		v := next.StartedMatchID
		patch.StartedMatchID = &v
	}
	prevMembers := map[string]contracts.LobbyMember{}
	nextMembers := map[string]contracts.LobbyMember{}
	for _, member := range prev.Members {
		prevMembers[member.UserID] = member
	}
	for _, member := range next.Members {
		nextMembers[member.UserID] = member
		if lobbyMemberFingerprint(prevMembers[member.UserID]) != lobbyMemberFingerprint(member) {
			patch.UpsertMembers = append(patch.UpsertMembers, member)
		}
	}
	for id := range prevMembers {
		if _, ok := nextMembers[id]; !ok {
			patch.RemoveMemberIDs = append(patch.RemoveMemberIDs, id)
		}
	}
	return patch
}

func lobbyMemberFingerprint(member contracts.LobbyMember) string {
	b, _ := json.Marshal(member)
	return string(b)
}

func normalizeLobbyTeam(teamID string) string {
	switch strings.ToLower(strings.TrimSpace(teamID)) {
	case "b":
		return "b"
	default:
		return "a"
	}
}

func lobbyPresenceField(userID, connID string) string {
	return strings.TrimSpace(userID)
}

func lobbyPresenceKey(lobbyID string) string {
	return "lobby:presence:v2:" + strings.TrimSpace(lobbyID)
}

func lobbyPresenceUserID(field string) string {
	if before, _, ok := strings.Cut(field, "|"); ok {
		return before
	}
	return field
}

func (q *matchCoordinator) runLobbyCleanupLoop(interval, inactivityTTL time.Duration) {
	if interval <= 0 || inactivityTTL <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		q.cleanupOpenLobbies(inactivityTTL)
		<-ticker.C
	}
}

func (q *matchCoordinator) cleanupOpenLobbies(_ time.Duration) {
	if reopened, err := q.persist.ReopenEndedLobbies(); err != nil {
		observability.Log("warn", "ended lobby reopen failed", map[string]any{"error": err.Error()})
	} else if reopened > 0 {
		observability.Log("info", "ended lobbies reopened", map[string]any{"members": reopened})
	}
	if err := q.persist.ExpireOpenLobbies(); err != nil {
		observability.Log("warn", "lobby expiry cleanup failed", map[string]any{"error": err.Error()})
		return
	}
}

func lobbyHasMember(snap contracts.LobbySnapshot, userID string) bool {
	for _, member := range snap.Members {
		if member.UserID == userID {
			return true
		}
	}
	return false
}

func strconvTimeID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func defaultLobbyMapScope(v string) string {
	if strings.TrimSpace(v) == "" {
		return "world"
	}
	return v
}
