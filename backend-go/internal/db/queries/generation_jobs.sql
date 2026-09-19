-- name: EnqueueGenerationJob :one
INSERT INTO generation_jobs (
  is_current, operation_version, supersedes_job_id,
  generation_attempt,
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
  failure_handled_at,
  created_at,
  updated_at
)
VALUES (
  @is_current, @operation_version, sqlc.narg('supersedes_job_id'),
  @generation_attempt,
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
  sqlc.narg('failure_handled_at'),
  @created_at,
  @updated_at
)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetGenerationJobByID :one
SELECT *
FROM generation_jobs
WHERE id = @id;

-- name: LockGenerationJobClaim :one
SELECT id FROM generation_jobs
WHERE id = @id AND locked_by = @worker_id AND attempt_count = @attempt_count
  AND is_current AND status = 'running' AND locked_until > clock_timestamp()
FOR UPDATE;

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
WITH request_lock AS MATERIALIZED (
  SELECT r.id FROM generation_requests r JOIN LATERAL (
    SELECT j.priority, j.available_at, j.created_at, j.id FROM generation_jobs j
    WHERE j.request_id = r.id AND j.generation_attempt = r.generation_attempt AND j.is_current
      AND j.status IN ('queued', 'retry_scheduled') AND j.available_at <= CURRENT_TIMESTAMP AND j.attempt_count < j.max_attempts
    ORDER BY j.priority DESC, j.available_at, j.created_at, j.id LIMIT 1
  ) next_job ON true
  ORDER BY next_job.priority DESC, next_job.available_at, next_job.created_at, next_job.id
  FOR UPDATE OF r SKIP LOCKED LIMIT 1
), candidate AS (
  SELECT id
  FROM generation_jobs
  WHERE request_id IN (SELECT id FROM request_lock) AND is_current AND status IN ('queued', 'retry_scheduled')
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
WITH request_lock AS MATERIALIZED (
 SELECT r.id FROM generation_requests r JOIN generation_jobs j ON j.request_id = r.id WHERE j.id = @id FOR UPDATE OF r
)
UPDATE generation_jobs AS job
SET
  status = 'completed',
  locked_by = NULL,
  locked_until = NULL,
  completed_at = @completed_at,
  last_error_code = NULL,
  last_error_message = NULL,
  updated_at = @completed_at
WHERE job.id = @id AND job.request_id IN (SELECT id FROM request_lock)
  AND job.status = 'running'
  AND job.locked_by = @worker_id
  AND job.attempt_count = @attempt_count
  AND job.locked_until > CURRENT_TIMESTAMP;

-- name: RetryGenerationJob :execrows
WITH request_lock AS MATERIALIZED (
 SELECT r.id FROM generation_requests r JOIN generation_jobs j ON j.request_id = r.id WHERE j.id = @id FOR UPDATE OF r
)
UPDATE generation_jobs AS job
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
WHERE job.id = @id AND job.request_id IN (SELECT id FROM request_lock)
  AND job.status = 'running'
  AND job.locked_by = @worker_id
  AND job.attempt_count = @attempt_count
  AND job.locked_until > CURRENT_TIMESTAMP;

-- name: FailGenerationJob :execrows
WITH request_lock AS MATERIALIZED (
 SELECT r.id FROM generation_requests r JOIN generation_jobs j ON j.request_id = r.id WHERE j.id = @id FOR UPDATE OF r
)
UPDATE generation_jobs AS job
SET
  status = 'failed',
  locked_by = NULL,
  locked_until = NULL,
  completed_at = @failed_at,
  last_error_code = @error_code,
  last_error_message = @error_message,
  updated_at = @failed_at
WHERE job.id = @id AND job.request_id IN (SELECT id FROM request_lock)
  AND job.status = 'running'
  AND job.locked_by = @worker_id
  AND job.attempt_count = @attempt_count
  AND job.locked_until > CURRENT_TIMESTAMP;

-- name: CancelGenerationJob :execrows
WITH request_lock AS MATERIALIZED (
 SELECT r.id FROM generation_requests r JOIN generation_jobs j ON j.request_id = r.id WHERE j.id = @id FOR UPDATE OF r
)
UPDATE generation_jobs AS job
SET
  status = 'cancelled',
  completed_at = @cancelled_at,
  updated_at = @cancelled_at
WHERE job.id = @id AND job.request_id IN (SELECT id FROM request_lock)
  AND job.status IN ('queued', 'retry_scheduled');

-- name: RequeueExpiredGenerationJobs :execrows
WITH request_locks AS MATERIALIZED (
 SELECT r.id FROM generation_requests r WHERE EXISTS (
   SELECT 1 FROM generation_jobs j WHERE j.request_id = r.id AND j.status = 'running' AND j.locked_until <= @requeued_at
 ) ORDER BY r.id FOR UPDATE OF r
)
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
WHERE request_id IN (SELECT id FROM request_locks) AND status = 'running'
  AND locked_until <= @requeued_at;

-- name: ListUnreconciledFailedGenerationJobs :many
SELECT *
FROM generation_jobs
WHERE status = 'failed'
  AND failure_handled_at IS NULL
ORDER BY completed_at ASC, id ASC
LIMIT @limit_rows;

-- name: MarkGenerationJobFailureHandled :execrows
UPDATE generation_jobs AS job
SET failure_handled_at = @handled_at,
    updated_at = GREATEST(job.updated_at, @handled_at)
WHERE job.id = @id
  AND job.status = 'failed'
  AND job.failure_handled_at IS NULL;

-- name: PurgeTerminalGenerationJobsBefore :execrows
DELETE FROM generation_jobs WHERE id IN (
 SELECT j.id FROM generation_jobs j JOIN generation_requests r ON r.id = j.request_id
 WHERE (NOT j.is_current OR j.generation_attempt < r.generation_attempt)
 AND j.status IN ('completed', 'failed', 'cancelled') AND j.updated_at < @cutoff
 AND NOT EXISTS (SELECT 1 FROM generation_jobs child WHERE child.parent_job_id = j.id)
 ORDER BY j.updated_at, j.id LIMIT @limit_rows
);

-- name: GetGenerationQueueMetrics :one
SELECT
  count(*) FILTER (WHERE status = 'queued')::bigint AS queued,
  count(*) FILTER (WHERE status = 'retry_scheduled')::bigint AS retry_scheduled,
  count(*) FILTER (WHERE status = 'running')::bigint AS running,
  count(*) FILTER (WHERE status = 'completed')::bigint AS completed,
  count(*) FILTER (WHERE status = 'failed')::bigint AS failed,
  count(*) FILTER (WHERE status = 'cancelled')::bigint AS cancelled,
  count(*) FILTER (WHERE status = 'running' AND locked_until <= CURRENT_TIMESTAMP)::bigint AS expired_leases,
  count(*) FILTER (WHERE status = 'failed' AND failure_handled_at IS NULL)::bigint AS unreconciled_failures
FROM generation_jobs;

-- name: CountPendingGenerationJobsWithAdmissionLock :one
WITH admission_lock AS (
  SELECT pg_advisory_xact_lock(4931529157321281::bigint)
)
SELECT count(*)::bigint
FROM generation_jobs, admission_lock
WHERE status IN ('queued', 'retry_scheduled', 'running');
