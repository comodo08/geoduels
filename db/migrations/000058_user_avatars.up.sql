-- Custom user avatars are stored as Postgres rows and served through a
-- dedicated public endpoint. Setting users.avatar_url to the served URL
-- overrides the OAuth provider avatar (existing coalesce precedence handles
-- the fallback when the custom avatar is cleared).
create table if not exists user_avatars (
  user_id      uuid primary key references users (id) on delete cascade,
  content_type text not null,
  data         bytea not null,
  width        int not null default 0,
  height       int not null default 0,
  created_at   timestamptz not null default now()
);
