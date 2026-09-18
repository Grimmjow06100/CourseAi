# Backend architecture

This document is the entry point for developers discovering the Course AI backend. It explains package ownership, dependency rules, runtime flows and where new code belongs.

## Dependency rule

Dependencies point toward stable application concepts:

```text
cmd/api
  |-- infrastructure/http ----> contract <---- service
  |-- infrastructure/jobs ----> contract         |
  |-- infrastructure/openai --> contract         v
  |-- infrastructure/postgres -> contract ----> domain
  `-- infrastructure/prompts --> contract
```

`domain` imports neither services nor infrastructure. `contract` exposes ports using domain types. `service` implements use cases against those ports. Infrastructure packages implement the ports for specific technologies. `cmd/api` is the server composition root. The operational `cmd/reconcile-generations` entry point only wires database/clock adapters to the same reconciliation service; it does not implement business transitions.

## Package map

| Package | Responsibility | Must not contain |
| --- | --- | --- |
| `internal/domain` | Entities, invariants, lifecycle transitions, pedagogical policies | SQL, HTTP, OpenAI DTOs, environment access |
| `internal/contract` | Repository ports, use-case interfaces, pagination, principals, application errors | Concrete adapters or business workflows |
| `internal/service` | Use cases, authorization, generation orchestration, transaction boundaries | Gin, pgx/sqlc rows, OpenAI SDK types |
| `internal/infrastructure/http` | Router, middleware, request/response DTOs, OpenAPI | Persistence and AI orchestration |
| `internal/infrastructure/postgres` | Repository implementations, sqlc mapping, unit of work | HTTP and generation workflow decisions |
| `internal/infrastructure/openai` | Prompt payloads, strict schemas, provider error mapping | Database operations and HTTP DTOs |
| `internal/infrastructure/jobs` | Worker concurrency, claims, leases, heartbeat, retry and maintenance | Course-generation business decisions |
| `internal/infrastructure/prompts` | Loading `.prompt.md` files | Prompt interpretation |
| `internal/db/sqlc` | Generated typed SQL methods | Manual changes |
| `internal/shared` | Small technology-neutral helpers reused by several packages | Domain policies or adapter-specific helpers |
| `cmd/api` | Configuration, adapter construction, process lifecycle | Business rules |

## HTTP request flow

```text
Gin router
  -> Clerk middleware validates the bearer token
  -> handler validates path/query/body DTOs
  -> service authorizes ownership and executes a use case
  -> UnitOfWork opens a transaction
  -> repository calls generated sqlc methods
  -> handler maps domain output to a public DTO
  -> error middleware logs the full trace and returns a sanitized response
```

Handlers do not load repositories directly. Repositories do not decide whether a Clerk user may access a resource. Ownership checks and transaction scope stay in application services.

## Durable generation flow

`POST /api/generations` does not execute OpenAI synchronously. It persists a generation request and an analysis job in one transaction, then returns `202 Accepted`.

```text
analysis
  |-- out of scope -> completed request without course
  |-- missing context -> awaiting_clarification
  `-- confirmed brief -> architecture
                            -> one lesson_plan job per module
                            -> lesson_content jobs
                            -> finalize_course
```

The worker pool claims jobs atomically in PostgreSQL. A claim contains a worker ID, an attempt number and a lease deadline. The heartbeat renews the lease during long OpenAI calls. Completion, retry or terminal failure uses the claim as a fencing token so an obsolete worker cannot overwrite a newer attempt.

Application job behavior belongs to `service.GenerationJobExecutor` and `CourseGeneratorService`. Generic execution mechanics belong to `infrastructure/jobs`.

Each request and job also carries `generation_attempt`, a business retry epoch distinct from the job claim's `attempt_count` and the clarification version. Worker transactions lock the request before checking the epoch and locking the live claim, then recheck claim expiry before commit. AI calls remain outside transactions. Old attempts cannot persist content or propagate failure into a newer retry.

Finalization validates persisted completeness and updates the request/course pair in one transaction. Maintenance reconciles interrupted completions only after current-attempt work is terminal. The read-only-by-default reconciliation command uses the same service and requires `-apply` to write. See [generation status fixes and rollout](generation-status-fixes-2026-09-18.md) before applying migration 00010 or repairing historical records.

## Persistence model

Services use `contract.UnitOfWork` to obtain transaction-scoped repositories. Repository writes return scalar entities without automatically hydrating the full graph. Generated collections use sqlc bulk operations where the expected cardinality justifies them.

Complete course reads use a fixed number of batched queries:

1. course;
2. modules for all course IDs;
3. lessons for all module IDs;
4. exercises for all lesson IDs;
5. quizzes for all lesson IDs.

The graph is reconstructed in Go. This avoids N+1 query growth while keeping SQL rows easy to map and validate.

## Database change workflow

1. Create a new Goose migration in `migrations`.
2. Add or update named SQL in `internal/db/queries`.
3. Run `sqlc vet`.
4. Run `sqlc generate`.
5. Adapt only the PostgreSQL repositories and mappers.
6. Add repository or integration coverage.
7. Apply migrations from an empty database before deployment.

Never modify `internal/db/sqlc` manually. Generated diffs are reviewed as a consequence of migration/query changes, not as source code design.

## Where to add code

| Need | Location |
| --- | --- |
| New entity rule or transition | `internal/domain` |
| New repository capability | interface in `internal/contract`, implementation in `internal/infrastructure/postgres` |
| New generation use case | `internal/service` |
| New OpenAI response shape | `internal/infrastructure/openai/dto` and strict schema in `openai` |
| New endpoint | handler/DTO/router in `internal/infrastructure/http`, plus OpenAPI and route-coverage tests |
| New background job kind | domain enum/invariants, executor dispatch, service runner, SQL persistence if needed |
| Generic reusable helper | `internal/shared` only after at least two real package-level uses |
| Environment variable | owning infrastructure config or `cmd/api/config.go`, plus `.env.example` and tests |

Do not create a shared abstraction merely because two blocks look similar. Extract it when it owns one concept, removes meaningful duplication and gives the operation a clearer name.

## Testing strategy

- Unit tests remain beside production files and can exercise unexported package behavior.
- HTTP tests verify validation, status codes, sanitization and route/OpenAPI parity.
- Domain tests cover invariants and state transitions without mocks.
- Service tests use in-memory ports to cover complete job chains and idempotency.
- PostgreSQL integration tests live in `tests/integration/postgres` and run against real migrations.
- Race tests cover worker concurrency and shared state.

The minimum pre-merge checks are `go vet ./...`, `staticcheck ./...`, `go test -count=1 ./...`, `go test -race ./...`, `sqlc vet` and the PostgreSQL integration suite.

## Recommended reading order

1. `internal/domain/generation_request.go`
2. `internal/domain/generation_jobs.go`
3. `internal/contract/generation_pipeline.go`
4. `internal/service/generation_commands.go`
5. `internal/service/course_generator_jobs.go`
6. `internal/service/generation_job_executor.go`
7. `internal/infrastructure/jobs/worker_pool.go`
8. `internal/infrastructure/postgres/unit_of_work.go`
9. `internal/infrastructure/http/router.go`
10. `cmd/api/app.go`
