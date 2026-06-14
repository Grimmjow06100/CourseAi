# TASKS

## Done

- [x] Fix Prisma PostgreSQL connection by moving the Docker host port from `5432` to `5433` to avoid the local PostgreSQL conflict.
- [x] Apply the initial Prisma migration against the Docker PostgreSQL database.
- [x] Fix backend TypeScript build errors around users controller DTOs/service calls and harden auth password verification.
- [x] Resolve backend ESLint `no-unsafe-*` diagnostics in auth/users flow and verify lint globally.
- [x] Split the vault `System Prompt prototype` note into root prompt files for needs analysis, course architecture, module lessons, and generation payload.
- [x] Implement AI service prompt loading from `backend/prompts/*.prompt.md` into an in-memory map on module initialization.
- [x] Add a class-validator DTO for the AI needs-analysis JSON response contract.
- [x] Write a vault proposal to improve `analysis.prompt.md` so AI outputs match `AnalysisResponseDto`.
- [x] Add `forceConsistentCasingInFileNames` to the frontend root TypeScript project config.
- [x] Write a learning note explaining environment variables, scopes, `.env` files, and Node/NestJS loading behavior.
- [x] Fix the generator service unsafe assignment diagnostic by replacing generic DTO validation with concrete analysis-response validation.
- [x] Improve the architecture generation prompt and add a class-validator DTO for architecture JSON responses.
- [x] Implement and type-check `promptArchitecture` with concrete architecture response validation.
- [x] Add the class-validator DTO for the module lessons plan response JSON.
- [x] Implement module lessons plan generation in controller, service, and parser validation service.
- [x] Analyze DTO and Prisma schema alignment for persistence at each generation step and document recommendations in the vault.
- [x] Scaffold a minimal Go backend in `backend-go` with environment loading, prompt loading, and HTTP health endpoints.
- [x] Finalize the NestJS generation pipeline with Prisma persistence, aligned DTO/schema models, Swagger route documentation, CRUD endpoints, and README updates.
- [x] Add the lesson content generation step with prompt, DTO validation, Prisma persistence, API documentation, migration, and tests.
- [x] Complete the backend MVP without auth by adding full pipeline orchestration, status/result/retry endpoints, Zod structured outputs, environment validation, rate limiting, OpenAI retry handling, and parser tests.
