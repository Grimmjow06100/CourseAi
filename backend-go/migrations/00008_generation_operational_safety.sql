-- +goose Up
ALTER TABLE generation_jobs
ADD COLUMN failure_handled_at TIMESTAMPTZ(3),
ADD CONSTRAINT generation_jobs_failure_handled_state_check
  CHECK (failure_handled_at IS NULL OR status = 'failed');

CREATE INDEX generation_jobs_unreconciled_failure_idx
ON generation_jobs (completed_at, id)
WHERE status = 'failed' AND failure_handled_at IS NULL;

CREATE INDEX generation_jobs_status_updated_idx
ON generation_jobs (status, updated_at);

CREATE INDEX generation_requests_clerk_status_created_idx
ON generation_requests (clerk_user_id, pipeline_status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS generation_requests_clerk_status_created_idx;
DROP INDEX IF EXISTS generation_jobs_status_updated_idx;
DROP INDEX IF EXISTS generation_jobs_unreconciled_failure_idx;

ALTER TABLE generation_jobs
DROP CONSTRAINT IF EXISTS generation_jobs_failure_handled_state_check,
DROP COLUMN IF EXISTS failure_handled_at;
