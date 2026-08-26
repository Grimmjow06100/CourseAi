-- +goose Up
-- Clerk authenticates requests directly. PostgreSQL only stores the verified
-- Clerk subject needed to own generation requests and courses.
ALTER TABLE "generation_requests"
  RENAME COLUMN "owner_subject" TO "clerk_user_id";

ALTER TABLE "generation_requests"
  DROP CONSTRAINT IF EXISTS "generation_requests_owner_subject_not_blank_check";

UPDATE "generation_requests"
SET "clerk_user_id" = 'user_legacy_unowned'
WHERE "clerk_user_id" = 'legacy_unowned';

ALTER TABLE "generation_requests"
  ADD CONSTRAINT "generation_requests_clerk_user_id_check"
    CHECK (
      char_length("clerk_user_id") BETWEEN 6 AND 64
      AND left("clerk_user_id", 5) = 'user_'
      AND "clerk_user_id" = btrim("clerk_user_id")
    );

ALTER INDEX IF EXISTS "generation_requests_owner_subject_idx"
  RENAME TO "generation_requests_clerk_user_id_idx";

ALTER TABLE "courses"
  ADD COLUMN "clerk_user_id" VARCHAR(64);

UPDATE "courses" AS course
SET "clerk_user_id" = request."clerk_user_id"
FROM "generation_requests" AS request
WHERE request."id" = course."request_id";

ALTER TABLE "courses"
  ALTER COLUMN "clerk_user_id" SET NOT NULL,
  ADD CONSTRAINT "courses_clerk_user_id_check"
    CHECK (
      char_length("clerk_user_id") BETWEEN 6 AND 64
      AND left("clerk_user_id", 5) = 'user_'
      AND "clerk_user_id" = btrim("clerk_user_id")
    );

CREATE INDEX "courses_clerk_user_id_idx"
  ON "courses" ("clerk_user_id");

DROP TABLE IF EXISTS "clerk_webhook_events";
DROP TABLE IF EXISTS "users";

-- +goose Down
CREATE TABLE "users" (
  "id" VARCHAR(64) NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "first_name" VARCHAR(100),
  "last_name" VARCHAR(100),
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" TIMESTAMPTZ,
  CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "users_clerk_id_format_check"
    CHECK (char_length("id") BETWEEN 6 AND 64 AND left("id", 5) = 'user_'),
  CONSTRAINT "users_email_not_blank_check"
    CHECK (char_length("email") > 0 AND "email" = btrim("email"))
);

CREATE UNIQUE INDEX "users_email_lower_key" ON "users" (lower("email"));

CREATE TABLE "clerk_webhook_events" (
  "event_id" VARCHAR(255) NOT NULL,
  "event_type" VARCHAR(100) NOT NULL,
  "processed_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "clerk_webhook_events_pkey" PRIMARY KEY ("event_id"),
  CONSTRAINT "clerk_webhook_events_event_id_not_blank_check"
    CHECK (char_length(btrim("event_id")) > 0),
  CONSTRAINT "clerk_webhook_events_event_type_not_blank_check"
    CHECK (char_length(btrim("event_type")) > 0)
);

DROP INDEX IF EXISTS "courses_clerk_user_id_idx";

ALTER TABLE "courses"
  DROP CONSTRAINT IF EXISTS "courses_clerk_user_id_check",
  DROP COLUMN IF EXISTS "clerk_user_id";

DROP INDEX IF EXISTS "generation_requests_clerk_user_id_idx";

ALTER TABLE "generation_requests"
  DROP CONSTRAINT IF EXISTS "generation_requests_clerk_user_id_check";

UPDATE "generation_requests"
SET "clerk_user_id" = 'legacy_unowned'
WHERE "clerk_user_id" = 'user_legacy_unowned';

ALTER TABLE "generation_requests"
  RENAME COLUMN "clerk_user_id" TO "owner_subject";

ALTER TABLE "generation_requests"
  ADD CONSTRAINT "generation_requests_owner_subject_not_blank_check"
    CHECK (char_length(btrim("owner_subject")) > 0);

CREATE INDEX "generation_requests_owner_subject_idx"
  ON "generation_requests" ("owner_subject");
