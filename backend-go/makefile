DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?schema=public

.PHONY: migrate-up migrate-down migrate-status migrate-create sqlc-generate sqlc-vet db-generate

migrate-up:
	goose up postgres "$(DATABASE_URL)" -dir migrations

migrate-down:
	goose down postgres "$(DATABASE_URL)" -dir migrations

migrate-status:
	goose status postgres "$(DATABASE_URL)" -dir migrations

migrate-create:
	goose create $(name) sql -dir migrations

sqlc-generate:
	sqlc generate

sqlc-vet:
	sqlc vet

db-generate: sqlc-generate
