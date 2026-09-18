-- +goose Up
ALTER TABLE generation_requests ADD COLUMN generation_attempt integer NOT NULL DEFAULT 1 CHECK (generation_attempt > 0);
ALTER TABLE generation_jobs ADD COLUMN generation_attempt integer NOT NULL DEFAULT 1 CHECK (generation_attempt > 0);
CREATE INDEX generation_jobs_request_attempt ON generation_jobs(request_id, generation_attempt, status);

CREATE VIEW course_content_states AS
SELECT c.id AS course_id,
  EXISTS (SELECT 1 FROM modules m WHERE m.course_id = c.id)
  AND NOT EXISTS (
    SELECT 1 FROM modules m WHERE m.course_id = c.id
    AND NOT EXISTS (SELECT 1 FROM lessons l WHERE l.module_id = m.id)
  )
  AND NOT EXISTS (
    SELECT 1 FROM modules m JOIN lessons l ON l.module_id = m.id
    WHERE m.course_id = c.id
      AND (l.content_markdown IS NULL OR btrim(l.content_markdown) = '')
      AND NOT EXISTS (SELECT 1 FROM lesson_exercises e WHERE e.lesson_id = l.id)
      AND NOT EXISTS (SELECT 1 FROM lesson_quizzes q WHERE q.lesson_id = l.id)
  ) AS content_complete
FROM courses c;

-- +goose Down
DROP VIEW course_content_states;
DROP INDEX generation_jobs_request_attempt;
ALTER TABLE generation_jobs DROP COLUMN generation_attempt;
ALTER TABLE generation_requests DROP COLUMN generation_attempt;
