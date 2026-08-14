package persistence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxAvatarBytes = 2 << 20 // 2 MiB

const maxAvatarDimension = 512

var allowedAvatarContentTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
}

// SetUserAvatar stores the decoded avatar bytes, validates that the payload is a
// supported raster image within the size/dimension limits, and points
// users.avatar_url at the served URL so all existing consumers pick it up.
// baseURL is the absolute API origin used to build the served URL.
func (s *pgStore) SetUserAvatar(userID, contentType string, data []byte, baseURL string) (string, error) {
	if userID == "" {
		return "", errors.New("user id required")
	}
	if len(data) == 0 {
		return "", errors.New("avatar data required")
	}
	if !isAllowedAvatarContentType(contentType) {
		return "", errors.New("unsupported avatar content type")
	}
	if len(data) > maxAvatarBytes {
		return "", errors.New("avatar exceeds size limit")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("avatar must be a valid PNG or JPEG image")
	}
	if cfg.Width == 0 || cfg.Height == 0 {
		return "", errors.New("avatar image has no dimensions")
	}
	if cfg.Width > maxAvatarDimension || cfg.Height > maxAvatarDimension {
		return "", fmt.Errorf("avatar must be at most %dx%d pixels", maxAvatarDimension, maxAvatarDimension)
	}
	if !isAllowedAvatarFormat(format) {
		return "", errors.New("avatar must be a valid PNG or JPEG image")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var accountType string
	if err := s.pool.QueryRow(ctx, `select coalesce(account_type, 'registered') from users where id = $1`, userID).Scan(&accountType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("user not found")
		}
		return "", err
	}
	if accountType == "guest" {
		return "", errors.New("guests cannot upload avatars")
	}

	if _, err := s.pool.Exec(ctx, `
		insert into user_avatars (user_id, content_type, data, width, height, created_at)
		values ($1, $2, $3, $4, $5, now())
		on conflict (user_id) do update set
			content_type = excluded.content_type,
			data = excluded.data,
			width = excluded.width,
			height = excluded.height,
			created_at = now()
	`, userID, contentType, data, cfg.Width, cfg.Height); err != nil {
		return "", err
	}

	var createdAt time.Time
	if err := s.pool.QueryRow(ctx, `select created_at from user_avatars where user_id = $1`, userID).Scan(&createdAt); err != nil {
		return "", err
	}

	url := buildUserAvatarURL(baseURL, userID, createdAt)
	if _, err := s.pool.Exec(ctx, `
		update users set avatar_url = $2
		where id = $1 and coalesce(account_type, 'registered') <> 'guest'
	`, userID, url); err != nil {
		return "", err
	}
	return url, nil
}

// ResetUserAvatar removes the stored avatar bytes and clears users.avatar_url so
// the provider avatar (coalesce precedence) takes over again.
func (s *pgStore) ResetUserAvatar(userID string) error {
	if userID == "" {
		return errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := s.pool.Exec(ctx, `delete from user_avatars where user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `update users set avatar_url = null where id = $1`, userID); err != nil {
		return err
	}
	return nil
}

// GetUserAvatar returns the stored content type and bytes for a user's custom
// avatar. It returns pgx.ErrNoRows when no custom avatar exists.
func (s *pgStore) GetUserAvatar(userID string) (string, []byte, error) {
	if userID == "" {
		return "", nil, errors.New("user id required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var contentType string
	var data []byte
	err := s.pool.QueryRow(ctx, `select content_type, data from user_avatars where user_id = $1`, userID).Scan(&contentType, &data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, pgx.ErrNoRows
		}
		return "", nil, err
	}
	return contentType, data, nil
}

func isAllowedAvatarContentType(contentType string) bool {
	_, ok := allowedAvatarContentTypes[strings.TrimSpace(strings.ToLower(contentType))]
	return ok
}

func isAllowedAvatarFormat(format string) bool {
	switch strings.ToLower(format) {
	case "png", "jpeg":
		return true
	default:
		return false
	}
}

func buildUserAvatarURL(base, userID string, createdAt time.Time) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path := fmt.Sprintf("/v1/avatars/%s?v=%d", userID, createdAt.Unix())
	if base == "" {
		return path
	}
	return base + path
}
