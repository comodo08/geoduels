package main

import (
	"testing"

	"geoduels/pkg/auth"
)

func TestGoogleAccountEmailRequiresVerifiedClaim(t *testing.T) {
	tests := []struct {
		name   string
		claims auth.IdentityTokenClaims
		want   string
	}{
		{
			name:   "verified",
			claims: auth.IdentityTokenClaims{Sub: "google-sub", Email: " person@example.com ", EmailVerified: true},
			want:   "person@example.com",
		},
		{
			name:   "unverified",
			claims: auth.IdentityTokenClaims{Sub: "google-sub", Email: "person@example.com"},
			want:   "google-sub@oidc.invalid",
		},
		{
			name:   "missing",
			claims: auth.IdentityTokenClaims{Sub: "google-sub", EmailVerified: true},
			want:   "google-sub@oidc.invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := googleAccountEmail(tt.claims); got != tt.want {
				t.Fatalf("googleAccountEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}
