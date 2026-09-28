package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "geoduels/pkg/persistence/sqlc/db"
)

func deletionUUID(s string) (pgtype.UUID, error) { var u pgtype.UUID; return u, u.Scan(s) }

// DeletionUser is the primitive ban-state read used before deleting an account.
func (s *PGStore) DeletionUser(ctx context.Context, userID string) (bool, string, error) {
	id, err := deletionUUID(userID)
	if err != nil {
		return false, "", err
	}
	u, err := s.q().GetDeletionUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", errors.New("user not found")
		}
		return false, "", err
	}
	banned, _ := u.IsBanned.(bool)
	return banned, u.BanReason, nil
}

func (s *PGStore) BanUserOAuthIdentities(ctx context.Context, userID, reason string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	return s.q().BanUserOAuthIdentities(ctx, db.BanUserOAuthIdentitiesParams{BannedUserID: id, Reason: reason})
}

func (s *PGStore) RevokeDeletionSessions(ctx context.Context, userID string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	return s.q().RevokeDeletionSessions(ctx, id)
}

func (s *PGStore) ListDeletionDiscordIdentities(ctx context.Context, userID string) ([]string, error) {
	id, err := deletionUUID(userID)
	if err != nil {
		return nil, err
	}
	return s.q().ListDeletionDiscordIdentities(ctx, db.ListDeletionDiscordIdentitiesParams{UserID: id, Provider: db.GdOauthProvider(IdentityProviderDiscord)})
}

func (s *PGStore) ArchiveDeletionIdentities(ctx context.Context, userID string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	return s.q().ArchiveDeletionIdentities(ctx, id)
}

func (s *PGStore) DeleteDeletionIdentities(ctx context.Context, userID string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	return s.q().DeleteDeletionIdentities(ctx, id)
}

func (s *PGStore) DeleteAccountRoles(ctx context.Context, userID string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	return s.q().DeleteAccountRoles(ctx, id)
}

func (s *PGStore) AnonymizeDeletedUser(ctx context.Context, userID string) error {
	id, err := deletionUUID(userID)
	if err != nil {
		return err
	}
	_, err = s.q().AnonymizeDeletedUser(ctx, id)
	return err
}

// DeleteOldGuestAccounts prunes stale guests; the service owns the TTL/batch
// bounds and returns only the count.
func (s *PGStore) DeleteOldGuestAccounts(ctx context.Context, ttl time.Duration, limit int) (int, error) {
	rows, err := s.q().DeleteOldGuestAccounts(ctx, db.DeleteOldGuestAccountsParams{TtlSeconds: ttl.Seconds(), AccountLimit: int32(limit)})
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}
