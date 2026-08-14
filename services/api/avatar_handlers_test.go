package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"

	"geoduels/pkg/auth"
	"geoduels/pkg/persistence"
)

type avatarTestStore struct {
	persistence.Store
	identity         persistence.Identity
	storedUserID     string
	storedContentType string
	storedData       []byte
	storedURL        string
	resetCalled      bool
	serveContentType string
	serveData        []byte
	serveErr         error
}

func (s *avatarTestStore) GetIdentity(sub string) (persistence.Identity, error) {
	return s.identity, nil
}

func (s *avatarTestStore) SyncLoginBadges(userID string) error { return nil }

func (s *avatarTestStore) SuggestNickname(sub, displayName string) (string, error) {
	return "Player", nil
}

func (s *avatarTestStore) SetUserAvatar(userID, contentType string, data []byte, baseURL string) (string, error) {
	s.storedUserID = userID
	s.storedContentType = contentType
	s.storedData = data
	url := baseURL + "/v1/avatars/" + userID
	s.storedURL = url
	s.identity.AvatarURL = url
	return url, nil
}

func (s *avatarTestStore) ResetUserAvatar(userID string) error {
	s.resetCalled = true
	s.identity.AvatarURL = ""
	return nil
}

func (s *avatarTestStore) GetUserAvatar(userID string) (string, []byte, error) {
	if s.serveErr != nil {
		return "", nil, s.serveErr
	}
	return s.serveContentType, s.serveData, nil
}

func newAvatarTestAPI(store *avatarTestStore) *api {
	return &api{
		store:            store,
		appAuthSecret:    []byte("01234567890123456789012345678901"),
		accessTokenTTL:   15 * time.Minute,
		apiPublicBaseURL: "https://api.example.com",
	}
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func avatarAuthRequest(t *testing.T, a *api, method, path string, body *bytes.Buffer, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	token, err := auth.IssueAppAccessToken(a.appAuthSecret, "user-1", "session-1", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func multipartAvatar(t *testing.T, data []byte, filename string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("avatar", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestUpdateAvatarStoresAndReturnsPayload(t *testing.T) {
	store := &avatarTestStore{
		identity: persistence.Identity{Sub: "user-1", AccountType: "registered"},
	}
	a := newAvatarTestAPI(store)
	pngData := encodePNG(t, 64, 64)
	body, ct := multipartAvatar(t, pngData, "avatar.png")
	req := avatarAuthRequest(t, a, http.MethodPost, "/v1/me/avatar", body, ct)
	rec := httptest.NewRecorder()

	a.updateAvatar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.storedUserID != "user-1" {
		t.Fatalf("stored user id = %q", store.storedUserID)
	}
	if store.storedContentType != "image/png" {
		t.Fatalf("stored content type = %q", store.storedContentType)
	}
	if !bytes.Equal(store.storedData, pngData) {
		t.Fatal("stored avatar data does not match upload")
	}
	if store.storedURL != "https://api.example.com/v1/avatars/user-1" {
		t.Fatalf("stored url = %q", store.storedURL)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	user, _ := payload["user"].(map[string]any)
	if user == nil {
		t.Fatal("expected user in payload")
	}
	if user["avatar_url"] != store.storedURL {
		t.Fatalf("payload avatar_url = %v, want %q", user["avatar_url"], store.storedURL)
	}
}

func TestUpdateAvatarRejectsGuest(t *testing.T) {
	store := &avatarTestStore{
		identity: persistence.Identity{Sub: "guest-1", AccountType: "guest"},
	}
	a := newAvatarTestAPI(store)
	pngData := encodePNG(t, 32, 32)
	body, ct := multipartAvatar(t, pngData, "avatar.png")
	req := avatarAuthRequest(t, a, http.MethodPost, "/v1/me/avatar", body, ct)
	rec := httptest.NewRecorder()

	a.updateAvatar(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestUpdateAvatarRejectsOversizedImage(t *testing.T) {
	store := &avatarTestStore{
		identity: persistence.Identity{Sub: "user-1", AccountType: "registered"},
	}
	a := newAvatarTestAPI(store)
	body, ct := multipartAvatar(t, encodePNG(t, 600, 600), "avatar.png")
	req := avatarAuthRequest(t, a, http.MethodPost, "/v1/me/avatar", body, ct)
	rec := httptest.NewRecorder()

	a.updateAvatar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateAvatarRejectsSvg(t *testing.T) {
	store := &avatarTestStore{
		identity: persistence.Identity{Sub: "user-1", AccountType: "registered"},
	}
	a := newAvatarTestAPI(store)
	svg := []byte("<svg xmlns='http://www.w3.org/2000/svg'><rect width='10' height='10'/></svg>")
	body, ct := multipartAvatar(t, svg, "avatar.svg")
	req := avatarAuthRequest(t, a, http.MethodPost, "/v1/me/avatar", body, ct)
	rec := httptest.NewRecorder()

	a.updateAvatar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestResetAvatarClearsAndReturnsPayload(t *testing.T) {
	store := &avatarTestStore{
		identity: persistence.Identity{
			Sub:        "user-1",
			AccountType: "registered",
			AvatarURL:  "https://api.example.com/v1/avatars/user-1",
		},
	}
	a := newAvatarTestAPI(store)
	req := avatarAuthRequest(t, a, http.MethodDelete, "/v1/me/avatar", bytes.NewBuffer(nil), "")
	rec := httptest.NewRecorder()

	a.resetAvatar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !store.resetCalled {
		t.Fatal("expected ResetUserAvatar to be called")
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	user, _ := payload["user"].(map[string]any)
	if user["avatar_url"] == "https://api.example.com/v1/avatars/user-1" {
		t.Fatal("expected custom avatar_url to be cleared")
	}
}

func TestServeAvatarReturnsStoredContentType(t *testing.T) {
	store := &avatarTestStore{
		serveContentType: "image/png",
		serveData:        encodePNG(t, 8, 8),
	}
	a := newAvatarTestAPI(store)
	r := mux.NewRouter()
	r.HandleFunc("/v1/avatars/{userId}", a.serveAvatar).Methods(http.MethodGet)
	req := httptest.NewRequest(http.MethodGet, "/v1/avatars/user-1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Disposition") != "inline" {
		t.Fatalf("content disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	if rec.Header().Get("Cache-Control") == "" {
		t.Fatal("expected cache-control header")
	}
}

func TestServeAvatarNotFound(t *testing.T) {
	store := &avatarTestStore{serveErr: pgx.ErrNoRows}
	a := newAvatarTestAPI(store)
	r := mux.NewRouter()
	r.HandleFunc("/v1/avatars/{userId}", a.serveAvatar).Methods(http.MethodGet)
	req := httptest.NewRequest(http.MethodGet, "/v1/avatars/missing", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDetectAvatarContentTypeValidation(t *testing.T) {
	if ct, err := detectAvatarContentType(encodePNG(t, 10, 10)); err != nil || ct != "image/png" {
		t.Fatalf("png: ct=%q err=%v", ct, err)
	}
	if _, err := detectAvatarContentType([]byte("<svg></svg>")); err == nil {
		t.Fatal("svg should be rejected")
	}
	if _, err := detectAvatarContentType([]byte("not an image")); err == nil {
		t.Fatal("random bytes should be rejected")
	}
	if _, err := detectAvatarContentType(encodePNG(t, 600, 600)); err == nil {
		t.Fatal("oversized image should be rejected")
	}
}
