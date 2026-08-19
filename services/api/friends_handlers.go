package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var errInvalidInt = errors.New("invalid integer")

// publishSocialEvent publishes a realtime social event to the target user's
// Redis pub/sub channel. The social websocket subscribes to social:{userID}
// and forwards these events to the connected client.
func (a *api) publishSocialEvent(userID, kind string, payload any) {
	if a.redis == nil || strings.TrimSpace(userID) == "" {
		return
	}
	body, err := json.Marshal(map[string]any{
		"type":    kind,
		"payload": payload,
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = a.redis.Publish(ctx, "social:"+userID, string(body)).Err()
}

func (a *api) searchPlayers(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"players": []any{}})
		return
	}
	limit := 8
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := atoiSafe(raw); err == nil && parsed > 0 && parsed <= 25 {
			limit = parsed
		}
	}
	players, err := a.store.SearchPlayersForFriends(query, claims.Sub, limit)
	if err != nil {
		http.Error(w, "search unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"players": players})
}

func (a *api) listFriends(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	friends, err := a.store.ListFriends(claims.Sub)
	if err != nil {
		http.Error(w, "friends unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"friends": friends})
}

func (a *api) listFriendRequests(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	incoming, outgoing, err := a.store.ListFriendRequests(claims.Sub)
	if err != nil {
		http.Error(w, "requests unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"incoming": incoming,
		"outgoing": outgoing,
	})
}

func (a *api) createFriendRequest(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		TargetUserId string `json:"targetUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	targetID := a.resolveEntityID("user", req.TargetUserId)
	if targetID == "" || targetID == claims.Sub {
		http.Error(w, "invalid target", http.StatusBadRequest)
		return
	}
	targetProfile, err := a.store.GetProfile(targetID)
	if err != nil || targetProfile.UserID == "" {
		http.Error(w, "player not found", http.StatusNotFound)
		return
	}
	if targetProfile.IsGuest {
		http.Error(w, "cannot friend guest accounts", http.StatusBadRequest)
		return
	}
	requesterProfile, err := a.store.GetProfile(claims.Sub)
	if err != nil {
		http.Error(w, "profile unavailable", http.StatusInternalServerError)
		return
	}
	if requesterProfile.IsGuest {
		http.Error(w, "guests cannot send friend requests", http.StatusForbidden)
		return
	}
	if err := a.store.CreateFriendshipRequest(claims.Sub, targetID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "yourself") {
			http.Error(w, "cannot friend yourself", http.StatusBadRequest)
			return
		}
		http.Error(w, "request unavailable", http.StatusInternalServerError)
		return
	}
	notificationID, nerr := a.store.InsertUserNotification(
		targetID,
		"friend_request",
		"fr:"+claims.Sub+":"+targetID,
		map[string]any{
			"requesterId":   claims.Sub,
			"requesterName": requesterProfile.DisplayName,
			"targetId":      targetID,
		},
	)
	if nerr == nil && notificationID > 0 {
		a.publishSocialEvent(targetID, "friend_request", map[string]any{
			"notificationId": notificationID,
			"requesterId":    claims.Sub,
			"requesterName":  requesterProfile.DisplayName,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "pending", "targetUserId": targetID})
}

func (a *api) acceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	requesterID := a.resolveEntityID("user", mux.Vars(r)["requesterId"])
	if requesterID == "" {
		http.Error(w, "invalid requester", http.StatusBadRequest)
		return
	}
	if err := a.store.AcceptFriendship(claims.Sub, requesterID); err != nil {
		http.Error(w, "could not accept", http.StatusInternalServerError)
		return
	}
	a.publishSocialEvent(requesterID, "friend_accepted", map[string]any{
		"friendId": claims.Sub,
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
}

func (a *api) declineFriendRequest(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	requesterID := a.resolveEntityID("user", mux.Vars(r)["requesterId"])
	if requesterID == "" {
		http.Error(w, "invalid requester", http.StatusBadRequest)
		return
	}
	if err := a.store.DeclineFriendship(claims.Sub, requesterID); err != nil {
		http.Error(w, "could not decline", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "declined"})
}

func (a *api) removeFriend(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	friendID := a.resolveEntityID("user", mux.Vars(r)["friendUserId"])
	if friendID == "" {
		http.Error(w, "invalid friend", http.StatusBadRequest)
		return
	}
	if err := a.store.RemoveFriend(claims.Sub, friendID); err != nil {
		http.Error(w, "could not remove", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "removed"})
}

func (a *api) inviteFriendToParty(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	partyID := strings.TrimSpace(mux.Vars(r)["partyId"])
	if partyID == "" {
		http.Error(w, "invalid party", http.StatusBadRequest)
		return
	}
	snapshot, ok, err := a.store.GetPartyByID(partyID)
	if err != nil || !ok {
		http.Error(w, "party not found", http.StatusNotFound)
		return
	}
	if snapshot.OwnerUserID != claims.Sub {
		http.Error(w, "only the party owner can invite", http.StatusForbidden)
		return
	}
	var req struct {
		FriendUserId string `json:"friendUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	friendID := a.resolveEntityID("user", req.FriendUserId)
	if friendID == "" || friendID == claims.Sub {
		http.Error(w, "invalid friend", http.StatusBadRequest)
		return
	}
	friends, err := a.store.AreFriends(claims.Sub, friendID)
	if err != nil {
		http.Error(w, "invite unavailable", http.StatusInternalServerError)
		return
	}
	if !friends {
		http.Error(w, "you can only invite friends", http.StatusForbidden)
		return
	}
	inviterProfile, _ := a.store.GetProfile(claims.Sub)
	inviterName := strings.TrimSpace(inviterProfile.DisplayName)
	if inviterName == "" {
		inviterName = claims.Sub
	}
	dedupeKey := "pi:" + partyID + ":" + friendID
	existing, alreadySent, _ := a.store.GetUserNotificationByDedupeKey(friendID, dedupeKey)
	// Only send when there is no pending invite yet, or when the friend
	// previously dismissed the last one (read_at set). Repeating an
	// already-pending invite is a no-op to avoid flooding the friend.
	shouldSend := !alreadySent || existing.ReadAt != nil
	notificationID := int64(0)
	var nerr error
	if shouldSend {
		notificationID, nerr = a.store.InsertUserNotification(
			friendID,
			"party_invite",
			dedupeKey,
			map[string]any{
				"partyId":     partyID,
				"inviteCode":  snapshot.InviteCode,
				"inviterId":   claims.Sub,
				"inviterName": inviterName,
			},
		)
		if nerr != nil {
			http.Error(w, "invite unavailable", http.StatusInternalServerError)
			return
		}
	} else {
		notificationID = existing.ID
	}
	if shouldSend {
		a.publishSocialEvent(friendID, "party_invite", map[string]any{
			"notificationId": notificationID,
			"partyId":        partyID,
			"inviteCode":     snapshot.InviteCode,
			"inviterId":      claims.Sub,
			"inviterName":    inviterName,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "invited",
		"inviteCode": snapshot.InviteCode,
	})
}

func atoiSafe(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errInvalidInt
	}
	n := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
