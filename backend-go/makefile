DATABASE_URL ?= postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable
GOOSE_DRIVER ?= postgres
GOOSE_DBSTRING ?= $(DATABASE_URL)
GOOSE_MIGRATION_DIR ?= migrations

export GOOSE_DRIVER
export GOOSE_DBSTRING
export GOOSE_MIGRATION_DIR
export DATABASE_URL

.PHONY: tools test test-race test-integration test-all migrate-up migrate-down migrate-status migrate-create sqlc-generate sqlc-vet db-generate

tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
	go install -tags="no_clickhouse no_mssql no_mysql no_sqlite3 no_libsql no_vertica no_ydb" github.com/pressly/goose/v3/cmd/goose@v3.27.1
	go install golang.org/x/vuln/cmd/govulncheck@latest

test:
	go test ./...

test-race:
	go test -race ./...

test-integration: export COURSE_AI_INTEGRATION_TEST := 1
test-integration:
	go test -count=1 -tags=integration ./tests/integration/...

test-all: test test-integration

migrate-up:
	goose up

migrate-down:
	goose down

migrate-status:
	goose status

migrate-create:
	goose create $(name) sql

sqlc-generate:
	sqlc generate

sqlc-vet:
	sqlc vet

db-generate: sqlc-generate
