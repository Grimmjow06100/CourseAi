# Course AI Backend Go

Backend Go officiel de Course AI.

Ce service expose l'API HTTP de l'application, gere l'authentification, accede a PostgreSQL via `pgx`, et execute la pipeline de generation dans des workers durables adosses a PostgreSQL.

## Stack backend

- Go 1.26+
- Gin pour HTTP
- pgx/pgxpool pour PostgreSQL
- Goose pour les migrations SQL
- JWT + bcrypt pour l'authentification
- OpenAI Responses API avec Structured Outputs
- Architecture par couches : `domain`, `contract`, `service`, `infrastructure`

## Installation

Depuis la racine du projet, demarrer PostgreSQL :

```powershell
Copy-Item .env.example .env
docker compose up -d
```

Puis dans ce dossier :

```powershell
cd backend-go
Copy-Item .env.example .env
go mod download
make migrate-up
go run ./cmd/api
```

Serveur local :

```txt
http://localhost:8080
```

## Variables d'environnement

Voir `.env.example` pour la liste complete.

Variables principales :

```env
HTTP_ADDR=:8080
DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable
JWT_SECRET=change_me_in_local_env
JWT_TOKEN_TTL=24h
OPENAI_API_KEY=sk-your-api-key
OPENAI_MODEL=gpt-5.6
OPENAI_MAX_OUTPUT_TOKENS=12000
PROMPTS_DIR=./prompts
GENERATION_WORKER_ENABLED=true
GENERATION_WORKER_CONCURRENCY=1
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

`OPENAI_API_KEY` est requise pour les routes de generation IA. `OPENAI_MODEL` et `OPENAI_MAX_OUTPUT_TOKENS` pilotent le modele et la taille maximale des reponses structurees.

## Migrations

Appliquer :

```powershell
make migrate-up
```

Rollback :

```powershell
make migrate-down
```

Statut :

```powershell
make migrate-status
```

## Routes

Sante :

```http
GET /health
```

Auth :

```http
POST /api/auth/signup
POST /api/auth/login
```

Catalogue :

```http
GET    /api/courses
GET    /api/courses/:courseID
DELETE /api/courses/:courseID
GET    /api/courses/:courseID/modules
GET    /api/modules/:moduleID
GET    /api/modules/:moduleID/lessons
GET    /api/lessons/:lessonID
```

Generation IA :

```http
POST /api/generations
POST /api/generations/analyze
POST /api/generations/:requestID/clarifications
POST /api/generations/:requestID/structure
POST /api/generations/:requestID/structure/retry
POST /api/generations/lessons/:lessonID/content
POST /api/generations/modules/:moduleID/contents
GET  /api/generations/:requestID/status
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
GET  /api/generation-jobs/:jobID
```

`POST /api/generations` persiste atomiquement une request et un job durable `analysis`. Apres l'analyse, la pipeline continue automatiquement uniquement si le brief est complet. Sinon, la request passe a `awaiting_clarification` et aucun job en aval n'est cree.
`POST /api/generations/analyze` cree une `GenerationRequest`, analyse le prompt, puis retourne le premier jet : hors scope eventuel, titre, synopsis, niveaux detectes, objectif, langue et questions de clarification.
`POST /api/generations/:requestID/clarifications` valide les reponses face aux questions persistees, confirme le brief et cree le job `architecture` dans la meme transaction.
`POST /api/generations/:requestID/structure` enfile la generation de la formation, des modules et des plans de lessons a partir du contexte confirme.
`POST /api/generations/:requestID/structure/retry` nettoie atomiquement les donnees partielles et enfile une nouvelle tentative de structure.
`POST /api/generations/lessons/:lessonID/content` enfile la generation et la persistance du contenu d'une lesson.
`POST /api/generations/modules/:moduleID/contents` enfile le contenu manquant des lessons du module.

Toutes les routes longues retournent immediatement `202 Accepted` :

```json
{
  "jobId": "uuid",
  "requestId": "uuid",
  "status": "queued",
  "jobStatus": "queued",
  "statusUrl": "/api/generations/{requestId}/status",
  "jobStatusUrl": "/api/generation-jobs/{jobId}",
  "resultUrl": "/api/generations/{requestId}/result"
}
```

Envoyer un header `Idempotency-Key` sur `POST /api/generations` permet de rejouer la commande sans creer une seconde request. Utiliser `GET /api/generation-jobs/:jobID` pour suivre `queued`, `running`, `retry_scheduled`, `completed`, `failed` ou `cancelled`.

Lorsque `GET /api/generations/:requestID/status` retourne `pipelineStatus: "awaiting_clarification"`, il contient le premier jet (`suggestedTitle`, `shortSynopsis`, niveaux, objectif et langue detectes), les questions avec des options `{ "value", "label" }` et :

```json
{
  "actionRequired": {
    "type": "submit_clarifications",
    "url": "/api/generations/{requestID}/clarifications"
  }
}
```

Le frontend affiche `label` et renvoie `value` dans `selectedValues`. Exemple :

```json
{
  "answers": [
    { "questionId": "currentLevel", "selectedValues": ["beginner"] },
    { "questionId": "targetLevel", "selectedValues": ["advanced"] },
    { "questionId": "goals", "selectedValues": ["administer_linux"] }
  ],
  "title": "Administration Linux",
  "synopsis": "Une formation progressive pour administrer Linux en production.",
  "language": "fr"
}
```

## Organisation

```txt
cmd/api                         point d'entree HTTP
internal/config                 chargement .env et variables d'environnement
internal/db                     ouverture du pool PostgreSQL et code sqlc genere
internal/domain                 entites et regles metier pures
internal/contract               interfaces entre couches
internal/service                use cases applicatifs
internal/shared                 helpers generiques sans dependance metier ou infrastructure
internal/infrastructure/auth    JWT et bcrypt
internal/infrastructure/clock   horloge systeme
internal/infrastructure/http    router, handlers, DTOs, middlewares Gin
internal/infrastructure/openai  adapter OpenAI CourseAIGenerator
internal/infrastructure/postgres repositories pgx et unit of work
internal/infrastructure/prompts implementation PromptStore
tests/integration/postgres      tests reels contre PostgreSQL, actives par build tag
tests/testkit                   setup partage reserve aux tests externes
prompts                         fichiers .prompt.md utilises par la generation
migrations                      migrations SQL Goose
```

Les tests unitaires restent a cote du code teste afin de conserver l'acces aux details du package et une navigation directe. Les tests qui exigent PostgreSQL sont isoles dans `tests/integration/postgres` et manipulent uniquement les API exportees. Le package `tests/testkit` centralise seulement le cycle de vie du pool et des transactions de test.

## Commandes utiles

```powershell
make test
make test-race
make test-integration
make test-all
go run ./cmd/api
go fmt ./...
```

`make test` n'utilise ni Docker ni PostgreSQL. `make test-integration` attend une base disponible via `DATABASE_URL`, active le build tag `integration`, desactive le cache des resultats et annule automatiquement les transactions de test. Le `DATABASE_URL` local par defaut pointe sur le port `5433` du Docker Compose.

## Pipeline IA

Le flux complet durable est :

1. `POST /api/generations` cree la request et le job `analysis`, puis retourne `202`.
2. Le worker analyse le prompt. Un hors scope termine la request sans cours. Un brief complet enfile `architecture`. Un brief incomplet place la request en `awaiting_clarification` et termine le job d'analyse.
3. Le frontend lit les questions avec `GET /api/generations/:requestID/status`, puis appelle `POST /api/generations/:requestID/clarifications`. Reponses, brief confirme, transition vers `queued` et job `architecture` sont persistes atomiquement.
4. Le job `architecture` persiste le cours et ses modules, puis cree un job `lesson_plan` par module.
5. Une fois tous les plans persistants, les jobs `lesson_content` generent theorie, exercices et quiz. Le dernier contenu cree le job `finalize_course`.
6. La finalisation marque le cours et la request `completed`.

Aucun job, lease ou worker ne reste reserve pendant l'attente utilisateur. Les routes `/structure` et de contenu restent disponibles pour les generations partielles et les reprises ciblees.

## Logs des workers

Le worker pool emet des logs JSON structures sur `stdout`. Chaque execution de job contient les champs de correlation suivants :

- `worker_id`, `job_id`, `request_id` et `parent_job_id` ;
- `job_kind` et `target_id` ;
- `attempt`, `max_attempts` et `priority` ;
- `started_at`, `finished_at` et `duration_ms` ;
- `outcome` et `job_status` ;
- `error`, `error_type` et `error_code` lorsqu'une erreur existe.

Evenements principaux :

```text
generation_worker_started
generation_worker_stopped
generation_job_started
generation_job_completed
generation_job_retry_scheduled
generation_job_failed
generation_job_interrupted
generation_job_claim_lost
generation_job_heartbeat_failed
```

Exemple d'une execution terminee :

```json
{
  "level": "INFO",
  "msg": "generation job completed",
  "event": "generation_job_completed",
  "component": "generation_worker",
  "worker_id": "course-ai-worker:host:1234:0:uuid",
  "job_id": "uuid",
  "request_id": "uuid",
  "parent_job_id": "uuid",
  "job_kind": "lesson_content",
  "target_id": "uuid",
  "attempt": 1,
  "max_attempts": 3,
  "outcome": "success",
  "job_status": "completed",
  "duration_ms": 8421
}
```

Les prompts, payloads et contenus generes ne sont pas journalises. Les logs restent ainsi utilisables pour la correlation et la mesure de latence sans exposer les donnees de formation.

L'implementation concrete de l'IA est dans `internal/infrastructure/openai` et elle est injectee dans `cmd/api/main.go`. Les prompts sont charges depuis `PROMPTS_DIR` par `internal/infrastructure/prompts`.

## Deploiement Railway

Configurer le service avec une racine `backend-go` et le fichier `railway.toml`. Le `Dockerfile` construit l'API et Goose. Railway execute les migrations en pre-deploy, verifie `/health`, puis lance l'API et le worker dans le meme processus.

Variables minimales :

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=${{Postgres.DATABASE_URL}}
GOOSE_MIGRATION_DIR=/app/migrations
OPENAI_API_KEY=...
OPENAI_MODEL=...
OPENAI_MAX_OUTPUT_TOKENS=12000
JWT_SECRET=...
JWT_TOKEN_TTL=24h
PROMPTS_DIR=/app/prompts
CORS_ALLOWED_ORIGINS=https://votre-frontend.example
GENERATION_WORKER_ENABLED=true
GENERATION_WORKER_CONCURRENCY=1
```

Railway fournit `PORT`; le backend ecoute automatiquement sur `:$PORT` lorsque `HTTP_ADDR` n'est pas defini. Garder une seule replica et une concurrence de `1` pour le premier deploiement, puis augmenter apres mesure des limites OpenAI et PostgreSQL.

