-- name: GetIdentity :one
SELECT u.id AS user_id,coalesce(u.email,ui.email,'') AS email,coalesce(ui.provider_name,'') AS provider_name,coalesce(u.avatar_url,ui.avatar_url,'') AS avatar_url,coalesce(gd_is_registered(u.id) AND u.nickname_claimed_at IS NULL,false) AS needs_nickname,gd_display_name(u.display_name,ui.provider_name,u.id) AS display_name,gd_is_registered(u.id) AS has_identity,gd_is_admin(u.id) AS is_admin,gd_is_judge(u.id) AS is_moderator,gd_is_banned(u.id) AS is_banned,coalesce(u.ban_reason,'') AS ban_reason FROM users u LEFT JOIN LATERAL (SELECT email,provider_name,avatar_url FROM user_identities WHERE user_id=u.id AND provider IN ('discord','google') ORDER BY CASE provider WHEN 'discord' THEN 0 WHEN 'google' THEN 1 ELSE 2 END,created_at ASC LIMIT 1) ui ON true WHERE u.id=sqlc.arg(user_id);

-- name: ListIdentityProviders :many
SELECT provider FROM user_identities WHERE user_id=$1 ORDER BY CASE provider WHEN 'discord' THEN 0 WHEN 'google' THEN 1 ELSE 2 END,provider;

-- name: NicknameTaken :one
SELECT EXISTS(
  SELECT 1 FROM users
  WHERE id <> $1
    AND nickname_claimed_at IS NOT NULL
    AND lower(display_name) = lower($2)
) AS taken;

-- name: ProviderIdentityBanned :one
SELECT coalesce(reason,'') FROM oauth_identity_bans WHERE provider=$1 AND provider_user_id=$2 AND revoked_at IS NULL LIMIT 1;

-- name: ProviderIdentityExists :one
SELECT EXISTS(SELECT 1 FROM user_identities WHERE provider=$1 AND provider_user_id=$2);

-- name: SetNickname :execrows
UPDATE users
SET display_name = $2,
    nickname_claimed_at = coalesce(nickname_claimed_at, now())
WHERE id = $1
  AND gd_is_registered(users.id);
