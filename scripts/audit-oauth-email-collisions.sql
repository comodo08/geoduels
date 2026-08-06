-- Read-only report of verified OAuth emails associated with more than one
-- active GeoDuels user. Email values are represented by an MD5 fingerprint so
-- routine operational output does not disclose them.
with email_claims as (
  select lower(btrim(email)) as normalized_email, id as user_id
  from users
  where email is not null and btrim(email) <> '' and deleted_at is null
  union
  select lower(btrim(ui.email)), ui.user_id
  from user_identities ui
  join users u on u.id = ui.user_id
  where ui.email is not null and btrim(ui.email) <> '' and u.deleted_at is null
), collision_users as (
  select c.normalized_email, c.user_id
  from email_claims c
  join (
    select normalized_email
    from email_claims
    group by normalized_email
    having count(distinct user_id) > 1
  ) collisions using (normalized_email)
  group by c.normalized_email, c.user_id
)
select
  md5(c.normalized_email) as email_fingerprint,
  u.id as user_id,
  u.created_at,
  u.account_type,
  coalesce((
    select string_agg(ui.provider, ',' order by ui.provider)
    from user_identities ui
    where ui.user_id = u.id
  ), '') as providers,
  (select count(*) from auth_sessions s where s.user_id = u.id) as session_count,
  (select max(s.last_used_at) from auth_sessions s where s.user_id = u.id) as latest_session_at,
  (select count(*) from match_players mp where mp.user_id = u.id) as match_count,
  (select count(*) from maps m where m.owner_user_id = u.id) as map_count,
  coalesce((select us.games_played from user_stats us where us.user_id = u.id), 0) as games_played,
  u.banned_at,
  u.deleted_at
from collision_users c
join users u on u.id = c.user_id
order by email_fingerprint, u.created_at, u.id;
