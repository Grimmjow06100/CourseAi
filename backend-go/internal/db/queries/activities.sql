-- name: CreateLessonExercise :one
INSERT INTO lesson_exercises (
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  instructions_markdown,
  content_markdown,
  correction_markdown,
  payload,
  raw_ai_output,
  created_at,
  updated_at
)
VALUES (
  @id,
  @lesson_id,
  @type::exercise_type,
  @difficulty::activity_difficulty,
  @title,
  @objective,
  @instructions_markdown,
  @content_markdown,
  @correction_markdown,
  @payload::jsonb,
  sqlc.narg('raw_ai_output')::jsonb,
  @created_at,
  @updated_at
)
RETURNING
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  instructions_markdown,
  content_markdown,
  correction_markdown,
  payload,
  raw_ai_output,
  created_at,
  updated_at;

-- name: CreateLessonExercises :copyfrom
INSERT INTO lesson_exercises (
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  instructions_markdown,
  content_markdown,
  correction_markdown,
  payload,
  raw_ai_output,
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

-- name: ListLessonExercisesByLessonID :many
SELECT
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  instructions_markdown,
  content_markdown,
  correction_markdown,
  payload,
  raw_ai_output,
  created_at,
  updated_at
FROM lesson_exercises
WHERE lesson_id = @lesson_id
ORDER BY created_at ASC, id ASC;

-- name: ListLessonExercisesByLessonIDs :many
SELECT
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  instructions_markdown,
  content_markdown,
  correction_markdown,
  payload,
  raw_ai_output,
  created_at,
  updated_at
FROM lesson_exercises
WHERE lesson_id = ANY(@lesson_ids::uuid[])
ORDER BY lesson_id ASC, created_at ASC, id ASC;

-- name: DeleteLessonExercisesByLessonID :exec
DELETE FROM lesson_exercises
WHERE lesson_id = @lesson_id;

-- name: CreateLessonQuiz :one
INSERT INTO lesson_quizzes (
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  questions,
  raw_ai_output,
  created_at,
  updated_at
)
VALUES (
  @id,
  @lesson_id,
  @type::quiz_type,
  @difficulty::activity_difficulty,
  @title,
  @objective,
  @questions::jsonb,
  sqlc.narg('raw_ai_output')::jsonb,
  @created_at,
  @updated_at
)
RETURNING
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  questions,
  raw_ai_output,
  created_at,
  updated_at;

-- name: CreateLessonQuizzes :copyfrom
INSERT INTO lesson_quizzes (
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  questions,
  raw_ai_output,
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
  $10
);

-- name: ListLessonQuizzesByLessonID :many
SELECT
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  questions,
  raw_ai_output,
  created_at,
  updated_at
FROM lesson_quizzes
WHERE lesson_id = @lesson_id
ORDER BY created_at ASC, id ASC;

-- name: ListLessonQuizzesByLessonIDs :many
SELECT
  id,
  lesson_id,
  type,
  difficulty,
  title,
  objective,
  questions,
  raw_ai_output,
  created_at,
  updated_at
FROM lesson_quizzes
WHERE lesson_id = ANY(@lesson_ids::uuid[])
ORDER BY lesson_id ASC, created_at ASC, id ASC;

-- name: DeleteLessonQuizzesByLessonID :exec
DELETE FROM lesson_quizzes
WHERE lesson_id = @lesson_id;
