package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
	"geoduels/pkg/duel"
	"geoduels/pkg/observability"
	"geoduels/pkg/persistence"
)

type stubStore struct {
	persistence.Store
	finalizeCalls int
}

func (s *stubStore) FinalizeMatch(snap contracts.MatchSnapshot, ownerEpoch int64) (contracts.MatchSnapshot, error) {
	s.finalizeCalls++
	return snap, nil
}

func (s *stubStore) Close() {}

func newTestNode(t *testing.T) *gameplayNode {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	roundForPlan := func(matchID string, roundIndex int) (contracts.LocationPoint, error) {
		return contracts.LocationPoint{Lat: 40.7128, Lng: -74.006}, nil
	}

	g := &gameplayNode{
		nodeID:    "node-test",
		nodeEpoch: 1,
		persist:   &stubStore{},
		coord:     coordinator.NewStore(rdb, 10*time.Second, 2*time.Hour, 24*time.Hour, 5*time.Second),
		redis:     rdb,
		runtimes: map[contracts.MatchMode]gameplayRuntime{
			contracts.ModeDuel: duelRuntime{
				mode:    contracts.ModeDuel,
				engine:  duel.New(roundForPlan),
				configs: newMatchConfigRegistry(),
			},
		},
		conns:        map[string]*websocket.Conn{},
		connWrite:    map[string]*sync.Mutex{},
		connID:       map[string]string{},
		userMatch:    map[string]string{},
		matchUsers:   map[string][]string{},
		matchModes:   map[string]contracts.MatchMode{},
		finalizing:   map[string]bool{},
		endedMatches: map[string]time.Time{},
		metrics:      observability.NewRuntimeMetrics(),
	}
	return g
}

func attachSocket(t *testing.T, g *gameplayNode, userID, connID, matchID string) *websocket.Conn {
	t.Helper()
	done := make(chan struct{})
	registered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		g.mu.Lock()
		g.conns[userID] = conn
		g.connWrite[userID] = &sync.Mutex{}
		g.connID[userID] = connID
		g.userMatch[userID] = matchID
		g.mu.Unlock()
		close(registered)
		<-done
		_ = conn.Close()
	}))
	t.Cleanup(func() {
		close(done)
		srv.Close()
	})

	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	<-registered
	return client
}

func readEvent(t *testing.T, ws *websocket.Conn) map[string]any {
	t.Helper()
	type result struct {
		msg []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		_, msg, err := ws.ReadMessage()
		ch <- result{msg: msg, err: err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("read event: %v", r.err)
		}
		var payload map[string]any
		if err := json.Unmarshal(r.msg, &payload); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		return payload
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
		return nil
	}
}

func startDuel(t *testing.T, g *gameplayNode, matchID string) gameplayRuntime {
	t.Helper()
	runtime := g.runtimes[contracts.ModeDuel]
	if err := runtime.CreateMatch(matchID, []string{"u1", "u2"}, map[string]contracts.PlayerProfile{
		"u1": {UserID: "u1", DisplayName: "P1"},
		"u2": {UserID: "u2", DisplayName: "P2"},
	}, true, "", contracts.MatchConfig{}, nil); err != nil {
		t.Fatalf("create match: %v", err)
	}
	g.mu.Lock()
	g.matchUsers[matchID] = []string{"u1", "u2"}
	g.matchModes[matchID] = contracts.ModeDuel
	g.mu.Unlock()
	return runtime
}

func TestPostEndDisconnectStillReachesRemainingPlayer(t *testing.T) {
	g := newTestNode(t)
	matchID := "m-1"
	runtime := startDuel(t, g, matchID)

	u1WS := attachSocket(t, g, "u1", "c1", matchID)

	endedSnap, err := runtime.Forfeit(matchID, "u2")
	if err != nil {
		t.Fatalf("forfeit: %v", err)
	}
	if endedSnap.State != contracts.MatchEnded {
		t.Fatalf("state = %q, want ended", endedSnap.State)
	}

	g.terminalize(matchID, endedSnap)
	finalEvt := readEvent(t, u1WS)
	if finalEvt["kind"] != "event" || finalEvt["type"] != string(contracts.EventMatchState) {
		t.Fatalf("unexpected finalized event: %v", finalEvt)
	}

	g.mu.RLock()
	if _, ended := g.endedMatches[matchID]; !ended {
		t.Fatal("match not marked ended-retained")
	}
	if got := len(g.matchUsers[matchID]); got != 2 {
		t.Fatalf("matchUsers after terminalize = %d, want 2", got)
	}
	if g.userMatch["u1"] != matchID {
		t.Fatal("userMatch entry dropped after terminalize")
	}
	g.mu.RUnlock()

	// The opponent leaves on the results screen; their socket closes.
	g.onDisconnect("u2", matchID, "c2")
	evt := readEvent(t, u1WS)
	if evt["kind"] != "event" || evt["type"] != string(contracts.EventMatchState) {
		t.Fatalf("unexpected disconnect event: %v", evt)
	}
	payload, _ := evt["payload"].(map[string]any)
	players, _ := payload["players"].(map[string]any)
	u2p, _ := players["u2"].(map[string]any)
	if u2p["disconnected"] != true {
		t.Fatalf("u2 disconnected = %v, want true", u2p["disconnected"])
	}

	if got := g.activeMatchCount(); got != 0 {
		t.Fatalf("activeMatchCount = %d, want 0", got)
	}

	g.onDisconnect("u1", matchID, "c1")
	g.mu.RLock()
	if _, ok := g.matchUsers[matchID]; ok {
		t.Fatal("routing not released after all sockets gone")
	}
	if _, ok := g.endedMatches[matchID]; ok {
		t.Fatal("endedMatches not released after all sockets gone")
	}
	if g.userMatch["u1"] != "" || g.userMatch["u2"] != "" {
		t.Fatal("userMatch entries not released after all sockets gone")
	}
	g.mu.RUnlock()
}

func TestTerminalizeDoesNotRefinalize(t *testing.T) {
	g := newTestNode(t)
	matchID := "m-2"
	runtime := startDuel(t, g, matchID)

	endedSnap, err := runtime.Forfeit(matchID, "u2")
	if err != nil {
		t.Fatalf("forfeit: %v", err)
	}
	g.terminalize(matchID, endedSnap)
	g.terminalize(matchID, endedSnap)

	stub, ok := g.persist.(*stubStore)
	if !ok {
		t.Fatal("persist is not stubStore")
	}
	if stub.finalizeCalls != 1 {
		t.Fatalf("finalize calls = %d, want 1", stub.finalizeCalls)
	}
}

func TestSweepEndedMatchesReleasesExpiredRetention(t *testing.T) {
	g := newTestNode(t)

	g.mu.Lock()
	expired := "m-3"
	g.matchUsers[expired] = []string{"u1", "u2"}
	g.matchModes[expired] = contracts.ModeDuel
	g.userMatch["u1"] = expired
	g.userMatch["u2"] = expired
	g.endedMatches[expired] = time.Now().Add(-endedMatchRetention - time.Second)

	fresh := "m-4"
	g.matchUsers[fresh] = []string{"u3", "u4"}
	g.matchModes[fresh] = contracts.ModeDuel
	g.userMatch["u3"] = fresh
	g.userMatch["u4"] = fresh
	g.endedMatches[fresh] = time.Now()
	g.mu.Unlock()

	g.sweepEndedMatches()

	g.mu.RLock()
	if _, ok := g.matchUsers[expired]; ok {
		t.Fatal("expired retained match not released")
	}
	if g.userMatch["u1"] != "" || g.userMatch["u2"] != "" {
		t.Fatal("userMatch entries for expired match not released")
	}
	if _, ok := g.matchUsers[fresh]; !ok {
		t.Fatal("fresh retained match released early")
	}
	g.mu.RUnlock()
}

func TestReleaseEndedMatchIfIdleKeepsWhileAnyConnected(t *testing.T) {
	g := newTestNode(t)
	matchID := "m-5"

	g.mu.Lock()
	g.matchUsers[matchID] = []string{"u1", "u2"}
	g.matchModes[matchID] = contracts.ModeDuel
	g.userMatch["u1"] = matchID
	g.endedMatches[matchID] = time.Now()
	g.conns["u1"] = &websocket.Conn{}
	g.connID["u1"] = "c1"
	g.mu.Unlock()

	g.releaseEndedMatchIfIdle(matchID)
	g.mu.RLock()
	if _, ok := g.matchUsers[matchID]; !ok {
		t.Fatal("released while u1 still connected")
	}
	g.mu.RUnlock()

	g.mu.Lock()
	delete(g.conns, "u1")
	g.mu.Unlock()
	g.releaseEndedMatchIfIdle(matchID)
	g.mu.RLock()
	if _, ok := g.matchUsers[matchID]; ok {
		t.Fatal("not released after last socket gone")
	}
	if _, ok := g.endedMatches[matchID]; ok {
		t.Fatal("endedMatches not released after last socket gone")
	}
	g.mu.RUnlock()
}

func TestReleaseEndedMatchIfIdleIgnoresLiveMatches(t *testing.T) {
	g := newTestNode(t)
	matchID := "m-6"

	g.mu.Lock()
	g.matchUsers[matchID] = []string{"u1", "u2"}
	g.matchModes[matchID] = contracts.ModeDuel
	g.mu.Unlock()

	g.releaseEndedMatchIfIdle(matchID)
	g.mu.RLock()
	if _, ok := g.matchUsers[matchID]; !ok {
		t.Fatal("live match routing was released")
	}
	g.mu.RUnlock()
}
