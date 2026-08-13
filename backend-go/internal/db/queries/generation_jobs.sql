-- name: EnqueueGenerationJob :one
INSERT INTO generation_jobs (
  id,
  request_id,
  parent_job_id,
  kind,
  status,
  target_id,
  idempotency_key,
  payload,
  priority,
  attempt_count,
  max_attempts,
  available_at,
  locked_by,
  locked_until,
  started_at,
  completed_at,
  last_error_code,
  last_error_message,
  created_at,
  updated_at
)
VALUES (
  @id,
  @request_id,
  sqlc.narg('parent_job_id'),
  @kind::generation_job_kind,
  @status::generation_job_status,
  sqlc.narg('target_id'),
  @idempotency_key,
  @payload::jsonb,
  @priority,
  @attempt_count,
  @max_attempts,
  @available_at,
  sqlc.narg('locked_by'),
  sqlc.narg('locked_until'),
  sqlc.narg('started_at'),
  sqlc.narg('completed_at'),
  sqlc.narg('last_error_code'),
  sqlc.narg('last_error_message'),
  @created_at,
  @updated_at
)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING *;

-- name: GetGenerationJobByID :one
SELECT *
FROM generation_jobs
WHERE id = @id;

-- name: GetGenerationJobByIdempotencyKey :one
SELECT *
FROM generation_jobs
WHERE idempotency_key = @idempotency_key;

-- name: ListGenerationJobsByRequestID :many
SELECT *
FROM generation_jobs
WHERE request_id = @request_id
ORDER BY created_at ASC, id ASC;

-- name: ClaimNextGenerationJob :one
WITH candidate AS (
  SELECT id
  FROM generation_jobs
  WHERE status IN ('queued', 'retry_scheduled')
    AND available_at <= CURRENT_TIMESTAMP
    AND attempt_count < max_attempts
  ORDER BY priority DESC, available_at ASC, created_at ASC, id ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE generation_jobs AS job
SET
  status = 'running',
  attempt_count = job.attempt_count + 1,
  locked_by = @worker_id,
  locked_until = @locked_until,
  started_at = COALESCE(job.started_at, CURRENT_TIMESTAMP),
  updated_at = CURRENT_TIMESTAMP,
  last_error_code = NULL,
  last_error_message = NULL
FROM candidate
WHERE job.id = candidate.id
  AND @locked_until > CURRENT_TIMESTAMP
RETURNING job.*;

-- name: RenewGenerationJobLease :execrows
UPDATE generation_jobs
SET
  locked_until = @locked_until,
  updated_at = CURRENT_TIMESTAMP
WHERE id = @id
  AND status = 'running'
  AND locked_by = @worker_id
  AND attempt_count = @attempt_count
  AND locked_until > CURRENT_TIMESTAMP
  AND @locked_until > CURRENT_TIMESTAMP;

-- name: CompleteGenerationJob :execrows
UPDATE generation_jobs
SET
  status = 'completed',
  locked_by = NULL,
  locked_until = NULL,
  completed_at = @completed_at,
  last_error_code = NULL,
  last_error_message = NULL,
  updated_at = @completed_at
WHERE id = @id
  AND status = 'running'
  AND locked_by = @worker_id
  AND attempt_count = @attempt_count
  AND locked_until > CURRENT_TIMESTAMP;

-- name: RetryGenerationJob :execrows
UPDATE generation_jobs
SET
  status = CASE
    WHEN attempt_count < max_attempts THEN 'retry_scheduled'::generation_job_status
    ELSE 'failed'::generation_job_status
  END,
  available_at = @retry_at,
  locked_by = NULL,
  locked_until = NULL,
  completed_at = CASE WHEN attempt_count >= max_attempts THEN CURRENT_TIMESTAMP ELSE NULL END,
  last_error_code = CASE
    WHEN attempt_count < max_attempts THEN @error_code
    ELSE 'max_attempts_exhausted'
  END,
  last_error_message = @error_message,
  updated_at = CURRENT_TIMESTAMP
WHERE id = @id
  AND status = 'running'
  AND locked_by = @worker_id
  AND attempt_count = @attempt_count
  AND locked_until > CURRENT_TIMESTAMP;

-- name: FailGenerationJob :execrows
UPDATE generation_jobs
SET
  status = 'failed',
  locked_by = NULL,
  locked_until = NULL,
  completed_at = @failed_at,
  last_error_code = @error_code,
  last_error_message = @error_message,
  updated_at = @failed_at
WHERE id = @id
  AND status = 'running'
  AND locked_by = @worker_id
  AND attempt_count = @attempt_count
  AND locked_until > CURRENT_TIMESTAMP;

-- name: CancelGenerationJob :execrows
UPDATE generation_jobs
SET
  status = 'cancelled',
  completed_at = @cancelled_at,
  updated_at = @cancelled_at
WHERE id = @id
  AND status IN ('queued', 'retry_scheduled');

-- name: RequeueExpiredGenerationJobs :execrows
UPDATE generation_jobs
SET
  status = CASE
    WHEN attempt_count < max_attempts THEN 'retry_scheduled'::generation_job_status
    ELSE 'failed'::generation_job_status
  END,
  available_at = @requeued_at,
  locked_by = NULL,
  locked_until = NULL,
  completed_at = CASE WHEN attempt_count >= max_attempts THEN @requeued_at ELSE NULL END,
  last_error_code = CASE
    WHEN attempt_count < max_attempts THEN 'lease_expired'
    ELSE 'max_attempts_exhausted'
  END,
  last_error_message = 'worker lease expired before the job completed',
  updated_at = @requeued_at
WHERE status = 'running'
  AND locked_until <= @requeued_at;
