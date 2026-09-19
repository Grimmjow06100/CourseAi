-- +goose Up
ALTER TABLE generation_requests ADD COLUMN tracking_revision bigint NOT NULL DEFAULT 0;
ALTER TABLE generation_requests ADD COLUMN history_complete boolean NOT NULL DEFAULT false;
ALTER TABLE generation_requests ALTER COLUMN history_complete SET DEFAULT true;
ALTER TABLE generation_jobs ADD COLUMN is_current boolean NOT NULL DEFAULT true;
ALTER TABLE generation_jobs ADD COLUMN operation_version integer NOT NULL DEFAULT 1 CHECK (operation_version > 0);
ALTER TABLE generation_jobs ADD COLUMN supersedes_job_id uuid REFERENCES generation_jobs(id) ON DELETE SET NULL;

WITH ranked AS (
  SELECT id, row_number() OVER (
    PARTITION BY request_id, generation_attempt, kind, target_id
    ORDER BY created_at, id
  ) AS version, row_number() OVER (
    PARTITION BY request_id, generation_attempt, kind, target_id
    ORDER BY created_at DESC, id DESC
  ) AS latest FROM generation_jobs
)
UPDATE generation_jobs j SET is_current = (r.latest = 1), operation_version = r.version
FROM ranked r WHERE r.id = j.id;

-- Fence obsolete work before enabling the new uniqueness constraint. A running
-- old process loses its lease and cannot persist its provider response.
UPDATE generation_jobs j SET status = 'cancelled', locked_by = NULL, locked_until = NULL,
  completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP,
  last_error_code = NULL, last_error_message = NULL
FROM generation_requests r WHERE j.request_id = r.id
  AND (NOT j.is_current OR j.generation_attempt < r.generation_attempt)
  AND j.status IN ('queued', 'running', 'retry_scheduled');

-- Repair requests poisoned by a local error while independent work still exists.
UPDATE generation_requests r SET pipeline_status = 'running', completed_at = NULL, failure_message = NULL
WHERE r.pipeline_status = 'failed' AND NOT r.is_out_of_scope AND EXISTS (
  SELECT 1 FROM generation_jobs j WHERE j.request_id = r.id AND j.generation_attempt = r.generation_attempt
    AND j.is_current AND j.status IN ('queued', 'running', 'retry_scheduled')
);
UPDATE courses c SET status = CASE WHEN EXISTS (
  SELECT 1 FROM modules m WHERE m.course_id = c.id AND NOT EXISTS (SELECT 1 FROM lessons l WHERE l.module_id = m.id)
) THEN 'lessons_generating'::course_generation_status ELSE 'content_generating'::course_generation_status END
FROM generation_requests r WHERE r.id = c.request_id AND r.pipeline_status = 'running' AND c.status = 'failed';

CREATE UNIQUE INDEX generation_jobs_current_operation ON generation_jobs
  (request_id, generation_attempt, kind, COALESCE(target_id, '00000000-0000-0000-0000-000000000000'::uuid))
  WHERE is_current;

CREATE TABLE generation_events (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  request_id uuid NOT NULL REFERENCES generation_requests(id) ON DELETE CASCADE,
  generation_attempt integer NOT NULL,
  job_id uuid REFERENCES generation_jobs(id) ON DELETE SET NULL,
  kind text NOT NULL,
  status text NOT NULL,
  target_id uuid,
  operation_version integer,
  attempt_count integer,
  occurred_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX generation_events_request_cursor ON generation_events(request_id, id DESC);

CREATE TABLE generation_retry_commands (
  request_id uuid NOT NULL REFERENCES generation_requests(id) ON DELETE CASCADE,
  idempotency_key text NOT NULL,
  payload jsonb NOT NULL,
  result jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (request_id, idempotency_key)
);

-- Revisions are updated in the transaction that changes public state. The
-- snapshot reads revision and state in a single SQL statement. Lease heartbeats
-- and internal provider errors do not produce public events or revision churn.
-- +goose StatementBegin
CREATE FUNCTION record_generation_job_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND (NEW.status, NEW.is_current, NEW.attempt_count)
    IS NOT DISTINCT FROM (OLD.status, OLD.is_current, OLD.attempt_count) THEN RETURN NEW; END IF;
  UPDATE generation_requests SET tracking_revision = tracking_revision + 1 WHERE id = NEW.request_id;
  INSERT INTO generation_events(request_id, generation_attempt, job_id, kind, status, target_id, operation_version, attempt_count)
    VALUES (NEW.request_id, NEW.generation_attempt, NEW.id, NEW.kind::text,
      CASE WHEN NOT NEW.is_current THEN 'superseded' ELSE NEW.status::text END,
      NEW.target_id, NEW.operation_version, NEW.attempt_count);
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER generation_job_event AFTER INSERT OR UPDATE ON generation_jobs
  FOR EACH ROW EXECUTE FUNCTION record_generation_job_event();

-- +goose StatementBegin
CREATE FUNCTION record_generation_request_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND (NEW.pipeline_status, NEW.current_step, NEW.progress_percent, NEW.generation_attempt)
    IS NOT DISTINCT FROM (OLD.pipeline_status, OLD.current_step, OLD.progress_percent, OLD.generation_attempt)
    THEN RETURN NEW; END IF;
  NEW.tracking_revision := NEW.tracking_revision + 1;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER generation_request_revision BEFORE INSERT OR UPDATE ON generation_requests
  FOR EACH ROW EXECUTE FUNCTION record_generation_request_event();

-- +goose StatementBegin
CREATE FUNCTION record_generation_pipeline_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND (NEW.pipeline_status, NEW.generation_attempt)
    IS NOT DISTINCT FROM (OLD.pipeline_status, OLD.generation_attempt) THEN RETURN NEW; END IF;
  INSERT INTO generation_events(request_id, generation_attempt, kind, status)
    VALUES (NEW.id, NEW.generation_attempt, 'pipeline', NEW.pipeline_status::text);
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER generation_pipeline_event AFTER INSERT OR UPDATE ON generation_requests
  FOR EACH ROW EXECUTE FUNCTION record_generation_pipeline_event();

-- +goose StatementBegin
CREATE FUNCTION revise_generation_content() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE request uuid;
BEGIN
  IF TG_TABLE_NAME = 'courses' THEN request := NEW.request_id;
  ELSIF TG_TABLE_NAME = 'modules' THEN SELECT request_id INTO request FROM courses WHERE id = NEW.course_id;
  ELSE SELECT c.request_id INTO request FROM courses c JOIN modules m ON m.course_id = c.id WHERE m.id = NEW.module_id;
  END IF;
  UPDATE generation_requests SET tracking_revision = tracking_revision + 1 WHERE id = request;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER generation_course_revision AFTER INSERT OR UPDATE ON courses FOR EACH ROW EXECUTE FUNCTION revise_generation_content();
CREATE TRIGGER generation_module_revision AFTER INSERT OR UPDATE ON modules FOR EACH ROW EXECUTE FUNCTION revise_generation_content();
CREATE TRIGGER generation_lesson_revision AFTER INSERT OR UPDATE ON lessons FOR EACH ROW EXECUTE FUNCTION revise_generation_content();

-- +goose Down
DROP TRIGGER generation_lesson_revision ON lessons;
DROP TRIGGER generation_module_revision ON modules;
DROP TRIGGER generation_course_revision ON courses;
DROP FUNCTION revise_generation_content();
DROP TRIGGER generation_pipeline_event ON generation_requests;
DROP FUNCTION record_generation_pipeline_event();
DROP TRIGGER generation_request_revision ON generation_requests;
DROP FUNCTION record_generation_request_event();
DROP TRIGGER generation_job_event ON generation_jobs;
DROP FUNCTION record_generation_job_event();
DROP TABLE generation_events;
DROP TABLE generation_retry_commands;
DROP INDEX generation_jobs_current_operation;
ALTER TABLE generation_jobs DROP COLUMN supersedes_job_id, DROP COLUMN operation_version, DROP COLUMN is_current;
ALTER TABLE generation_requests DROP COLUMN history_complete, DROP COLUMN tracking_revision;
