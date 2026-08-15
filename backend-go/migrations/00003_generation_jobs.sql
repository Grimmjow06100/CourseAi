-- +goose Up
CREATE TYPE "generation_job_status" AS ENUM (
  'queued',
  'running',
  'retry_scheduled',
  'completed',
  'failed',
  'cancelled'
);

CREATE TYPE "generation_job_kind" AS ENUM (
  'analysis',
  'architecture',
  'lesson_plan',
  'lesson_content',
  'module_content',
  'finalize_course'
);

CREATE TABLE "generation_jobs" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "request_id" UUID NOT NULL,
  "parent_job_id" UUID,
  "kind" "generation_job_kind" NOT NULL,
  "status" "generation_job_status" NOT NULL DEFAULT 'queued',
  "target_id" UUID,
  "idempotency_key" TEXT NOT NULL,
  "payload" JSONB NOT NULL DEFAULT '{}'::jsonb,
  "priority" INTEGER NOT NULL DEFAULT 0,
  "attempt_count" INTEGER NOT NULL DEFAULT 0,
  "max_attempts" INTEGER NOT NULL DEFAULT 3,
  "available_at" TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "locked_by" TEXT,
  "locked_until" TIMESTAMPTZ(3),
  "started_at" TIMESTAMPTZ(3),
  "completed_at" TIMESTAMPTZ(3),
  "last_error_code" TEXT,
  "last_error_message" TEXT,
  "created_at" TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "generation_jobs_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "generation_jobs_request_fkey"
    FOREIGN KEY ("request_id")
    REFERENCES "generation_requests"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT "generation_jobs_parent_fkey"
    FOREIGN KEY ("parent_job_id")
    REFERENCES "generation_jobs"("id")
    ON DELETE CASCADE
    ON UPDATE CASCADE,
  CONSTRAINT "generation_jobs_attempts_check"
    CHECK ("attempt_count" >= 0 AND "max_attempts" > 0 AND "attempt_count" <= "max_attempts"),
  CONSTRAINT "generation_jobs_payload_is_object_check"
    CHECK (jsonb_typeof("payload") = 'object'),
  CONSTRAINT "generation_jobs_parent_is_not_self_check"
    CHECK ("parent_job_id" IS NULL OR "parent_job_id" <> "id"),
  CONSTRAINT "generation_jobs_idempotency_key_not_blank_check"
    CHECK (btrim("idempotency_key") <> ''),
  CONSTRAINT "generation_jobs_target_check"
    CHECK (
      "kind" NOT IN ('lesson_plan', 'lesson_content', 'module_content', 'finalize_course')
      OR "target_id" IS NOT NULL
    ),
  CONSTRAINT "generation_jobs_lock_state_check"
    CHECK (
      (
        "status" = 'running'
        AND "attempt_count" > 0
        AND "locked_by" IS NOT NULL
        AND btrim("locked_by") <> ''
        AND "locked_until" IS NOT NULL
        AND "started_at" IS NOT NULL
        AND "completed_at" IS NULL
      )
      OR (
        "status" <> 'running'
        AND "locked_by" IS NULL
        AND "locked_until" IS NULL
      )
    ),
  CONSTRAINT "generation_jobs_completion_state_check"
    CHECK (
      ("status" IN ('completed', 'failed', 'cancelled') AND "completed_at" IS NOT NULL)
      OR ("status" NOT IN ('completed', 'failed', 'cancelled') AND "completed_at" IS NULL)
    ),
  CONSTRAINT "generation_jobs_attempt_state_check"
    CHECK (
      (
        "attempt_count" = 0
        AND "started_at" IS NULL
        AND "status" IN ('queued', 'cancelled')
      )
      OR (
        "attempt_count" > 0
        AND "started_at" IS NOT NULL
        AND "status" <> 'queued'
      )
    ),
  CONSTRAINT "generation_jobs_error_state_check"
    CHECK (
      (
        "status" IN ('retry_scheduled', 'failed')
        AND "last_error_message" IS NOT NULL
        AND btrim("last_error_message") <> ''
      )
      OR (
        "status" IN ('queued', 'running', 'completed')
        AND "last_error_code" IS NULL
        AND "last_error_message" IS NULL
      )
      OR "status" = 'cancelled'
    ),
  CONSTRAINT "generation_jobs_idempotency_key_unique"
    UNIQUE ("idempotency_key")
);

CREATE INDEX "generation_jobs_claim_idx"
ON "generation_jobs" ("priority" DESC, "available_at", "created_at", "id")
WHERE "status" IN ('queued', 'retry_scheduled');

CREATE INDEX "generation_jobs_request_idx"
ON "generation_jobs" ("request_id", "created_at", "id");

CREATE INDEX "generation_jobs_expired_lease_idx"
ON "generation_jobs" ("locked_until")
WHERE "status" = 'running';

-- +goose Down
DROP TABLE IF EXISTS "generation_jobs";
DROP TYPE IF EXISTS "generation_job_kind";
DROP TYPE IF EXISTS "generation_job_status";
