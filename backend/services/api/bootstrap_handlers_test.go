package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	socialdomain "geoduels/internal/social"
	"geoduels/pkg/auth"
	"geoduels/pkg/coordinator"
)

type friendsPageTestStore struct{ testRepositories }

func (s *friendsPageTestStore) GetSocialAccount(context.Context, string) (bool, bool, bool, error) {
	return false, true, true, nil
}
func (s *friendsPageTestStore) ListFriends(_ context.Context, _ string, _ int) ([]socialdomain.CompactPlayer, error) {
	return []socialdomain.CompactPlayer{{UserID: "friend-1", DisplayName: "Friend"}}, nil
}
func (s *friendsPageTestStore) ListFriendRequests(_ context.Context, _ string, direction string, _ int) ([]socialdomain.FriendRequest, error) {
	return []socialdomain.FriendRequest{{ID: direction + "-1", Direction: direction}}, nil
}
func (s *friendsPageTestStore) ListRecentPlayers(_ context.Context, _ string, _ int) ([]socialdomain.CompactPlayer, error) {
	return []socialdomain.CompactPlayer{{UserID: "recent-1", DisplayName: "Recent"}}, nil
}
func (s *friendsPageTestStore) ListPartyInviteStatus(_ context.Context, _ string, partyID string) (map[string]socialdomain.CompactPartyInvite, error) {
	if partyID != "party-1" {
		return map[string]socialdomain.CompactPartyInvite{}, nil
	}
	created := time.Unix(1_700_000_000, 0).UTC()
	return map[string]socialdomain.CompactPartyInvite{
		"friend-1": {ID: "invite-1", CreatedAt: created, ExpiresAt: created.Add(20 * time.Minute)},
	}, nil
}

func TestFriendsPageReturnsOneCohesiveReadModel(t *testing.T) {
	store := &friendsPageTestStore{}
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/me/friends-page", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	a := &api{social: socialdomain.NewService(store), appAuthSecret: secret}

	dispatch(a, response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Friends       []socialdomain.CompactPlayer            `json:"friends"`
		Requests      map[string][]socialdomain.FriendRequest `json:"requests"`
		RecentPlayers []socialdomain.CompactPlayer            `json:"recentPlayers"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Friends) != 1 || len(body.Requests["incoming"]) != 1 || len(body.Requests["outgoing"]) != 1 || len(body.RecentPlayers) != 1 {
		t.Fatalf("unexpected friends page response: %+v", body)
	}
	if body.Friends[0].PartyInvite != nil {
		t.Fatal("friends page without partyId must omit party invite status")
	}
}

func TestFriendsPageAttachesPartyInviteStatus(t *testing.T) {
	store := &friendsPageTestStore{}
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/me/friends-page?partyId=party-1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	a := &api{social: socialdomain.NewService(store), appAuthSecret: secret}

	dispatch(a, response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Friends []socialdomain.CompactPlayer `json:"friends"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Friends[0].PartyInvite == nil || body.Friends[0].PartyInvite.ID != "invite-1" {
		t.Fatalf("expected party invite status, got %+v", body.Friends[0].PartyInvite)
	}
}

func TestApplySocialPresenceUsesCoordinatorOnlineSet(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	coordStore := coordinator.NewStore(rdb, time.Minute, time.Hour, time.Hour, time.Second)
	if err := coordStore.TouchPresence(t.Context(), "online-friend"); err != nil {
		t.Fatal(err)
	}
	seen := time.Now()
	players := []socialdomain.CompactPlayer{
		{UserID: "online-friend", LastSeenAt: &seen},
		{UserID: "offline-friend", LastSeenAt: &seen},
		{UserID: "hidden-friend"},
	}
	a := &api{coord: coordStore}
	a.applySocialPresence(t.Context(), players)
	if players[0].PresenceStatus != "online" {
		t.Fatalf("online friend status = %q", players[0].PresenceStatus)
	}
	if players[1].PresenceStatus != "offline" {
		t.Fatalf("offline friend status = %q", players[1].PresenceStatus)
	}
	if players[2].PresenceStatus != "" {
		t.Fatalf("hidden friend status = %q", players[2].PresenceStatus)
	}
}
