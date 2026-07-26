# Course AI

Course AI est une application de generation de formations IT avec l'IA.

Le backend officiel du projet est le backend Go situe dans `backend-go/`.

## Etat actuel

Le backend Go contient aujourd'hui :

- un modele de domaine pour `Course`, `Module`, `Lesson`, `GenerationRequest` et `User` ;
- des contracts applicatifs pour les repositories, services, auth, transactions et generation IA ;
- une couche service avec auth, catalogue de cours et orchestration de pipeline de generation ;
- une infrastructure PostgreSQL basee sur `pgx` ;
- un adapter OpenAI qui implemente `contract.CourseAIGenerator` avec Structured Outputs ;
- une implementation `PromptStore` dans l'infrastructure ;
- des migrations SQL Goose ;
- une API HTTP Gin avec DTOs, handlers, router et middlewares ;
- une auth JWT + bcrypt ;
- une base PostgreSQL locale via Docker Compose.

La generation IA est cablee dans `cmd/api/main.go` via `internal/infrastructure/openai` et les prompts Markdown de `backend-go/prompts`.

## Stack

- Backend : Go, Gin, pgx, Goose
- Base de donnees : PostgreSQL 16
- Auth : JWT, bcrypt
- IA : OpenAI SDK Responses API avec Structured Outputs
- Frontend : React + Vite dans `frontend/`
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
│   │   │   ├── auth/     # JWT et bcrypt
│   │   │   ├── clock/    # horloge systeme
│   │   │   ├── http/     # Gin router, handlers, DTOs, middlewares
│   │   │   ├── openai/   # adapter CourseAIGenerator
│   │   │   ├── postgres/ # repositories pgx et unit of work
│   │   │   └── prompts/  # implementation PromptStore
│   │   ├── config/       # chargement des variables d'environnement
│   │   └── database/     # ouverture du pool PostgreSQL
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

- Go 1.26+
- Docker Desktop
- Goose CLI pour les migrations
- Une cle OpenAI valide

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
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" up
go run ./cmd/api
```

L'API demarre par defaut sur :

```txt
http://localhost:8080
```

Healthcheck :

```txt
GET http://localhost:8080/health
```

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
JWT_SECRET=change_me_in_local_env
JWT_TOKEN_TTL=24h
OPENAI_API_KEY=sk-your-api-key

```

`OPENAI_API_KEY` est requise pour utiliser les routes de generation IA.

## Routes HTTP

Auth :

```http
POST /api/auth/signup
POST /api/auth/login
```

Catalogue de cours :

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
POST /api/generations/structure
POST /api/generations/lessons/:lessonID/content
POST /api/generations/modules/:moduleID/contents
GET  /api/generations/:requestID/status
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
```

`POST /api/generations` genere la formation complete avec contenu de toutes les lessons.
`POST /api/generations/structure` genere seulement la formation, ses modules et ses lessons, sans contenu Markdown.
`POST /api/generations/lessons/:lessonID/content` genere et persiste le contenu d'une lesson.
`POST /api/generations/modules/:moduleID/contents` genere et persiste le contenu de toutes les lessons d'un module.

## Exemples rapides

Creer un utilisateur :

```http
POST /api/auth/signup
Content-Type: application/json

{
  "username": "samy",
  "password": "Password!"
}
```

Generer la structure d'une formation :

```http
POST /api/generations/structure
Content-Type: application/json

{
  "prompt": "Je veux apprendre Docker pour deployer une API backend."
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

Appliquer les migrations :

```powershell
cd backend-go
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" up
```

Rollback d'une migration :

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" down
```

Afficher le statut :

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" status
```

## Commandes utiles

Depuis `backend-go/` :

```powershell
go test ./...
go run ./cmd/api
go fmt ./...
```

Depuis la racine :

```powershell
docker compose up -d
docker compose down
```
