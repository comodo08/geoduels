package accounts

import "time"

type Store interface {
	UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error)
	UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (Identity, error)
	ProviderIdentityExists(provider, providerUserID string) (bool, error)
	GoogleIdentityExists(googleSub string) (bool, error)
	IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error)
	UnlinkProviderIdentity(userID, provider string) (Identity, error)
	CreateGuestIdentity() (Identity, error)
	GetIdentity(sub string) (Identity, error)
	SetNickname(sub, displayName string) error
	SuggestNickname(sub, displayName string) (string, error)
	DeleteAccount(userID string) error
	DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error)
	SetUserAdmin(userID string, isAdmin bool) error
	SetUserModerator(userID string, isModerator bool) error
}

type Service struct{ Store }

func NewService(store Store) *Service { return &Service{Store: store} }
