package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"geoduels/internal/httpx"
	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
	"geoduels/pkg/entityid"
	"geoduels/pkg/matchlaunch"
	"geoduels/pkg/observability"
	"geoduels/pkg/partyevents"
	"geoduels/pkg/sessionpolicy"
)

var partyUpgrader = websocket.Upgrader{CheckOrigin: httpx.WSOriginAllowed}

const (
	defaultPartyTTL        = 2 * time.Hour
	partyPresenceOnlineTTL = 15 * time.Second
	partyPresenceAwayTTL   = 60 * time.Second
	partyPresenceTTL       = 90 * time.Second
)

func (q *matchCoordinator) createParty(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	var req contracts.PartyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	mode := req.Mode
	if mode == "" {
		mode = contracts.ModeDuel
	}
	if !contracts.IsPrivatePartyMode(mode) {
		return httpx.PlainTextError(c, http.StatusBadRequest, "unsupported party mode")
	}
	snap, err := q.parties.CreateParty(userID, mode, req.MapScope, defaultPartyTTL)
	if err != nil {
		observability.Log("error", "create party failed", map[string]any{"userId": userID, "mode": string(mode), "error": err.Error()})
		return httpx.PlainTextError(c, http.StatusInternalServerError, "party unavailable")
	}
	if req.Config.Ruleset != "" || req.Config.MapID != "" || req.Config.MapKey != "" || req.Config.RoundTimerMode != "" || req.Config.RoundTimeLimitMS > 0 || req.Config.PressureTimeLimitMS > 0 || req.Config.MultiplierMode != "" {
		if req.Config.MapID == "" && req.Config.MapKey == "" {
			req.Config.MapID = snap.Config.MapID
		}
		snap, err = q.parties.SetPartyConfig(snap.ID, req.Config)
		if err != nil {
			observability.Log("error", "create party settings save failed", map[string]any{"userId": userID, "partyId": snap.ID, "error": err.Error()})
			return httpx.PlainTextError(c, http.StatusInternalServerError, "party unavailable")
		}
	}
	if q.touchPartyPresence(snap.ID, userID, "") {
		q.publishPartyChanged(r.Context(), snap.ID)
	}
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) getParty(c echo.Context) error {
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	code := strings.TrimSpace(c.Param("code"))
	snap, ok, err := q.parties.GetPartyByInviteCode(code)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusInternalServerError, "party unavailable")
	}
	if !ok {
		return httpx.PlainTextError(c, http.StatusNotFound, "party not found")
	}
	isMember := false
	for _, member := range snap.Members {
		if member.UserID == userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return httpx.PlainTextError(c, http.StatusForbidden, "party membership required")
	}
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) joinParty(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	code := strings.TrimSpace(c.Param("code"))
	snap, found, err := q.parties.GetPartyByInviteCode(code)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusInternalServerError, "party unavailable")
	}
	if !found {
		return httpx.PlainTextError(c, http.StatusNotFound, "party not found")
	}
	snap, err = q.parties.JoinParty(snap.ID, userID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusConflict, err.Error())
	}
	q.touchPartyPresence(snap.ID, userID, "")
	q.applyPartyPresence(&snap)
	q.publishPartyChanged(r.Context(), snap.ID)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) leaveParty(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	snap, err := q.parties.LeaveParty(id, userID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "party unavailable")
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) kickPartyMember(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	var req contracts.PartyMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	id := strings.TrimSpace(c.Param("id"))
	snap, err := q.parties.KickPartyMember(id, userID, strings.TrimSpace(req.UserID))
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "party unavailable")
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) transferPartyOwner(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	var req contracts.PartyMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	id := strings.TrimSpace(c.Param("id"))
	snap, err := q.parties.TransferPartyOwner(id, userID, strings.TrimSpace(req.UserID))
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "party unavailable")
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) updatePartyTeam(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	var req contracts.PartyTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	id := strings.TrimSpace(c.Param("id"))
	snap, err := q.parties.SetPartyMemberTeam(id, userID, strings.TrimSpace(req.TeamID))
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, err.Error())
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) updatePartySettings(c echo.Context) error {
	r := c.Request()
	userID, err := q.requirePlayableUser(c)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	snap, found, err := q.parties.GetPartyByID(id)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusInternalServerError, "party unavailable")
	}
	if !found {
		return httpx.PlainTextError(c, http.StatusNotFound, "party not found")
	}
	if snap.OwnerUserID != userID {
		return httpx.PlainTextError(c, http.StatusForbidden, "forbidden")
	}
	if snap.State != contracts.PartyOpen {
		return httpx.PlainTextError(c, http.StatusConflict, "party settings are locked")
	}
	var req struct {
		Mode   contracts.MatchMode   `json:"mode"`
		Config contracts.MatchConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpx.PlainTextError(c, http.StatusBadRequest, "invalid payload")
	}
	if req.Mode != "" && !contracts.IsPrivatePartyMode(req.Mode) {
		return httpx.PlainTextError(c, http.StatusBadRequest, "unsupported party mode")
	}
	if req.Mode != "" && req.Mode != snap.Mode {
		if err := q.parties.SetPartyMode(snap.ID, req.Mode); err != nil {
			return httpx.PlainTextError(c, http.StatusBadGateway, "party settings unavailable")
		}
		snap.Mode = req.Mode
	}
	snap, err = q.parties.SetPartyConfig(snap.ID, req.Config)
	if err != nil {
		if errors.Is(err, errPartyMapUnavailable) {
			return httpx.PlainTextError(c, http.StatusUnprocessableEntity, err.Error())
		}
		observability.Log("error", "update party settings save failed", map[string]any{"userId": userID, "partyId": snap.ID, "mapId": req.Config.MapID, "error": err.Error()})
		return httpx.PlainTextError(c, http.StatusBadGateway, "party settings unavailable")
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	q.applyPartyPresence(&snap)
	return httpx.JSON(c, http.StatusOK, snap)
}

func (q *matchCoordinator) partyPresence(c echo.Context) error {
	r := c.Request()
	claims, _, err := q.requireActiveAccount(c)
	if err != nil {
		return err
	}
	partyID := strings.TrimSpace(c.Param("id"))
	snap, ok, err := q.parties.GetPartyByID(partyID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadGateway, "party unavailable")
	}
	if !ok || !partyHasMember(snap, claims.Sub) {
		return httpx.PlainTextError(c, http.StatusForbidden, "forbidden")
	}
	if q.touchPartyPresence(partyID, claims.Sub, "") {
		q.publishPartyChanged(r.Context(), partyID)
	}
	return httpx.JSON(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (q *matchCoordinator) partyWS(c echo.Context) error {
	r := c.Request()
	claims, _, err := q.requireActiveAccount(c)
	if err != nil {
		return err
	}
	partyID := strings.TrimSpace(c.Param("id"))
	snap, ok, err := q.parties.GetPartyByID(partyID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadGateway, "party unavailable")
	}
	if !ok || !partyHasMember(snap, claims.Sub) {
		return httpx.PlainTextError(c, http.StatusForbidden, "forbidden")
	}
	conn, err := partyUpgrader.Upgrade(c.Response().Writer, r, nil)
	if err != nil {
		return nil
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	connID := strconvTimeID()
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
		if q.touchPartyPresence(partyID, claims.Sub, connID) {
			q.publishPartyChanged(r.Context(), partyID)
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

	var partyEvents <-chan *redis.Message
	if q.redis != nil {
		pubsub := q.redis.Subscribe(ctx, partyevents.Channel(partyID))
		defer pubsub.Close()
		if _, err := pubsub.Receive(ctx); err != nil {
			observability.Log("warn", "party event subscribe failed", map[string]any{"partyId": partyID, "error": err.Error()})
		} else {
			partyEvents = pubsub.Channel()
		}
	}

	var writeMu sync.Mutex
	if q.touchPartyPresence(partyID, claims.Sub, connID) {
		q.publishPartyChanged(r.Context(), partyID)
	}
	if latest, ok, err := q.parties.GetPartyByID(partyID); err != nil || !ok {
		q.writeQueueMessage(conn, &writeMu, "party_error", map[string]string{"message": "Party unavailable"})
		return nil
	} else {
		snap = latest
		if !partyHasMember(snap, claims.Sub) {
			q.writeQueueMessage(conn, &writeMu, "party_error", map[string]string{"message": "You left this party"})
			return nil
		}
	}
	q.applyPartyPresence(&snap)
	q.writePartySnapshot(conn, &writeMu, snap)
	lastParty := snap
	lastPartyFingerprint := partyFingerprint(snap)
	revision := int64(1)

	refreshParty := func() bool {
		next, ok, err := q.parties.GetPartyByID(partyID)
		if err != nil || !ok {
			q.writeQueueMessage(conn, &writeMu, "party_error", map[string]string{"message": "Party unavailable"})
			return false
		}
		q.applyPartyPresence(&next)
		if !partyHasMember(next, claims.Sub) {
			q.writeQueueMessage(conn, &writeMu, "party_error", map[string]string{"message": "You left this party"})
			return false
		}
		nextFingerprint := partyFingerprint(next)
		if nextFingerprint != lastPartyFingerprint {
			revision++
			q.writePartyPatch(conn, &writeMu, partyPatch(lastParty, next, revision))
			lastParty = next
			lastPartyFingerprint = nextFingerprint
		}
		activeMatchID := next.ActiveMatchID
		if activeMatchID == "" {
			activeMatchID = next.StartedMatchID
		}
		if (next.State == contracts.PartyInMatch || next.State == contracts.PartyStarted) && activeMatchID != "" {
			if assigned, ok, err := q.state.GetAssignmentByMatch(ctx, activeMatchID); err == nil && ok {
				if payload, ok, err := q.launcher().AssignedPayload(claims.Sub, assigned); err == nil && ok {
					q.writeQueueMessage(conn, &writeMu, "match_assigned", payload)
					return false
				}
			}
		}
		return next.State == contracts.PartyOpen || next.State == contracts.PartyInMatch || next.State == contracts.PartyStarted
	}

	presenceTicker := time.NewTicker(10 * time.Second)
	defer presenceTicker.Stop()
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-partyEvents:
			if !ok {
				return nil
			}
			if event == nil || event.Payload != partyevents.KindChanged {
				continue
			}
			if !refreshParty() {
				return nil
			}
		case <-presenceTicker.C:
			if q.touchPartyPresence(partyID, claims.Sub, connID) {
				q.publishPartyChanged(r.Context(), partyID)
			}
		case <-pingTicker.C:
			if !q.writeQueuePing(conn, &writeMu) {
				return nil
			}
		}
	}
}

func (q *matchCoordinator) requirePlayableUser(c echo.Context) (string, error) {
	appClaims, identity, err := q.requireActiveAccount(c)
	if err != nil {
		return "", err
	}
	if identity.NicknameRequired {
		return "", httpx.PlainTextError(c, http.StatusForbidden, "nickname required")
	}
	if identity.AuthMigrationRequired {
		return "", httpx.PlainTextError(c, http.StatusForbidden, "connect discord to continue")
	}
	return appClaims.Sub, nil
}

func (q *matchCoordinator) startParty(c echo.Context) error {
	r := c.Request()
	claims, _, err := q.requireActiveAccount(c)
	if err != nil {
		return err
	}
	partyID := strings.TrimSpace(c.Param("id"))
	snap, ok, err := q.parties.GetPartyByID(partyID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadGateway, "party unavailable")
	}
	if !ok {
		return httpx.PlainTextError(c, http.StatusNotFound, "party not found")
	}
	if snap.OwnerUserID != claims.Sub {
		return httpx.PlainTextError(c, http.StatusForbidden, "forbidden")
	}
	if q.redis != nil {
		q.applyPartyPresence(&snap)
		if err := requirePartyPresence(snap); err != nil {
			return httpx.PlainTextError(c, http.StatusConflict, err.Error())
		}
	}
	found, err := q.partyMatchFound(snap)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusConflict, err.Error())
	}
	for _, userID := range found.Players {
		if assigned, ok, err := q.state.GetAssignmentByUser(r.Context(), userID); err == nil && ok {
			mode := sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID)
			switch q.launcher().ValidateAssignment(r.Context(), assigned) {
			case matchlaunch.AssignmentValid, matchlaunch.AssignmentPending:
				if contracts.IsPrivatePartyMode(mode) {
					return httpx.PlainTextError(c, http.StatusConflict, activePartyMatchConflict(userID, assigned, found.Profiles[userID]))
				}
				q.clearSupersededAssignment(context.Background(), assigned)
			case matchlaunch.AssignmentAbandoned, matchlaunch.AssignmentInvalid:
				_ = q.state.ClearAssignment(context.Background(), assigned)
			}
		}
	}
	assigned, err := q.launcher().EnsureAssignment(r.Context(), found)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusBadGateway, "party start failed")
	}
	snap, err = q.parties.MarkPartyInMatch(partyID, found.MatchID)
	if err != nil {
		return httpx.PlainTextError(c, http.StatusConflict, "party start failed")
	}
	q.publishPartyChanged(r.Context(), snap.ID)
	payload, ok, err := q.launcher().AssignedPayload(claims.Sub, assigned)
	if err != nil || !ok {
		return httpx.PlainTextError(c, http.StatusBadGateway, "unable to issue gameplay ticket")
	}
	return httpx.JSON(c, http.StatusOK, contracts.PartyStartResponse{Assignment: payload})
}

func (q *matchCoordinator) clearSupersededAssignment(ctx context.Context, assigned coordinator.Assignment) {
	if sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID) == contracts.ModeSingleplayer {
		q.terminateSupersededMatch(ctx, assigned)
	}
	_ = q.state.ClearAssignment(ctx, assigned)
	if q.matches != nil && sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID) == contracts.ModeSingleplayer {
		_ = q.matches.RecordRuntimeMatch(ctx, assigned.MatchID, string(contracts.MatchEnded), assigned.NodeEpoch, true)
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
		observability.Log("warn", "superseded party match terminate failed", map[string]any{"matchId": assigned.MatchID, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		observability.Log("warn", "superseded party match terminate rejected", map[string]any{"matchId": assigned.MatchID, "status": resp.StatusCode})
	}
}

func activePartyMatchConflict(userID string, assigned coordinator.Assignment, profile contracts.PlayerProfile) string {
	name := strings.TrimSpace(profile.DisplayName)
	if name == "" {
		name = userID
	}
	mode := sessionpolicy.NormalizeMode(assigned.Mode, assigned.MatchID)
	return "player " + name + " (" + userID + ") already has an active " + string(mode) + " match " + assigned.MatchID
}

func (q *matchCoordinator) partyMatchFound(snap contracts.PartySnapshot) (contracts.MatchFound, error) {
	if snap.State != contracts.PartyOpen {
		return contracts.MatchFound{}, errors.New("party is not open")
	}
	active := make([]contracts.PartyMember, 0, len(snap.Members))
	for _, member := range snap.Members {
		if strings.TrimSpace(member.UserID) != "" {
			active = append(active, member)
		}
	}
	if len(active) < contracts.MinPartyMembers || len(active) > contracts.MaxPartyMembers {
		return contracts.MatchFound{}, fmt.Errorf("party requires %d to %d players", contracts.MinPartyMembers, contracts.MaxPartyMembers)
	}
	switch snap.Mode {
	case contracts.ModeDuel:
		if len(active) != 2 {
			return contracts.MatchFound{}, errors.New("duel party requires exactly two players")
		}
	case contracts.ModeTeamDuel:
		teamCounts := map[string]int{}
		for _, member := range active {
			teamCounts[normalizePartyTeam(member.TeamID)]++
		}
		if teamCounts["a"] == 0 || teamCounts["b"] == 0 {
			return contracts.MatchFound{}, errors.New("team duel requires players on both teams")
		}
	case contracts.ModeFreeForAll:
	default:
		return contracts.MatchFound{}, errors.New("unsupported party mode")
	}
	match := contracts.MatchFound{
		MatchID:               entityid.New(),
		Mode:                  snap.Mode,
		Unranked:              true,
		Players:               []string{},
		Profiles:              map[string]contracts.PlayerProfile{},
		Teams:                 map[string]string{},
		Config:                contracts.NormalizeMatchConfig(snap.Config),
		MapAccessUserID:       snap.OwnerUserID,
		MapScope:              defaultPartyMapScope(snap.MapScope),
		SourcePartyID:         snap.ID,
		SourcePartyInviteCode: snap.InviteCode,
		ReturnTarget:          &contracts.MatchReturnTarget{Kind: contracts.MatchReturnParty, PartyID: snap.ID},
	}
	for _, member := range active {
		match.Players = append(match.Players, member.UserID)
		if snap.Mode == contracts.ModeTeamDuel {
			match.Teams[member.UserID] = normalizePartyTeam(member.TeamID)
		}
		match.Profiles[member.UserID] = contracts.PlayerProfile{
			UserID:        member.UserID,
			DisplayName:   member.DisplayName,
			AvatarURL:     member.AvatarURL,
			IsGuest:       member.IsGuest,
			IsAdmin:       member.IsAdmin,
			SelectedBadge: member.SelectedBadge,
		}
		if profile, err := q.profiles.GetProfile(member.UserID); err == nil {
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

func (q *matchCoordinator) writePartySnapshot(conn *websocket.Conn, writeMu *sync.Mutex, snap contracts.PartySnapshot) bool {
	q.applyPartyPresence(&snap)
	return q.writeQueueMessage(conn, writeMu, "party_snapshot", snap)
}

func (q *matchCoordinator) writePartyPatch(conn *websocket.Conn, writeMu *sync.Mutex, patch contracts.PartyPatch) bool {
	return q.writeQueueMessage(conn, writeMu, "party_patch", patch)
}

func (q *matchCoordinator) touchPartyPresence(partyID, userID, connID string) bool {
	if q.redis == nil || strings.TrimSpace(partyID) == "" || strings.TrimSpace(userID) == "" {
		return false
	}
	key := partyPresenceKey(partyID)
	now := time.Now().UnixMilli()
	field := partyPresenceField(userID, connID)
	previous, _ := q.redis.HGet(context.Background(), key, field).Result()
	var addedCmd *redis.IntCmd
	_, err := q.redis.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		addedCmd = pipe.HSet(context.Background(), key, field, now)
		pipe.Expire(context.Background(), key, partyPresenceTTL)
		return nil
	})
	if err != nil {
		observability.Log("warn", "party presence touch failed", map[string]any{"partyId": partyID, "userId": userID, "error": err.Error()})
		return false
	}
	if addedCmd != nil && addedCmd.Val() > 0 {
		return true
	}
	prevMS, err := strconv.ParseInt(previous, 10, 64)
	if err != nil {
		return true
	}
	return partyPresenceStatus(now, prevMS) != contracts.PartyPresenceOnline
}

func (q *matchCoordinator) clearPartyPresence(partyID, userID, connID string) {
	if q.redis == nil || strings.TrimSpace(partyID) == "" || strings.TrimSpace(userID) == "" {
		return
	}
	removed, err := q.redis.HDel(context.Background(), partyPresenceKey(partyID), partyPresenceField(userID, connID)).Result()
	if err != nil {
		observability.Log("warn", "party presence clear failed", map[string]any{"partyId": partyID, "userId": userID, "error": err.Error()})
		return
	}
	if removed > 0 {
		q.publishPartyChanged(context.Background(), partyID)
	}
}

func (q *matchCoordinator) publishPartyChanged(ctx context.Context, partyID string) {
	if q.redis == nil || strings.TrimSpace(partyID) == "" {
		return
	}
	_ = q.redis.Publish(ctx, partyevents.Channel(partyID), partyevents.KindChanged).Err()
}

func (q *matchCoordinator) applyPartyPresence(snap *contracts.PartySnapshot) {
	if snap == nil {
		return
	}
	for i := range snap.Members {
		snap.Members[i].InActiveMatch = false
	}
	activeMatchID := snap.ActiveMatchID
	if activeMatchID == "" {
		activeMatchID = snap.StartedMatchID
	}
	if q.state != nil && activeMatchID != "" {
		if assigned, ok, err := q.state.GetAssignmentByMatch(context.Background(), activeMatchID); err == nil && ok {
			players := make(map[string]struct{}, len(assigned.Players))
			for _, userID := range assigned.Players {
				players[userID] = struct{}{}
			}
			for i := range snap.Members {
				_, snap.Members[i].InActiveMatch = players[snap.Members[i].UserID]
			}
		}
	}
	if q.redis == nil {
		return
	}
	fields := make([]string, 0, len(snap.Members))
	for _, member := range snap.Members {
		userID := strings.TrimSpace(member.UserID)
		if userID != "" {
			fields = append(fields, partyPresenceField(userID, ""))
		}
	}
	if len(fields) == 0 {
		return
	}
	values, err := q.redis.HMGet(context.Background(), partyPresenceKey(snap.ID), fields...).Result()
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
		userID := partyPresenceUserID(fields[i])
		if userID != "" {
			seen[userID] = ms
		}
	}
	for i := range snap.Members {
		lastSeen := seen[snap.Members[i].UserID]
		switch partyPresenceStatus(now, lastSeen) {
		case contracts.PartyPresenceOnline:
			snap.Members[i].Connected = true
			snap.Members[i].PresenceStatus = contracts.PartyPresenceOnline
		case contracts.PartyPresenceAway:
			snap.Members[i].Connected = false
			snap.Members[i].PresenceStatus = contracts.PartyPresenceAway
		default:
			snap.Members[i].Connected = false
			snap.Members[i].PresenceStatus = contracts.PartyPresenceOffline
		}
	}
}

func partyPresenceStatus(now, lastSeen int64) contracts.PartyPresenceStatus {
	if lastSeen <= 0 {
		return contracts.PartyPresenceOffline
	}
	age := now - lastSeen
	switch {
	case age <= partyPresenceOnlineTTL.Milliseconds():
		return contracts.PartyPresenceOnline
	case age <= partyPresenceAwayTTL.Milliseconds():
		return contracts.PartyPresenceAway
	default:
		return contracts.PartyPresenceOffline
	}
}

func requirePartyPresence(snap contracts.PartySnapshot) error {
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
		return errors.New("all players must be in the party to start: " + strings.Join(missing, ", "))
	}
	return nil
}

func partyFingerprint(snap contracts.PartySnapshot) string {
	b, _ := json.Marshal(snap)
	return string(b)
}

func partyPatch(prev, next contracts.PartySnapshot, revision int64) contracts.PartyPatch {
	patch := contracts.PartyPatch{Revision: revision}
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
	prevMembers := map[string]contracts.PartyMember{}
	nextMembers := map[string]contracts.PartyMember{}
	for _, member := range prev.Members {
		prevMembers[member.UserID] = member
	}
	for _, member := range next.Members {
		nextMembers[member.UserID] = member
		if partyMemberFingerprint(prevMembers[member.UserID]) != partyMemberFingerprint(member) {
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

func partyMemberFingerprint(member contracts.PartyMember) string {
	b, _ := json.Marshal(member)
	return string(b)
}

func normalizePartyTeam(teamID string) string {
	switch strings.ToLower(strings.TrimSpace(teamID)) {
	case "b":
		return "b"
	default:
		return "a"
	}
}

func partyPresenceField(userID, connID string) string {
	return strings.TrimSpace(userID)
}

func partyPresenceKey(partyID string) string {
	return "party:presence:v2:" + strings.TrimSpace(partyID)
}

func partyPresenceUserID(field string) string {
	if before, _, ok := strings.Cut(field, "|"); ok {
		return before
	}
	return field
}

func (q *matchCoordinator) runPartyCleanupLoop(interval, inactivityTTL time.Duration) {
	if interval <= 0 || inactivityTTL <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		q.cleanupOpenParties(inactivityTTL)
		<-ticker.C
	}
}

func (q *matchCoordinator) cleanupOpenParties(_ time.Duration) {
	if reopened, err := q.parties.ReopenEndedParties(); err != nil {
		observability.Log("warn", "ended party reopen failed", map[string]any{"error": err.Error()})
	} else if reopened > 0 {
		observability.Log("info", "ended parties reopened", map[string]any{"members": reopened})
	}
	if err := q.parties.ExpireOpenParties(); err != nil {
		observability.Log("warn", "party expiry cleanup failed", map[string]any{"error": err.Error()})
		return
	}
}

func partyHasMember(snap contracts.PartySnapshot, userID string) bool {
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

func defaultPartyMapScope(v string) string {
	if strings.TrimSpace(v) == "" {
		return "world"
	}
	return v
}
