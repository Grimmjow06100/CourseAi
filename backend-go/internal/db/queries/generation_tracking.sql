-- name: GetGenerationTrackingSnapshot :one
SELECT jsonb_build_object(
  'requestId', r.id, 'generationAttempt', r.generation_attempt,
  'revision', r.tracking_revision::text, 'observedAt', clock_timestamp(),
  'pipelineStatus', r.pipeline_status, 'title', COALESCE(c.title, r.suggested_title, ''),
  'courseId', c.id, 'createdAt', r.created_at AT TIME ZONE 'UTC',
  'completedAt', r.completed_at AT TIME ZONE 'UTC',
  'isOutOfScope', r.is_out_of_scope, 'historyComplete', r.history_complete,
  'contentComplete', COALESCE(cs.content_complete, false),
  'modules', COALESCE((SELECT jsonb_agg(jsonb_build_object(
    'id', m.id, 'title', m.title, 'order', m.module_order,
    'lessons', COALESCE((SELECT jsonb_agg(jsonb_build_object(
      'id', l.id, 'title', l.title, 'order', l.lesson_order,
      'hasContent', (COALESCE(btrim(l.content_markdown), '') <> ''
        OR EXISTS (SELECT 1 FROM lesson_exercises e WHERE e.lesson_id = l.id)
        OR EXISTS (SELECT 1 FROM lesson_quizzes q WHERE q.lesson_id = l.id))
    ) ORDER BY l.lesson_order) FROM lessons l WHERE l.module_id = m.id), '[]'::jsonb)
  ) ORDER BY m.module_order) FROM modules m WHERE m.course_id = c.id), '[]'::jsonb),
  'operations', COALESCE((SELECT jsonb_agg(jsonb_build_object(
    'id', j.id, 'parentJobId', j.parent_job_id, 'targetId', j.target_id,
    'kind', j.kind, 'status', j.status, 'operationVersion', j.operation_version,
    'supersedesJobId', j.supersedes_job_id, 'attemptCount', j.attempt_count,
    'maxAttempts', j.max_attempts, 'availableAt', j.available_at,
    'startedAt', j.started_at, 'completedAt', j.completed_at,
    'failureCode', CASE
      WHEN j.status = 'cancelled' THEN 'cancelled'
      WHEN j.status NOT IN ('failed', 'retry_scheduled') THEN NULL
      WHEN j.last_error_code IN ('invalid_command', 'invalid_api_key', 'insufficient_quota', 'model_not_found', 'invalid_request_error', 'openai_http_400', 'openai_http_401', 'openai_http_403', 'openai_http_404') THEN 'operator_action_required'
      WHEN j.status = 'failed' AND j.attempt_count >= j.max_attempts THEN 'retry_exhausted'
      ELSE 'generation_failed' END
  ) ORDER BY j.created_at, j.id) FROM generation_jobs j
    WHERE j.request_id = r.id AND j.generation_attempt = r.generation_attempt AND j.is_current), '[]'::jsonb)
)::jsonb AS snapshot
FROM generation_requests r LEFT JOIN courses c ON c.request_id = r.id
LEFT JOIN course_content_states cs ON cs.course_id = c.id WHERE r.id = @request_id;

-- name: ListPublicGenerationEvents :many
SELECT id, generation_attempt, job_id, kind, status, target_id, operation_version, attempt_count, occurred_at
FROM generation_events WHERE request_id = @request_id AND (@cursor_id::bigint = 0 OR id < @cursor_id)
ORDER BY id DESC LIMIT @limit_rows;

-- name: SupersedeGenerationOperation :execrows
UPDATE generation_jobs SET is_current = false
WHERE id = @id AND operation_version = @operation_version AND is_current
  AND status IN ('failed', 'cancelled');

-- name: GetGenerationRetryReceipt :one
SELECT payload, result FROM generation_retry_commands WHERE request_id = @request_id AND idempotency_key = @idempotency_key;

-- name: SaveGenerationRetryReceipt :exec
INSERT INTO generation_retry_commands(request_id, idempotency_key, payload, result)
VALUES (@request_id, @idempotency_key, @payload, @result);

-- name: GetCurrentGenerationOperation :one
SELECT * FROM generation_jobs WHERE request_id = @request_id AND generation_attempt = @generation_attempt
AND kind = @kind AND target_id IS NOT DISTINCT FROM sqlc.narg('target_id')::uuid AND is_current;

-- name: PurgePublicGenerationEventsBefore :execrows
WITH candidates AS MATERIALIZED (
  SELECT e.id, e.request_id FROM generation_events e WHERE e.occurred_at < @cutoff
  ORDER BY e.occurred_at, e.id LIMIT @limit_rows
), request_locks AS MATERIALIZED (
  SELECT r.id FROM generation_requests r WHERE r.id IN (SELECT request_id FROM candidates)
  ORDER BY r.id FOR UPDATE
), removed AS (
  DELETE FROM generation_events WHERE id IN (SELECT id FROM candidates)
    AND request_id IN (SELECT id FROM request_locks) RETURNING request_id
)
UPDATE generation_requests SET history_complete = false, tracking_revision = tracking_revision + 1
WHERE id IN (SELECT request_id FROM removed);

-- name: PurgeGenerationRetryReceiptsBefore :execrows
DELETE FROM generation_retry_commands WHERE (request_id, idempotency_key) IN (
 SELECT receipt.request_id, receipt.idempotency_key FROM generation_retry_commands receipt WHERE receipt.created_at < @cutoff
 ORDER BY receipt.created_at LIMIT @limit_rows
);
