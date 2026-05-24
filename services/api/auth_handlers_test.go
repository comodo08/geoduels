package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"geoduels/pkg/auth"
	"geoduels/pkg/persistence"
)

type guestAuthTestStore struct {
	persistence.Store
	createdGuests int
	identity      persistence.Identity
	sessions      map[string]persistence.RefreshTokenRecord
}

func (s *guestAuthTestStore) CreateGuestIdentity() (persistence.Identity, error) {
	s.createdGuests++
	s.identity = persistence.Identity{
		Sub:         "guest-1",
		DisplayName: "Guest",
		Onboarded:   true,
		AccountType: "guest",
	}
	return s.identity, nil
}

func (s *guestAuthTestStore) IsSignupIPBanned(ipAddress string) (bool, error) {
	return false, nil
}

func (s *guestAuthTestStore) CreateAuthSession(userID, refreshTokenHash string, expiresAt time.Time, params persistence.AuthSessionParams) (persistence.RefreshTokenRecord, error) {
	if s.sessions == nil {
		s.sessions = map[string]persistence.RefreshTokenRecord{}
	}
	rec := persistence.RefreshTokenRecord{
		ID:               "session-1",
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
		LastUsedAt:       time.Now(),
	}
	s.sessions[refreshTokenHash] = rec
	return rec, nil
}

func (s *guestAuthTestStore) GetAuthSessionByRefreshToken(hash string) (persistence.RefreshTokenRecord, bool, error) {
	rec, ok := s.sessions[hash]
	return rec, ok, nil
}

func (s *guestAuthTestStore) RotateAuthSession(sessionID, currentHash, nextHash string, expiresAt time.Time, usedAt time.Time) (persistence.RefreshTokenRecord, bool, error) {
	rec, ok := s.sessions[currentHash]
	if !ok {
		return persistence.RefreshTokenRecord{}, false, nil
	}
	delete(s.sessions, currentHash)
	rec.RefreshTokenHash = nextHash
	rec.ExpiresAt = expiresAt
	rec.LastUsedAt = usedAt
	s.sessions[nextHash] = rec
	return rec, true, nil
}

func (s *guestAuthTestStore) GetIdentity(sub string) (persistence.Identity, error) {
	return s.identity, nil
}

func TestGuestLoginReusesExistingRefreshSession(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                 store,
		redis:                 rdb,
		appAuthSecret:         []byte("01234567890123456789012345678901"),
		accessTokenTTL:        15 * time.Minute,
		refreshTokenTTL:       30 * 24 * time.Hour,
		refreshCookieName:     "geoduels_refresh",
		refreshCookieSameSite: http.SameSiteLaxMode,
		guestSignupIPLimit:    1,
		guestSignupIPWindow:   time.Minute,
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
	firstRec := httptest.NewRecorder()
	a.guestLogin(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first guest login status = %d", firstRec.Code)
	}
	if store.createdGuests != 1 {
		t.Fatalf("created guests after first login = %d", store.createdGuests)
	}
	cookie := firstRec.Result().Cookies()[0]
	if cookie.Value == "" || auth.RefreshTokenHash(cookie.Value) == "" {
		t.Fatal("expected refresh cookie")
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
	secondReq.AddCookie(cookie)
	secondRec := httptest.NewRecorder()
	a.guestLogin(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second guest login status = %d", secondRec.Code)
	}
	if store.createdGuests != 1 {
		t.Fatalf("guest login should reuse cookie session, created guests = %d", store.createdGuests)
	}
}

func TestGuestLoginIgnoresNicknamePayload(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                 store,
		redis:                 rdb,
		appAuthSecret:         []byte("01234567890123456789012345678901"),
		accessTokenTTL:        15 * time.Minute,
		refreshTokenTTL:       30 * 24 * time.Hour,
		refreshCookieName:     "geoduels_refresh",
		refreshCookieSameSite: http.SameSiteLaxMode,
		guestSignupIPLimit:    1,
		guestSignupIPWindow:   time.Minute,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", strings.NewReader(`{"nickname":"Custom"}`))
	rec := httptest.NewRecorder()
	a.guestLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("guest login status = %d", rec.Code)
	}
	if store.identity.DisplayName != "Guest" {
		t.Fatalf("guest display name = %q, want Guest", store.identity.DisplayName)
	}
}

func TestGuestLoginRequiresTurnstileWhenEnabled(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                  store,
		redis:                  rdb,
		appAuthSecret:          []byte("01234567890123456789012345678901"),
		accessTokenTTL:         15 * time.Minute,
		refreshTokenTTL:        30 * 24 * time.Hour,
		refreshCookieName:      "geoduels_refresh",
		refreshCookieSameSite:  http.SameSiteLaxMode,
		guestSignupIPLimit:     10,
		guestSignupIPWindow:    time.Minute,
		guestTurnstileRequired: true,
		turnstileSecret:        "secret",
		httpClient:             http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
	rec := httptest.NewRecorder()
	a.guestLogin(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("guest login status = %d", rec.Code)
	}
	if store.createdGuests != 0 {
		t.Fatalf("guest should not be created, created guests = %d", store.createdGuests)
	}
}

func TestGuestLoginValidatesTurnstileToken(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	var sawRemoteIP string
	verifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse verify body: %v", err)
		}
		if values.Get("secret") != "secret" {
			t.Fatalf("secret = %q", values.Get("secret"))
		}
		if values.Get("response") != "token-123" {
			t.Fatalf("response = %q", values.Get("response"))
		}
		sawRemoteIP = values.Get("remoteip")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":  true,
			"hostname": "play.example.com",
			"action":   guestTurnstileAction,
		})
	}))
	defer verifyServer.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                  store,
		redis:                  rdb,
		appAuthSecret:          []byte("01234567890123456789012345678901"),
		accessTokenTTL:         15 * time.Minute,
		refreshTokenTTL:        30 * 24 * time.Hour,
		refreshCookieName:      "geoduels_refresh",
		refreshCookieSameSite:  http.SameSiteLaxMode,
		guestSignupIPLimit:     10,
		guestSignupIPWindow:    time.Minute,
		guestTurnstileRequired: true,
		turnstileSecret:        "secret",
		turnstileVerifyURL:     verifyServer.URL,
		turnstileHostname:      "play.example.com",
		httpClient:             verifyServer.Client(),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", strings.NewReader(`{"turnstileToken":"token-123"}`))
	req.RemoteAddr = "203.0.113.44:12345"
	rec := httptest.NewRecorder()
	a.guestLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("guest login status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.createdGuests != 1 {
		t.Fatalf("created guests = %d", store.createdGuests)
	}
	if sawRemoteIP != "203.0.113.44" {
		t.Fatalf("remoteip = %q", sawRemoteIP)
	}
}

func TestSessionUserIncludesProfileFields(t *testing.T) {
	user := sessionUser(persistence.Identity{
		Sub:          "user-1",
		Email:        "player@example.com",
		DisplayName:  "Player",
		ProviderName: "discord-player",
		AvatarURL:    "https://cdn.example/avatar.png",
		AccountType:  "registered",
		IsAdmin:      true,
	})

	if user.ID != "user-1" {
		t.Fatalf("user id = %q, want user-1", user.ID)
	}
	if user.Email != "player@example.com" {
		t.Fatalf("email = %q, want player@example.com", user.Email)
	}
	if user.DisplayName != "Player" {
		t.Fatalf("display name = %q, want Player", user.DisplayName)
	}
	if user.AvatarURL != "https://cdn.example/avatar.png" {
		t.Fatalf("avatar url = %q, want profile avatar", user.AvatarURL)
	}
	if user.IsGuest {
		t.Fatal("registered user should not be marked as guest")
	}
	if !user.IsAdmin {
		t.Fatal("expected admin flag to be preserved")
	}
}

func TestGuestLoginRateLimitsNewGuestsByIP(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                 store,
		redis:                 rdb,
		appAuthSecret:         []byte("01234567890123456789012345678901"),
		accessTokenTTL:        15 * time.Minute,
		refreshTokenTTL:       30 * 24 * time.Hour,
		refreshCookieName:     "geoduels_refresh",
		refreshCookieSameSite: http.SameSiteLaxMode,
		guestSignupIPLimit:    2,
		guestSignupIPWindow:   time.Minute,
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
		req.RemoteAddr = "203.0.113.10:12345"
		rec := httptest.NewRecorder()
		a.guestLogin(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("guest login %d status = %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	a.guestLogin(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third guest login status = %d", rec.Code)
	}
	if retryAfter := rec.Header().Get("Retry-After"); retryAfter == "" {
		t.Fatal("expected Retry-After header")
	}
	if store.createdGuests != 2 {
		t.Fatalf("rate-limited request should not create a guest, created guests = %d", store.createdGuests)
	}
}

func TestGuestLoginDailyRateLimitsNewGuestsByIP(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	store := &guestAuthTestStore{}
	a := &api{
		store:                  store,
		redis:                  rdb,
		appAuthSecret:          []byte("01234567890123456789012345678901"),
		accessTokenTTL:         15 * time.Minute,
		refreshTokenTTL:        30 * 24 * time.Hour,
		refreshCookieName:      "geoduels_refresh",
		refreshCookieSameSite:  http.SameSiteLaxMode,
		guestSignupIPLimit:     100,
		guestSignupIPWindow:    time.Minute,
		guestSignupDailyLimit:  2,
		guestSignupDailyWindow: 24 * time.Hour,
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
		req.RemoteAddr = "203.0.113.20:12345"
		rec := httptest.NewRecorder()
		a.guestLogin(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("guest login %d status = %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/guest", nil)
	req.RemoteAddr = "203.0.113.20:12345"
	rec := httptest.NewRecorder()
	a.guestLogin(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third daily guest login status = %d", rec.Code)
	}
	if retryAfter := rec.Header().Get("Retry-After"); retryAfter == "" {
		t.Fatal("expected Retry-After header")
	}
	if store.createdGuests != 2 {
		t.Fatalf("daily rate-limited request should not create a guest, created guests = %d", store.createdGuests)
	}
}
