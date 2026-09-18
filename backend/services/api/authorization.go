package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"geoduels/internal/accounts"
	"geoduels/pkg/auth"
)

type requestPrincipalKey struct{}

type requestPrincipal struct {
	claims   auth.AppClaims
	identity accounts.Identity
}

func (a *api) authenticatedAccount(r *http.Request) (auth.AppClaims, accounts.Identity, error) {
	if principal, ok := r.Context().Value(requestPrincipalKey{}).(requestPrincipal); ok {
		return principal.claims, principal.identity, nil
	}
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		return auth.AppClaims{}, accounts.Identity{}, err
	}
	identity, err := a.accounts.GetIdentity(claims.Sub)
	return claims, identity, err
}

func (a *api) authenticatedIdentity(r *http.Request) (accounts.Identity, error) {
	_, identity, err := a.authenticatedAccount(r)
	return identity, err
}

func (a *api) accountBanned(userID string) (bool, error) {
	identity, err := a.accounts.GetIdentity(userID)
	return identity.IsBanned, err
}

// active protects actions that a signed-in but banned account may not
// perform. Authentication, account management, notifications, and read-only
// routes deliberately do not use this wrapper.
func (a *api) active(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims, identity, err := a.authenticatedAccount(c.Request())
		if err != nil {
			return plainTextError(c, http.StatusUnauthorized, "unauthorized")
		}
		if identity.IsBanned {
			return writeAPIError(c, http.StatusForbidden, "account_banned", "user is banned")
		}
		ctx := context.WithValue(c.Request().Context(), requestPrincipalKey{}, requestPrincipal{claims: claims, identity: identity})
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

func writeAPIError(c echo.Context, status int, code, message string) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(status)
	return json.NewEncoder(c.Response()).Encode(map[string]string{"error": message, "code": code})
}
