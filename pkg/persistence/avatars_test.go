package persistence

import (
	"testing"
	"time"
)

func TestBuildUserAvatarURL(t *testing.T) {
	got := buildUserAvatarURL("https://api.example.com/", "user-1", time.Unix(123, 0))
	if got != "https://api.example.com/v1/avatars/user-1?v=123" {
		t.Fatalf("url = %q", got)
	}
	got = buildUserAvatarURL("", "user-1", time.Unix(123, 0))
	if got != "/v1/avatars/user-1?v=123" {
		t.Fatalf("relative url = %q", got)
	}
}

func TestIsAllowedAvatarContentType(t *testing.T) {
	if !isAllowedAvatarContentType("image/png") {
		t.Fatal("png should be allowed")
	}
	if !isAllowedAvatarContentType("IMAGE/JPEG") {
		t.Fatal("jpeg should be allowed")
	}
	if isAllowedAvatarContentType("image/svg+xml") {
		t.Fatal("svg should be rejected")
	}
	if isAllowedAvatarContentType("") {
		t.Fatal("empty should be rejected")
	}
}
