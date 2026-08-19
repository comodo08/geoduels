package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"geoduels/pkg/auth"
	"geoduels/pkg/contracts"
	"geoduels/pkg/persistence"
)

type friendsTestStore struct {
	persistence.Store
	profiles        map[string]persistence.Profile
	parties         map[string]contracts.PartySnapshot
	friends         map[string]bool
	removed         []string
	accepted        []string
	declined        []string
	requests        []string
	notifications   []persistence.UserNotification
	notificationTargets []string
	areFriendsPairs map[string]bool
}

func (s *friendsTestStore) GetProfile(userID string) (persistence.Profile, error) {
	if p, ok := s.profiles[userID]; ok {
		return p, nil
	}
	return persistence.Profile{UserID: userID, DisplayName: userID}, nil
}

func (s *friendsTestStore) GetPartyByID(partyID string) (contracts.PartySnapshot, bool, error) {
	if p, ok := s.parties[partyID]; ok {
		return p, true, nil
	}
	return contracts.PartySnapshot{}, false, nil
}

func (s *friendsTestStore) CreateFriendshipRequest(requesterID, addresseeID string) error {
	s.requests = append(s.requests, requesterID+":"+addresseeID)
	return nil
}

func (s *friendsTestStore) ListFriends(userID string) ([]persistence.FriendRow, error) {
	return nil, nil
}

func (s *friendsTestStore) ListFriendRequests(userID string) ([]persistence.FriendRow, []persistence.FriendRow, error) {
	return nil, nil, nil
}

func (s *friendsTestStore) AcceptFriendship(userID, requesterID string) error {
	s.accepted = append(s.accepted, userID+":"+requesterID)
	return nil
}

func (s *friendsTestStore) DeclineFriendship(userID, requesterID string) error {
	s.declined = append(s.declined, userID+":"+requesterID)
	return nil
}

func (s *friendsTestStore) RemoveFriend(userID, friendID string) error {
	s.removed = append(s.removed, userID+":"+friendID)
	return nil
}

func (s *friendsTestStore) AreFriends(userID, otherID string) (bool, error) {
	key := userID + "|" + otherID
	if v, ok := s.areFriendsPairs[key]; ok {
		return v, nil
	}
	return false, nil
}

func (s *friendsTestStore) InsertUserNotification(userID, notificationType, dedupeKey string, payload any) (int64, error) {
	s.notificationTargets = append(s.notificationTargets, userID)
	s.notifications = append(s.notifications, persistence.UserNotification{
		Type: notificationType,
	})
	return int64(len(s.notifications)), nil
}

func (s *friendsTestStore) GetUserNotificationByDedupeKey(userID, dedupeKey string) (persistence.UserNotification, bool, error) {
	return persistence.UserNotification{}, false, nil
}

func newFriendsTestAPI(store *friendsTestStore) *api {
	secret := []byte("test-secret-32-bytes-long-1234567890")
	return &api{
		store:          store,
		appAuthSecret:  secret,
		coord:          nil,
		redis:          nil,
	}
}

func friendsTestToken(t *testing.T, a *api, sub string) string {
	t.Helper()
	tok, err := auth.IssueAppAccessToken(a.appAuthSecret, sub, "session-1", time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

func TestCreateFriendRequestRejectsGuests(t *testing.T) {
	store := &friendsTestStore{
		profiles: map[string]persistence.Profile{
			"guest-1": {UserID: "guest-1", DisplayName: "Guest", IsGuest: true},
			"user-2":  {UserID: "user-2", DisplayName: "Target"},
		},
	}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "guest-1")

	body := strings.NewReader(`{"targetUserId":"user-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/me/friends/requests", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	a.createFriendRequest(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (guest rejected), body=%q", rec.Code, rec.Body.String())
	}
	if len(store.requests) != 0 {
		t.Fatal("guest must not create a friendship request")
	}
}

func TestCreateFriendRequestInsertsNotification(t *testing.T) {
	store := &friendsTestStore{
		profiles: map[string]persistence.Profile{
			"user-1": {UserID: "user-1", DisplayName: "Sender"},
			"user-2": {UserID: "user-2", DisplayName: "Target"},
		},
	}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	body := strings.NewReader(`{"targetUserId":"user-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/me/friends/requests", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	a.createFriendRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", rec.Code, rec.Body.String())
	}
	if len(store.notifications) != 1 || store.notifications[0].Type != "friend_request" {
		t.Fatalf("expected one friend_request notification, got %+v", store.notifications)
	}
	if store.notificationTargets[0] != "user-2" {
		t.Fatalf("notification should target the addressee, got %q", store.notificationTargets[0])
	}
}

func TestInviteFriendRequiresPartyOwner(t *testing.T) {
	store := &friendsTestStore{
		parties: map[string]contracts.PartySnapshot{
			"party-1": {ID: "party-1", InviteCode: "ABCD12", OwnerUserID: "other-owner"},
		},
	}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	body := strings.NewReader(`{"friendUserId":"user-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/parties/party-1/invite-friend", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req = mux.SetURLVars(req, map[string]string{"partyId": "party-1"})
	rec := httptest.NewRecorder()
	a.inviteFriendToParty(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (non-owner), body=%q", rec.Code, rec.Body.String())
	}
	if len(store.notifications) != 0 {
		t.Fatal("non-owner must not send party invite")
	}
}

func TestInviteFriendRequiresFriendship(t *testing.T) {
	store := &friendsTestStore{
		parties: map[string]contracts.PartySnapshot{
			"party-1": {ID: "party-1", InviteCode: "ABCD12", OwnerUserID: "user-1"},
		},
		areFriendsPairs: map[string]bool{},
	}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	body := strings.NewReader(`{"friendUserId":"user-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/parties/party-1/invite-friend", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req = mux.SetURLVars(req, map[string]string{"partyId": "party-1"})
	rec := httptest.NewRecorder()
	a.inviteFriendToParty(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (not friends), body=%q", rec.Code, rec.Body.String())
	}
}

func TestInviteFriendSuccess(t *testing.T) {
	store := &friendsTestStore{
		parties: map[string]contracts.PartySnapshot{
			"party-1": {ID: "party-1", InviteCode: "ABCD12", OwnerUserID: "user-1"},
		},
		areFriendsPairs: map[string]bool{"user-1|user-2": true},
	}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	body := strings.NewReader(`{"friendUserId":"user-2"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/parties/party-1/invite-friend", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req = mux.SetURLVars(req, map[string]string{"partyId": "party-1"})
	rec := httptest.NewRecorder()
	a.inviteFriendToParty(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", rec.Code, rec.Body.String())
	}
	if len(store.notifications) != 1 || store.notifications[0].Type != "party_invite" {
		t.Fatalf("expected one party_invite notification, got %+v", store.notifications)
	}
}

func TestAcceptFriendRequestCallsAccept(t *testing.T) {
	store := &friendsTestStore{}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	req := httptest.NewRequest(http.MethodPost, "/v1/me/friends/requests/user-2/accept", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req = mux.SetURLVars(req, map[string]string{"requesterId": "user-2"})
	rec := httptest.NewRecorder()
	a.acceptFriendRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", rec.Code, rec.Body.String())
	}
	if len(store.accepted) != 1 || store.accepted[0] != "user-1:user-2" {
		t.Fatalf("expected accept for user-1:user-2, got %+v", store.accepted)
	}
}

func TestRemoveFriendCallsRemove(t *testing.T) {
	store := &friendsTestStore{}
	a := newFriendsTestAPI(store)
	tok := friendsTestToken(t, a, "user-1")

	req := httptest.NewRequest(http.MethodDelete, "/v1/me/friends/user-2", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req = mux.SetURLVars(req, map[string]string{"friendUserId": "user-2"})
	rec := httptest.NewRecorder()
	a.removeFriend(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%q", rec.Code, rec.Body.String())
	}
	if len(store.removed) != 1 || store.removed[0] != "user-1:user-2" {
		t.Fatalf("expected remove for user-1:user-2, got %+v", store.removed)
	}
}
