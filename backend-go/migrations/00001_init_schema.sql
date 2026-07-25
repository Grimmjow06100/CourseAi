-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE TYPE "course_generation_status" AS ENUM (
  'analysis_pending',
  'needs_clarification',
  'analysis_completed',
  'architecture_generating',
  'structure_generated',
  'lessons_generating',
  'lessons_generated',
  'content_generating',
  'completed',
  'failed'
);

CREATE TYPE "generation_pipeline_status" AS ENUM (
  'queued',
  'running',
  'completed',
  'failed'
);

CREATE TYPE "course_language" AS ENUM ('fr', 'en');

CREATE TYPE "lesson_type" AS ENUM (
  'theory',
  'practice',
  'mixed',
  'quiz'
);

CREATE TYPE "level" AS ENUM (
  'beginner',
  'intermediate',
  'advanced',
  'expert',
  'unknown'
);

CREATE TABLE "generation_requests" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "initial_user_prompt" TEXT NOT NULL,
  "pipeline_status" "generation_pipeline_status" NOT NULL DEFAULT 'queued',
  "current_step" TEXT,
  "progress_percent" INTEGER NOT NULL DEFAULT 0,
  "failure_message" TEXT,
  "started_at" TIMESTAMP(3),
  "completed_at" TIMESTAMP(3),
  "is_out_of_scope" BOOLEAN NOT NULL DEFAULT false,
  "error_message" TEXT,
  "warning_message" TEXT,
  "suggested_title" TEXT,
  "short_synopsis" TEXT,
  "detected_current_level" "level",
  "detected_target_level" "level",
  "detected_goal" TEXT,
  "detected_language" "course_language",
  "clarification_questions" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "raw_analysis_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "generation_requests_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "users" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "username" TEXT NOT NULL,
  "password" TEXT NOT NULL,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "courses" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "request_id" UUID NOT NULL,
  "language" "course_language" NOT NULL,
  "status" "course_generation_status" NOT NULL DEFAULT 'analysis_completed',
  "initial_user_prompt" TEXT NOT NULL,
  "title" TEXT NOT NULL,
  "synopsis" TEXT NOT NULL,
  "target_audience" TEXT,
  "current_level" "level" NOT NULL,
  "target_level" "level" NOT NULL,
  "prerequisites" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "goals" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "acquired_skills" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "final_project_title" TEXT,
  "final_project_description" TEXT,
  "final_project_constraints" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "generation_payload" JSONB,
  "raw_architecture_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "courses_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "courses_request_id_fkey"
    FOREIGN KEY ("request_id")
    REFERENCES "generation_requests"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE TABLE "modules" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "course_id" UUID NOT NULL,
  "module_order" INTEGER NOT NULL,
  "title" TEXT NOT NULL,
  "description" TEXT NOT NULL,
  "key_learning_points" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "raw_module_output" JSONB,
  "raw_lessons_plan_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "modules_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "modules_course_id_fkey"
    FOREIGN KEY ("course_id")
    REFERENCES "courses"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE TABLE "lessons" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "module_id" UUID NOT NULL,
  "lesson_order" INTEGER NOT NULL,
  "title" TEXT NOT NULL,
  "type" "lesson_type" NOT NULL,
  "estimated_duration_minutes" INTEGER NOT NULL,
  "learning_goal" TEXT NOT NULL,
  "requires_diagram" BOOLEAN NOT NULL DEFAULT false,
  "technical_keywords" JSONB NOT NULL DEFAULT '[]'::jsonb,
  "content_markdown" TEXT,
  "raw_content_output" JSONB,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "lessons_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "lessons_module_id_fkey"
    FOREIGN KEY ("module_id")
    REFERENCES "modules"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE UNIQUE INDEX "users_username_key" ON "users"("username");
CREATE UNIQUE INDEX "courses_request_id_key" ON "courses"("request_id");
CREATE INDEX "generation_requests_pipeline_status_idx" ON "generation_requests"("pipeline_status");
CREATE INDEX "courses_status_idx" ON "courses"("status");
CREATE INDEX "courses_language_idx" ON "courses"("language");
CREATE INDEX "modules_course_id_idx" ON "modules"("course_id");
CREATE UNIQUE INDEX "modules_course_order_unique" ON "modules"("course_id", "module_order");
CREATE INDEX "lessons_module_id_idx" ON "lessons"("module_id");
CREATE UNIQUE INDEX "lessons_module_order_unique" ON "lessons"("module_id", "lesson_order");

-- +goose Down
DROP TABLE IF EXISTS "lessons";
DROP TABLE IF EXISTS "modules";
DROP TABLE IF EXISTS "courses";
DROP TABLE IF EXISTS "users";
DROP TABLE IF EXISTS "generation_requests";

DROP TYPE IF EXISTS "level";
DROP TYPE IF EXISTS "lesson_type";
DROP TYPE IF EXISTS "course_language";
DROP TYPE IF EXISTS "generation_pipeline_status";
DROP TYPE IF EXISTS "course_generation_status";


