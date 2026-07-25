DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?schema=public

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-create:
	goose -dir migrations create $(name) sql