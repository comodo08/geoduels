package accounts

import (
	"strings"
)

// chooseProviderIdentityUser resolves which users row an OAuth identity attaches
// to. A users row without any linked sign-in identity is a guest and is upgraded
// in place; a row that already has an identity is reused as-is. The bool reports
// whether the returned row was an identity-less guest.
func chooseProviderIdentityUser(existingProviderUserID, existingEmailUserID string, existingEmailHasIdentity bool, linkUserID string, linkHasIdentity bool) (string, bool) {
	if existingProviderUserID != "" {
		return existingProviderUserID, false
	}
	if existingEmailUserID != "" {
		return existingEmailUserID, !existingEmailHasIdentity
	}
	if linkUserID != "" {
		return linkUserID, !linkHasIdentity
	}
	return "", false
}

func providerUsesAccountEmail(provider string) bool {
	return provider == IdentityProviderGoogle || provider == IdentityProviderDiscord
}

func isSyntheticOAuthEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	return strings.HasSuffix(email, "@oauth.invalid") || strings.HasSuffix(email, ".oauth.invalid")
}

func providerAccountEmail(provider, email string) string {
	email = strings.TrimSpace(email)
	if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
		return email
	}
	return ""
}
