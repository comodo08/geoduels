package accounts

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"geoduels/pkg/contentfilter"
	"geoduels/pkg/entityid"
)

var ErrNicknameTaken = errors.New("nickname already taken")

var ErrOAuthEmailConflict = errors.New("verified email is linked to multiple accounts")

// VerifiedEmailUser is one row of the verified-email account-discovery read.
type VerifiedEmailUser struct {
	UserID      string
	HasIdentity bool
}

// IdentityPresence reports whether a users row exists and already has a linked
// sign-in identity. Found is false when the row is missing.
type IdentityPresence struct {
	Found       bool
	HasIdentity bool
}

const (
	accountOpTimeout      = 4 * time.Second
	accountCleanupTimeout = 30 * time.Second
)

// Service owns account orchestration. Policy, identity merge rules, nickname
// allocation and transaction boundaries live here; persistence exposes
// the store handles persistence.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) op() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), accountOpTimeout)
}

func (s *Service) cleanupOp() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), accountCleanupTimeout)
}

// findUserByVerifiedEmail resolves the single account a verified email belongs
// to, rejecting the ambiguous case. Synthetic emails carry no account claim.
func (s *Service) findUserByVerifiedEmail(ctx context.Context, store Store, email string) (string, bool, error) {
	if email == "" || isSyntheticOAuthEmail(email) {
		return "", false, nil
	}
	rows, err := store.FindUsersByVerifiedEmail(ctx, email)
	if err != nil {
		return "", false, err
	}
	if len(rows) > 1 {
		return "", false, ErrOAuthEmailConflict
	}
	if len(rows) == 1 {
		return rows[0].UserID, rows[0].HasIdentity, nil
	}
	return "", false, nil
}

func (s *Service) UpsertGoogleIdentity(googleSub, email, googleName, avatarURL, linkUserID string) (Identity, error) {
	return s.UpsertProviderIdentity(IdentityProviderGoogle, googleSub, email, googleName, avatarURL, linkUserID)
}

// UpsertProviderIdentity owns validation, the OAuth email lock, the merge
// decision and the guest-to-registered upgrade inside one transaction.
func (s *Service) UpsertProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return Identity{}, errors.New("provider required")
	}
	if providerUserID == "" {
		return Identity{}, errors.New("provider subject required")
	}
	if email == "" {
		email = providerUserID + "@oauth.invalid"
	}
	if providerName == "" {
		providerName = providerUserID
	}
	ctx, cancel := s.op()
	defer cancel()
	var userID string
	err := s.store.WithinTx(ctx, func(store Store) error {
		seasonID, err := store.ActiveSeasonID(ctx)
		if err != nil {
			return err
		}
		providerIdentityBanned, _, err := store.ProviderIdentityBanned(ctx, provider, providerUserID)
		if err != nil {
			return err
		}
		if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
			if err := store.LockOAuthEmail(ctx, email); err != nil {
				return err
			}
		}
		existingProviderUserID, err := store.FindProviderIdentityUser(ctx, provider, providerUserID)
		if err != nil {
			return err
		}
		// A provider ban prevents account creation/evasion, but an identity that
		// is still attached to its banned account may authenticate into it.
		if providerIdentityBanned && existingProviderUserID == "" {
			return errors.New("provider identity banned")
		}
		var previousLinkedProviderUserID string
		if provider == IdentityProviderDiscord && linkUserID != "" {
			previousLinkedProviderUserID, err = store.FindProviderUserIdentity(ctx, provider, linkUserID)
			if err != nil {
				return err
			}
		}
		var existingEmailUserID string
		var existingEmailHasIdentity bool
		if providerUsesAccountEmail(provider) && existingProviderUserID == "" {
			existingEmailUserID, existingEmailHasIdentity, err = s.findUserByVerifiedEmail(ctx, store, email)
			if err != nil {
				return err
			}
		}
		var linkHasIdentity bool
		if existingProviderUserID == "" && existingEmailUserID == "" && linkUserID != "" {
			presence, err := store.UserIdentityPresence(ctx, linkUserID)
			if err != nil {
				return err
			}
			linkHasIdentity = presence.HasIdentity
		}
		userID, _ = chooseProviderIdentityUser(existingProviderUserID, existingEmailUserID, existingEmailHasIdentity, linkUserID, linkHasIdentity)
		if userID == "" {
			userID = entityid.New()
		}
		userEmail := providerAccountEmail(provider, email)
		if err := store.UpsertRegisteredUser(ctx, userID, userEmail, providerName, avatarURL); err != nil {
			return err
		}
		if existingProviderUserID != "" {
			if err := store.UpsertIdentityByProviderSubject(ctx, userID, provider, providerUserID, email, providerName, avatarURL); err != nil {
				return err
			}
		} else {
			if err := store.UpsertIdentityByUserProvider(ctx, userID, provider, providerUserID, email, providerName, avatarURL); err != nil {
				return err
			}
		}
		if err := store.RecordIdentityHistory(ctx, userID, provider, providerUserID, email, providerName); err != nil {
			return err
		}
		if err := store.EnsureAccountRank(ctx, userID, modeDuel, seasonID, initialMMR); err != nil {
			return err
		}
		if err := store.EnsureAccountStats(ctx, userID); err != nil {
			return err
		}
		if err := store.EnsureAccountRankedStats(ctx, userID, modeDuel, seasonID); err != nil {
			return err
		}
		if provider == IdentityProviderDiscord {
			if previousLinkedProviderUserID != "" && previousLinkedProviderUserID != providerUserID {
				if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionCleanupRoles, previousLinkedProviderUserID); err != nil {
					return err
				}
			}
			if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionSync, providerUserID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Identity{}, err
	}
	return s.store.GetIdentity(ctx, userID)
}

// LinkProviderIdentity owns the "already linked" checks and the account-linking
// decision inside one transaction.
func (s *Service) LinkProviderIdentity(provider, providerUserID, email, providerName, avatarURL, linkUserID string) (Identity, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	linkUserID = strings.TrimSpace(linkUserID)
	if provider == "" {
		return Identity{}, errors.New("provider required")
	}
	if providerUserID == "" {
		return Identity{}, errors.New("provider subject required")
	}
	if linkUserID == "" {
		return Identity{}, errors.New("link user required")
	}
	if email == "" {
		email = providerUserID + "@oauth.invalid"
	}
	if providerName == "" {
		providerName = providerUserID
	}
	if _, err := profileUUID(linkUserID); err != nil {
		return Identity{}, errors.New("link user not found")
	}
	ctx, cancel := s.op()
	defer cancel()
	err := s.store.WithinTx(ctx, func(store Store) error {
		banned, _, err := store.ProviderIdentityBanned(ctx, provider, providerUserID)
		if err != nil {
			return err
		}
		if banned {
			return errors.New("provider identity banned")
		}
		seasonID, err := store.ActiveSeasonID(ctx)
		if err != nil {
			return err
		}
		if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
			if err := store.LockOAuthEmail(ctx, email); err != nil {
				return err
			}
		}
		presence, err := store.UserIdentityPresence(ctx, linkUserID)
		if err != nil {
			return err
		}
		if !presence.Found {
			return errors.New("link user not found")
		}
		existingProviderUserID, err := store.FindProviderIdentityUser(ctx, provider, providerUserID)
		if err != nil {
			return err
		}
		if existingProviderUserID != "" && existingProviderUserID != linkUserID {
			return errors.New("provider identity already linked")
		}
		var previousLinkedProviderUserID string
		if provider == IdentityProviderDiscord {
			previousLinkedProviderUserID, err = store.FindProviderUserIdentity(ctx, provider, linkUserID)
			if err != nil {
				return err
			}
		}
		if providerUsesAccountEmail(provider) && email != "" && !isSyntheticOAuthEmail(email) {
			existingEmailUserID, _, err := s.findUserByVerifiedEmail(ctx, store, email)
			if err != nil && !errors.Is(err, ErrOAuthEmailConflict) {
				return err
			}
			if errors.Is(err, ErrOAuthEmailConflict) {
				return errors.New("provider identity already linked")
			}
			if existingEmailUserID != "" && existingEmailUserID != linkUserID {
				return errors.New("provider identity already linked")
			}
		}
		userEmail := providerAccountEmail(provider, email)
		if err := store.PromoteLinkedUser(ctx, linkUserID, userEmail, providerName, avatarURL); err != nil {
			return err
		}
		if err := store.UpsertIdentityByUserProvider(ctx, linkUserID, provider, providerUserID, email, providerName, avatarURL); err != nil {
			return err
		}
		if err := store.RecordIdentityHistory(ctx, linkUserID, provider, providerUserID, email, providerName); err != nil {
			return err
		}
		if err := store.EnsureAccountRank(ctx, linkUserID, modeDuel, seasonID, initialMMR); err != nil {
			return err
		}
		if err := store.EnsureAccountRankedStats(ctx, linkUserID, modeDuel, seasonID); err != nil {
			return err
		}
		if provider == IdentityProviderDiscord {
			if previousLinkedProviderUserID != "" && previousLinkedProviderUserID != providerUserID {
				if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionCleanupRoles, previousLinkedProviderUserID); err != nil {
					return err
				}
			}
			if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionSync, providerUserID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Identity{}, err
	}
	return s.store.GetIdentity(ctx, linkUserID)
}

func (s *Service) GoogleIdentityExists(googleSub string) (bool, error) {
	return s.ProviderIdentityExists(IdentityProviderGoogle, googleSub)
}

func (s *Service) ProviderIdentityExists(provider, providerUserID string) (bool, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	if provider == "" || providerUserID == "" {
		return false, errors.New("provider and subject required")
	}
	ctx, cancel := s.op()
	defer cancel()
	return s.store.ProviderIdentityExists(ctx, provider, providerUserID)
}

func (s *Service) IsProviderIdentityBanned(provider, providerUserID string) (bool, string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	if provider == "" || providerUserID == "" {
		return false, "", errors.New("provider and subject required")
	}
	ctx, cancel := s.op()
	defer cancel()
	return s.store.ProviderIdentityBanned(ctx, provider, providerUserID)
}

// UnlinkProviderIdentity owns the last-sign-in-method rule and the Discord
// role-cleanup enqueue inside one transaction.
func (s *Service) UnlinkProviderIdentity(userID, provider string) (Identity, error) {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(strings.ToLower(provider))
	if userID == "" || provider == "" {
		return Identity{}, errors.New("user and provider required")
	}
	ctx, cancel := s.op()
	defer cancel()
	err := s.store.WithinTx(ctx, func(store Store) error {
		providerCount, err := store.CountUserProviders(ctx, userID)
		if err != nil {
			return err
		}
		if providerCount <= 1 {
			return errors.New("cannot unlink the last sign-in method")
		}
		var unlinkedProviderUserID string
		if provider == IdentityProviderDiscord {
			unlinkedProviderUserID, err = store.FindProviderUserIdentity(ctx, provider, userID)
			if err != nil {
				return err
			}
		}
		affected, err := store.DeleteUserProvider(ctx, userID, provider)
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("provider is not linked")
		}
		if err := store.MarkIdentityHistoryDeleted(ctx, userID, provider); err != nil {
			return err
		}
		if provider == IdentityProviderGoogle {
			if err := store.ClearUserEmailWithoutGoogle(ctx, userID); err != nil {
				return err
			}
		}
		if provider == IdentityProviderDiscord {
			if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionCleanupRoles, unlinkedProviderUserID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Identity{}, err
	}
	return s.store.GetIdentity(ctx, userID)
}

// CreateGuestIdentity owns guest row/stat creation and identifier allocation.
func (s *Service) CreateGuestIdentity() (Identity, error) {
	userID := entityid.New()
	ctx, cancel := s.op()
	defer cancel()
	err := s.store.WithinTx(ctx, func(store Store) error {
		seasonID, err := store.ActiveSeasonID(ctx)
		if err != nil {
			return err
		}
		if err := store.UpsertUser(ctx, userID, "Guest"); err != nil {
			return err
		}
		if err := store.EnsureUserRank(ctx, userID, modeDuel, seasonID, initialMMR); err != nil {
			return err
		}
		if err := store.EnsureUserStats(ctx, userID); err != nil {
			return err
		}
		if err := store.EnsureRankedStats(ctx, userID, modeDuel, seasonID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Identity{}, err
	}
	return s.store.GetIdentity(ctx, userID)
}

func (s *Service) GetIdentity(sub string) (Identity, error) {
	if sub == "" {
		return Identity{}, errors.New("subject required")
	}
	ctx, cancel := s.op()
	defer cancel()
	return s.store.GetIdentity(ctx, sub)
}

func (s *Service) SetNickname(sub, displayName string) error {
	if sub == "" {
		return errors.New("subject required")
	}
	if displayName == "" {
		return errors.New("display name required")
	}
	ctx, cancel := s.op()
	defer cancel()
	return s.store.WithinTx(ctx, func(store Store) error {
		return store.SetNickname(ctx, sub, displayName)
	})
}

// SuggestNickname owns the candidate-generation policy, including the random
// suffix fallback, and delegates only the availability read.
func (s *Service) SuggestNickname(sub, displayName string) (string, error) {
	base := contentfilter.NicknameSuggestionBase(displayName)
	if _, err := contentfilter.ValidateNickname(base); err != nil {
		base = "Player"
	}
	ctx, cancel := s.op()
	defer cancel()
	available := func(candidate string) (bool, error) {
		taken, err := s.store.NicknameTaken(ctx, sub, candidate)
		return !taken, err
	}
	if ok, err := available(base); err != nil {
		return "", err
	} else if ok {
		return base, nil
	}
	prefix := base
	if len(prefix) > contentfilter.MaxNicknameLength-4 {
		prefix = prefix[:contentfilter.MaxNicknameLength-4]
	}
	for range 32 {
		value, err := rand.Int(rand.Reader, big.NewInt(9000))
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("%s%04d", prefix, value.Int64()+1000)
		if ok, err := available(candidate); err != nil {
			return "", err
		} else if ok {
			return candidate, nil
		}
	}
	return "", errors.New("nickname suggestion unavailable")
}

// DeleteAccount owns the ban cascade, session revocation, Discord cleanup
// fan-out and anonymization steps inside one transaction.
func (s *Service) DeleteAccount(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("userID required")
	}
	ctx, cancel := s.op()
	defer cancel()
	return s.store.WithinTx(ctx, func(store Store) error {
		banned, reason, err := store.DeletionUser(ctx, userID)
		if err != nil {
			return err
		}
		if banned {
			reason = strings.TrimSpace(reason)
			if reason == "" {
				reason = "account deleted while banned"
			}
			if err := store.BanUserOAuthIdentities(ctx, userID, reason); err != nil {
				return err
			}
		}
		if err := store.RevokeDeletionSessions(ctx, userID); err != nil {
			return err
		}
		identities, err := store.ListDeletionDiscordIdentities(ctx, userID)
		if err != nil {
			return err
		}
		for _, discordUserID := range identities {
			if err := store.EnqueueDiscordSync(ctx, DiscordSyncActionCleanupRoles, discordUserID); err != nil {
				return err
			}
		}
		if err := store.ArchiveDeletionIdentities(ctx, userID); err != nil {
			return err
		}
		if err := store.DeleteDeletionIdentities(ctx, userID); err != nil {
			return err
		}
		if err := store.DeleteAccountRoles(ctx, userID); err != nil {
			return err
		}
		if err := store.AnonymizeDeletedUser(ctx, userID); err != nil {
			return err
		}
		return nil
	})
}

// DeleteGuestAccountsOlderThan owns the TTL/batch bounds; the store only runs
// the prune primitive.
func (s *Service) DeleteGuestAccountsOlderThan(ttl time.Duration, limit int) (int, error) {
	if ttl <= 0 || limit <= 0 {
		return 0, nil
	}
	ctx, cancel := s.cleanupOp()
	defer cancel()
	var deleted int
	err := s.store.WithinTx(ctx, func(store Store) error {
		n, err := store.DeleteOldGuestAccounts(ctx, ttl, limit)
		if err != nil {
			return err
		}
		deleted = n
		return nil
	})
	return deleted, err
}
