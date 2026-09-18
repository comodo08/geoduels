package maps

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"geoduels/internal/storekit"
	"geoduels/pkg/contracts"
	"geoduels/pkg/entityid"
	db "geoduels/pkg/persistence/sqlc/db"
)

func (s *PGStore) CreateCustomMap(userID, displayName, description, visibility, difficulty, thumbnailKey string, thumbnailVariant int, source io.Reader) (contracts.CustomMap, error) {
	mapID := entityid.New()
	return s.ingestCustomMap(userID, mapID, displayName, description, visibility, difficulty, thumbnailKey, thumbnailVariant, source, true)
}

func (s *PGStore) ImportOfficialMap(adminUserID string, input OfficialMapImportInput, source io.Reader) (contracts.CustomMap, error) {
	mapKey := strings.TrimSpace(input.MapKey)
	displayName := strings.TrimSpace(input.DisplayName)
	if mapKey == "" {
		return contracts.CustomMap{}, errors.New("map key required")
	}
	if displayName == "" || len(displayName) > maxMapDisplayNameRunes {
		return contracts.CustomMap{}, errors.New("map name must be 1 to 80 characters")
	}
	if len(input.Description) > maxMapDescriptionRunes {
		return contracts.CustomMap{}, errors.New("description must be at most 500 characters")
	}
	visibility := normalizeMapVisibility(input.Visibility)
	difficulty := normalizeMapDifficulty(input.Difficulty)
	thumbnailVariant := normalizeThumbnailVariant(input.ThumbnailVariant)
	thumbnailKey := normalizeThumbnailKey(input.ThumbnailKey, thumbnailVariant)
	regionType := strings.ToLower(strings.TrimSpace(input.OfficialRegionType))
	regionCode := strings.ToUpper(strings.TrimSpace(input.OfficialRegionCode))
	if regionType == "" && regionCode != "" {
		regionType = "country"
	}
	if regionType != "" && regionType != "country" && regionType != "continent" {
		return contracts.CustomMap{}, errors.New("unsupported official region type")
	}
	if len(regionCode) > maxOfficialRegionCodeRunes {
		return contracts.CustomMap{}, errors.New("official region code must be at most 32 characters")
	}

	parsed, digest, rejected, err := decodeMapRows(source, absoluteMaxMapLocations)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if len(parsed) < minMapLocations {
		return contracts.CustomMap{}, fmt.Errorf("map requires at least %d valid locations", minMapLocations)
	}
	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return contracts.CustomMap{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	defer tx.Rollback(ctx)

	mapID := entityid.Derive("map", mapKey)
	mapUUID, err := storekit.ProfileUUID(mapID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	q := db.New(tx)
	if err := q.UpsertOfficialMap(ctx, db.UpsertOfficialMapParams{
		MapID: mapUUID, DisplayName: displayName, Description: strings.TrimSpace(input.Description),
		Visibility: db.GdMapVisibility(visibility), Difficulty: db.GdMapDifficulty(difficulty), ThumbnailVariant: int32(thumbnailVariant),
		ThumbnailKey: thumbnailKey, ContentHash: digestBytes, RejectedLocationCount: int32(rejected),
		OfficialBy: strings.TrimSpace(adminUserID), OfficialRegionType: regionType, OfficialRegionCode: regionCode,
	}); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.UpsertAlias(ctx, db.UpsertAliasParams{Alias: mapKey, MapID: mapUUID}); err != nil {
		return contracts.CustomMap{}, err
	}
	storageID, err := q.StorageID(ctx, mapUUID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.DeleteLocations(ctx, storageID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := insertIngestLocations(ctx, q, storageID, parsed); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.DeleteCountryStats(ctx, mapUUID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.InsertCountryStats(ctx, db.InsertCountryStatsParams{MapID: mapUUID, MapStorageID: storageID}); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.MarkMapReady(ctx, db.MarkMapReadyParams{MapID: mapUUID, LocationCount: int32(len(parsed)), ContentHash: digestBytes, RejectedLocationCount: int32(rejected)}); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.UpsertAlias(ctx, db.UpsertAliasParams{Alias: mapKey, MapID: mapUUID}); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return contracts.CustomMap{}, err
	}
	details, ok, err := s.GetMap(adminUserID, mapID)
	if err != nil || !ok {
		return contracts.CustomMap{}, err
	}
	return details.Map, nil
}

func (s *PGStore) ReplaceCustomMapLocations(userID, mapID string, source io.Reader) (contracts.CustomMap, error) {
	return s.ingestCustomMap(userID, strings.TrimSpace(mapID), "", "", "", "", "", 0, source, false)
}

func (s *PGStore) ingestCustomMap(userID, mapID, displayName, description, visibility, difficulty, thumbnailKey string, thumbnailVariant int, source io.Reader, create bool) (contracts.CustomMap, error) {
	userID, mapID = strings.TrimSpace(userID), strings.TrimSpace(mapID)
	if userID == "" || mapID == "" {
		return contracts.CustomMap{}, errors.New("user and map required")
	}
	if create {
		displayName = strings.TrimSpace(displayName)
		if displayName == "" || len(displayName) > maxMapDisplayNameRunes {
			return contracts.CustomMap{}, errors.New("map name must be 1 to 80 characters")
		}
		if len(description) > maxMapDescriptionRunes {
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
	q := db.New(tx)
	trustQ := db.New(tx)
	if err := trustQ.LockMapUpload(ctx, "map-upload:"+userID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := enforceMapUploadQuota(ctx, tx, userID, mapID, len(parsed), create); err != nil {
		return contracts.CustomMap{}, err
	}
	userUUID, err := storekit.ProfileUUID(userID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if create {
		mapUUID, parseErr := storekit.ProfileUUID(mapID)
		if parseErr != nil {
			return contracts.CustomMap{}, parseErr
		}
		err = q.CreateMap(ctx, db.CreateMapParams{MapID: mapUUID, OwnerUserID: userUUID, DisplayName: displayName, Description: strings.TrimSpace(description), Visibility: db.GdMapVisibility(visibility), Difficulty: db.GdMapDifficulty(difficulty), ThumbnailVariant: int32(thumbnailVariant), ThumbnailKey: thumbnailKey})
	} else {
		var canonicalID string
		if canonicalID, _, err = resolveMapIdentity(ctx, tx, mapID); err == nil {
			mapID = canonicalID
		}
		if err == nil {
			mapUUID, parseErr := storekit.ProfileUUID(mapID)
			if parseErr != nil {
				return contracts.CustomMap{}, parseErr
			}
			var ownerUUID pgtype.UUID
			ownerUUID, err = q.LockMapOwner(ctx, mapUUID)
			owner := storekit.UUIDVal(ownerUUID)
			if err == nil && owner != userID {
				err = errors.New("map is not owned by this account")
			}
		}
	}
	if err != nil {
		return contracts.CustomMap{}, err
	}
	mapUUID, err := storekit.ProfileUUID(mapID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	storageID, err := q.StorageID(ctx, mapUUID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.DeleteLocations(ctx, storageID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := insertIngestLocations(ctx, q, storageID, parsed); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.DeleteCountryStats(ctx, mapUUID); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.InsertCountryStats(ctx, db.InsertCountryStatsParams{MapID: mapUUID, MapStorageID: storageID}); err != nil {
		return contracts.CustomMap{}, err
	}
	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.MarkMapReady(ctx, db.MarkMapReadyParams{MapID: mapUUID, LocationCount: int32(len(parsed)), ContentHash: digestBytes, RejectedLocationCount: int32(rejected)}); err != nil {
		return contracts.CustomMap{}, err
	}
	if err := q.InsertUploadEvent(ctx, db.InsertUploadEventParams{UserID: userUUID, MapID: mapUUID, LocationCount: int32(len(parsed))}); err != nil {
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

func (s *PGStore) UpdateCustomMap(userID, mapID string, update contracts.CustomMapUpdate) (contracts.CustomMap, error) {
	name := strings.TrimSpace(update.DisplayName)
	if name == "" || len(name) > maxMapDisplayNameRunes || len(update.Description) > maxMapDescriptionRunes {
		return contracts.CustomMap{}, errors.New("invalid map details")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	canonicalID, _, err := resolveMapIdentity(ctx, s.pool, mapID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	mapID = canonicalID
	mapUUID, err := storekit.ProfileUUID(strings.TrimSpace(mapID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	userUUID, err := storekit.ProfileUUID(strings.TrimSpace(userID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	rows, err := s.q().UpdateMapDetails(ctx, db.UpdateMapDetailsParams{DisplayName: name, Description: strings.TrimSpace(update.Description), Visibility: db.GdMapVisibility(normalizeMapVisibility(update.Visibility)), Difficulty: db.GdMapDifficulty(normalizeMapDifficulty(update.Difficulty)), ThumbnailVariant: int32(normalizeThumbnailVariant(update.ThumbnailVariant)), ThumbnailKey: normalizeThumbnailKey(update.ThumbnailKey, update.ThumbnailVariant), MapID: mapUUID, OwnerUserID: userUUID})
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if rows == 0 {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	details, _, err := s.GetMap(userID, mapID)
	return details.Map, err
}

func (s *PGStore) PublishCustomMap(userID, mapID string) (contracts.CustomMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	canonicalID, _, err := resolveMapIdentity(ctx, s.pool, mapID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	mapID = canonicalID
	mapUUID, err := storekit.ProfileUUID(strings.TrimSpace(mapID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	userUUID, err := storekit.ProfileUUID(strings.TrimSpace(userID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	rows, err := s.q().PublishMap(ctx, db.PublishMapParams{ID: mapUUID, OwnerUserID: userUUID})
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if rows == 0 {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	details, _, err := s.GetMap(userID, mapID)
	return details.Map, err
}

func (s *PGStore) SetMapFavorite(userID, mapID string, favorite bool) (contracts.CustomMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	defer tx.Rollback(ctx)
	canonicalID, _, err := resolveMapIdentity(ctx, tx, mapID)
	if err != nil {
		return contracts.CustomMap{}, err
	}
	mapID = canonicalID
	mapUUID, err := storekit.ProfileUUID(strings.TrimSpace(mapID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	userUUID, err := storekit.ProfileUUID(strings.TrimSpace(userID))
	if err != nil {
		return contracts.CustomMap{}, err
	}
	q := db.New(tx)
	visibility, err := q.FavoriteVisibility(ctx, db.FavoriteVisibilityParams{ID: mapUUID, OwnerUserID: userUUID})
	if err != nil {
		return contracts.CustomMap{}, err
	}
	if !visibility.Visible {
		return contracts.CustomMap{}, pgx.ErrNoRows
	}
	changed := false
	if favorite {
		rows, err := q.AddFavorite(ctx, db.AddFavoriteParams{MapID: mapUUID, UserID: userUUID})
		if err != nil {
			return contracts.CustomMap{}, err
		}
		if rows > 0 {
			changed = true
			if err := incrementMapFavoriteStats(ctx, tx, strings.TrimSpace(mapID), strings.TrimSpace(userID)); err != nil {
				return contracts.CustomMap{}, err
			}
		}
	} else {
		rows, err := q.RemoveFavorite(ctx, db.RemoveFavoriteParams{MapID: mapUUID, UserID: userUUID})
		if err != nil {
			return contracts.CustomMap{}, err
		}
		if rows > 0 {
			changed = true
			if err := q.DecrementFavoriteCount(ctx, mapUUID); err != nil {
				return contracts.CustomMap{}, err
			}
			if err := refreshMapTrendingScore(ctx, tx, strings.TrimSpace(mapID)); err != nil {
				return contracts.CustomMap{}, err
			}
		}
	}
	if changed && visibility.OwnerUserID.Valid {
		if _, err := refreshMapCreatorTrust(ctx, tx, storekit.UUIDVal(visibility.OwnerUserID)); err != nil {
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

func (s *PGStore) ArchiveCustomMap(userID, mapID string, allowAnyMap bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	canonicalID, _, err := resolveMapIdentity(ctx, tx, mapID)
	if err != nil {
		return err
	}
	mapID = canonicalID
	mapUUID, err := storekit.ProfileUUID(strings.TrimSpace(mapID))
	if err != nil {
		return err
	}
	userUUID, err := storekit.ProfileUUID(strings.TrimSpace(userID))
	if err != nil {
		return err
	}
	q := db.New(tx)
	params := db.DeleteArchivableMapParams{MapID: mapUUID, UserID: userUUID, AllowAny: allowAnyMap}
	rows, err := q.DeleteArchivableMap(ctx, params)
	if err != nil {
		return err
	}
	if rows == 0 {
		rows, err = q.ArchiveMap(ctx, db.ArchiveMapParams{MapID: mapUUID, UserID: userUUID, AllowAny: allowAnyMap})
	}
	if err == nil && rows == 0 {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertIngestLocations(ctx context.Context, q *db.Queries, storageID int32, rows []mapRow) error {
	arg := db.InsertLocationsParams{MapStorageID: storageID}
	for _, row := range rows {
		heading, pitch, pano := int16(0), int16(0), ""
		if row.HeadingCDeg != nil {
			heading = *row.HeadingCDeg
		}
		if row.PitchCDeg != nil {
			pitch = *row.PitchCDeg
		}
		if row.PanoID != nil {
			pano = *row.PanoID
		}
		arg.LatE7 = append(arg.LatE7, row.LatE7)
		arg.LngE7 = append(arg.LngE7, row.LngE7)
		arg.Country = append(arg.Country, row.Country)
		arg.PanoID = append(arg.PanoID, pano)
		arg.HeadingCdeg = append(arg.HeadingCdeg, heading)
		arg.PitchCdeg = append(arg.PitchCdeg, pitch)
		arg.RandKeyI = append(arg.RandKeyI, row.RandKey)
	}
	return q.InsertLocations(ctx, arg)
}
