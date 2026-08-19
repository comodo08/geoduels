package persistence

import (
	"context"
	"strings"
	"time"
)

// SearchPlayersForFriends finds registered players whose claimed nickname
// matches the query, excluding only the caller. Existing friends and pending
// requests are still returned so the UI can show their current relationship
// state (e.g. a "Friends" or "Requested" badge).
func (s *pgStore) SearchPlayersForFriends(query, excludeUserID string, limit int) ([]PlayerSearchResult, error) {
	query = strings.TrimSpace(query)
	excludeUserID = strings.TrimSpace(excludeUserID)
	if query == "" {
		return []PlayerSearchResult{}, nil
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > 25 {
		limit = 25
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select
			u.id::text,
			coalesce(nullif(u.display_name, ''), ui.provider_name, u.id::text),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(u.account_type = 'guest', false),
			coalesce(u.selected_badge_code, 0),
			coalesce(u.selected_badge_season_id, '')
		from users u
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = u.id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		where u.account_type = 'registered'
		  and u.id::text <> $3
		  and lower(u.display_name) like lower($1)
		order by
		  case when lower(u.display_name) = lower($2) then 0 else 1 end,
		  char_length(u.display_name),
		  lower(u.display_name)
		limit $4
	`, "%"+query+"%", query, excludeUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlayerSearchResult{}
	for rows.Next() {
		var row PlayerSearchResult
		var badgeCode int16
		var badgeSeason string
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.AvatarURL, &row.IsGuest, &badgeCode, &badgeSeason); err != nil {
			return nil, err
		}
		if row.IsGuest {
			continue
		}
		_, selected, err := s.profileBadges(ctx, row.UserID, badgeIDFromParts(badgeCode, badgeSeason))
		if err != nil {
			return nil, err
		}
		row.SelectedBadge = selected
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
