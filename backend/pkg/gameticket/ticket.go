package gameticket

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"geoduels/pkg/contracts"
)

func Issue(secret []byte, userID, matchID, node string, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("missing gameplay ticket secret")
	}
	if userID == "" || matchID == "" || node == "" {
		return "", errors.New("userID, matchID, and node are required")
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	now := time.Now()
	claims := contracts.GameplayTicketClaims{
		Node:    node,
		MatchID: matchID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func Validate(secret []byte, tokenStr string) (contracts.GameplayTicketClaims, error) {
	if tokenStr == "" {
		return contracts.GameplayTicketClaims{}, errors.New("missing gameplay ticket")
	}
	claims := contracts.GameplayTicketClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected jwt signing method")
		}
		return secret, nil
	})
	if err != nil || !tok.Valid {
		return contracts.GameplayTicketClaims{}, errors.New("invalid gameplay ticket")
	}
	if claims.Subject == "" || claims.MatchID == "" || claims.Node == "" {
		return contracts.GameplayTicketClaims{}, errors.New("invalid gameplay ticket claims")
	}
	return claims, nil
}
