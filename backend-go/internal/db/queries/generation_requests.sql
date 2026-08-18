-- name: CreateGenerationRequest :one
INSERT INTO generation_requests (
  id,
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
  @id,
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
WHERE gr.id = @id;

-- name: GetGenerationRequestByCourseID :one
SELECT gr.*
FROM generation_requests gr
JOIN courses c ON c.request_id = gr.id
WHERE c.id = @course_id;
