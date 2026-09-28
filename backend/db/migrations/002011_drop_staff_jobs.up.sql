-- The hand-rolled staff job queue is replaced by River (internal/jobs). Drop it
-- if an earlier development build applied it.
DROP TABLE IF EXISTS public.staff_jobs;
