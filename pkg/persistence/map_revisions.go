package persistence

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"geoduels/pkg/entityid"
)

func (s *pgStore) ActivateMapRevision(mapKey, displayName string, dataset []byte) (MapRevisionSummary, error) {
	if strings.TrimSpace(mapKey) == "" {
		return MapRevisionSummary{}, errors.New("map key required")
	}
	rows, err := parseMapRows(dataset)
	if err != nil {
		return MapRevisionSummary{}, err
	}
	if len(rows) == 0 {
		return MapRevisionSummary{}, errors.New("no valid rows")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = mapKey
	}
	sum := sha1.Sum(dataset)
	contentHash := hex.EncodeToString(sum[:])
	revisionID := entityid.Derive("map-revision", mapKey+":"+contentHash)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MapRevisionSummary{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		insert into maps(map_key, display_name)
		values($1, $2)
		on conflict (map_key) do update set
			display_name = excluded.display_name
	`, mapKey, displayName); err != nil {
		return MapRevisionSummary{}, err
	}

	inserted := true
	var existing string
	var revisionStorageID int32
	err = tx.QueryRow(ctx, `select id,storage_id from map_revisions where map_key = $1 and content_hash = $2 limit 1`, mapKey, contentHash).Scan(&existing, &revisionStorageID)
	if err == nil {
		revisionID = existing
		inserted = false
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return MapRevisionSummary{}, err
	} else {
		if err := tx.QueryRow(ctx, `
			insert into map_revisions(id, map_key, content_hash, status, row_count)
			values($1, $2, $3, 'validated', 0)
			returning storage_id
		`, revisionID, mapKey, contentHash).Scan(&revisionStorageID); err != nil {
			return MapRevisionSummary{}, err
		}
	}

	if inserted {
		block := make([][]any, 0, len(rows))
		for _, r := range rows {
			block = append(block, []any{revisionStorageID, r.LatE7, r.LngE7, r.Country, r.PanoID, r.HeadingCDeg, r.PitchCDeg, r.RandKey})
		}
		if _, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{"locations"},
			[]string{"revision_storage_id", "lat_e7", "lng_e7", "country", "pano_id", "heading_cdeg", "pitch_cdeg", "rand_key_i"},
			pgx.CopyFromRows(block),
		); err != nil {
			return MapRevisionSummary{}, err
		}
	}

	if _, err := tx.Exec(ctx, `update map_revisions set row_count = $2, status = 'active' where id = $1`, revisionID, len(rows)); err != nil {
		return MapRevisionSummary{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into map_aliases(map_key, active_revision_id, updated_at)
		values($1, $2, now())
		on conflict (map_key) do update set
			rollback_revision_id = map_aliases.active_revision_id,
			active_revision_id = excluded.active_revision_id,
			updated_at = now()
	`, mapKey, revisionID); err != nil {
		return MapRevisionSummary{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return MapRevisionSummary{}, err
	}
	return MapRevisionSummary{
		MapKey:      mapKey,
		RevisionID:  revisionID,
		RowCount:    len(rows),
		Inserted:    inserted,
		DisplayName: displayName,
	}, nil
}
