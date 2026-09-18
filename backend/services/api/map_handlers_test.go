package main

import (
	"geoduels/internal/profiles"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	socialdomain "geoduels/internal/social"
	"geoduels/pkg/auth"
)

type mapUserTestStore struct {
	testRepositories
	profile profiles.Profile
}

func (s *mapUserTestStore) GetProfile(userID string) (profiles.Profile, error) {
	p := s.profile
	p.UserID = userID
	return p, nil
}

func TestMapUserRejectsGuestWhenRegisteredAccountRequired(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "guest-1", "session-1", time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	a := &api{
		accounts: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, sessions: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, profiles: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, badges: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, matchStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, moderation: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, admin: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, content: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, seasons: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, gameplayMaps: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, runtimeStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, chatStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, parties: &mapUserTestStore{profile: profiles.Profile{IsGuest: true}}, social: socialdomain.NewService(&mapUserTestStore{profile: profiles.Profile{IsGuest: true}}),
		appAuthSecret: secret,
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/maps/map-1/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	userID, ok := a.mapUser(c, true)

	if ok || userID != "" {
		t.Fatalf("guest map user unexpectedly allowed: ok=%v userID=%q", ok, userID)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "guest accounts cannot interact with maps") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestMapUserAllowsRegisteredAccountForMapInteractions(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	a := &api{
		accounts: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, sessions: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, profiles: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, badges: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, matchStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, moderation: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, admin: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, content: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, seasons: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, gameplayMaps: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, runtimeStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, chatStore: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, parties: &mapUserTestStore{profile: profiles.Profile{IsGuest: false}}, social: socialdomain.NewService(&mapUserTestStore{profile: profiles.Profile{IsGuest: false}}),
		appAuthSecret: secret,
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/maps/map-1/favorite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	userID, ok := a.mapUser(c, true)

	if !ok || userID != "user-1" {
		t.Fatalf("registered map user rejected: ok=%v userID=%q", ok, userID)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
