-- name: CreateChangelogPost :one
INSERT INTO changelog_posts(slug,title,markdown,published,updated_at) VALUES($1,$2,$3,$4,now()) RETURNING id,slug,title,markdown,published,created_at,updated_at;

-- name: GetChangelogPost :one
SELECT id,slug,title,markdown,published,created_at,updated_at FROM changelog_posts WHERE slug=sqlc.arg(slug) AND (sqlc.arg(include_unpublished)::boolean=false OR published=true);

-- name: GetSetting :one
SELECT value_json FROM site_settings WHERE key=$1;

-- name: ListChangelogPosts :many
SELECT id,slug,title,markdown,published,created_at,updated_at FROM changelog_posts WHERE ($1::boolean OR published=true) ORDER BY updated_at DESC,id DESC;

-- name: SetSetting :exec
INSERT INTO site_settings(key,value_json,updated_at) VALUES(sqlc.arg(setting_key),convert_from(sqlc.arg(value_json), 'UTF8')::jsonb,now()) ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json,updated_at=now();

-- name: UpdateChangelogPost :one
UPDATE changelog_posts SET slug=$2,title=$3,markdown=$4,published=$5,updated_at=now() WHERE id=$1 RETURNING id,slug,title,markdown,published,created_at,updated_at;
