package persistence

import (
	"context"
	"strings"
	"time"

	"geoduels/pkg/contracts"
)

type customMapScanner interface {
	Scan(dest ...any) error
}

func scanCustomMap(row customMapScanner) (contracts.CustomMap, error) {
	var item contracts.CustomMap
	var publishedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.OwnerUserID,
		&item.AuthorName,
		&item.DisplayName,
		&item.Description,
		&item.Visibility,
		&item.Status,
		&item.Difficulty,
		&item.ThumbnailVariant,
		&item.ThumbnailKey,
		&item.LocationCount,
		&item.ActiveRevisionID,
		&item.System,
		&item.Official,
		&publishedAt,
		&item.PlayCount,
		&item.FavoriteCount,
		&item.CommentCount,
		&item.TrendingScore,
		&item.Favorited,
		&item.OfficialRegion,
		&item.RankedMoving,
		&item.RankedNMPZ,
		&item.DefaultMoving,
		&item.DefaultNMPZ,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return contracts.CustomMap{}, err
	}
	if !publishedAt.IsZero() && publishedAt.Year() > 1 {
		item.PublishedAt = &publishedAt
	}
	return item, nil
}

func (s *pgStore) mapCountryStats(ctx context.Context, revisionID string) ([]contracts.MapCountryStat, error) {
	if strings.TrimSpace(revisionID) == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		select country, location_count
		from map_revision_country_stats
		where map_revision_id=$1
		order by location_count desc, country asc
		limit 64
	`, strings.TrimSpace(revisionID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []contracts.MapCountryStat{}
	for rows.Next() {
		var item contracts.MapCountryStat
		if err := rows.Scan(&item.Country, &item.LocationCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
