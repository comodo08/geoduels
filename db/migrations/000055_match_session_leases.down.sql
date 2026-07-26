-- Revert durable ownership leases.
drop index if exists idx_match_sessions_expired_lease;

alter table match_sessions
  drop column if exists lease_expires_at;
