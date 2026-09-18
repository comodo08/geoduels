package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"geoduels/pkg/auth"
)

const oauthStateTTL = 5 * time.Minute

type oauthStateClaims struct {
	Intent   string `json:"intent,omitempty"`
	Origin   string `json:"origin"`
	ReturnTo string `json:"returnTo,omitempty"`
	LinkSub  string `json:"linkSub,omitempty"`
	Nonce    string `json:"nonce"`
	jwt.RegisteredClaims
}

var oauthPopupTemplate = template.Must(template.New("oauth-popup").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>GeoDuels Sign In</title>
</head>
<body style="background:#0d1216;color:#f4f9ff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0">
  <div style="max-width:420px;padding:24px;text-align:center">
    <p id="status" style="margin:0 0 12px;font-size:16px;font-weight:600">Finishing sign-in...</p>
    <p id="detail" style="margin:0;color:#9db2c7;font-size:14px;line-height:1.5"></p>
    <button id="close" type="button" style="display:none;margin-top:18px;padding:10px 16px;border-radius:999px;border:1px solid rgba(255,255,255,0.12);background:rgba(255,255,255,0.06);color:#f4f9ff;cursor:pointer">Close</button>
  </div>
  <script>
    (function () {
      var payload = {{.Payload}};
      var targetOrigin = {{.TargetOrigin}};
      var message = { type: 'geoduels:auth', payload: payload };
      var statusEl = document.getElementById('status');
      var detailEl = document.getElementById('detail');
      var closeButton = document.getElementById('close');
      if (closeButton) {
        closeButton.addEventListener('click', function () {
          window.close();
        });
      }
      if (window.opener && targetOrigin) {
        window.opener.postMessage(message, targetOrigin);
      }
      if (!window.opener && targetOrigin) {
        var nextURL = new URL(targetOrigin);
        if (payload && payload.returnTo) {
          try {
            nextURL = new URL(String(payload.returnTo), targetOrigin);
          } catch (_error) {}
        }
        if (payload && payload.ok) {
          nextURL.searchParams.set('auth', 'success');
          if (payload && payload.provider) {
            nextURL.searchParams.set('provider', String(payload.provider));
          }
          nextURL.searchParams.delete('authError');
          nextURL.searchParams.delete('googleAuth');
          nextURL.searchParams.delete('googleAuthError');
        } else {
          nextURL.searchParams.set('auth', 'error');
          if (payload && payload.provider) {
            nextURL.searchParams.set('provider', String(payload.provider));
          }
          nextURL.searchParams.set('authError', payload && payload.error ? String(payload.error) : 'Login failed');
          nextURL.searchParams.delete('googleAuth');
          nextURL.searchParams.delete('googleAuthError');
        }
        window.location.replace(nextURL.toString());
        return;
      }
      if (payload && payload.ok) {
        if (statusEl) statusEl.textContent = 'Sign-in complete.';
        if (detailEl) detailEl.textContent = 'This window will close automatically.';
        window.setTimeout(function () {
          window.close();
        }, 400);
        return;
      }
      if (statusEl) statusEl.textContent = 'Sign-in failed.';
      if (detailEl) detailEl.textContent = payload && payload.error ? String(payload.error) : 'The callback did not complete successfully.';
      if (closeButton) closeButton.style.display = 'inline-flex';
    })();
  </script>
</body>
</html>`))

func (a *api) googleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if !a.googleOAuthEnabled() {
		http.Error(w, "google sign-in unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		ReturnTo string `json:"returnTo"`
		Intent   string `json:"intent"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		http.Error(w, "missing origin", http.StatusBadRequest)
		return
	}
	allowedOrigins := allowedOriginsSet()
	if !allowedOrigins[origin] && !allowedOrigins["*"] {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	intent := normalizeOAuthIntent(req.Intent)
	linkSub, err := a.oauthLinkSubject(r, intent)
	if err != nil {
		http.Error(w, oauthStartError(intent), http.StatusUnauthorized)
		return
	}

	state := oauthStateClaims{
		Intent:   intent,
		Origin:   origin,
		ReturnTo: sanitizeOAuthReturnPath(req.ReturnTo),
		LinkSub:  linkSub,
		Nonce:    randomHex(16),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	stateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, state)
	signedState, err := stateToken.SignedString(a.appAuthSecret)
	if err != nil {
		http.Error(w, "failed to create oauth state", http.StatusInternalServerError)
		return
	}

	redirectURI := a.googleRedirectURI(r)
	authURL := a.googleOAuthConfig(redirectURI).AuthCodeURL(
		signedState,
		oauth2.SetAuthURLParam("nonce", state.Nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	_ = json.NewEncoder(w).Encode(map[string]string{"authURL": authURL})
}

func (a *api) googleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if !a.googleOAuthEnabled() {
		http.Error(w, "google sign-in unavailable", http.StatusServiceUnavailable)
		return
	}
	payload := map[string]any{"ok": false, "error": "Sign-in failed", "provider": "google"}
	targetOrigin := ""
	defer func() {
		renderOAuthPopup(w, targetOrigin, payload)
	}()

	if errParam := strings.TrimSpace(r.URL.Query().Get("error")); errParam != "" {
		payload["error"] = errParam
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	stateToken := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || stateToken == "" {
		payload["error"] = "missing oauth response"
		return
	}
	state, err := a.parseOAuthState(stateToken)
	if err != nil {
		payload["error"] = "invalid oauth state"
		return
	}
	targetOrigin = state.Origin
	payload["returnTo"] = state.ReturnTo

	idToken, err := a.exchangeGoogleCode(r.Context(), code, a.googleRedirectURI(r))
	if err != nil {
		payload["error"] = "google exchange failed"
		return
	}
	idClaims, err := a.googleVerifier.ValidateIDToken(r.Context(), idToken, state.Nonce)
	if err != nil {
		payload["error"] = "invalid google identity"
		return
	}
	email := googleAccountEmail(idClaims)
	displayName := strings.TrimSpace(idClaims.Name)
	if displayName == "" {
		displayName = email
	}
	identity, err := a.resolveOAuthIdentity(
		r,
		state,
		"google",
		idClaims.Sub,
		email,
		displayName,
		strings.TrimSpace(idClaims.Picture),
	)
	if err != nil {
		log.Printf("google oauth callback: persist identity failed: %v", err)
		payload["error"] = oauthUserError(err)
		return
	}
	refreshToken, sessionRecord, err := a.createSession(identity.Sub, r)
	if err != nil {
		log.Printf("google oauth callback: create session failed for user %s: %v", identity.Sub, err)
		payload["error"] = "issue session failed"
		return
	}
	accessToken, err := auth.IssueAppAccessToken(a.appAuthSecret, identity.Sub, sessionRecord.ID, a.accessTokenTTL)
	if err != nil {
		log.Printf("google oauth callback: issue access token failed for user %s session %s: %v", identity.Sub, sessionRecord.ID, err)
		payload["error"] = "issue session failed"
		return
	}
	a.setRefreshCookie(w, r, refreshToken)
	payload = a.oauthSessionPayload("google", accessToken, identity, displayName, state.ReturnTo)
}

func googleAccountEmail(claims auth.IdentityTokenClaims) string {
	email := strings.TrimSpace(claims.Email)
	if email == "" || !claims.EmailVerified {
		return claims.Sub + "@oidc.invalid"
	}
	return email
}

func sanitizeOAuthReturnPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/"
	}
	return raw
}

func (a *api) googleOAuthEnabled() bool {
	return a.googleVerifier != nil && a.googleClientID != "" && a.googleSecret != ""
}

func (a *api) parseOAuthState(raw string) (oauthStateClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &oauthStateClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return a.appAuthSecret, nil
	})
	if err != nil || !token.Valid {
		return oauthStateClaims{}, errors.New("invalid state")
	}
	claims, ok := token.Claims.(*oauthStateClaims)
	if !ok || claims.Origin == "" || claims.Nonce == "" {
		return oauthStateClaims{}, errors.New("invalid state claims")
	}
	return *claims, nil
}

func (a *api) exchangeGoogleCode(ctx context.Context, code, redirectURI string) (string, error) {
	token, err := a.googleOAuthConfig(redirectURI).Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	idToken, _ := token.Extra("id_token").(string)
	if strings.TrimSpace(idToken) == "" {
		return "", errors.New("missing id_token")
	}
	return idToken, nil
}

func (a *api) googleOAuthConfig(redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.googleClientID,
		ClientSecret: a.googleSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}

func (a *api) googleRedirectURI(r *http.Request) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s/v1/auth/google/callback", scheme, host)
}

func renderOAuthPopup(w http.ResponseWriter, targetOrigin string, payload map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	payloadJSON, _ := json.Marshal(payload)
	originJSON, _ := json.Marshal(targetOrigin)
	_ = oauthPopupTemplate.Execute(w, map[string]template.JS{
		"Payload":      template.JS(payloadJSON),
		"TargetOrigin": template.JS(originJSON),
	})
}

func randomHex(numBytes int) string {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
