package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"geoduels/internal/storekit"
	"geoduels/pkg/contracts"
	db "geoduels/pkg/persistence/sqlc/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type notificationTxAdapter struct{ queries *db.Queries }

func newNotificationTxAdapter(tx pgx.Tx) notificationTxAdapter {
	return notificationTxAdapter{queries: db.New(tx)}
}

func uuidValue(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(strings.TrimSpace(value)); err != nil {
		return id, err
	}
	return id, nil
}

func (a notificationTxAdapter) upsert(ctx context.Context, userID, typ, dedupe string, payload any, id *int64) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	u, err := uuidValue(userID)
	if err != nil {
		return err
	}
	row, err := a.queries.UpsertUserNotification(ctx, db.UpsertUserNotificationParams{UserID: u, Type: db.GdNotificationType(typ), DedupeKey: dedupe, PayloadJson: body})
	if err != nil {
		return err
	}
	*id = row
	return nil
}

func (a notificationTxAdapter) reporters(ctx context.Context, subject string) ([]string, error) {
	u, err := uuidValue(subject)
	if err != nil {
		return nil, err
	}
	rows, err := a.queries.ListReporters(ctx, u)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, storekit.UUIDVal(row))
	}
	return result, nil
}
func (a notificationTxAdapter) enqueue(ctx context.Context, typ, dedupe string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return a.queries.EnqueueNotificationOutbox(ctx, db.EnqueueNotificationOutboxParams{Type: db.GdNotificationOutboxType(typ), DedupeKey: dedupe, PayloadJson: body})
}

func notificationItem(row db.ClaimPendingNotificationRow) contracts.NotificationOutboxItem {
	return contracts.NotificationOutboxItem{ID: row.ID, Type: string(row.Type), PayloadJSON: json.RawMessage(row.PayloadJson), Attempts: int(row.Attempts)}
}
func upsertUserNotification(ctx context.Context, tx pgx.Tx, userID, notificationType, dedupeKey string, payload any, id *int64) error {
	return storekit.UpsertUserNotificationTx(ctx, tx, userID, notificationType, dedupeKey, payload, id)
}

func notifyAccountEnforcement(ctx context.Context, tx pgx.Tx, userID, action, reason string, moderationLogID int64, endsAt any) error {
	notificationType := "account_banned"
	if action == "unban" {
		notificationType = "account_unbanned"
	}
	var notificationID int64
	return upsertUserNotification(ctx, tx, userID, notificationType, fmt.Sprintf("%s:%d", notificationType, moderationLogID), map[string]any{
		"reason":          strings.TrimSpace(reason),
		"action":          action,
		"moderationLogId": moderationLogID,
		"endsAt":          endsAt,
	}, &notificationID)
}

func notifyReportersOfBan(ctx context.Context, tx pgx.Tx, subjectUserID, action string, logID int64) error {
	reporterIDs, err := newNotificationTxAdapter(tx).reporters(ctx, subjectUserID)
	if err != nil {
		return err
	}
	for _, reporterID := range reporterIDs {
		var notificationID int64
		if err := upsertUserNotification(ctx, tx, reporterID, "reported_player_banned", fmt.Sprintf("reported_player_banned:%d:%s", logID, reporterID), map[string]any{
			"action":          action,
			"moderationLogId": logID,
		}, &notificationID); err != nil {
			return err
		}
	}
	return nil
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(body)
}

func enqueueNotificationOutbox(ctx context.Context, tx pgx.Tx, notificationType, dedupeKey string, payload any) error {
	return newNotificationTxAdapter(tx).enqueue(ctx, notificationType, dedupeKey, payload)
}

func requireNotificationID(id int64) (int64, error) {
	if id <= 0 {
		return 0, fmt.Errorf("notification id required")
	}
	return id, nil
}
func timestamptz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
