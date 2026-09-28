package accounts

import (
	"context"
	"time"
)

// Store persists accounts data. WithinTx supplies a store bound to one
// transaction; callback errors roll back all its writes.
type Store interface {
	WithinTx(ctx context.Context, fn func(Store) error) error
	GetIdentity(ctx context.Context, sub string) (Identity, error)
	FindProviderIdentityUser(ctx context.Context, provider, providerUserID string) (string, error)
	FindProviderUserIdentity(ctx context.Context, provider, userID string) (string, error)
	UserIdentityPresence(ctx context.Context, userID string) (IdentityPresence, error)
	FindUsersByVerifiedEmail(ctx context.Context, email string) ([]VerifiedEmailUser, error)
	LockOAuthEmail(ctx context.Context, email string) error
	UpsertRegisteredUser(ctx context.Context, userID, email, displayName, avatarURL string) error
	PromoteLinkedUser(ctx context.Context, userID, email, displayName, avatarURL string) error
	UpsertIdentityByProviderSubject(ctx context.Context, userID, provider, providerUserID, email, providerName, avatarURL string) error
	UpsertIdentityByUserProvider(ctx context.Context, userID, provider, providerUserID, email, providerName, avatarURL string) error
	RecordIdentityHistory(ctx context.Context, userID, provider, providerUserID, email, providerName string) error
	EnsureAccountRank(ctx context.Context, userID, mode, seasonID string, mmr int) error
	EnsureAccountStats(ctx context.Context, userID string) error
	EnsureAccountRankedStats(ctx context.Context, userID, mode, seasonID string) error
	UpsertUser(ctx context.Context, userID, displayName string) error
	EnsureUserRank(ctx context.Context, userID, mode, seasonID string, mmr int) error
	EnsureUserStats(ctx context.Context, userID string) error
	EnsureRankedStats(ctx context.Context, userID, mode, seasonID string) error
	ProviderIdentityExists(ctx context.Context, provider, providerUserID string) (bool, error)
	ProviderIdentityBanned(ctx context.Context, provider, providerUserID string) (bool, string, error)
	CountUserProviders(ctx context.Context, userID string) (int, error)
	DeleteUserProvider(ctx context.Context, userID, provider string) (int, error)
	MarkIdentityHistoryDeleted(ctx context.Context, userID, provider string) error
	ClearUserEmailWithoutGoogle(ctx context.Context, userID string) error
	SetNickname(ctx context.Context, sub, displayName string) error
	NicknameTaken(ctx context.Context, sub, candidate string) (bool, error)
	DeletionUser(ctx context.Context, userID string) (bool, string, error)
	BanUserOAuthIdentities(ctx context.Context, userID, reason string) error
	RevokeDeletionSessions(ctx context.Context, userID string) error
	ListDeletionDiscordIdentities(ctx context.Context, userID string) ([]string, error)
	ArchiveDeletionIdentities(ctx context.Context, userID string) error
	DeleteDeletionIdentities(ctx context.Context, userID string) error
	DeleteAccountRoles(ctx context.Context, userID string) error
	AnonymizeDeletedUser(ctx context.Context, userID string) error
	DeleteOldGuestAccounts(ctx context.Context, ttl time.Duration, limit int) (int, error)
	EnqueueDiscordSync(ctx context.Context, action, discordUserID string) error
	ActiveSeasonID(ctx context.Context) (string, error)
}
