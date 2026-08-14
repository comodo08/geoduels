package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

const maxAvatarUploadBytes = int64(2 << 20)

const maxAvatarDimension = 512

var allowedAvatarContentTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
}

func isAllowedAvatarContentType(contentType string) bool {
	_, ok := allowedAvatarContentTypes[strings.TrimSpace(strings.ToLower(contentType))]
	return ok
}

func (a *api) avatarBaseURL(r *http.Request) string {
	base := strings.TrimSpace(a.apiPublicBaseURL)
	if base != "" {
		return strings.TrimRight(base, "/")
	}
	scheme := "http"
	if requestIsHTTPS(r) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// updateAvatar accepts a multipart PNG/JPEG avatar, validates it, stores it, and
// re-issues the auth session payload so the new avatar propagates immediately.
func (a *api) updateAvatar(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	identity, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		http.Error(w, "identity not found", http.StatusUnauthorized)
		return
	}
	if identity.AccountType == "guest" {
		http.Error(w, "guests cannot upload avatars", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarUploadBytes)
	if err := r.ParseMultipartForm(maxAvatarUploadBytes); err != nil {
		http.Error(w, "avatar must be multipart form data under 2 MiB", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "avatar file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Size > maxAvatarUploadBytes {
		http.Error(w, "avatar must be under 2 MiB", http.StatusBadRequest)
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxAvatarUploadBytes+1))
	if err != nil {
		http.Error(w, "could not read avatar", http.StatusBadRequest)
		return
	}
	if int64(len(data)) > maxAvatarUploadBytes {
		http.Error(w, "avatar must be under 2 MiB", http.StatusBadRequest)
		return
	}

	contentType, err := detectAvatarContentType(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := a.store.SetUserAvatar(claims.Sub, contentType, data, a.avatarBaseURL(r))
	if err != nil {
		log.Printf("avatar upload failed for user %s: %v", claims.Sub, err)
		a.writeAvatarStoreError(w, err)
		return
	}
	_ = url

	updated, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		http.Error(w, "identity not found", http.StatusUnauthorized)
		return
	}
	payload, err := a.issueAuthSessionPayload(updated, claims.SessionID)
	if err != nil {
		http.Error(w, "issue session failed", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// resetAvatar clears the custom avatar and falls back to the provider avatar.
func (a *api) resetAvatar(w http.ResponseWriter, r *http.Request) {
	claims, err := a.authenticatedClaims(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := a.store.ResetUserAvatar(claims.Sub); err != nil {
		http.Error(w, "failed to reset avatar", http.StatusInternalServerError)
		return
	}
	updated, err := a.store.GetIdentity(claims.Sub)
	if err != nil {
		http.Error(w, "identity not found", http.StatusUnauthorized)
		return
	}
	payload, err := a.issueAuthSessionPayload(updated, claims.SessionID)
	if err != nil {
		http.Error(w, "issue session failed", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// serveAvatar serves a stored avatar publicly. The content type comes strictly
// from the stored whitelist to avoid content-type spoofing or SVG execution.
func (a *api) serveAvatar(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(mux.Vars(r)["userId"])
	if userID == "" {
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	contentType, data, err := a.store.GetUserAvatar(userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "avatar unavailable", http.StatusInternalServerError)
		return
	}
	if !isAllowedAvatarContentType(contentType) {
		http.Error(w, "avatar unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(data)
}

// adminResetPlayerAvatar force-resets a player's avatar (moderation takedown).
func (a *api) adminResetPlayerAvatar(w http.ResponseWriter, r *http.Request) {
	admin, err := a.adminIdentity(r)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	userID := strings.TrimSpace(mux.Vars(r)["id"])
	if userID == "" {
		http.Error(w, "player id required", http.StatusBadRequest)
		return
	}
	if userID == admin.Sub {
		http.Error(w, "use the account settings reset instead", http.StatusBadRequest)
		return
	}
	if err := a.store.ResetUserAvatar(userID); err != nil {
		http.Error(w, "failed to reset avatar", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) writeAvatarStoreError(w http.ResponseWriter, err error) {
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "guest"):
		http.Error(w, err.Error(), http.StatusForbidden)
	case strings.Contains(msg, "size limit") || strings.Contains(msg, "under 2 mib"):
		http.Error(w, "avatar must be under 2 MiB", http.StatusBadRequest)
	case strings.Contains(msg, "512"):
		http.Error(w, "avatar must be at most 512x512 pixels", http.StatusBadRequest)
	case strings.Contains(msg, "valid png or jpeg") || strings.Contains(msg, "unsupported avatar") || strings.Contains(msg, "no dimensions"):
		http.Error(w, "avatar must be a PNG or JPEG image", http.StatusBadRequest)
	default:
		http.Error(w, "failed to save avatar", http.StatusInternalServerError)
	}
}

// detectAvatarContentType fully decodes the image (rejecting SVG and other
// unsupported formats) and returns the normalized content type after enforcing
// the maximum dimensions.
func detectAvatarContentType(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("avatar must be a PNG or JPEG image")
	}
	if cfg.Width == 0 || cfg.Height == 0 {
		return "", errors.New("avatar image has no dimensions")
	}
	if cfg.Width > maxAvatarDimension || cfg.Height > maxAvatarDimension {
		return "", fmt.Errorf("avatar must be at most %dx%d pixels", maxAvatarDimension, maxAvatarDimension)
	}
	switch strings.ToLower(format) {
	case "png":
		return "image/png", nil
	case "jpeg":
		return "image/jpeg", nil
	default:
		return "", errors.New("avatar must be a PNG or JPEG image")
	}
}
