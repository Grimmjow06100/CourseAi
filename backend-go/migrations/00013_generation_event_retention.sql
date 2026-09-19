-- +goose Up
CREATE INDEX generation_events_retention ON generation_events(occurred_at, id);
CREATE INDEX generation_retry_commands_retention ON generation_retry_commands(created_at);

-- +goose Down
DROP INDEX generation_retry_commands_retention;
DROP INDEX generation_events_retention;
