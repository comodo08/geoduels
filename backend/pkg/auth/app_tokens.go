package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppClaims struct {
	Sub       string `json:"sub"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func IssueAppAccessToken(secret []byte, sub, sessionID string, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("missing app auth secret")
	}
	if sub == "" {
		return "", errors.New("missing subject")
	}
	if sessionID == "" {
		return "", errors.New("missing session id")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	now := time.Now()
	claims := AppClaims{
		Sub:       sub,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func ValidateAppAccessToken(secret []byte, tokenStr string) (AppClaims, error) {
	if tokenStr == "" {
		return AppClaims{}, errors.New("missing access token")
	}
	claims := AppClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected jwt signing method")
		}
		return secret, nil
	})
	if err != nil || !tok.Valid {
		return AppClaims{}, errors.New("invalid access token")
	}
	if claims.Sub == "" {
		claims.Sub = claims.Subject
	}
	if claims.Sub == "" {
		return AppClaims{}, errors.New("missing subject")
	}
	if claims.SessionID == "" {
		return AppClaims{}, errors.New("missing session id")
	}
	return claims, nil
}

func NewRefreshToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	hash := RefreshTokenHash(token)
	return token, hash, nil
}

func RefreshTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
