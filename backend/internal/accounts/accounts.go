package accounts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
	"geoduels/pkg/staff"
)

func accountNullableText(value any) pgtype.Text {
	var result pgtype.Text
	_ = result.Scan(value)
	return result
}

// GetIdentity is the identity read primitive. It returns the account view
// callers expect, including staff roles and linked providers.
func (s *PGStore) GetIdentity(ctx context.Context, sub string) (Identity, error) {
	if sub == "" {
		return Identity{}, errors.New("subject required")
	}
	u, err := profileUUID(sub)
	if err != nil {
		return Identity{}, err
	}
	r, err := s.q().GetIdentity(ctx, u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, errors.New("identity not found")
		}
		return Identity{}, err
	}
	out := Identity{}
	out.Sub = storekit.UUIDVal(r.UserID)
	out.Email = r.Email
	out.GoogleName = r.ProviderName
	out.AvatarURL = r.AvatarUrl
	out.NicknameRequired, _ = r.NeedsNickname.(bool)
	out.DisplayName = r.DisplayName
	out.IsGuest = !r.HasIdentity
	out.IsAdmin = r.IsAdmin
	out.IsModerator = r.IsModerator
	roles, err := s.q().GetStaffRoles(ctx, u)
	if err != nil {
		return Identity{}, err
	}
	out.Roles = staff.FromStrings(roles)
	out.IsBanned = r.IsBanned
	out.BanReason = r.BanReason
	out.ProviderName = out.GoogleName
	out.LinkedProviders, _ = s.userProviders(ctx, sub)
	out.AuthMigrationRequired = false
	out.RecoveryAvailable = false
	return out, nil
}

func (s *PGStore) userProviders(ctx context.Context, userID string) ([]string, error) {
	u, err := profileUUID(userID)
	if err != nil {
		return nil, err
	}
	providers, err := s.q().ListIdentityProviders(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(providers))
	for _, p := range providers {
		out = append(out, string(p))
	}
	return out, nil
}

func (s *PGStore) ProviderIdentityExists(ctx context.Context, provider, providerUserID string) (bool, error) {
	exists, err := s.q().ProviderIdentityExists(ctx, db.ProviderIdentityExistsParams{Provider: db.GdOauthProvider(provider), ProviderUserID: providerUserID})
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PGStore) ProviderIdentityBanned(ctx context.Context, provider, providerUserID string) (bool, string, error) {
	reason, err := s.q().ProviderIdentityBanned(ctx, db.ProviderIdentityBannedParams{Provider: db.GdOauthProvider(provider), ProviderUserID: providerUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, reason, nil
}

func (s *PGStore) FindProviderIdentityUser(ctx context.Context, provider, providerUserID string) (string, error) {
	u, err := s.q().FindProviderIdentityUser(ctx, db.FindProviderIdentityUserParams{Provider: db.GdOauthProvider(provider), ProviderUserID: providerUserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return storekit.UUIDVal(u), nil
}

func (s *PGStore) FindProviderUserIdentity(ctx context.Context, provider, userID string) (string, error) {
	u, err := profileUUID(userID)
	if err != nil {
		return "", err
	}
	id, err := s.q().FindProviderUserIdentity(ctx, db.FindProviderUserIdentityParams{UserID: u, Provider: db.GdOauthProvider(provider)})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *PGStore) UserIdentityPresence(ctx context.Context, userID string) (IdentityPresence, error) {
	u, err := profileUUID(userID)
	if err != nil {
		return IdentityPresence{}, err
	}
	hasIdentity, err := s.q().GetUserIdentityPresence(ctx, u)
	if errors.Is(err, pgx.ErrNoRows) {
		return IdentityPresence{}, nil
	}
	if err != nil {
		return IdentityPresence{}, err
	}
	return IdentityPresence{Found: true, HasIdentity: hasIdentity}, nil
}

func (s *PGStore) FindUsersByVerifiedEmail(ctx context.Context, email string) ([]VerifiedEmailUser, error) {
	rows, err := s.q().FindUserByVerifiedEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	out := make([]VerifiedEmailUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, VerifiedEmailUser{UserID: storekit.UUIDVal(row.ID), HasIdentity: row.HasIdentity})
	}
	return out, nil
}

func (s *PGStore) LockOAuthEmail(ctx context.Context, email string) error {
	return s.q().LockOAuthEmail(ctx, email)
}

func (s *PGStore) UpsertRegisteredUser(ctx context.Context, userID, email, displayName, avatarURL string) error {
	id, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().UpsertRegisteredUser(ctx, db.UpsertRegisteredUserParams{
		ID:          id,
		Email:       accountNullableText(storekit.Nullable(email)),
		DisplayName: displayName,
		AvatarUrl:   accountNullableText(storekit.Nullable(avatarURL)),
	})
}

func (s *PGStore) PromoteLinkedUser(ctx context.Context, userID, email, displayName, avatarURL string) error {
	id, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().PromoteLinkedUser(ctx, db.PromoteLinkedUserParams{
		ID:          id,
		Email:       accountNullableText(storekit.Nullable(email)),
		DisplayName: displayName,
		AvatarUrl:   accountNullableText(storekit.Nullable(avatarURL)),
	})
}

func (s *PGStore) UpsertIdentityByProviderSubject(ctx context.Context, userID, provider, providerUserID, email, providerName, avatarURL string) error {
	id, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().UpsertIdentityByProviderSubject(ctx, db.UpsertIdentityByProviderSubjectParams{
		UserID:         id,
		Provider:       db.GdOauthProvider(provider),
		ProviderUserID: providerUserID,
		Email:          accountNullableText(email),
		ProviderName:   accountNullableText(providerName),
		AvatarUrl:      accountNullableText(storekit.Nullable(avatarURL)),
	})
}

func (s *PGStore) UpsertIdentityByUserProvider(ctx context.Context, userID, provider, providerUserID, email, providerName, avatarURL string) error {
	id, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().UpsertIdentityByUserProvider(ctx, db.UpsertIdentityByUserProviderParams{
		UserID:         id,
		Provider:       db.GdOauthProvider(provider),
		ProviderUserID: providerUserID,
		Email:          accountNullableText(email),
		ProviderName:   accountNullableText(providerName),
		AvatarUrl:      accountNullableText(storekit.Nullable(avatarURL)),
	})
}

func (s *PGStore) RecordIdentityHistory(ctx context.Context, userID, provider, providerUserID, email, providerName string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().RecordUserIdentityHistory(ctx, db.RecordUserIdentityHistoryParams{
		UserID:         u,
		Provider:       db.GdOauthProvider(provider),
		ProviderUserID: providerUserID,
		Email:          pgtype.Text{String: email, Valid: true},
		ProviderName:   pgtype.Text{String: providerName, Valid: true},
	})
}

func (s *PGStore) EnsureAccountRank(ctx context.Context, userID, mode, seasonID string, mmr int) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureAccountRank(ctx, db.EnsureAccountRankParams{UserID: u, Mode: db.GdMatchMode(mode), SeasonID: seasonID, Mmr: int32(mmr)})
}

func (s *PGStore) EnsureAccountStats(ctx context.Context, userID string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureAccountStats(ctx, u)
}

func (s *PGStore) EnsureAccountRankedStats(ctx context.Context, userID, mode, seasonID string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureAccountRankedStats(ctx, db.EnsureAccountRankedStatsParams{UserID: u, Mode: db.GdMatchMode(mode), SeasonID: seasonID})
}

func (s *PGStore) UpsertUser(ctx context.Context, userID, displayName string) error {
	id, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().UpsertUser(ctx, db.UpsertUserParams{ID: id, DisplayName: displayName})
}

func (s *PGStore) EnsureUserRank(ctx context.Context, userID, mode, seasonID string, mmr int) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureUserRank(ctx, db.EnsureUserRankParams{UserID: u, Mode: db.GdMatchMode(mode), SeasonID: seasonID, Mmr: int32(mmr)})
}

func (s *PGStore) EnsureUserStats(ctx context.Context, userID string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureUserStats(ctx, u)
}

func (s *PGStore) EnsureRankedStats(ctx context.Context, userID, mode, seasonID string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().EnsureRankedStats(ctx, db.EnsureRankedStatsParams{UserID: u, Mode: db.GdMatchMode(mode), SeasonID: seasonID})
}

func (s *PGStore) SetNickname(ctx context.Context, sub, displayName string) error {
	u, err := profileUUID(sub)
	if err != nil {
		return err
	}
	tag, err := s.q().SetNickname(ctx, db.SetNicknameParams{ID: u, DisplayName: displayName})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_claimed_nickname_unique" {
			return ErrNicknameTaken
		}
		return err
	}
	if tag == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *PGStore) CountUserProviders(ctx context.Context, userID string) (int, error) {
	u, err := profileUUID(userID)
	if err != nil {
		return 0, err
	}
	count, err := s.q().CountUserProviders(ctx, u)
	return int(count), err
}

func (s *PGStore) DeleteUserProvider(ctx context.Context, userID, provider string) (int, error) {
	u, err := profileUUID(userID)
	if err != nil {
		return 0, err
	}
	affected, err := s.q().DeleteUserProvider(ctx, db.DeleteUserProviderParams{UserID: u, Provider: db.GdOauthProvider(provider)})
	return int(affected), err
}

func (s *PGStore) MarkIdentityHistoryDeleted(ctx context.Context, userID, provider string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().MarkIdentityHistoryDeleted(ctx, db.MarkIdentityHistoryDeletedParams{UserID: u, Provider: db.GdOauthProvider(provider)})
}

func (s *PGStore) ClearUserEmailWithoutGoogle(ctx context.Context, userID string) error {
	u, err := profileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().ClearUserEmailWithoutGoogle(ctx, u)
}

func (s *PGStore) NicknameTaken(ctx context.Context, sub, candidate string) (bool, error) {
	u, err := profileUUID(sub)
	if err != nil {
		return false, err
	}
	return s.q().NicknameTaken(ctx, db.NicknameTakenParams{ID: u, Lower: candidate})
}

// EnqueueDiscordSync queues a Discord role sync in the current transaction.
func (s *PGStore) EnqueueDiscordSync(ctx context.Context, action, discordUserID string) error {
	if s.jobs == nil {
		return nil
	}
	tx, err := s.requireTx()
	if err != nil {
		return err
	}
	return s.jobs.EnqueueDiscordSync(ctx, tx, action, discordUserID)
}

// ActiveSeasonID resolves the active ranked season for account bootstrap.
func (s *PGStore) ActiveSeasonID(ctx context.Context) (string, error) {
	return storekit.ActiveSeasonID(ctx, s.q())
}
