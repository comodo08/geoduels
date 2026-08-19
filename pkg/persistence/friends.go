package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"geoduels/pkg/contracts"
)

type FriendRow struct {
	UserID       string                 `json:"userId"`
	DisplayName  string                 `json:"displayName"`
	AvatarURL    string                 `json:"avatarUrl,omitempty"`
	IsGuest      bool                   `json:"isGuest"`
	SelectedBadge *contracts.PlayerBadge `json:"selectedBadge,omitempty"`
}

type FriendsRepository interface {
	CreateFriendshipRequest(requesterID, addresseeID string) error
	ListFriends(userID string) ([]FriendRow, error)
	ListFriendRequests(userID string) (incoming []FriendRow, outgoing []FriendRow, err error)
	AcceptFriendship(userID, requesterID string) error
	DeclineFriendship(userID, requesterID string) error
	RemoveFriend(userID, friendID string) error
	AreFriends(userID, otherID string) (bool, error)
}

func (s *pgStore) CreateFriendshipRequest(requesterID, addresseeID string) error {
	requesterID = strings.TrimSpace(requesterID)
	addresseeID = strings.TrimSpace(addresseeID)
	if requesterID == "" || addresseeID == "" {
		return errors.New("requester and addressee are required")
	}
	if requesterID == addresseeID {
		return errors.New("cannot friend yourself")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	friends, err := s.AreFriends(requesterID, addresseeID)
	if err != nil {
		return err
	}
	if friends {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		insert into friendships(user_id, friend_id, status)
		values($1, $2, 'pending')
		on conflict (user_id, friend_id) do nothing
	`, requesterID, addresseeID)
	return err
}

func (s *pgStore) ListFriends(userID string) ([]FriendRow, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
		select
			friend_user_id,
			coalesce(nullif(u.display_name, ''), ui.provider_name, u.id::text),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(u.account_type = 'guest', false),
			coalesce(u.selected_badge_code, 0),
			coalesce(u.selected_badge_season_id, '')
		from (
			select distinct case when f.user_id = $1 then f.friend_id else f.user_id end as friend_user_id
			from friendships f
			where (f.user_id = $1 or f.friend_id = $1) and f.status = 'accepted'
		) friends
		join users u on u.id::text = friends.friend_user_id
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = u.id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		order by lower(display_name)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FriendRow{}
	for rows.Next() {
		var row FriendRow
		var badgeCode int16
		var badgeSeason string
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.AvatarURL, &row.IsGuest, &badgeCode, &badgeSeason); err != nil {
			return nil, err
		}
		badges, selected, err := s.profileBadges(ctx, row.UserID, badgeIDFromParts(badgeCode, badgeSeason))
		if err != nil {
			return nil, err
		}
		_ = badges
		row.SelectedBadge = selected
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *pgStore) ListFriendRequests(userID string) ([]FriendRow, []FriendRow, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	incoming, err := s.loadPendingFriends(ctx, "user_id", "friend_id = $1", userID)
	if err != nil {
		return nil, nil, err
	}
	outgoing, err := s.loadPendingFriends(ctx, "friend_id", "user_id = $1", userID)
	if err != nil {
		return nil, nil, err
	}
	return incoming, outgoing, nil
}

func (s *pgStore) loadPendingFriends(ctx context.Context, otherColumn, where string, userID string) ([]FriendRow, error) {
	query := `
		select
			friend_user_id,
			coalesce(nullif(u.display_name, ''), ui.provider_name, u.id::text),
			coalesce(u.avatar_url, ui.avatar_url, ''),
			coalesce(u.account_type = 'guest', false),
			coalesce(u.selected_badge_code, 0),
			coalesce(u.selected_badge_season_id, '')
		from (
			select distinct ` + otherColumn + ` as friend_user_id
			from friendships f
			where ` + where + ` and f.status = 'pending'
		) pending
		join users u on u.id::text = friend_user_id
		left join lateral (
			select provider_name, avatar_url
			from user_identities
			where user_id = u.id and provider = 'google'
			order by created_at asc
			limit 1
		) ui on true
		order by lower(display_name)
	`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FriendRow{}
	for rows.Next() {
		var row FriendRow
		var badgeCode int16
		var badgeSeason string
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.AvatarURL, &row.IsGuest, &badgeCode, &badgeSeason); err != nil {
			return nil, err
		}
		_, selected, err := s.profileBadges(ctx, row.UserID, badgeIDFromParts(badgeCode, badgeSeason))
		if err != nil {
			return nil, err
		}
		row.SelectedBadge = selected
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *pgStore) AcceptFriendship(userID, requesterID string) error {
	userID = strings.TrimSpace(userID)
	requesterID = strings.TrimSpace(requesterID)
	if userID == "" || requesterID == "" || userID == requesterID {
		return errors.New("invalid friendship")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, pair := range [][2]string{{requesterID, userID}, {userID, requesterID}} {
		if _, err := tx.Exec(ctx, `
			insert into friendships(user_id, friend_id, status, updated_at)
			values($1, $2, 'accepted', now())
			on conflict (user_id, friend_id) do update set status = 'accepted', updated_at = now()
		`, pair[0], pair[1]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *pgStore) DeclineFriendship(userID, requesterID string) error {
	userID = strings.TrimSpace(userID)
	requesterID = strings.TrimSpace(requesterID)
	if userID == "" || requesterID == "" {
		return errors.New("invalid friendship")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		delete from friendships
		where user_id = $1 and friend_id = $2 and status = 'pending'
	`, requesterID, userID)
	return err
}

func (s *pgStore) RemoveFriend(userID, friendID string) error {
	userID = strings.TrimSpace(userID)
	friendID = strings.TrimSpace(friendID)
	if userID == "" || friendID == "" || userID == friendID {
		return errors.New("invalid friendship")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
		delete from friendships
		where (user_id = $1 and friend_id = $2) or (user_id = $2 and friend_id = $1)
	`, userID, friendID)
	return err
}

func (s *pgStore) AreFriends(userID, otherID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	otherID = strings.TrimSpace(otherID)
	if userID == "" || otherID == "" {
		return false, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var exists bool
	err := s.pool.QueryRow(ctx, `
		select exists(
			select 1 from friendships
			where status = 'accepted'
			  and ((user_id = $1 and friend_id = $2) or (user_id = $2 and friend_id = $1))
		)
	`, userID, otherID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
