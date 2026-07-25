# Course AI Backend Go

Minimal Go backend scaffold for rebuilding the Course AI API.

## Requirements

- Go 1.26+
- PostgreSQL running from the root `docker-compose.yml`
- A local `.env` file copied from `.env.example`

## Setup

```powershell
cd backend-go
Copy-Item .env.example .env
go run ./cmd/api
```

## Database Migrations

The initial Goose migration mirrors the current Prisma schema from the NestJS backend.
The local `.env` exposes both the full `DATABASE_URL` and split database settings
(`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`) so the
configuration can evolve without rewriting every command. The pgx pool is also
configured through `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_IDLE_TIME`,
`DB_MAX_CONN_LIFETIME`, and `DB_HEALTH_CHECK_PERIOD`.

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" up
```

To rollback the initial schema:

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" down
```

## Endpoints

- `GET /health` returns service status.
- `GET /prompts` returns loaded prompt names.

## Project Layout

```txt
cmd/api              application entrypoint
internal/config      environment loading and app config
internal/envfile     minimal .env loader
internal/httpapi     HTTP routes and handlers
internal/prompts     prompt loading from markdown files
prompts              system prompts used by generation pipeline
migrations           Goose SQL migrations
```

This scaffold intentionally uses the Go standard library first. Add framework or database dependencies only when the core boundaries are clear.

