-- Merge an empty duplicate OAuth account into an established survivor.
--
-- Usage:
--   psql "$POSTGRES_URL" -X -v ON_ERROR_STOP=1 \
--     -v survivor_user_id='...' -v duplicate_user_id='...' \
--     -f scripts/merge-empty-oauth-account.sql
--
-- This script deliberately aborts when the duplicate owns any material data,
-- is moderated, does not share an OAuth email with the survivor, or has a
-- provider that would conflict on the survivor.
begin;

create temporary table oauth_merge_params (
  survivor_id uuid primary key,
  duplicate_id uuid unique not null,
  duplicate_email text
) on commit drop;

insert into oauth_merge_params(survivor_id, duplicate_id)
values (:'survivor_user_id'::uuid, :'duplicate_user_id'::uuid);

select 1
from users u
join oauth_merge_params p on u.id in (p.survivor_id, p.duplicate_id)
for update;

do $merge_guard$
declare
  survivor_count integer;
  duplicate_count integer;
  shared_email boolean;
  provider_conflict boolean;
  material_rows boolean;
  user_reference record;
begin
  select count(*) filter (where u.id = p.survivor_id),
         count(*) filter (where u.id = p.duplicate_id)
  into survivor_count, duplicate_count
  from oauth_merge_params p
  left join users u on u.id in (p.survivor_id, p.duplicate_id);

  if survivor_count <> 1 or duplicate_count <> 1 then
    raise exception 'both survivor and duplicate users must exist';
  end if;

  select exists (
    select 1
    from oauth_merge_params p
    join users d on d.id = p.duplicate_id
    where d.deleted_at is not null or d.banned_at is not null
  ) into material_rows;
  if material_rows then
    raise exception 'duplicate is deleted or has moderation state';
  end if;

  select exists (
    select 1
    from oauth_merge_params p
    join user_identities a on a.user_id = p.survivor_id
    join user_identities b on b.user_id = p.duplicate_id
      and lower(btrim(b.email)) = lower(btrim(a.email))
    where a.email is not null and b.email is not null
  ) into shared_email;
  if not shared_email then
    raise exception 'accounts do not share an active OAuth email';
  end if;

  select exists (
    select 1
    from oauth_merge_params p
    join user_identities a on a.user_id = p.survivor_id
    join user_identities b on b.user_id = p.duplicate_id and b.provider = a.provider
  ) into provider_conflict;
  if provider_conflict then
    raise exception 'survivor already has one of the duplicate providers';
  end if;

  -- Refuse every user reference except the records this script explicitly
  -- knows how to move, revoke, or discard. This automatically stays strict
  -- when a future migration adds another user-owned table.
  for user_reference in
    select c.conrelid::regclass::text as table_name, a.attname as column_name
    from pg_constraint c
    cross join lateral unnest(c.conkey) key_column(attnum)
    join pg_attribute a on a.attrelid = c.conrelid and a.attnum = key_column.attnum
    where c.contype = 'f'
      and c.confrelid = 'users'::regclass
      and c.conrelid not in (
        'auth_sessions'::regclass,
        'user_identities'::regclass,
        'user_identity_history'::regclass,
        'ranks'::regclass,
        'ranked_stats'::regclass,
        'user_stats'::regclass
      )
  loop
    execute format(
      'select exists (select 1 from %s x, oauth_merge_params p where x.%I = p.duplicate_id)',
      user_reference.table_name,
      user_reference.column_name
    ) into material_rows;
    if material_rows then
      raise exception 'duplicate owns material data in %.%; manual merge required',
        user_reference.table_name, user_reference.column_name;
    end if;
  end loop;

  select
    exists(select 1 from user_stats x, oauth_merge_params p where x.user_id = p.duplicate_id and (x.games_played <> 0 or x.wins <> 0))
    or exists(select 1 from ranked_stats x, oauth_merge_params p where x.user_id = p.duplicate_id and (x.games_played <> 0 or x.wins <> 0))
  into material_rows;
  if material_rows then
    raise exception 'duplicate has nonzero seeded statistics; manual merge required';
  end if;
end
$merge_guard$;

update oauth_merge_params p
set duplicate_email = d.email
from users d
where d.id = p.duplicate_id;

update auth_sessions s
set revoked_at = coalesce(s.revoked_at, now())
from oauth_merge_params p
where s.user_id = p.duplicate_id;

update user_identities ui
set user_id = p.survivor_id
from oauth_merge_params p
where ui.user_id = p.duplicate_id;

insert into user_identity_history(
  user_id, provider, provider_user_id, email, provider_name,
  first_seen_at, last_seen_at, deleted_at
)
select
  p.survivor_id, h.provider, h.provider_user_id, h.email, h.provider_name,
  h.first_seen_at, h.last_seen_at, null
from user_identity_history h
join oauth_merge_params p on h.user_id = p.duplicate_id
on conflict (user_id, provider, provider_user_id) do update set
  email = excluded.email,
  provider_name = excluded.provider_name,
  first_seen_at = least(user_identity_history.first_seen_at, excluded.first_seen_at),
  last_seen_at = greatest(user_identity_history.last_seen_at, excluded.last_seen_at),
  deleted_at = null;

delete from user_identity_history h
using oauth_merge_params p
where h.user_id = p.duplicate_id;

delete from ranked_stats x using oauth_merge_params p where x.user_id = p.duplicate_id;
delete from ranks x using oauth_merge_params p where x.user_id = p.duplicate_id;
delete from user_stats x using oauth_merge_params p where x.user_id = p.duplicate_id;
delete from users u using oauth_merge_params p where u.id = p.duplicate_id;

update users u
set email = coalesce(u.email, p.duplicate_email)
from oauth_merge_params p
where u.id = p.survivor_id;

commit;
