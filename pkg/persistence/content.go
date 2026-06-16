package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) GetLobbyChangelog(defaultContent LobbyChangelogContent) (LobbyChangelogContent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	posts, err := s.ListChangelogPosts(false)
	if err == nil && len(posts) > 0 {
		post := posts[0]
		preview := strings.TrimSpace(post.Summary)
		if preview == "" {
			preview = post.Markdown
		}
		return LobbyChangelogContent{
			Eyebrow:   "Latest News",
			Title:     post.Title,
			Markdown:  preview,
			Slug:      post.Slug,
			UpdatedAt: post.UpdatedAt,
		}, nil
	}
	var raw string
	err = s.pool.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'lobby_changelog'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultContent, nil
		}
		return LobbyChangelogContent{}, err
	}
	var content LobbyChangelogContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return defaultContent, nil
	}
	if strings.TrimSpace(content.Eyebrow) == "" {
		content.Eyebrow = defaultContent.Eyebrow
	}
	if strings.TrimSpace(content.Title) == "" {
		content.Title = defaultContent.Title
	}
	if strings.TrimSpace(content.Markdown) == "" {
		content.Markdown = defaultContent.Markdown
	}
	return content, nil
}

func (s *pgStore) SetLobbyChangelog(content LobbyChangelogContent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	payload, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('lobby_changelog', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload))
	return err
}

func (s *pgStore) ListChangelogPosts(includeUnpublished bool) ([]ChangelogPost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	query := `
		select id, slug, title, summary, markdown, published, created_at, updated_at
		from changelog_posts
		where ($1::boolean or published = true)
		order by updated_at desc, id desc
	`
	rows, err := s.pool.Query(ctx, query, includeUnpublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []ChangelogPost
	for rows.Next() {
		var post ChangelogPost
		if err := rows.Scan(
			&post.ID,
			&post.Slug,
			&post.Title,
			&post.Summary,
			&post.Markdown,
			&post.Published,
			&post.CreatedAt,
			&post.UpdatedAt,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *pgStore) GetChangelogPostBySlug(slug string, publishedOnly bool) (ChangelogPost, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		select id, slug, title, summary, markdown, published, created_at, updated_at
		from changelog_posts
		where slug = $1 and ($2::boolean = false or published = true)
	`, slug, publishedOnly).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChangelogPost{}, false, nil
		}
		return ChangelogPost{}, false, err
	}
	return post, true, nil
}

func (s *pgStore) CreateChangelogPost(input ChangelogPostInput) (ChangelogPost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		insert into changelog_posts(slug, title, summary, markdown, published, updated_at)
		values($1, $2, $3, $4, $5, now())
		returning id, slug, title, summary, markdown, published, created_at, updated_at
	`, input.Slug, input.Title, input.Summary, input.Markdown, input.Published).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	return post, err
}

func (s *pgStore) UpdateChangelogPost(id int64, input ChangelogPostInput) (ChangelogPost, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var post ChangelogPost
	err := s.pool.QueryRow(ctx, `
		update changelog_posts
		set slug = $2,
			title = $3,
			summary = $4,
			markdown = $5,
			published = $6,
			updated_at = now()
		where id = $1
		returning id, slug, title, summary, markdown, published, created_at, updated_at
	`, id, input.Slug, input.Title, input.Summary, input.Markdown, input.Published).Scan(
		&post.ID,
		&post.Slug,
		&post.Title,
		&post.Summary,
		&post.Markdown,
		&post.Published,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChangelogPost{}, false, nil
		}
		return ChangelogPost{}, false, err
	}
	return post, true, nil
}

func (s *pgStore) GetModerationSettings() (ModerationSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var raw string
	err := s.pool.QueryRow(ctx, `
		select value_json::text
		from site_settings
		where key = 'moderation_settings'
	`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ModerationSettings{}, nil
		}
		return ModerationSettings{}, err
	}
	var settings ModerationSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return ModerationSettings{}, nil
	}
	settings.DiscordWebhookURL = strings.TrimSpace(settings.DiscordWebhookURL)
	return settings, nil
}

func (s *pgStore) SetModerationSettings(settings ModerationSettings) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	settings.DiscordWebhookURL = strings.TrimSpace(settings.DiscordWebhookURL)
	payload, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into site_settings(key, value_json, updated_at)
		values('moderation_settings', $1::jsonb, now())
		on conflict (key) do update set
			value_json = excluded.value_json,
			updated_at = now()
	`, string(payload))
	return err
}
