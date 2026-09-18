package main

import (
	"geoduels/internal/accounts"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	socialdomain "geoduels/internal/social"
	"geoduels/pkg/auth"
)

func TestActiveAccountRejectsBannedUserWithStructuredError(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	store := &adminModerationTestStore{identity: accounts.Identity{Sub: "user-1", IsBanned: true}}
	a := &api{accounts: store, sessions: store, profiles: store, badges: store, matchStore: store, moderation: store, admin: store, content: store, seasons: store, gameplayMaps: store, runtimeStore: store, chatStore: store, parties: store, social: socialdomain.NewService(store), appAuthSecret: secret}
	called := false
	e := echo.New()
	e.POST("/action", func(c echo.Context) error { called = true; return nil }, a.active)
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v", rec.Code, called)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"code":"account_banned"`) || !strings.Contains(body, `"error":"user is banned"`) {
		t.Fatalf("body=%q", body)
	}
}

func TestActiveAccountPassesResolvedIdentityToHandler(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	token, err := auth.IssueAppAccessToken(secret, "user-1", "session-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	store := &adminModerationTestStore{identity: accounts.Identity{Sub: "user-1"}}
	a := &api{accounts: store, sessions: store, profiles: store, badges: store, matchStore: store, moderation: store, admin: store, content: store, seasons: store, gameplayMaps: store, runtimeStore: store, chatStore: store, parties: store, social: socialdomain.NewService(store), appAuthSecret: secret}
	e := echo.New()
	e.POST("/action", func(c echo.Context) error {
		_, identity, err := a.authenticatedAccount(c.Request())
		if err != nil || identity.Sub != "user-1" {
			t.Fatalf("request identity=%+v err=%v", identity, err)
		}
		return c.NoContent(http.StatusNoContent)
	}, a.active)
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
