package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	preferencesdomain "geoduels/internal/preferences"
)

const maxPreferencesBytes = 32 * 1024

func writePreferences(c echo.Context, preferences preferencesdomain.UserPreferences) error {
	c.Response().Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.Response()).Encode(struct {
		Preferences json.RawMessage `json:"preferences"`
		Revision    int64           `json:"revision"`
	}{
		Preferences: preferences.Preferences,
		Revision:    preferences.Revision,
	})
}

func (a *api) updateUserPreferences(c echo.Context) error {
	r := c.Request()
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		return plainTextError(c, http.StatusUnauthorized, "unauthorized")
	}
	var req struct {
		Preferences json.RawMessage `json:"preferences"`
		Revision    int64           `json:"revision"`
	}
	r.Body = http.MaxBytesReader(c.Response(), r.Body, maxPreferencesBytes)
	if err := decodeJSONBody(r, &req); err != nil || len(req.Preferences) == 0 || !json.Valid(req.Preferences) {
		return plainTextError(c, http.StatusBadRequest, "invalid preferences")
	}
	trimmed := bytes.TrimSpace(req.Preferences)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' || req.Revision < 0 {
		return plainTextError(c, http.StatusBadRequest, "invalid preferences")
	}
	var header struct {
		Version int `json:"version"`
	}
	if a.preferences == nil {
		return plainTextError(c, http.StatusNotImplemented, "preferences unavailable")
	}
	if json.Unmarshal(req.Preferences, &header) != nil {
		return plainTextError(c, http.StatusBadRequest, "unsupported preference version")
	}
	preferences, err := a.preferences.Update(r.Context(), claims.Sub, header.Version, req.Preferences, req.Revision)
	if errors.Is(err, preferencesdomain.ErrUnsupportedVersion) {
		return plainTextError(c, http.StatusBadRequest, "unsupported preference version")
	}
	if errors.Is(err, preferencesdomain.ErrRevisionConflict) {
		return plainTextError(c, http.StatusConflict, "preferences changed in another session")
	}
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "failed to save preferences")
	}
	return writePreferences(c, preferences)
}
