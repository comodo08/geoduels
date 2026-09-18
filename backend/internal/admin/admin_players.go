package admin

import (
	"context"
	"errors"
	"geoduels/internal/badges"
	"geoduels/internal/storekit"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/pkg/contracts"
	db "geoduels/pkg/persistence/sqlc/db"
)

func (s *PGStore) ListUserRoles() ([]UserRoleGrant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.db.ListUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := []UserRoleGrant{}
	for _, row := range rows {
		var item UserRoleGrant
		item.UserID = row.ID.String()
		item.DisplayName, _ = row.DisplayName.(string)
		item.Email, item.Role, item.GrantedBy, item.Reason = row.Email, row.Role, "", row.LastReason
		item.GrantedBy = storekit.UUIDVal(row.ActorUserID)
		if row.GrantedAt.Valid {
			item.GrantedAt = row.GrantedAt.Time
		}
		out = append(out, item)
	}
	return out, nil
}

func normalizeAdminRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "admin", "moderator":
		return role, nil
	default:
		return "", errors.New("unsupported role")
	}
}

func (s *PGStore) GrantUserRole(userID, role, grantedBy, reason string) error {
	userID = strings.TrimSpace(userID)
	role, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	uid, err := profileUUID(userID)
	if err != nil {
		return err
	}
	q := s.db.WithTx(tx)
	tag, err := q.GrantUserRole(ctx, db.GrantUserRoleParams{UserID: uid, Role: role})
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	if err := q.GrantRoleLog(ctx, db.GrantRoleLogParams{SubjectUserID: uid, ActorUserID: strings.TrimSpace(grantedBy), Reason: strings.TrimSpace(reason), Role: role}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PGStore) RevokeUserRole(userID, role, revokedBy, reason string) error {
	userID = strings.TrimSpace(userID)
	role, err := normalizeAdminRole(role)
	if err != nil {
		return err
	}
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.db.WithTx(tx)
	uid, err := profileUUID(userID)
	if err != nil {
		return err
	}
	tag, err := q.RevokeUserRole(ctx, db.RevokeUserRoleParams{UserID: uid, Role: role})
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	hasTeamRoleValue, err := q.HasTeamRole(ctx, uid)
	if err != nil {
		return err
	}
	hasTeamRole := hasTeamRoleValue.Bool
	if !hasTeamRole {
		if err := badges.RemoveGeoDuelsTeamBadgeTx(ctx, tx, userID); err != nil {
			return err
		}
	}
	if err := q.RevokeRoleLog(ctx, db.RevokeRoleLogParams{SubjectUserID: uid, ActorUserID: strings.TrimSpace(revokedBy), Reason: strings.TrimSpace(reason), Role: role}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PGStore) SearchPlayers(query string, limit int) ([]AdminPlayerSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	pattern := "%"
	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		pattern = "%" + strings.ToLower(trimmed) + "%"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.SearchAdminPlayers(ctx, db.SearchAdminPlayersParams{
		Mode:       db.GdMatchMode(modeDuel),
		SeasonID:   seasonID,
		DefaultMmr: int32(initialMMR),
		Search:     pattern,
		RowLimit:   int32(limit),
		CreatorID:  pgtype.UUID{},
	})
	if err != nil {
		return nil, err
	}
	result := make([]AdminPlayerSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, adminPlayerSummaryFromRow(row))
	}
	if err := s.populateAdminPlayerIdentities(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func adminPlayerSummaryFromRow(row db.SearchAdminPlayersRow) AdminPlayerSummary {
	var item AdminPlayerSummary
	item.UserID = storekit.UUIDVal(row.UserID)
	item.Email = row.Email
	if row.DisplayName.Valid {
		item.DisplayName = row.DisplayName.String
	}
	item.AvatarURL = row.AvatarUrl
	item.MMR = int(row.Mmr)
	item.GamesPlayed = int(row.GamesPlayed)
	item.Wins = int(row.Wins)
	item.RankedGamesPlayed = int(row.RankedGamesPlayed)
	item.IsGuest, _ = row.IsGuest.(bool)
	item.IsAdmin = row.IsAdmin
	item.IsModerator = row.IsModerator
	item.IsBanned, _ = row.IsBanned.(bool)
	item.BanReason = row.BanReason
	item.BannedAt = row.BannedAt.Time
	item.BanExpiresAt = row.BanExpiresAt.Time
	item.ChatMutedAt = row.ChatMutedAt.Time
	item.ChatMuteReason = row.ChatMuteReason
	item.ChatMutedUntil = row.ChatMuteExpiresAt.Time
	item.ReportMutedAt = row.ReportMutedAt.Time
	item.ReportMuteReason = row.ReportMuteReason
	item.ReportMutedUntil = row.ReportMuteExpiresAt.Time
	item.LastIPAddress = row.LastIpAddress
	return item
}

func (s *PGStore) GetPlayerSummary(ctx context.Context, userID string) (AdminPlayerSummary, error) {
	return s.getAdminPlayerSummary(ctx, userID)
}

func (s *PGStore) GetPlayerStats(ctx context.Context, userID string) (AdminPlayerStats, error) {
	return s.adminPlayerStats(ctx, userID)
}

func ApplyPlayerStats(player *AdminPlayerSummary, stats AdminPlayerStats) {
	applyAdminPlayerStats(player, stats)
}

func (s *PGStore) getAdminPlayerSummary(ctx context.Context, userID string) (AdminPlayerSummary, error) {
	seasonID, err := storekit.ActiveSeasonID(ctx, s.pool)
	if err != nil {
		return AdminPlayerSummary{}, err
	}
	uid, err := profileUUID(userID)
	if err != nil {
		return AdminPlayerSummary{}, err
	}
	rows, err := s.db.SearchAdminPlayers(ctx, db.SearchAdminPlayersParams{
		Mode:       db.GdMatchMode(modeDuel),
		SeasonID:   seasonID,
		DefaultMmr: int32(initialMMR),
		Search:     "%",
		RowLimit:   1,
		CreatorID:  uid,
	})
	if err != nil {
		return AdminPlayerSummary{}, err
	}
	if len(rows) == 0 {
		return AdminPlayerSummary{}, pgx.ErrNoRows
	}
	item := adminPlayerSummaryFromRow(rows[0])
	items := []AdminPlayerSummary{item}
	if err := s.populateAdminPlayerIdentities(ctx, items); err != nil {
		return AdminPlayerSummary{}, err
	}
	return items[0], nil
}

func (s *PGStore) GetAdminPlayerDetail(userID string) (AdminPlayerDetail, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AdminPlayerDetail{}, errors.New("userID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	player, err := s.getAdminPlayerSummary(ctx, userID)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	stats, err := s.adminPlayerStats(ctx, userID)
	if err != nil {
		return AdminPlayerDetail{}, err
	}
	applyAdminPlayerStats(&player, stats)
	return AdminPlayerDetail{Player: player}, nil
}

func applyAdminPlayerStats(player *AdminPlayerSummary, stats AdminPlayerStats) {
	player.TrackedMatches = stats.TotalMatches
	player.RankedMatches = stats.RankedMatches
	player.DuelMatches = stats.DuelMatches
	player.SingleplayerRuns = stats.SingleplayerRuns
	player.Losses = stats.Losses
	if stats.Wins > player.Wins {
		player.Wins = stats.Wins
	}
}

func (s *PGStore) adminPlayerStats(ctx context.Context, userID string) (AdminPlayerStats, error) {
	var stats AdminPlayerStats
	u, err := profileUUID(userID)
	if err != nil {
		return stats, err
	}
	row, err := s.db.AdminPlayerStats(ctx, db.AdminPlayerStatsParams{WinnerUserID: u, Mode: db.GdMatchMode(modeDuel)})
	if err == nil {
		stats.TotalMatches, stats.RankedMatches, stats.DuelMatches, stats.SingleplayerRuns, stats.Wins, stats.Losses = int(row.TotalMatches), int(row.RankedMatches), int(row.DuelMatches), int(row.SingleplayerRuns), int(row.Wins), int(row.Losses)
	}
	return stats, err
}

func (s *PGStore) populateAdminPlayerIdentities(ctx context.Context, players []AdminPlayerSummary) error {
	if len(players) == 0 {
		return nil
	}
	userIDs := make([]pgtype.UUID, 0, len(players))
	byUserID := make(map[string]int, len(players))
	for i := range players {
		uid := chatUUID(players[i].UserID)
		userIDs = append(userIDs, uid)
		byUserID[uid.String()] = i
	}
	rows, err := s.db.AdminPlayerIdentities(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var identity contracts.AdminUserIdentity
		identity.Provider = string(row.Provider)
		identity.ProviderUserID = row.ProviderUserID
		identity.Email = row.Email
		identity.ProviderName = row.ProviderName
		identity.LastSeenAt = row.LastSeenAt.Time
		if row.DeletedAt.Valid {
			identity.DeletedAt = row.DeletedAt.Time
		}
		if idx, ok := byUserID[row.UserID.String()]; ok {
			players[idx].Identities = append(players[idx].Identities, identity)
		}
	}
	return nil
}
