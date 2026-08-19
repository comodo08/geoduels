package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/auth"
	"geoduels/pkg/observability"
)

var socialWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

func (a *api) socialWS(w http.ResponseWriter, r *http.Request) {
	accessToken := strings.TrimSpace(r.URL.Query().Get("accessToken"))
	if accessToken == "" {
		accessToken = strings.TrimSpace(r.Header.Get("Authorization"))
		accessToken = strings.TrimPrefix(accessToken, "Bearer ")
	}
	if accessToken == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ValidateAppAccessToken(a.appAuthSecret, accessToken)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := a.resolveEntityID("user", claims.Sub)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := socialWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	_ = conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
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

	channel := "social:" + userID
	var events <-chan *redis.Message
	if a.redis != nil {
		pubsub := a.redis.Subscribe(ctx, channel)
		defer pubsub.Close()
		if _, err := pubsub.Receive(ctx); err != nil {
			observability.Log("warn", "social event subscribe failed", map[string]any{"userId": userID, "error": err.Error()})
		} else {
			events = pubsub.Channel()
		}
	}

	var writeMu sync.Mutex
	write := func(messageType string, payload any) {
		body, err := json.Marshal(map[string]any{"type": messageType, "payload": payload})
		if err != nil {
			return
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_ = conn.WriteMessage(websocket.TextMessage, body)
	}

	if err := a.coord.TouchPresence(ctx, userID); err != nil {
		observability.Log("warn", "social presence touch failed", map[string]any{"userId": userID, "error": err.Error()})
	}

	var friendIDs []string
	var lastStatuses map[string]string

	presenceTicker := time.NewTicker(5 * time.Second)
	defer presenceTicker.Stop()
	pingTicker := time.NewTicker(20 * time.Second)
	defer pingTicker.Stop()
	friendRefresh := time.NewTicker(15 * time.Second)
	defer friendRefresh.Stop()

	refreshFriends := func() {
		friends, err := a.store.ListFriends(userID)
		if err != nil {
			return
		}
		ids := make([]string, 0, len(friends))
		for _, f := range friends {
			if f.UserID != "" {
				ids = append(ids, f.UserID)
			}
		}
		friendIDs = ids
	}
	pushPresence := func() {
		if len(friendIDs) == 0 {
			return
		}
		statuses, err := a.coord.GetPresenceStatuses(ctx, friendIDs)
		if err != nil {
			return
		}
		changed := map[string]string{}
		for id, status := range statuses {
			s := string(status)
			if lastStatuses[id] != s {
				changed[id] = s
			}
		}
		if len(changed) == 0 {
			return
		}
		if lastStatuses == nil {
			lastStatuses = map[string]string{}
		}
		for id, s := range changed {
			lastStatuses[id] = s
		}
		write("presence", changed)
	}

	refreshFriends()
	pushPresence()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event == nil || strings.TrimSpace(event.Payload) == "" {
				continue
			}
			writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = conn.WriteMessage(websocket.TextMessage, []byte(event.Payload))
			writeMu.Unlock()
		case <-friendRefresh.C:
			refreshFriends()
		case <-presenceTicker.C:
			if err := a.coord.TouchPresence(ctx, userID); err != nil {
				observability.Log("warn", "social presence touch failed", map[string]any{"userId": userID, "error": err.Error()})
			}
			pushPresence()
		case <-pingTicker.C:
			writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			pingErr := conn.WriteMessage(websocket.PingMessage, nil)
			writeMu.Unlock()
			if pingErr != nil {
				return
			}
		}
	}
}
