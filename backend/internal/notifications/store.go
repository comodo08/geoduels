package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	"geoduels/pkg/contracts"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore owns PostgreSQL access for the notifications feature: the user
// notification inbox and the delivery outbox claimed by workers.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) q() *db.Queries { return db.New(s.pool) }

func (s *PGStore) ListUserNotifications(userID string, limit int) ([]contracts.UserNotification, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	u, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q().ListUserNotifications(ctx, db.ListUserNotificationsParams{UserID: u, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.UserNotification, 0, len(rows))
	for _, row := range rows {
		out = append(out, contracts.UserNotification{
			ID: row.ID, Type: string(row.Type), Payload: json.RawMessage(row.PayloadJson), CreatedAt: row.CreatedAt.Time,
			ActorUserID: storekit.UUIDVal(row.ActorUserID), ActorDisplayName: strings.TrimSpace(row.ActorDisplayName),
		})
	}
	return out, nil
}

func (s *PGStore) MarkUserNotificationRead(userID string, notificationID int64) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || notificationID <= 0 {
		return errors.New("userID and notificationID required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	u, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	id, err := requireNotificationID(notificationID)
	if err != nil {
		return err
	}
	return s.q().MarkUserNotificationRead(ctx, db.MarkUserNotificationReadParams{ID: id, UserID: u})
}

func (s *PGStore) ListNotificationInbox(userID string, limit int, beforeID int64) ([]contracts.UserNotification, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	u, err := storekit.ProfileUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.q().ListNotificationInbox(ctx, db.ListNotificationInboxParams{UserID: u, BeforeID: beforeID, RowLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.UserNotification, 0, len(rows))
	for _, row := range rows {
		item := contracts.UserNotification{
			ID: row.ID, Type: string(row.Type), Category: string(row.Category), Payload: json.RawMessage(row.PayloadJson), CreatedAt: row.CreatedAt.Time,
			ActorUserID: storekit.UUIDVal(row.ActorUserID), ActorDisplayName: strings.TrimSpace(row.ActorDisplayName),
		}
		if row.ReadAt.Valid {
			value := row.ReadAt.Time
			item.ReadAt = &value
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *PGStore) MarkAllUserNotificationsRead(userID string) error {
	u, err := storekit.ProfileUUID(userID)
	if err != nil {
		return err
	}
	return s.q().MarkAllUserNotificationsRead(context.Background(), u)
}

func requireNotificationID(id int64) (int64, error) {
	if id <= 0 {
		return 0, errors.New("notification id required")
	}
	return id, nil
}
