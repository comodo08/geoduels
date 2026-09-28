package staff

import (
	"context"

	"geoduels/internal/badges"
	"geoduels/pkg/contracts"
)

func (a *PGStore) AwardBadge(ctx context.Context, userID, badgeID string) (bool, error) {
	tx, err := a.requireTx()
	if err != nil {
		return false, err
	}
	return badges.AwardBadgeTx(ctx, tx, userID, badgeID)
}

// RemoveBadge removes the GeoDuels team badge (the only staff-managed revocable badge).
func (a *PGStore) RemoveBadge(ctx context.Context, userID, _ string) error {
	tx, err := a.requireTx()
	if err != nil {
		return err
	}
	return badges.RemoveGeoDuelsTeamBadgeTx(ctx, tx, userID)
}

// BadgeCatalog lists badges an admin may grant.
func (a *PGStore) BadgeCatalog() []contracts.AdminBadgeDefinition {
	return a.badges.ListAdminGrantableBadges()
}

// GrantBadgeByNickname grants a badge by nickname, writing its own audit entry.
func (a *PGStore) GrantBadgeByNickname(ctx context.Context, nickname, badgeID, actorID string) (contracts.PlayerBadge, bool, error) {
	tx, err := a.requireTx()
	if err != nil {
		return contracts.PlayerBadge{}, false, err
	}
	return badges.GrantBadgeToUserTx(ctx, tx, nickname, badgeID, actorID)
}
