-- +goose Up
-- Legacy UUIDs and password hashes cannot be mapped to Clerk identities safely.
-- Refuse to discard existing accounts silently; migrate or remove them explicitly
-- before applying this migration.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "users" LIMIT 1) THEN
    RAISE EXCEPTION 'cannot migrate non-empty users table to Clerk identities'
      USING HINT = 'Export or remove legacy users, then run the migration again.';
  END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE "users";

CREATE TABLE "users" (
  "id" VARCHAR(64) NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "first_name" VARCHAR(100),
  "last_name" VARCHAR(100),
  "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
  CONSTRAINT "users_clerk_id_format_check"
    CHECK (char_length("id") BETWEEN 6 AND 64 AND left("id", 5) = 'user_'),
  CONSTRAINT "users_email_not_blank_check"
    CHECK (char_length("email") > 0 AND "email" = btrim("email"))
);

-- Clerk may preserve the original casing of an address. This index enforces the
-- application invariant that two users cannot own the same email address.
CREATE UNIQUE INDEX "users_email_lower_key" ON "users" (lower("email"));

-- +goose Down
-- A Clerk identity has no local password hash, so rolling back cannot preserve it.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "users" LIMIT 1) THEN
    RAISE EXCEPTION 'cannot roll back a non-empty Clerk users table'
      USING HINT = 'Export or remove Clerk users before rolling back this migration.';
  END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE "users";

CREATE TABLE "users" (
  "id" UUID NOT NULL DEFAULT gen_random_uuid(),
  "username" TEXT NOT NULL,
  "password" TEXT NOT NULL,
  "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP(3) NOT NULL,
  CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "users_username_key" ON "users" ("username");
