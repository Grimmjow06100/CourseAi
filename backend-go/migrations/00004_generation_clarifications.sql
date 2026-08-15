-- +goose Up
ALTER TYPE generation_pipeline_status
ADD VALUE IF NOT EXISTS 'awaiting_clarification';

ALTER TABLE generation_requests
ADD COLUMN analysis_completed_at TIMESTAMP(3),
ADD COLUMN clarification_answers JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN confirmed_title TEXT,
ADD COLUMN confirmed_synopsis TEXT,
ADD COLUMN confirmed_current_level level,
ADD COLUMN confirmed_target_level level,
ADD COLUMN confirmed_goals JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN confirmed_language course_language,
ADD COLUMN brief_confirmed_at TIMESTAMP(3),
ADD COLUMN clarifications_submitted_at TIMESTAMP(3),
ADD COLUMN clarification_version INTEGER NOT NULL DEFAULT 0,
ADD CONSTRAINT generation_requests_questions_array_check
  CHECK (jsonb_typeof(clarification_questions) = 'array'),
ADD CONSTRAINT generation_requests_answers_array_check
  CHECK (jsonb_typeof(clarification_answers) = 'array'),
ADD CONSTRAINT generation_requests_confirmed_goals_array_check
  CHECK (jsonb_typeof(confirmed_goals) = 'array'),
ADD CONSTRAINT generation_requests_clarification_version_check
  CHECK (clarification_version >= 0),
ADD CONSTRAINT generation_requests_confirmed_brief_check
  CHECK (
    (
      brief_confirmed_at IS NULL
      AND clarification_version = 0
      AND confirmed_title IS NULL
      AND confirmed_synopsis IS NULL
      AND confirmed_current_level IS NULL
      AND confirmed_target_level IS NULL
      AND confirmed_goals = '[]'::jsonb
      AND confirmed_language IS NULL
      AND clarifications_submitted_at IS NULL
    )
    OR
    (
      brief_confirmed_at IS NOT NULL
      AND clarification_version > 0
      AND confirmed_title IS NOT NULL
      AND btrim(confirmed_title) <> ''
      AND confirmed_synopsis IS NOT NULL
      AND btrim(confirmed_synopsis) <> ''
      AND confirmed_current_level IS NOT NULL
      AND confirmed_target_level IS NOT NULL
      AND jsonb_array_length(confirmed_goals) > 0
      AND confirmed_language IS NOT NULL
    )
  );

-- +goose Down
ALTER TABLE generation_requests
DROP CONSTRAINT IF EXISTS generation_requests_confirmed_brief_check,
DROP CONSTRAINT IF EXISTS generation_requests_clarification_version_check,
DROP CONSTRAINT IF EXISTS generation_requests_confirmed_goals_array_check,
DROP CONSTRAINT IF EXISTS generation_requests_answers_array_check,
DROP CONSTRAINT IF EXISTS generation_requests_questions_array_check,
DROP COLUMN IF EXISTS clarification_version,
DROP COLUMN IF EXISTS clarifications_submitted_at,
DROP COLUMN IF EXISTS brief_confirmed_at,
DROP COLUMN IF EXISTS confirmed_language,
DROP COLUMN IF EXISTS confirmed_goals,
DROP COLUMN IF EXISTS confirmed_target_level,
DROP COLUMN IF EXISTS confirmed_current_level,
DROP COLUMN IF EXISTS confirmed_synopsis,
DROP COLUMN IF EXISTS confirmed_title,
DROP COLUMN IF EXISTS clarification_answers,
DROP COLUMN IF EXISTS analysis_completed_at;
