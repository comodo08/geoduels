-- The custom outbox tables are replaced by River (internal/jobs). Drop them and
-- the enums they owned.
DROP TABLE IF EXISTS public.notification_outbox;
DROP TABLE IF EXISTS public.discord_sync_outbox;
DROP TYPE IF EXISTS public.gd_notification_outbox_type;
DROP TYPE IF EXISTS public.gd_discord_sync_action;
