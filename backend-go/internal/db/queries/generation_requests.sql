-- name: CreateGenerationRequest :one
INSERT INTO generation_requests (
  generation_attempt,
  id,
  clerk_user_id,
  initial_user_prompt,
  pipeline_status,
  current_step,
  progress_percent,
  failure_message,
  started_at,
  completed_at,
  is_out_of_scope,
  error_message,
  warning_message,
  suggested_title,
  short_synopsis,
  detected_current_level,
  detected_target_level,
  detected_goal,
  detected_language,
  clarification_questions,
  raw_analysis_output,
  analysis_completed_at,
  clarification_answers,
  confirmed_title,
  confirmed_synopsis,
  confirmed_current_level,
  confirmed_target_level,
  confirmed_goals,
  confirmed_language,
  brief_confirmed_at,
  clarifications_submitted_at,
  clarification_version,
  created_at,
  updated_at
)
VALUES (
  @generation_attempt,
  @id,
  @clerk_user_id,
  @initial_user_prompt,
  @pipeline_status::generation_pipeline_status,
  sqlc.narg('current_step'),
  @progress_percent,
  sqlc.narg('failure_message'),
  sqlc.narg('started_at'),
  sqlc.narg('completed_at'),
  @is_out_of_scope,
  sqlc.narg('error_message'),
  sqlc.narg('warning_message'),
  sqlc.narg('suggested_title'),
  sqlc.narg('short_synopsis'),
  sqlc.narg('detected_current_level')::level,
  sqlc.narg('detected_target_level')::level,
  sqlc.narg('detected_goal'),
  sqlc.narg('detected_language')::course_language,
  @clarification_questions::jsonb,
  sqlc.narg('raw_analysis_output')::jsonb,
  sqlc.narg('analysis_completed_at'),
  @clarification_answers::jsonb,
  sqlc.narg('confirmed_title'),
  sqlc.narg('confirmed_synopsis'),
  sqlc.narg('confirmed_current_level')::level,
  sqlc.narg('confirmed_target_level')::level,
  @confirmed_goals::jsonb,
  sqlc.narg('confirmed_language')::course_language,
  sqlc.narg('brief_confirmed_at'),
  sqlc.narg('clarifications_submitted_at'),
  @clarification_version,
  @created_at,
  @updated_at
)
RETURNING *;

-- name: UpdateGenerationRequest :one
UPDATE generation_requests
SET
  generation_attempt = @generation_attempt,
  clerk_user_id = @clerk_user_id,
  initial_user_prompt = @initial_user_prompt,
  pipeline_status = @pipeline_status::generation_pipeline_status,
  current_step = sqlc.narg('current_step'),
  progress_percent = @progress_percent,
  failure_message = sqlc.narg('failure_message'),
  started_at = sqlc.narg('started_at'),
  completed_at = sqlc.narg('completed_at'),
  is_out_of_scope = @is_out_of_scope,
  error_message = sqlc.narg('error_message'),
  warning_message = sqlc.narg('warning_message'),
  suggested_title = sqlc.narg('suggested_title'),
  short_synopsis = sqlc.narg('short_synopsis'),
  detected_current_level = sqlc.narg('detected_current_level')::level,
  detected_target_level = sqlc.narg('detected_target_level')::level,
  detected_goal = sqlc.narg('detected_goal'),
  detected_language = sqlc.narg('detected_language')::course_language,
  clarification_questions = @clarification_questions::jsonb,
  raw_analysis_output = sqlc.narg('raw_analysis_output')::jsonb,
  analysis_completed_at = sqlc.narg('analysis_completed_at'),
  clarification_answers = @clarification_answers::jsonb,
  confirmed_title = sqlc.narg('confirmed_title'),
  confirmed_synopsis = sqlc.narg('confirmed_synopsis'),
  confirmed_current_level = sqlc.narg('confirmed_current_level')::level,
  confirmed_target_level = sqlc.narg('confirmed_target_level')::level,
  confirmed_goals = @confirmed_goals::jsonb,
  confirmed_language = sqlc.narg('confirmed_language')::course_language,
  brief_confirmed_at = sqlc.narg('brief_confirmed_at'),
  clarifications_submitted_at = sqlc.narg('clarifications_submitted_at'),
  clarification_version = @clarification_version,
  updated_at = @updated_at
WHERE id = @id
RETURNING *;

-- name: GetGenerationRequestByID :one
SELECT *
FROM generation_requests
WHERE id = @id;

-- name: GetGenerationRequestForUpdate :one
SELECT *
FROM generation_requests
WHERE id = @id
FOR UPDATE;

-- name: GetGenerationStatusByID :one
SELECT
  gr.id AS request_id,
  gr.generation_attempt,
  COALESCE(cs.content_complete, false)::boolean AS content_complete,
  gr.pipeline_status,
  gr.current_step,
  gr.progress_percent,
  gr.failure_message,
  gr.is_out_of_scope,
	gr.error_message,
	gr.warning_message,
	gr.suggested_title,
	gr.short_synopsis,
	gr.detected_current_level,
	gr.detected_target_level,
	gr.detected_goal,
	gr.detected_language,
  gr.clarification_questions,
  c.id AS course_id,
  c.status AS course_status
FROM generation_requests gr
LEFT JOIN courses c ON c.request_id = gr.id
LEFT JOIN course_content_states cs ON cs.course_id = c.id
WHERE gr.id = @id;

-- name: CountGenerationRequestsByOwner :one
SELECT count(*)::bigint
FROM generation_requests AS request
WHERE request.clerk_user_id = @clerk_user_id
  AND (
    sqlc.narg('pipeline_status')::generation_pipeline_status IS NULL
    OR request.pipeline_status = sqlc.narg('pipeline_status')::generation_pipeline_status
  );

-- name: ListGenerationRequestsByOwner :many
SELECT
  request.id AS request_id,
  request.generation_attempt,
  COALESCE(cs.content_complete, false)::boolean AS content_complete,
  course.id AS course_id,
  request.initial_user_prompt,
  COALESCE(course.title, request.confirmed_title, request.suggested_title, request.initial_user_prompt) AS title,
  request.pipeline_status,
  course.status AS course_status,
  request.current_step,
  request.progress_percent,
  request.is_out_of_scope,
  request.failure_message,
  request.created_at,
  request.updated_at
FROM generation_requests AS request
LEFT JOIN courses AS course ON course.request_id = request.id
LEFT JOIN course_content_states cs ON cs.course_id = course.id
WHERE request.clerk_user_id = @clerk_user_id
  AND (
    sqlc.narg('pipeline_status')::generation_pipeline_status IS NULL
    OR request.pipeline_status = sqlc.narg('pipeline_status')::generation_pipeline_status
  )
ORDER BY request.created_at DESC, request.id DESC
LIMIT @limit_rows
OFFSET @offset_rows;

-- name: GetGenerationRequestByCourseID :one
SELECT gr.*
FROM generation_requests gr
JOIN courses c ON c.request_id = gr.id
WHERE c.id = @course_id;

-- name: GetGenerationAdmissionUsage :one
WITH admission_lock AS (
  SELECT pg_advisory_xact_lock(4931529157321281::bigint)
)
SELECT
  (
    SELECT count(*)::bigint
    FROM generation_requests AS active_request
    WHERE active_request.clerk_user_id = @clerk_user_id
      AND active_request.pipeline_status IN ('queued', 'running', 'awaiting_clarification')
  ) AS active_requests,
  (
    SELECT count(*)::bigint
    FROM generation_requests AS daily_request
    WHERE daily_request.clerk_user_id = @clerk_user_id
      AND daily_request.created_at >= @created_since
  ) AS daily_requests,
  (
    SELECT count(*)::bigint
    FROM generation_jobs AS pending_job
    WHERE pending_job.status IN ('queued', 'retry_scheduled', 'running')
  ) AS pending_jobs
FROM admission_lock;

-- name: DeleteGenerationRequestByID :execrows
DELETE FROM generation_requests
WHERE id = @id;

-- name: ListGenerationCompletionCandidates :many
SELECT gr.id FROM generation_requests gr
JOIN courses c ON c.request_id = gr.id
JOIN course_content_states cs ON cs.course_id = c.id
WHERE cs.content_complete
  AND gr.pipeline_status <> 'awaiting_clarification'
  AND NOT gr.is_out_of_scope
  AND (gr.pipeline_status <> 'completed' OR c.status <> 'completed')
  AND NOT EXISTS (
    SELECT 1 FROM generation_jobs j
    WHERE j.request_id = gr.id AND j.generation_attempt = gr.generation_attempt
      AND j.status IN ('queued', 'running', 'retry_scheduled')
  )
ORDER BY gr.updated_at, gr.id
LIMIT @limit_rows;

-- name: PurgeRawGenerationOutputsBefore :one
WITH eligible_requests AS MATERIALIZED (
  SELECT retained_request.id
  FROM generation_requests AS retained_request
  WHERE retained_request.pipeline_status IN ('completed', 'failed')
    AND retained_request.updated_at < @cutoff
    AND (
      retained_request.raw_analysis_output IS NOT NULL
      OR EXISTS (
        SELECT 1
        FROM courses AS retained_course
        WHERE retained_course.request_id = retained_request.id
          AND retained_course.raw_architecture_output IS NOT NULL
      )
      OR EXISTS (
        SELECT 1
        FROM modules AS retained_module
        JOIN courses AS retained_course ON retained_course.id = retained_module.course_id
        WHERE retained_course.request_id = retained_request.id
          AND (retained_module.raw_module_output IS NOT NULL OR retained_module.raw_lessons_plan_output IS NOT NULL)
      )
      OR EXISTS (
        SELECT 1
        FROM lessons AS retained_lesson
        JOIN modules AS retained_module ON retained_module.id = retained_lesson.module_id
        JOIN courses AS retained_course ON retained_course.id = retained_module.course_id
        WHERE retained_course.request_id = retained_request.id
          AND retained_lesson.raw_content_output IS NOT NULL
      )
      OR EXISTS (
        SELECT 1
        FROM lesson_exercises AS retained_exercise
        JOIN lessons AS retained_lesson ON retained_lesson.id = retained_exercise.lesson_id
        JOIN modules AS retained_module ON retained_module.id = retained_lesson.module_id
        JOIN courses AS retained_course ON retained_course.id = retained_module.course_id
        WHERE retained_course.request_id = retained_request.id
          AND retained_exercise.raw_ai_output IS NOT NULL
      )
      OR EXISTS (
        SELECT 1
        FROM lesson_quizzes AS retained_quiz
        JOIN lessons AS retained_lesson ON retained_lesson.id = retained_quiz.lesson_id
        JOIN modules AS retained_module ON retained_module.id = retained_lesson.module_id
        JOIN courses AS retained_course ON retained_course.id = retained_module.course_id
        WHERE retained_course.request_id = retained_request.id
          AND retained_quiz.raw_ai_output IS NOT NULL
      )
    )
  ORDER BY retained_request.updated_at ASC, retained_request.id ASC
  LIMIT @limit_rows
),
purged_requests AS (
  UPDATE generation_requests
  SET raw_analysis_output = NULL
  WHERE id IN (SELECT id FROM eligible_requests)
),
purged_courses AS (
  UPDATE courses
  SET raw_architecture_output = NULL
  WHERE request_id IN (SELECT id FROM eligible_requests)
),
purged_modules AS (
  UPDATE modules
  SET raw_module_output = NULL,
      raw_lessons_plan_output = NULL
  WHERE course_id IN (SELECT id FROM courses WHERE request_id IN (SELECT id FROM eligible_requests))
),
purged_lessons AS (
  UPDATE lessons
  SET raw_content_output = NULL
  WHERE module_id IN (
    SELECT modules.id
    FROM modules
    JOIN courses ON courses.id = modules.course_id
    WHERE courses.request_id IN (SELECT id FROM eligible_requests)
  )
),
purged_exercises AS (
  UPDATE lesson_exercises
  SET raw_ai_output = NULL
  WHERE lesson_id IN (
    SELECT lessons.id
    FROM lessons
    JOIN modules ON modules.id = lessons.module_id
    JOIN courses ON courses.id = modules.course_id
    WHERE courses.request_id IN (SELECT id FROM eligible_requests)
  )
),
purged_quizzes AS (
  UPDATE lesson_quizzes
  SET raw_ai_output = NULL
  WHERE lesson_id IN (
    SELECT lessons.id
    FROM lessons
    JOIN modules ON modules.id = lessons.module_id
    JOIN courses ON courses.id = modules.course_id
    WHERE courses.request_id IN (SELECT id FROM eligible_requests)
  )
)
SELECT count(*)::bigint AS purged_requests
FROM eligible_requests;
