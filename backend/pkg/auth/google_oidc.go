package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

const defaultGoogleIssuer = "https://accounts.google.com"

type GoogleVerifier struct {
	verifier *oidc.IDTokenVerifier
}

type IdentityTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Nonce         string `json:"nonce"`
}

func NewGoogleVerifier(ctx context.Context, clientID, issuer string) (*GoogleVerifier, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID is required")
	}
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = defaultGoogleIssuer
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return &GoogleVerifier{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (v *GoogleVerifier) ValidateIDToken(ctx context.Context, tokenStr, nonce string) (IdentityTokenClaims, error) {
	if v == nil || v.verifier == nil {
		return IdentityTokenClaims{}, errors.New("google verifier unavailable")
	}
	if strings.TrimSpace(tokenStr) == "" {
		return IdentityTokenClaims{}, errors.New("missing id token")
	}
	token, err := v.verifier.Verify(ctx, tokenStr)
	if err != nil {
		return IdentityTokenClaims{}, err
	}
	var claims IdentityTokenClaims
	if err := token.Claims(&claims); err != nil {
		return IdentityTokenClaims{}, err
	}
	if claims.Sub == "" {
		return IdentityTokenClaims{}, errors.New("missing subject")
	}
	if nonce != "" && claims.Nonce != nonce {
		return IdentityTokenClaims{}, errors.New("invalid nonce")
	}
	return claims, nil
}
