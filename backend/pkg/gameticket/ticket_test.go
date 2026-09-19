package gameticket

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"geoduels/pkg/contracts"
)

var testSecret = []byte("local-test-secret-do-not-use-in-prod")

func TestIssueAndValidateRoundTrip(t *testing.T) {
	token, err := Issue(testSecret, "user-1", "match-1", "node-a", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := Validate(testSecret, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.Subject != "user-1" || claims.MatchID != "match-1" || claims.Node != "node-a" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatalf("issued ticket must carry issued-at and expires-at: %+v", claims.RegisteredClaims)
	}
	ttl := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if ttl != time.Minute {
		t.Fatalf("ttl mismatch: %v", ttl)
	}
}

func TestIssueRejectsBadInputs(t *testing.T) {
	if _, err := Issue(nil, "u", "m", "n", time.Minute); err == nil {
		t.Fatal("expected error for missing secret")
	}
	if _, err := Issue(testSecret, "", "m", "n", time.Minute); err == nil {
		t.Fatal("expected error for missing user id")
	}
	if _, err := Issue(testSecret, "u", "", "n", time.Minute); err == nil {
		t.Fatal("expected error for missing match id")
	}
	if _, err := Issue(testSecret, "u", "m", "", time.Minute); err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestValidateWrongSecret(t *testing.T) {
	token, err := Issue(testSecret, "u", "m", "n", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := Validate([]byte("a-different-secret"), token); err == nil {
		t.Fatal("expected validation failure with wrong secret")
	}
}

func TestValidateTamperedPayload(t *testing.T) {
	token, err := Issue(testSecret, "attacker-should-not-pass", "m", "n", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected compact JWS with three parts, got %d", len(parts))
	}
	tampered := parts[0] + "." + parts[1] + "x" + "." + parts[2]
	if _, err := Validate(testSecret, tampered); err == nil {
		t.Fatal("expected validation failure for tampered payload")
	}
}

// mint builds an application-shaped ticket outside the issuer so policy tests
// can exercise claims combinations the issuer never produces.
func mint(t *testing.T, alg jwt.SigningMethod, mutate func(*contracts.GameplayTicketClaims)) string {
	t.Helper()
	claims := contracts.GameplayTicketClaims{
		Node:    "node-a",
		MatchID: "match-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	if mutate != nil {
		mutate(&claims)
	}
	token, err := jwt.NewWithClaims(alg, claims).SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestValidateExpiredTicket(t *testing.T) {
	token := mint(t, jwt.SigningMethodHS256, func(c *contracts.GameplayTicketClaims) {
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	})
	if _, err := Validate(testSecret, token); err == nil {
		t.Fatal("expected validation failure for expired ticket")
	}
}

func TestValidateMissingRequiredClaims(t *testing.T) {
	cases := map[string]func(*contracts.GameplayTicketClaims){
		"missing subject":  func(c *contracts.GameplayTicketClaims) { c.Subject = "" },
		"missing match id": func(c *contracts.GameplayTicketClaims) { c.MatchID = "" },
		"missing node":     func(c *contracts.GameplayTicketClaims) { c.Node = "" },
		"missing expiry":   func(c *contracts.GameplayTicketClaims) { c.ExpiresAt = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			token := mint(t, jwt.SigningMethodHS256, mutate)
			if _, err := Validate(testSecret, token); err == nil {
				t.Fatalf("expected validation failure for %s", name)
			}
		})
	}
}

func TestValidateDisallowedAlgorithms(t *testing.T) {
	// Tokens signed with an algorithm other than HS256 must be rejected before
	// any key material is used.
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, contracts.GameplayTicketClaims{
		Node:    "node-a",
		MatchID: "match-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	unsigned, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("mint unsigned token: %v", err)
	}
	if _, err := Validate(testSecret, unsigned); err == nil {
		t.Fatal("expected rejection of alg=none ticket")
	}

	rs256Token := mintWithRSA(t)
	if _, err := Validate(testSecret, rs256Token); err == nil {
		t.Fatal("expected rejection of RS256 ticket")
	}
}

func mintWithRSA(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	claims := contracts.GameplayTicketClaims{
		Node:    "node-a",
		MatchID: "match-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign rsa token: %v", err)
	}
	return token
}
