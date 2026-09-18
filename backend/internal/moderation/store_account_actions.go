package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

func (s *PGStore) SetPlayerBan(userID, reason, actorUserID string, banned bool) error {
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
	q := db.New(tx)
	uid, err := profileUUID(userID)
	if err != nil {
		return err
	}
	bannedAt := pgtype.Timestamptz{}
	banReason := pgtype.Text{}
	if banned {
		bannedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		if strings.TrimSpace(reason) != "" {
			banReason = pgtype.Text{String: strings.TrimSpace(reason), Valid: true}
		}
	}
	tag, err := q.BanUser(ctx, db.BanUserParams{UserID: uid, BannedAt: bannedAt, BanReason: banReason})
	if err != nil {
		return err
	}
	if tag == 0 {
		return errors.New("user not found")
	}
	if banned {
		if err := q.BanUserOAuthIdentities(ctx, db.BanUserOAuthIdentitiesParams{BannedUserID: uid, Reason: strings.TrimSpace(reason), CreatedBy: strings.TrimSpace(actorUserID)}); err != nil {
			return err
		}
	} else {
		if _, err := q.RevokeOAuthIdentityBans(ctx, uid); err != nil {
			return err
		}
	}
	action := "unban"
	if banned {
		action = "permanent_ban"
	}
	logID, err := q.InsertModerationLog(ctx, db.InsertModerationLogParams{
		SubjectUserID: uid,
		ActorUserID:   strings.TrimSpace(actorUserID),
		Action:        db.GdModerationLogAction(action),
		Reason:        strings.TrimSpace(reason),
	})
	if err != nil {
		return err
	}
	if err := notifyAccountEnforcement(ctx, tx, userID, action, reason, logID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PGStore) PreviewCommunityPardon(olderThan time.Duration) (CommunityPardonSummary, error) {
	if olderThan <= 0 {
		olderThan = 7 * 24 * time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cutoff := time.Now().Add(-olderThan)
	candidates, err := s.db.ListCommunityPardonCandidates(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return CommunityPardonSummary{}, err
	}
	return CommunityPardonSummary{Eligible: len(candidates), Cutoff: cutoff}, nil
}

func (s *PGStore) PardonBannedPlayers(olderThan time.Duration, actorUserID string) (CommunityPardonSummary, error) {
	if olderThan <= 0 {
		olderThan = 7 * 24 * time.Hour
	}
	actorUserID = strings.TrimSpace(actorUserID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CommunityPardonSummary{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)

	cutoff := time.Now().Add(-olderThan)
	userIDs, err := q.PardonBannedPlayers(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return CommunityPardonSummary{}, err
	}
	metadata, _ := json.Marshal(map[string]any{"release": "v2", "policy": "active ban older than 7 days"})
	for _, uid := range userIDs {
		userID := storekit.UUIDVal(uid)
		if _, err := q.RevokeOAuthIdentityBans(ctx, uid); err != nil {
			return CommunityPardonSummary{}, err
		}
		logID, err := q.InsertModerationLog(ctx, db.InsertModerationLogParams{
			SubjectUserID: uid,
			ActorUserID:   actorUserID,
			Action:        db.GdModerationLogActionUnban,
			Reason:        "v2 community pardon",
			Metadata:      metadata,
		})
		if err != nil {
			return CommunityPardonSummary{}, err
		}
		if err := notifyAccountEnforcement(ctx, tx, userID, "unban", "v2 community pardon", logID, nil); err != nil {
			return CommunityPardonSummary{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunityPardonSummary{}, err
	}
	return CommunityPardonSummary{Eligible: len(userIDs), Pardoned: len(userIDs), Cutoff: cutoff}, nil
}

func (s *PGStore) ClearReporterMute(userID string) error {
	userID = strings.TrimSpace(userID)
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
	q := db.New(tx)
	uid, err := profileUUID(userID)
	if err != nil {
		return err
	}
	tag, err := q.ClearReporterMute(ctx, uid)
	if err != nil {
		return err
	}
	if tag == 0 {
		return errors.New("user not found")
	}
	if _, err := q.InsertModerationLog(ctx, db.InsertModerationLogParams{SubjectUserID: uid, Action: db.GdModerationLogActionReportUnmute}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PGStore) SetPlayerMute(userID, kind, reason, actorUserID string, until time.Time, muted bool) error {
	userID = strings.TrimSpace(userID)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if userID == "" {
		return errors.New("user id required")
	}
	if kind != "chat" && kind != "report" {
		return errors.New("unsupported mute kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	uid, err := profileUUID(userID)
	if err != nil {
		return err
	}
	var tag int64
	if muted {
		if until.IsZero() {
			until = time.Now().Add(7 * 24 * time.Hour)
		}
		if kind == "chat" {
			tag, err = q.SetChatMute(ctx, db.SetChatMuteParams{UserID: uid, Reason: strings.TrimSpace(reason), ChatMuteExpiresAt: pgtype.Timestamptz{Time: until, Valid: true}})
		} else {
			tag, err = q.SetReportMute(ctx, db.SetReportMuteParams{UserID: uid, Reason: strings.TrimSpace(reason), ReportMuteExpiresAt: pgtype.Timestamptz{Time: until, Valid: true}})
		}
	} else if kind == "chat" {
		tag, err = q.ClearChatMute(ctx, uid)
	} else {
		tag, err = q.ClearReportMute(ctx, uid)
	}
	if err != nil {
		return err
	}
	if tag == 0 {
		return errors.New("user not found")
	}
	action := kind + "_unmute"
	expiresAt := pgtype.Timestamptz{}
	if muted {
		action = kind + "_mute"
		expiresAt = pgtype.Timestamptz{Time: until, Valid: true}
	}
	if _, err := q.InsertModerationLog(ctx, db.InsertModerationLogParams{
		SubjectUserID: uid,
		ActorUserID:   strings.TrimSpace(actorUserID),
		Action:        db.GdModerationLogAction(action),
		Reason:        strings.TrimSpace(reason),
		ExpiresAt:     expiresAt,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
