package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type playerMatchesCursor struct {
	EndedAt string `json:"endedAt"`
	MatchID string `json:"matchId"`
}

func (a *api) publicPlayerProfile(c echo.Context) error {
	nickname := strings.TrimSpace(c.Param("nickname"))
	profile, err := a.profiles.GetPublicPlayerProfileByNickname(nickname)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return plainTextError(c, http.StatusNotFound, "player not found")
		}
		return plainTextError(c, http.StatusInternalServerError, "player profile unavailable")
	}
	c.Response().Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.Response()).Encode(profile)
}

func (a *api) publicPlayerMatches(c echo.Context) error {
	limit := 20
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return plainTextError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = parsed
	}
	profile, err := a.profiles.GetPublicPlayerProfileByNickname(strings.TrimSpace(c.Param("nickname")))
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return plainTextError(c, http.StatusNotFound, "player not found")
		}
		return plainTextError(c, http.StatusInternalServerError, "player profile unavailable")
	}
	var beforeEndedAt time.Time
	var beforeMatchID string
	rankedOnly := strings.EqualFold(strings.TrimSpace(c.QueryParam("filter")), "ranked") ||
		strings.EqualFold(strings.TrimSpace(c.QueryParam("ranked")), "true")
	if rawCursor := strings.TrimSpace(c.QueryParam("cursor")); rawCursor != "" {
		cursor, err := decodePlayerMatchesCursor(rawCursor)
		if err != nil {
			return plainTextError(c, http.StatusBadRequest, "invalid cursor")
		}
		beforeEndedAt, err = time.Parse(time.RFC3339Nano, cursor.EndedAt)
		if err != nil || strings.TrimSpace(cursor.MatchID) == "" {
			return plainTextError(c, http.StatusBadRequest, "invalid cursor")
		}
		beforeMatchID = a.resolveEntityID("match", cursor.MatchID)
	}
	page, err := a.matchStore.ListPlayerMatchHistoryPage(profile.UserID, limit, beforeEndedAt, beforeMatchID, rankedOnly)
	if err != nil {
		return plainTextError(c, http.StatusInternalServerError, "match history unavailable")
	}
	nextCursor := ""
	if page.HasMore {
		nextCursor = encodePlayerMatchesCursor(playerMatchesCursor{
			EndedAt: page.NextEndedAt.UTC().Format(time.RFC3339Nano),
			MatchID: page.NextMatchID,
		})
	}
	c.Response().Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.Response()).Encode(map[string]any{
		"matches":    page.Matches,
		"nextCursor": nextCursor,
	})
}

func encodePlayerMatchesCursor(cursor playerMatchesCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodePlayerMatchesCursor(raw string) (playerMatchesCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return playerMatchesCursor{}, err
	}
	var cursor playerMatchesCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return playerMatchesCursor{}, err
	}
	return cursor, nil
}
