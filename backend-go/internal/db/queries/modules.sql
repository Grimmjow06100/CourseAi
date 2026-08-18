-- name: CreateModule :one
INSERT INTO modules (
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_lessons_plan_output,
  created_at,
  updated_at
)
VALUES (
  @id,
  @course_id,
  @module_order,
  @title,
  @description,
  @key_learning_points::jsonb,
  sqlc.narg('raw_lessons_plan_output')::jsonb,
  @created_at,
  @updated_at
)
RETURNING
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_module_output,
  raw_lessons_plan_output,
  created_at,
  updated_at;

-- name: CreateModules :copyfrom
INSERT INTO modules (
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_lessons_plan_output,
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
  $9
);

-- name: UpdateModule :one
UPDATE modules
SET
  course_id = @course_id,
  module_order = @module_order,
  title = @title,
  description = @description,
  key_learning_points = @key_learning_points::jsonb,
  raw_lessons_plan_output = sqlc.narg('raw_lessons_plan_output')::jsonb,
  updated_at = @updated_at
WHERE id = @id
RETURNING
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_module_output,
  raw_lessons_plan_output,
  created_at,
  updated_at;

-- name: GetModuleByID :one
SELECT
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_module_output,
  raw_lessons_plan_output,
  created_at,
  updated_at
FROM modules
WHERE id = @id;

-- name: ListModulesByCourseID :many
SELECT
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_module_output,
  raw_lessons_plan_output,
  created_at,
  updated_at
FROM modules
WHERE course_id = @course_id
ORDER BY module_order ASC, id ASC;

-- name: ListModulesByCourseIDs :many
SELECT
  id,
  course_id,
  module_order,
  title,
  description,
  key_learning_points,
  raw_module_output,
  raw_lessons_plan_output,
  created_at,
  updated_at
FROM modules
WHERE course_id = ANY(@course_ids::uuid[])
ORDER BY course_id ASC, module_order ASC, id ASC;

-- name: DeleteModuleByID :execrows
DELETE FROM modules
WHERE id = @id;
