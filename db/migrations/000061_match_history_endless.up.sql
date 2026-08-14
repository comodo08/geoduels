alter table match_history add column if not exists endless boolean not null default false;

update match_history
set endless = true
where mode = 'singleplayer'
  and (round_count > 5 or (replay_zstd is null and round_count < 5));
