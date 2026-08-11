-- name: CreateLesson :one
INSERT INTO lessons (
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at
)
VALUES (
  @id,
  @module_id,
  @lesson_order,
  @title,
  @type::lesson_type,
  @estimated_duration_minutes,
  @learning_goal,
  @requires_diagram,
  @technical_keywords::jsonb,
  sqlc.narg('content_markdown'),
  sqlc.narg('raw_content_output')::jsonb,
  @created_at,
  @updated_at
)
RETURNING
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at;

-- name: CreateLessons :copyfrom
INSERT INTO lessons (
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at
)
VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13
);

-- name: UpdateLesson :one
UPDATE lessons AS l
SET
  module_id = @module_id,
  lesson_order = @lesson_order,
  title = @title,
  type = @type::lesson_type,
  estimated_duration_minutes = @estimated_duration_minutes,
  learning_goal = @learning_goal,
  requires_diagram = @requires_diagram,
  technical_keywords = @technical_keywords::jsonb,
  content_markdown = sqlc.narg('content_markdown'),
  raw_content_output = sqlc.narg('raw_content_output')::jsonb,
  updated_at = @updated_at
WHERE l.id = @id
RETURNING
  l.id,
  l.module_id,
  l.lesson_order,
  l.title,
  l.type,
  l.estimated_duration_minutes,
  l.learning_goal,
  l.requires_diagram,
  l.technical_keywords,
  l.content_markdown,
  l.raw_content_output,
  l.created_at,
  l.updated_at;

-- name: ReplaceLessonContent :one
WITH deleted_exercises AS (
  DELETE FROM lesson_exercises
  WHERE lesson_id = @id
), deleted_quizzes AS (
  DELETE FROM lesson_quizzes
  WHERE lesson_id = @id
)
UPDATE lessons AS l
SET
  content_markdown = sqlc.narg('content_markdown'),
  raw_content_output = sqlc.narg('raw_content_output')::jsonb,
  updated_at = @updated_at
WHERE l.id = @id
RETURNING
  l.id,
  l.module_id,
  l.lesson_order,
  l.title,
  l.type,
  l.estimated_duration_minutes,
  l.learning_goal,
  l.requires_diagram,
  l.technical_keywords,
  l.content_markdown,
  l.raw_content_output,
  l.created_at,
  l.updated_at;

-- name: GetLessonByID :one
SELECT
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at
FROM lessons
WHERE id = @id;

-- name: ListLessonsByModuleID :many
SELECT
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at
FROM lessons
WHERE module_id = @module_id
ORDER BY lesson_order ASC, id ASC;

-- name: ListLessonsByModuleIDs :many
SELECT
  id,
  module_id,
  lesson_order,
  title,
  type,
  estimated_duration_minutes,
  learning_goal,
  requires_diagram,
  technical_keywords,
  content_markdown,
  raw_content_output,
  created_at,
  updated_at
FROM lessons
WHERE module_id = ANY(@module_ids::uuid[])
ORDER BY module_id ASC, lesson_order ASC, id ASC;

-- name: DeleteLessonByID :execrows
DELETE FROM lessons
WHERE id = @id;
