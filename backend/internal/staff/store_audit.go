package staff

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

func (a *PGStore) RecordAudit(ctx context.Context, entry AuditEntry) (int64, error) {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}
	expiresAt := pgtype.Timestamptz{}
	if entry.ExpiresAt != nil {
		expiresAt = storekit.Timestamptz(*entry.ExpiresAt)
	}
	return a.q().InsertModerationLog(ctx, db.InsertModerationLogParams{
		SubjectUserID: storekit.MustUUID(entry.SubjectID),
		ActorUserID:   entry.ActorID,
		Action:        db.GdModerationLogAction(entry.Action),
		Reason:        entry.Reason,
		ExpiresAt:     expiresAt,
		Metadata:      metadata,
	})
}
