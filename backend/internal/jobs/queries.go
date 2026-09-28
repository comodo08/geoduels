package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

func listDiscordIdentities(ctx context.Context, tx pgx.Tx, userID string) ([]string, error) {
	u, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	return db.New(tx).ListDiscordIdentities(ctx, db.ListDiscordIdentitiesParams{UserID: u, Provider: db.GdOauthProvider("discord")})
}
