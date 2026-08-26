-- +goose Up
ALTER TABLE "users"
  ADD COLUMN "deleted_at" TIMESTAMPTZ;

ALTER TABLE "generation_requests"
  ADD COLUMN "owner_subject" VARCHAR(64);

-- Pre-production data has no Clerk owner. Preserve it without exposing it to
-- authenticated users; newly created requests always receive a Clerk subject.
UPDATE "generation_requests"
SET "owner_subject" = 'legacy_unowned'
WHERE "owner_subject" IS NULL;

ALTER TABLE "generation_requests"
  ALTER COLUMN "owner_subject" SET NOT NULL,
  ADD CONSTRAINT "generation_requests_owner_subject_not_blank_check"
    CHECK (char_length(btrim("owner_subject")) > 0);

CREATE INDEX "generation_requests_owner_subject_idx"
  ON "generation_requests" ("owner_subject");

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

-- +goose Down
DROP TABLE IF EXISTS "clerk_webhook_events";

DROP INDEX IF EXISTS "generation_requests_owner_subject_idx";

ALTER TABLE "generation_requests"
  DROP CONSTRAINT IF EXISTS "generation_requests_owner_subject_not_blank_check",
  DROP COLUMN IF EXISTS "owner_subject";

ALTER TABLE "users"
  DROP COLUMN IF EXISTS "deleted_at";
