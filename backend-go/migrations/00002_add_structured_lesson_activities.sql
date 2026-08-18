-- +goose Up
CREATE TYPE "activity_difficulty" AS ENUM (
  'beginner',
  'intermediate',
  'advanced'
);

CREATE TYPE "exercise_type" AS ENUM (
  'guided_lab',
  'coding',
  'debugging',
  'configuration',
  'scenario',
  'written_answer',
  'command_line',
  'mixed'
);

CREATE TYPE "quiz_type" AS ENUM (
  'single_choice',
  'multiple_choice',
  'true_false',
  'short_answer',
  'mixed'
);

CREATE TABLE "lesson_exercises" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "lesson_id" UUID NOT NULL,
  "type" "exercise_type" NOT NULL,
  "difficulty" "activity_difficulty" NOT NULL,
  "title" TEXT NOT NULL,
  "objective" TEXT NOT NULL,
  "instructions_markdown" TEXT NOT NULL,
  "content_markdown" TEXT NOT NULL,
  "correction_markdown" TEXT NOT NULL,
  "payload" JSONB NOT NULL DEFAULT '{}'::jsonb,
  "raw_ai_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "lesson_exercises_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "lesson_exercises_lesson_id_fkey"
    FOREIGN KEY ("lesson_id")
    REFERENCES "lessons"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT "lesson_exercises_payload_is_object_check"
    CHECK (jsonb_typeof("payload") = 'object')
);

CREATE TABLE "lesson_quizzes" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "lesson_id" UUID NOT NULL,
  "type" "quiz_type" NOT NULL,
  "difficulty" "activity_difficulty" NOT NULL,
  "title" TEXT NOT NULL,
  "objective" TEXT NOT NULL,
  "questions" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "raw_ai_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "lesson_quizzes_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "lesson_quizzes_lesson_id_fkey"
    FOREIGN KEY ("lesson_id")
    REFERENCES "lessons"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT "lesson_quizzes_questions_is_array_check"
    CHECK (jsonb_typeof("questions") = 'array')
);

CREATE INDEX "lesson_exercises_lesson_id_idx" ON "lesson_exercises"("lesson_id");
CREATE INDEX "lesson_exercises_type_idx" ON "lesson_exercises"("type");
CREATE INDEX "lesson_exercises_difficulty_idx" ON "lesson_exercises"("difficulty");

CREATE INDEX "lesson_quizzes_lesson_id_idx" ON "lesson_quizzes"("lesson_id");
CREATE INDEX "lesson_quizzes_type_idx" ON "lesson_quizzes"("type");
CREATE INDEX "lesson_quizzes_difficulty_idx" ON "lesson_quizzes"("difficulty");

-- +goose Down
DROP TABLE IF EXISTS "lesson_quizzes";
DROP TABLE IF EXISTS "lesson_exercises";

DROP TYPE IF EXISTS "quiz_type";
DROP TYPE IF EXISTS "exercise_type";
DROP TYPE IF EXISTS "activity_difficulty";
