create table if not exists friendships (
  user_id    text not null,
  friend_id  text not null,
  status     text not null check (status in ('pending', 'accepted', 'blocked')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, friend_id)
);

create index if not exists friendships_friend_id_idx on friendships (friend_id);
create index if not exists friendships_user_status_idx on friendships (user_id, status);
create index if not exists friendships_friend_status_idx on friendships (friend_id, status);
