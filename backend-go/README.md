# Course AI Backend Go

Backend Go officiel de Course AI.

Ce service expose l'API HTTP de l'application, gere l'authentification, accede a PostgreSQL via `pgx`, et contient la pipeline applicative de generation de formations.

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
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" up
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
```

`OPENAI_API_KEY` est requise pour les routes de generation IA. `OPENAI_MODEL` et `OPENAI_MAX_OUTPUT_TOKENS` pilotent le modele et la taille maximale des reponses structurees.

## Migrations

Appliquer :

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" up
```

Rollback :

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" down
```

Statut :

```powershell
goose -dir ./migrations postgres "postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable" status
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
POST /api/generations/structure
POST /api/generations/lessons/:lessonID/content
POST /api/generations/modules/:moduleID/contents
GET  /api/generations/:requestID/status
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
```

`POST /api/generations/structure` persiste la formation avec modules et lessons sans contenu Markdown.
`POST /api/generations/lessons/:lessonID/content` genere et persiste le contenu d'une lesson.
`POST /api/generations/modules/:moduleID/contents` genere et persiste le contenu de toutes les lessons du module.

## Organisation

```txt
cmd/api                         point d'entree HTTP
internal/config                 chargement .env et variables d'environnement
internal/database               ouverture du pool PostgreSQL
internal/domain                 entites et regles metier pures
internal/contract               interfaces entre couches
internal/service                use cases applicatifs
internal/infrastructure/auth    JWT et bcrypt
internal/infrastructure/clock   horloge systeme
internal/infrastructure/http    router, handlers, DTOs, middlewares Gin
internal/infrastructure/openai  adapter OpenAI CourseAIGenerator
internal/infrastructure/postgres repositories pgx et unit of work
internal/infrastructure/prompts implementation PromptStore
prompts                         fichiers .prompt.md utilises par la generation
migrations                      migrations SQL Goose
```

## Commandes utiles

```powershell
go test ./...
go run ./cmd/api
go fmt ./...
```

## Pipeline IA

`CourseGeneratorService` orchestre :

1. creation de `GenerationRequest` ;
2. analyse du prompt utilisateur ;
3. generation d'architecture de cours ;
4. generation des plans de lessons ;
5. generation optionnelle du contenu Markdown ;
6. persistance progressive dans PostgreSQL.

L'implementation concrete de l'IA est dans `internal/infrastructure/openai` et elle est injectee dans `cmd/api/main.go`. Les prompts sont charges depuis `PROMPTS_DIR` par `internal/infrastructure/prompts`.
