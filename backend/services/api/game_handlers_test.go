package main

import (
	"context"
	"encoding/json"
	"geoduels/internal/matches"
	"geoduels/internal/seasons"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"geoduels/internal/leaderboard"

	socialdomain "geoduels/internal/social"
	"geoduels/pkg/auth"
	"geoduels/pkg/contracts"
	"geoduels/pkg/coordinator"
)

type matchAccessTestStore struct {
	testRepositories
	snapshot []byte
}

const testMatchID = "00000000-0000-7000-8000-000000000101"

type leaderboardTestStore struct {
	testRepositories
	settings seasons.RankedSeasonSettings
}

func (s *leaderboardTestStore) GetRankedSeasonSettings() (seasons.RankedSeasonSettings, error) {
	return s.settings, nil
}

func (s *leaderboardTestStore) ListLeaderboard(context.Context, string, string, int, int) ([]leaderboard.Entry, error) {
	return []leaderboard.Entry{}, nil
}

func (s *leaderboardTestStore) GetLeaderboardOverview(_ context.Context, userID, mode, seasonID string, limit int) (leaderboard.Overview, error) {
	return leaderboard.Overview{
		Mode:         mode,
		SeasonID:     seasonID,
		TotalPlayers: 12,
	}, nil
}

func (s *matchAccessTestStore) GetFinalMatchSnapshot(matchID string) ([]byte, bool, error) {
	if matchID != testMatchID || len(s.snapshot) == 0 {
		return nil, false, nil
	}
	return s.snapshot, true, nil
}

func (s *matchAccessTestStore) GetIdentity(sub string) (Identity, error) {
	return Identity{Sub: sub}, nil
}

func (s *matchAccessTestStore) GetRuntimeMatch(_ context.Context, matchID string) (matches.RuntimeMatch, bool, error) {
	return matches.RuntimeMatch{}, false, nil
}

func (s *matchAccessTestStore) MatchSessionSourceParty(_ context.Context, matchID string) (string, string, bool, error) {
	return "", "", false, nil
}

func TestLeaderboardIncludesActiveSeasonResetTime(t *testing.T) {
	nextResetAt := time.Date(2026, time.July, 1, 21, 0, 0, 0, time.UTC)
	store := &leaderboardTestStore{
		settings: seasons.RankedSeasonSettings{
			ActiveSeasonID: "s3",
			NextResetAt:    &nextResetAt,
		},
	}
	a := &api{seasons: store}
	a.leaderboardService = leaderboard.NewService(store)
	req := httptest.NewRequest(http.MethodGet, "/v1/leaderboard", nil)
	rec := httptest.NewRecorder()

	dispatch(a, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	var response struct {
		Season       string     `json:"season"`
		NextResetAt  *time.Time `json:"nextResetAt"`
		TotalPlayers int        `json:"totalPlayers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Season != "s3" || response.TotalPlayers != 12 {
		t.Fatalf("unexpected leaderboard metadata: %+v", response)
	}
	if response.NextResetAt == nil || !response.NextResetAt.Equal(nextResetAt) {
		t.Fatalf("next reset = %v, want %s", response.NextResetAt, nextResetAt)
	}
}

func TestPublicFinalMatchSnapshotIsAvailableToAnyViewer(t *testing.T) {
	raw, err := json.Marshal(contracts.MatchSnapshot{
		MatchID: testMatchID,
		State:   contracts.MatchEnded,
		Players: map[string]contracts.PlayerState{
			"player-1": {UserID: "player-1"},
			"player-2": {UserID: "player-2"},
		},
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	a := &api{accounts: &matchAccessTestStore{snapshot: raw}, sessions: &matchAccessTestStore{snapshot: raw}, profiles: &matchAccessTestStore{snapshot: raw}, badges: &matchAccessTestStore{snapshot: raw}, matchStore: &matchAccessTestStore{snapshot: raw}, moderation: &matchAccessTestStore{snapshot: raw}, admin: &matchAccessTestStore{snapshot: raw}, content: &matchAccessTestStore{snapshot: raw}, seasons: &matchAccessTestStore{snapshot: raw}, gameplayMaps: &matchAccessTestStore{snapshot: raw}, runtimeStore: &matchAccessTestStore{snapshot: raw}, chatStore: &matchAccessTestStore{snapshot: raw}, parties: &matchAccessTestStore{snapshot: raw}, social: socialdomain.NewService(&matchAccessTestStore{snapshot: raw})}
	snapshot, found, err := a.getPublicFinalMatchSnapshot(testMatchID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if !found || snapshot == nil {
		t.Fatal("expected snapshot to be found")
	}
}

func TestMatchRouteReturnsPublicHistoryWithoutAuth(t *testing.T) {
	const matchID = testMatchID
	raw, err := json.Marshal(contracts.MatchSnapshot{
		MatchID: matchID,
		State:   contracts.MatchEnded,
		Phase:   contracts.PhaseEnded,
		Players: map[string]contracts.PlayerState{
			"player-1": {UserID: "player-1", LastGuessLat: 1, LastGuessLng: 2, Disconnected: true},
			"player-2": {UserID: "player-2"},
		},
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	a := &api{accounts: &matchAccessTestStore{snapshot: raw}, sessions: &matchAccessTestStore{snapshot: raw}, profiles: &matchAccessTestStore{snapshot: raw}, badges: &matchAccessTestStore{snapshot: raw}, matchStore: &matchAccessTestStore{snapshot: raw}, moderation: &matchAccessTestStore{snapshot: raw}, admin: &matchAccessTestStore{snapshot: raw}, content: &matchAccessTestStore{snapshot: raw}, seasons: &matchAccessTestStore{snapshot: raw}, gameplayMaps: &matchAccessTestStore{snapshot: raw}, runtimeStore: &matchAccessTestStore{snapshot: raw}, chatStore: &matchAccessTestStore{snapshot: raw}, parties: &matchAccessTestStore{snapshot: raw}, social: socialdomain.NewService(&matchAccessTestStore{snapshot: raw})}
	req := httptest.NewRequest(http.MethodGet, "/v1/matches/"+matchID+"/route", nil)
	rec := httptest.NewRecorder()

	dispatch(a, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	var resp contracts.MatchSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "history" || resp.Snapshot == nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if got := resp.Snapshot.Players["player-1"]; got.LastGuessLat != 0 || got.LastGuessLng != 0 || got.Disconnected {
		t.Fatalf("snapshot was not sanitized: %+v", got)
	}
}

func TestMatchSessionAllowsGuestAssignedToLiveMatch(t *testing.T) {
	const matchID = testMatchID
	appSecret := []byte("01234567890123456789012345678901")
	ticketSecret := []byte("abcdefghijklmnopqrstuvwxyz012345")

	gameplay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/internal/matches/"+matchID {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer gameplay.Close()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	coordStore := coordinator.NewStore(rdb, time.Minute, time.Hour, time.Hour, time.Second)
	if err := coordStore.RegisterNode(t.Context(), coordinator.NodeRecord{
		NodeID:      "node-1",
		PublicRoute: "node-1",
		InternalURL: gameplay.URL,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}
	if err := coordStore.SaveAssignment(t.Context(), coordinator.Assignment{
		MatchID:     matchID,
		Mode:        contracts.ModeDuel,
		NodeID:      "node-1",
		PublicRoute: "node-1",
		Players:     []string{"guest-1", "guest-2"},
	}); err != nil {
		t.Fatalf("save assignment: %v", err)
	}

	token, err := auth.IssueAppAccessToken(appSecret, "guest-2", "session-1", time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	a := &api{
		accounts: &matchAccessTestStore{}, sessions: &matchAccessTestStore{}, profiles: &matchAccessTestStore{}, badges: &matchAccessTestStore{}, matchStore: &matchAccessTestStore{}, moderation: &matchAccessTestStore{}, admin: &matchAccessTestStore{}, content: &matchAccessTestStore{}, seasons: &matchAccessTestStore{}, gameplayMaps: &matchAccessTestStore{}, runtimeStore: &matchAccessTestStore{}, chatStore: &matchAccessTestStore{}, parties: &matchAccessTestStore{}, social: socialdomain.NewService(&matchAccessTestStore{}),
		coord:          coordStore,
		appAuthSecret:  appSecret,
		ticketAuth:     ticketSecret,
		internalSecret: "",
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/matches/"+matchID+"/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	dispatch(a, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	var resp contracts.MatchSessionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "live_connectable" {
		t.Fatalf("status = %q, want live_connectable", resp.Status)
	}
	if resp.MatchID != matchID || resp.Node != "node-1" || resp.WSPath != "/ws/node-1" || resp.Ticket == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
