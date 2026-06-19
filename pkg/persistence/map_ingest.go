package persistence

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"geoduels/pkg/contracts"
	"geoduels/pkg/entityid"
)

func (s *pgStore) CreateCustomMap(userID, displayName, description, difficulty, thumbnailKey string, thumbnailVariant int, source io.Reader) (contracts.CustomMap, error) {
	mapID := entityid.New()
	return s.ingestCustomMap(userID, mapID, displayName, description, "private", difficulty, thumbnailKey, thumbnailVariant, source, true)
}

func (s *pgStore) UploadCustomMapRevision(userID, mapID string, source io.Reader) (contracts.CustomMap, error) {
	return s.ingestCustomMap(userID, strings.TrimSpace(mapID), "", "", "", "", "", 0, source, false)
}

func (s *pgStore) ingestCustomMap(userID, mapID, displayName, description, visibility, difficulty, thumbnailKey string, thumbnailVariant int, source io.Reader, create bool) (contracts.CustomMap, error) {
	userID, mapID = strings.TrimSpace(userID), strings.TrimSpace(mapID)
	if userID == "" || mapID == "" {
		return contracts.CustomMap{}, errors.New("user and map required")
	}
	if create {
		displayName = strings.TrimSpace(displayName)
		if displayName == "" || len(displayName) > 80 {
			return contracts.CustomMap{}, errors.New("map name must be 1 to 80 characters")
		}
		if len(description) > 500 {
			return contracts.CustomMap{}, errors.New("description must be at most 500 characters")
		}
		difficulty = normalizeMapDifficulty(difficulty)
		thumbnailVariant = normalizeThumbnailVariant(thumbnailVariant)
		visibility = normalizeMapVisibility(visibility)
		thumbnailKey = normalizeThumbnailKey(thumbnailKey, thumbnailVariant)
	}
	quota, err := s.GetMapUploadQuota(userID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	parsed, digest, rejected, err := decodeMapRows(source, quota.MaxMapLocations)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if len(parsed) < minMapLocations {
		return contracts.CustomMap{}, fmt.Errorf("map requires at least %d valid locations", minMapLocations)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext($1))`, "map-upload:"+userID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := enforceMapUploadQuota(ctx, tx, userID, mapID, len(parsed), create); err != nil {
		return contracts.CustomMap{}, err
	}
	if create {
		_, err = tx.Exec(ctx, `
			insert into maps(map_key, owner_user_id, display_name, description, visibility, status, difficulty, thumbnail_variant, thumbnail_key, location_count, created_at, updated_at)
			values($1, $2, $3, $4, $5, 'processing', $6, $7, $8, 0, now(), now())
		`, mapID, userID, displayName, strings.TrimSpace(description), visibility, difficulty, thumbnailVariant, thumbnailKey)
	} else {
		var owner string
		if err = tx.QueryRow(ctx, `select coalesce(owner_user_id::text, '') from maps where map_key=$1 and archived_at is null for update`, mapID).Scan(&owner); err == nil && owner != userID {
			err = errors.New("map is not owned by this account")
		}
	}
	if err != nil {
		return contracts.CustomMap{}, err
	}

	revisionID := entityid.Derive("map-revision", mapID+":"+digest)
	var revisionStorageID int32
	var existing string
	err = tx.QueryRow(ctx, `select id,storage_id from map_revisions where map_key=$1 and content_hash=$2`, mapID, digest).Scan(&existing, &revisionStorageID)
	if err == nil {
		revisionID = existing
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return contracts.CustomMap{}, err
	} else {
		if err := tx.QueryRow(ctx, `insert into map_revisions(id, map_key, content_hash, status, row_count, rejected_count) values($1,$2,$3,'processing',0,$4) returning storage_id`, revisionID, mapID, digest, rejected).Scan(&revisionStorageID); err != nil {
			return contracts.CustomMap{}, err
		}
		block := make([][]any, 0, len(parsed))
		for _, row := range parsed {
			block = append(block, []any{revisionStorageID, row.LatE7, row.LngE7, row.Country, row.PanoID, row.HeadingCDeg, row.PitchCDeg, row.RandKey})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"locations"}, []string{"revision_storage_id", "lat_e7", "lng_e7", "country", "pano_id", "heading_cdeg", "pitch_cdeg", "rand_key_i"}, pgx.CopyFromRows(block)); err != nil {
			return contracts.CustomMap{}, err
		}
	}
	if _, err := tx.Exec(ctx, `update map_revisions set status='active', row_count=$2, rejected_count=$3, activated_at=now() where id=$1`, revisionID, len(parsed), rejected); err != nil {
		return contracts.CustomMap{}, err
	}
	if _, err := tx.Exec(ctx, `delete from map_revision_country_stats where map_revision_id=$1`, revisionID); err != nil {
		return contracts.CustomMap{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into map_revision_country_stats(map_revision_id, country, location_count)
		select $1, coalesce(nullif(country,''), 'Unknown'), count(*)::int
		from locations
		where revision_storage_id=$2
		group by coalesce(nullif(country,''), 'Unknown')
	`, revisionID, revisionStorageID); err != nil {
		return contracts.CustomMap{}, err
	}
	if _, err := tx.Exec(ctx, `update maps set active_revision_id=$2, status='ready', location_count=$3, updated_at=now() where map_key=$1`, mapID, revisionID, len(parsed)); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := pruneMapRevisions(ctx, tx, mapID, revisionID); err != nil {
		return contracts.CustomMap{}, err
	}
	if _, err := tx.Exec(ctx, `insert into map_upload_events(user_id,map_id,location_count) values($1,$2,$3)`, userID, mapID, len(parsed)); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return contracts.CustomMap{}, err
	}
	details, ok, err := s.GetMap(userID, mapID)
	if err != nil || !ok {
		return contracts.CustomMap{}, err
	}
	return details.Map, nil
}

func (s *pgStore) UpdateCustomMap(userID, mapID string, update contracts.CustomMapUpdate) (contracts.CustomMap, error) {
	name := strings.TrimSpace(update.DisplayName)
	if name == "" || len(name) > 80 || len(update.Description) > 500 {
		return contracts.CustomMap{}, errors.New("invalid map details")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `
		update maps
		set display_name=$3, description=$4, visibility=$5, difficulty=$6, thumbnail_variant=$7, thumbnail_key=$8, updated_at=now()
		where map_key=$1 and owner_user_id=$2 and archived_at is null
	`, strings.TrimSpace(mapID), strings.TrimSpace(userID), name, strings.TrimSpace(update.Description), normalizeMapVisibility(update.Visibility), normalizeMapDifficulty(update.Difficulty), normalizeThumbnailVariant(update.ThumbnailVariant), normalizeThumbnailKey(update.ThumbnailKey, update.ThumbnailVariant))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if tag.RowsAffected() == 0 {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	details, _, err := s.GetMap(userID, mapID)
	return details.Map, err
}

func (s *pgStore) PublishCustomMap(userID, mapID string) (contracts.CustomMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tag, err := s.pool.Exec(ctx, `
		update maps
		set visibility='public', published_at=coalesce(published_at, now()), updated_at=now()
		where map_key=$1 and owner_user_id=$2 and archived_at is null and status='ready'
	`, strings.TrimSpace(mapID), strings.TrimSpace(userID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if tag.RowsAffected() == 0 {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	details, _, err := s.GetMap(userID, mapID)
	return details.Map, err
}

func (s *pgStore) SetMapFavorite(userID, mapID string, favorite bool) (contracts.CustomMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	defer tx.Rollback(ctx)
	var visible bool
	var ownerUserID string
	if err := tx.QueryRow(ctx, `
		select
			exists(select 1 from maps where map_key=$1 and archived_at is null and (published_at is not null or owner_user_id is null or owner_user_id=$2)),
			coalesce((select owner_user_id::text from maps where map_key=$1 and archived_at is null), '')
	`, strings.TrimSpace(mapID), strings.TrimSpace(userID)).Scan(&visible, &ownerUserID); err != nil {
		return contracts.CustomMap{}, err
	}
	if !visible {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	changed := false
	if favorite {
		tag, err := tx.Exec(ctx, `insert into map_favorites(map_id,user_id) values($1,$2) on conflict do nothing`, strings.TrimSpace(mapID), strings.TrimSpace(userID))
		if err != nil {
			return contracts.CustomMap{}, err
		}
		if tag.RowsAffected() > 0 {
			changed = true
			if err := incrementMapFavoriteStats(ctx, tx, strings.TrimSpace(mapID), strings.TrimSpace(userID)); err != nil {
				return contracts.CustomMap{}, err
			}
		}
	} else {
		tag, err := tx.Exec(ctx, `delete from map_favorites where map_id=$1 and user_id=$2`, strings.TrimSpace(mapID), strings.TrimSpace(userID))
		if err != nil {
			return contracts.CustomMap{}, err
		}
		if tag.RowsAffected() > 0 {
			changed = true
			if _, err := tx.Exec(ctx, `update maps set favorite_count=greatest(favorite_count-1,0), updated_at=now() where map_key=$1`, strings.TrimSpace(mapID)); err != nil {
				return contracts.CustomMap{}, err
			}
			if err := refreshMapTrendingScore(ctx, tx, strings.TrimSpace(mapID)); err != nil {
				return contracts.CustomMap{}, err
			}
		}
	}
	if changed && ownerUserID != "" {
		if _, err := refreshMapCreatorTrust(ctx, tx, ownerUserID); err != nil {
			return contracts.CustomMap{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return contracts.CustomMap{}, err
	}
	details, ok, err := s.GetMap(userID, mapID)
	if err != nil || !ok {
		return contracts.CustomMap{}, err
	}
	return details.Map, nil
}

func (s *pgStore) ArchiveCustomMap(userID, mapID string, allowAnyOwner bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		delete from maps m
		where m.map_key=$1
		  and (m.owner_user_id=$2 or ($3 and m.owner_user_id is not null))
		  and m.archived_at is null
		  and not exists(select 1 from match_round_plans p where p.map_id=m.map_key)
		  and not exists(select 1 from parties p where p.map_id=m.map_key)
	`, strings.TrimSpace(mapID), strings.TrimSpace(userID), allowAnyOwner)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		tag, err = tx.Exec(ctx, `
			update maps
			set status='archived', archived_at=now(), updated_at=now()
			where map_key=$1
			  and (owner_user_id=$2 or ($3 and owner_user_id is not null))
			  and archived_at is null
		`, strings.TrimSpace(mapID), strings.TrimSpace(userID), allowAnyOwner)
	}
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
