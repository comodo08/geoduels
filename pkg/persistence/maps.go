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
)

const (
	maxCustomMapsPerAccount      = 10
	maxCustomLocationsPerAccount = 250_000
	maxRevisionsPerMap           = 10
	maxInactiveRevisionsKept     = 5
	maxMapLocations              = 100_000
	minMapLocations              = 5
	maxUploadsPerHour            = 3
	maxUploadsPerDay             = 10
	plannedRoundCount            = 20
	mapTrendingWindowDays        = 7
)

type MapCatalog interface {
	ListMaps(userID string, opts contracts.MapListOptions) ([]contracts.CustomMap, error)
	GetMap(userID, mapID string) (contracts.MapDetails, bool, error)
	CreateCustomMap(userID, displayName, description, difficulty, thumbnailKey string, thumbnailVariant int, source io.Reader) (contracts.CustomMap, error)
	UploadCustomMapRevision(userID, mapID string, source io.Reader) (contracts.CustomMap, error)
	UpdateCustomMap(userID, mapID string, update contracts.CustomMapUpdate) (contracts.CustomMap, error)
	PublishCustomMap(userID, mapID string) (contracts.CustomMap, error)
	SetMapFavorite(userID, mapID string, favorite bool) (contracts.CustomMap, error)
	ListMapComments(userID, mapID string) ([]contracts.MapComment, error)
	CreateMapComment(userID, mapID string, input contracts.MapCommentCreate) (contracts.MapComment, error)
	DeleteMapComment(userID, mapID, commentID string, moderator bool) error
	ArchiveCustomMap(userID, mapID string) error
	PrepareMatchPlan(ctx context.Context, found *contracts.MatchFound) error
}

func (s *pgStore) ListMaps(userID string, opts contracts.MapListOptions) ([]contracts.CustomMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	scope := normalizeMapScope(opts.Scope)
	sortMode := normalizeMapSort(opts.Sort)
	searchPattern := mapSearchPattern(opts.Search)
	query := `
		select m.map_key, coalesce(m.owner_user_id, ''), coalesce(u.display_name, 'GeoDuels'), m.display_name, m.description, m.visibility, m.status,
		       m.difficulty, m.thumbnail_variant, coalesce(m.thumbnail_key, 'generic/variant-' || greatest(1, least(5, m.thumbnail_variant))::text), m.location_count, coalesce(m.active_revision_id, ''), m.owner_user_id is null,
		       coalesce(m.published_at, '0001-01-01'::timestamptz), m.play_count, m.favorite_count, m.comment_count, m.trending_score,
		       exists(select 1 from map_favorites mf where mf.map_id=m.map_key and mf.user_id=$1),
		       trim(both ':' from concat_ws(':', nullif(m.official_region_type,''), nullif(m.official_region_code,''))),
		       m.created_at, m.updated_at
		from maps m
		left join users u on u.id = m.owner_user_id
		where m.archived_at is null
	`
	args := []any{strings.TrimSpace(userID)}
	switch scope {
	case "official":
		query += ` and m.owner_user_id is null`
	case "community":
		query += ` and m.owner_user_id is not null and m.published_at is not null and m.status='ready'`
	case "favorites":
		query += ` and exists(select 1 from map_favorites mf where mf.map_id=m.map_key and mf.user_id=$1)`
	case "mine":
		query += ` and m.owner_user_id = $1`
	default:
		query += ` and (m.owner_user_id is null or m.owner_user_id = $1)`
	}
	if searchPattern != "" {
		args = append(args, searchPattern)
		searchArg := len(args)
		query += fmt.Sprintf(` and (
			m.display_name ilike $%[1]d escape '\'
			or m.description ilike $%[1]d escape '\'
			or m.map_key ilike $%[1]d escape '\'
			or coalesce(u.display_name, 'GeoDuels') ilike $%[1]d escape '\'
			or trim(both ':' from concat_ws(':', nullif(m.official_region_type,''), nullif(m.official_region_code,''))) ilike $%[1]d escape '\'
		)`, searchArg)
	}
	switch sortMode {
	case "popular":
		query += ` order by (m.play_count + m.favorite_count * 3) desc, m.published_at desc nulls last, m.updated_at desc`
	case "new":
		query += ` order by m.published_at desc nulls last, m.updated_at desc`
	default:
		query += ` order by m.trending_score desc, m.published_at desc nulls last, m.updated_at desc`
	}
	query += ` limit 72`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []contracts.CustomMap{}
	for rows.Next() {
		item, err := scanCustomMap(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *pgStore) GetMap(userID, mapID string) (contracts.MapDetails, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	row := s.pool.QueryRow(ctx, `
		select m.map_key, coalesce(m.owner_user_id, ''), coalesce(u.display_name, 'GeoDuels'), m.display_name, m.description, m.visibility, m.status,
		       m.difficulty, m.thumbnail_variant, coalesce(m.thumbnail_key, 'generic/variant-' || greatest(1, least(5, m.thumbnail_variant))::text), m.location_count, coalesce(m.active_revision_id, ''), m.owner_user_id is null,
		       coalesce(m.published_at, '0001-01-01'::timestamptz), m.play_count, m.favorite_count, m.comment_count, m.trending_score,
		       exists(select 1 from map_favorites mf where mf.map_id=m.map_key and mf.user_id=$2),
		       trim(both ':' from concat_ws(':', nullif(m.official_region_type,''), nullif(m.official_region_code,''))),
		       m.created_at, m.updated_at
		from maps m
		left join users u on u.id = m.owner_user_id
		where m.map_key = $1 and m.archived_at is null
		  and (m.owner_user_id is null or m.owner_user_id = $2 or m.published_at is not null or m.visibility = 'unlisted')
	`, strings.TrimSpace(mapID), strings.TrimSpace(userID))
	item, err := scanCustomMap(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return contracts.MapDetails{}, false, nil
	}
	if err != nil {
		return contracts.MapDetails{}, false, err
	}
	stats, err := s.mapCountryStats(ctx, item.ActiveRevisionID)
	if err != nil {
		return contracts.MapDetails{}, false, err
	}
	comments, err := s.listMapComments(ctx, strings.TrimSpace(userID), item.ID)
	if err != nil {
		return contracts.MapDetails{}, false, err
	}
	return contracts.MapDetails{Map: item, CountryStats: stats, Comments: comments}, true, nil
}
