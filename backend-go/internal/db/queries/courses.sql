-- name: CreateCourse :one
INSERT INTO courses (
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
)
VALUES (
  @id,
  @request_id,
  @language::course_language,
  @status::course_generation_status,
  @initial_user_prompt,
  @title,
  @synopsis,
  sqlc.narg('target_audience'),
  @current_level::level,
  @target_level::level,
  @prerequisites::jsonb,
  @goals::jsonb,
  @acquired_skills::jsonb,
  sqlc.narg('final_project_title'),
  sqlc.narg('final_project_description'),
  @final_project_constraints::jsonb,
  sqlc.narg('raw_architecture_output')::jsonb,
  @created_at,
  @updated_at,
  @clerk_user_id
)
RETURNING
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id;

-- name: UpdateCourse :one
UPDATE courses
SET
  request_id = @request_id,
  clerk_user_id = @clerk_user_id,
  language = @language::course_language,
  status = @status::course_generation_status,
  initial_user_prompt = @initial_user_prompt,
  title = @title,
  synopsis = @synopsis,
  target_audience = sqlc.narg('target_audience'),
  current_level = @current_level::level,
  target_level = @target_level::level,
  prerequisites = @prerequisites::jsonb,
  goals = @goals::jsonb,
  acquired_skills = @acquired_skills::jsonb,
  final_project_title = sqlc.narg('final_project_title'),
  final_project_description = sqlc.narg('final_project_description'),
  final_project_constraints = @final_project_constraints::jsonb,
  raw_architecture_output = sqlc.narg('raw_architecture_output')::jsonb,
  updated_at = @updated_at
WHERE id = @id
RETURNING
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id;

-- name: GetCourseByID :one
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE id = @id;

-- name: GetCourseByRequestID :one
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE request_id = @request_id;

-- name: IsCourseContentComplete :one
SELECT
  EXISTS (
    SELECT 1
    FROM modules m
    WHERE m.course_id = @course_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM modules m
    WHERE m.course_id = @course_id
      AND NOT EXISTS (
        SELECT 1
        FROM lessons l
        WHERE l.module_id = m.id
      )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM modules m
    JOIN lessons l ON l.module_id = m.id
    WHERE m.course_id = @course_id
      AND (l.content_markdown IS NULL OR btrim(l.content_markdown) = '')
      AND NOT EXISTS (
        SELECT 1
        FROM lesson_exercises e
        WHERE e.lesson_id = l.id
      )
      AND NOT EXISTS (
        SELECT 1
        FROM lesson_quizzes q
        WHERE q.lesson_id = l.id
      )
  ) AS is_complete;

-- name: CountCourses :one
SELECT count(*)::bigint
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  );

-- name: ListCoursesCreatedAtDesc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY created_at DESC, id DESC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesCreatedAtAsc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY created_at ASC, id ASC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesUpdatedAtDesc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY updated_at DESC, id DESC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesUpdatedAtAsc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY updated_at ASC, id ASC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesTitleDesc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY title DESC, id DESC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesTitleAsc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY title ASC, id ASC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesStatusDesc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY status DESC, id DESC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: ListCoursesStatusAsc :many
SELECT
  id,
  request_id,
  language,
  status,
  initial_user_prompt,
  title,
  synopsis,
  target_audience,
  current_level,
  target_level,
  prerequisites,
  goals,
  acquired_skills,
  final_project_title,
  final_project_description,
  final_project_constraints,
  generation_payload,
  raw_architecture_output,
  created_at,
  updated_at,
  clerk_user_id
FROM courses
WHERE clerk_user_id = @clerk_user_id
  AND (sqlc.narg('status')::course_generation_status IS NULL OR status = sqlc.narg('status')::course_generation_status)
  AND (sqlc.narg('language')::course_language IS NULL OR language = sqlc.narg('language')::course_language)
  AND (
    sqlc.narg('search')::text IS NULL
    OR btrim(sqlc.narg('search')::text) = ''
    OR title ILIKE '%' || sqlc.narg('search')::text || '%'
    OR synopsis ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY status ASC, id ASC
LIMIT @limit_rows OFFSET @offset_rows;

-- name: DeleteCourseByID :execrows
DELETE FROM courses
WHERE id = @id;

-- name: DeleteCourseByRequestID :execrows
DELETE FROM courses
WHERE request_id = @request_id;

-- name: DeleteCourseGenerationByCourseID :execrows
DELETE FROM generation_requests
WHERE id = (
  SELECT request_id
  FROM courses
  WHERE courses.id = @course_id
);
