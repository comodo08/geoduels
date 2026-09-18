package preferences

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"geoduels/internal/storekit"
	db "geoduels/pkg/persistence/sqlc/db"
)

// PGStore owns PostgreSQL access for the preferences feature.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) q() *db.Queries { return db.New(s.pool) }

func (s *PGStore) GetUserPreferences(ctx context.Context, userID string) (UserPreferences, error) {
	var result UserPreferences
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return result, err
	}
	row, err := s.q().GetUserPreferences(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		result.SchemaVersion = SupportedSchemaVersion
		result.Preferences = json.RawMessage(`{}`)
		return result, nil
	}
	if err == nil {
		result.SchemaVersion = int(row.SchemaVersion)
		result.Preferences = json.RawMessage(row.PreferencesJson)
		result.Revision = row.Revision
	}
	return result, err
}

func (s *PGStore) UpdateUserPreferences(ctx context.Context, userID string, schemaVersion int, preferences json.RawMessage, expectedRevision int64) (UserPreferences, error) {
	var result UserPreferences
	id, err := storekit.ProfileUUID(userID)
	if err != nil {
		return result, err
	}
	row, err := s.q().UpsertUserPreferences(ctx, db.UpsertUserPreferencesParams{UserID: id, SchemaVersion: int32(schemaVersion), PreferencesJson: preferences, ExpectedRevision: expectedRevision})
	if errors.Is(err, pgx.ErrNoRows) {
		return UserPreferences{}, ErrRevisionConflict
	}
	if err == nil {
		result.SchemaVersion = int(row.SchemaVersion)
		result.Preferences = json.RawMessage(row.PreferencesJson)
		result.Revision = row.Revision
	}
	return result, err
}
