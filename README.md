# Course AI

Course AI est une application de generation de formations IT avec l'IA.

Le backend officiel du projet est le backend Go situe dans `backend-go/`.

## Etat actuel

Le backend Go contient aujourd'hui :

- un modele de domaine pour `Course`, `Module`, `Lesson` et `GenerationRequest` ;
- des contracts applicatifs pour les repositories, l'identite, les transactions et la generation IA ;
- une couche service avec autorisation Clerk, catalogue, commandes de generation et executeur de jobs ;
- une infrastructure PostgreSQL basee sur `pgx` ;
- un adapter OpenAI qui implemente `contract.CourseAIGenerator` avec Structured Outputs ;
- une implementation `PromptStore` dans l'infrastructure ;
- des migrations SQL Goose ;
- une API HTTP Gin avec commandes asynchrones, statut de jobs, CORS et arret gracieux ;
- une queue PostgreSQL durable avec worker pool, leases, heartbeat, retries et reprise apres redemarrage ;
- des logs JSON structures pour chaque worker et chaque issue de job, avec correlation, tentative et duree ;
- une authentification Clerk avec ownership des demandes, jobs et formations ;
- une base PostgreSQL locale via Docker Compose.

La generation IA est cablee dans `cmd/api/app.go` via `internal/infrastructure/openai` et les prompts Markdown de `backend-go/prompts`.

## Stack

- Backend : Go, Gin, pgx, Goose
- Base de donnees : PostgreSQL 16
- Auth : Clerk et session token bearer
- IA : OpenAI SDK Responses API avec Structured Outputs
- Frontend : React, Vite, TanStack Router/Query, Clerk et Tailwind dans `frontend/`
- Infra locale : Docker Compose pour PostgreSQL

## Structure du repo

```txt
.
├── backend-go/           # backend Go officiel
│   ├── cmd/api/          # point d'entree HTTP
│   ├── internal/         # code applicatif Go
│   │   ├── contract/     # interfaces entre couches
│   │   ├── domain/       # entites et regles metier pures
│   │   ├── service/      # use cases applicatifs
│   │   ├── infrastructure/
│   │   │   ├── auth/     # configuration de la verification des sessions Clerk
│   │   │   ├── clock/    # horloge systeme
│   │   │   ├── http/     # Gin router, handlers, DTOs, middlewares
│   │   │   ├── openai/   # adapter CourseAIGenerator
│   │   │   ├── postgres/ # repositories pgx et unit of work
│   │   │   └── prompts/  # implementation PromptStore
│   │   ├── config/       # chargement des variables d'environnement
│   │   └── db/           # ouverture du pool PostgreSQL et code sqlc genere
│   ├── docs/             # architecture et onboarding backend
│   ├── migrations/       # migrations Goose
│   ├── prompts/          # prompts IA .prompt.md
│   ├── .env.example      # variables attendues par le backend Go
│   └── README.md         # documentation backend detaillee
├── frontend/             # frontend React/Vite
├── docker-compose.yml    # PostgreSQL local
├── .env.example          # variables Docker Compose locales
├── TASKS.md              # historique des taches terminees
└── README.md
```

## Prerequis

- Go 1.27+
- Docker Desktop
- Goose CLI pour les migrations
- Une cle OpenAI valide
- Node.js 22+ et une publishable key Clerk pour le frontend

Installer Goose si besoin :

```powershell
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## Installation locale

Depuis la racine du projet :

```powershell
Copy-Item .env.example .env
docker compose up -d
```

Puis cote backend Go :

```powershell
cd backend-go
Copy-Item .env.example .env
go mod download
make migrate-up
go run ./cmd/api
```

L'API demarre par defaut sur :

```txt
http://localhost:8080
```

Puis cote frontend :

```powershell
cd frontend
Copy-Item .env.example .env.local
npm ci
npm run api:generate
npm run dev
```

Configurer `VITE_CLERK_PUBLISHABLE_KEY` dans `frontend/.env.local`. Le client est disponible par defaut sur `http://localhost:5173`. Sa documentation d'architecture et de deploiement se trouve dans `frontend/README.md` et `frontend/docs/architecture.md`.

Healthcheck :

```txt
GET http://localhost:8080/health
```

Documentation interactive et contrat OpenAPI :

```txt
Swagger UI :    http://localhost:8080/docs
OpenAPI JSON :  http://localhost:8080/docs/openapi.json
```

La spécification documente toutes les routes, leurs paramètres, payloads, réponses, statuts asynchrones et erreurs. Un test compare automatiquement ce contrat avec les routes enregistrées dans Gin afin d'éviter qu'un nouvel endpoint reste non documenté.

## Variables d'environnement

A la racine du projet, pour PostgreSQL local :

```env
POSTGRES_USER=course_ai
POSTGRES_PASSWORD=course_ai_password
POSTGRES_DB=course_ai
POSTGRES_PORT=5433
```

Dans `backend-go/.env` :

```env
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable
DB_HOST=localhost
DB_PORT=5433
DB_NAME=course_ai
DB_USER=course_ai
DB_PASSWORD=course_ai_password
DB_SSLMODE=disable
DB_MAX_CONNS=10
DB_MIN_CONNS=1
DB_MAX_CONN_IDLE_TIME=30m
DB_MAX_CONN_LIFETIME=1h
DB_HEALTH_CHECK_PERIOD=1m
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable
GOOSE_MIGRATION_DIR=./migrations
PROMPTS_DIR=./prompts
CLERK_SECRET_KEY=sk_test_xxx
CLERK_AUTHORIZED_PARTIES=http://localhost:5173
OPENAI_API_KEY=sk-your-api-key
CORS_ALLOWED_ORIGINS=http://localhost:5173
GENERATION_WORKER_ENABLED=true
GENERATION_WORKER_CONCURRENCY=1
OPENAI_MAX_RETRIES=0
HTTP_MAX_BODY_BYTES=65536
GENERATION_MAX_ACTIVE_PER_USER=2
GENERATION_MAX_DAILY_PER_USER=10

```

`OPENAI_API_KEY` est requise pour utiliser les routes de generation IA.

## Routes HTTP

La référence détaillée et directement testable se trouve dans Swagger UI. La liste suivante sert d'aperçu rapide.

Toutes les routes metier `/api` exigent un session token Clerk dans `Authorization: Bearer <token>`. `/health`, `/health/live`, `/health/ready` et les preflights CORS restent publics.

Catalogue de cours :

```http
GET    /api/courses
GET    /api/courses/:courseID
DELETE /api/courses/:courseID
GET    /api/courses/:courseID/modules
GET    /api/modules/:moduleID
GET    /api/modules/:moduleID/lessons
GET    /api/lessons/:lessonID
GET    /api/lessons/:lessonID/solutions
```

Generation IA :

```http
GET  /api/generations
POST /api/generations
POST /api/generations/:requestID/clarifications
POST /api/generations/:requestID/structure
POST /api/generations/:requestID/structure/retry
POST /api/generations/lessons/:lessonID/content
POST /api/generations/modules/:moduleID/contents
GET  /api/generations/:requestID/status
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
DELETE /api/generations/:requestID
GET  /api/generation-jobs/:jobID
```

`POST /api/generations` persiste la demande, enfile un job d'analyse et retourne immediatement `202 Accepted`. Si l'analyse manque d'informations, le statut passe a `awaiting_clarification` sans conserver de worker actif. Sinon, la pipeline enfile automatiquement l'architecture.
`POST /api/generations/:requestID/clarifications` valide et persiste les reponses ainsi que le brief confirme, puis enfile atomiquement le job d'architecture. Les valeurs envoyees doivent correspondre aux champs `value` des options retournees par le statut.
`POST /api/generations/:requestID/structure` enfile la formation, ses modules et le plan des lessons. Cette route ne genere pas le contenu Markdown.
`POST /api/generations/:requestID/structure/retry` relance uniquement l'etape structure sur une request `failed` dont l'echec vient de `architecture_generation` ou `lesson_plan_generation`; le body est le meme que `/structure` et les donnees partielles sont supprimees avant relance.
`POST /api/generations/lessons/:lessonID/content` et `POST /api/generations/modules/:moduleID/contents` retournent aussi `202`; suivre leur etat avec `GET /api/generation-jobs/:jobID`.

Le header `Idempotency-Key` est recommande sur `POST /api/generations` et est scope par Clerk User ID. Les migrations `00001` a `00009` doivent etre appliquees avant le demarrage. `00008` ajoute la reconciliation durable des echecs de jobs et les index operationnels ; `00009` indexe l'historique chronologique par proprietaire.

## Exemples rapides

Les comptes sont crees dans le frontend avec Clerk. Pour appeler une route metier :

```http
Authorization: Bearer <session-token-clerk>
```

Demarrer l'analyse asynchrone d'une demande de formation :

```http
POST /api/generations
Content-Type: application/json

{
  "prompt": "Je veux apprendre Docker pour deployer une API backend."
}
```

Repondre aux questions retournees lorsque `pipelineStatus` vaut `awaiting_clarification` :

```http
POST /api/generations/:requestID/clarifications
Content-Type: application/json

{
  "answers": [
    { "questionId": "currentLevel", "selectedValues": ["beginner"] },
    { "questionId": "targetLevel", "selectedValues": ["intermediate"] },
    { "questionId": "goals", "selectedValues": ["deploy_applications"] }
  ],
  "title": "Formation Docker pour deployer une API backend",
  "synopsis": "Une progression pratique pour comprendre Docker, creer des images et deployer une API conteneurisee.",
  "language": "fr"
}
```

Le backend reconstruit les niveaux et objectifs du brief a partir des reponses validees. Cette commande retourne le `jobId` du job d'architecture avec un statut `queued`.

Le polling de `GET /api/generations/:requestID/status` expose aussi `suggestedTitle`, `shortSynopsis`, les niveaux, l'objectif et la langue detectes. Le frontend dispose ainsi du premier jet complet avant d'envoyer les clarifications.

Generer la structure apres confirmation du premier jet :

```http
POST /api/generations/:requestID/structure
Content-Type: application/json

{
  "title": "Formation Docker pour deployer une API backend",
  "synopsis": "Une progression pratique pour comprendre Docker, creer des images et deployer une API conteneurisee.",
  "currentLevel": "beginner",
  "targetLevel": "intermediate",
  "goals": ["Comprendre Docker", "Conteneuriser une API", "Preparer un deploiement"],
  "language": "fr"
}
```

Relancer uniquement la structure apres un echec `architecture_generation` ou `lesson_plan_generation` :

```http
POST /api/generations/:requestID/structure/retry
Content-Type: application/json

{
  "title": "Formation Docker pour deployer une API backend",
  "synopsis": "Une progression pratique pour comprendre Docker, creer des images et deployer une API conteneurisee.",
  "currentLevel": "beginner",
  "targetLevel": "intermediate",
  "goals": ["Comprendre Docker", "Conteneuriser une API", "Preparer un deploiement"],
  "language": "fr"
}
```
Generer le contenu d'une lesson :

```http
POST /api/generations/lessons/:lessonID/content
```

Generer le contenu de toutes les lessons d'un module :

```http
POST /api/generations/modules/:moduleID/contents
```

Lister les cours :

```http
GET /api/courses?page=1&pageSize=20&search=docker&orderBy=created_at&orderDirection=desc
```

## Migrations

Appliquer les migrations depuis `backend-go/` :

```powershell
make migrate-up
```

Rollback d'une migration :

```powershell
make migrate-down
```

Afficher le statut :

```powershell
make migrate-status
```

## Commandes utiles

Depuis `backend-go/` :

```powershell
make test
make test-race
make test-integration
make test-all
go run ./cmd/api
go fmt ./...
```

Les tests unitaires sont colocalises avec les packages Go. Les tests qui utilisent la base reelle sont regroupes dans `backend-go/tests/integration/postgres`; `make test-integration` les active explicitement et utilise la base definie par `DATABASE_URL`.

Depuis la racine :

```powershell
docker compose up -d
docker compose down
```

