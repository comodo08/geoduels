package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"geoduels/internal/accounts"
	preferencesdomain "geoduels/internal/preferences"
	socialdomain "geoduels/internal/social"
	"geoduels/pkg/auth"
)

type preferencesTestStore struct {
	testRepositories
	value preferencesdomain.UserPreferences
}

func (s *preferencesTestStore) GetUserPreferences(context.Context, string) (preferencesdomain.UserPreferences, error) {
	return s.value, nil
}

func (s *preferencesTestStore) UpdateUserPreferences(_ context.Context, _ string, _ int, preferences json.RawMessage, revision int64) (preferencesdomain.UserPreferences, error) {
	if revision != s.value.Revision {
		return preferencesdomain.UserPreferences{}, preferencesdomain.ErrRevisionConflict
	}
	s.value = preferencesdomain.UserPreferences{
		SchemaVersion: 1,
		Preferences:   preferences,
		Revision:      revision + 1,
	}
	return s.value, nil
}

func (s *preferencesTestStore) GetIdentity(sub string) (accounts.Identity, error) {
	return accounts.Identity{Sub: sub}, nil
}

func preferenceRequest(t *testing.T, method, body string, store *preferencesTestStore) (*api, *http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, "/v1/me/preferences", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	return &api{accounts: store, sessions: store, profiles: store, badges: store, matchStore: store, moderation: store, admin: store, content: store, seasons: store, gameplayMaps: store, runtimeStore: store, chatStore: store, parties: store, social: socialdomain.NewService(store), preferences: preferencesdomain.NewService(store), appAuthSecret: secret}, request, httptest.NewRecorder()
}

func TestUpdateUserPreferences(t *testing.T) {
	store := &preferencesTestStore{value: preferencesdomain.UserPreferences{
		SchemaVersion: 1,
		Preferences:   json.RawMessage(`{}`),
		Revision:      3,
	}}
	a, request, response := preferenceRequest(t, http.MethodPatch, `{"revision":3,"preferences":{"version":1,"audioMuted":true,"bindings":{}}}`, store)

	dispatch(a, response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.value.Revision != 4 {
		t.Fatalf("revision = %d, want 4", store.value.Revision)
	}
}

func TestUpdateUserPreferencesRejectsStaleRevision(t *testing.T) {
	store := &preferencesTestStore{value: preferencesdomain.UserPreferences{Revision: 2}}
	a, request, response := preferenceRequest(t, http.MethodPatch, `{"revision":1,"preferences":{"version":1}}`, store)

	dispatch(a, response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}
