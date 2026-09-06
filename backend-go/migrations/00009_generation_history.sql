-- +goose Up
CREATE INDEX generation_requests_clerk_created_id_idx
ON generation_requests (clerk_user_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS generation_requests_clerk_created_id_idx;
