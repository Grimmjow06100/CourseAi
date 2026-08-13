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
goose up
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
goose up
```

Rollback :

```powershell
goose down
```

Statut :

```powershell
goose status
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
POST /api/generations/:requestID/structure
POST /api/generations/:requestID/structure/retry
POST /api/generations/lessons/:lessonID/content
POST /api/generations/modules/:moduleID/contents
GET  /api/generations/:requestID/status
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
GET  /api/generation-jobs/:jobID
```

`POST /api/generations` persiste atomiquement une request et un job durable pour le mode automatique complet : prompt -> analyse -> structure -> contenu de toutes les lessons.
`POST /api/generations/analyze` cree une `GenerationRequest`, analyse le prompt, puis retourne le premier jet : hors scope eventuel, titre, synopsis, niveaux detectes, objectif, langue et questions de clarification.
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

Le flux interactif du MVP est separe en trois frontieres claires :

1. `POST /api/generations/analyze` : l'utilisateur envoie un prompt brut. Le backend persiste une `GenerationRequest`, appelle le prompt d'analyse, puis retourne soit un hors scope, soit un premier jet avec titre, synopsis, niveaux, objectif, langue et questions de clarification.
2. `POST /api/generations/:requestID/structure` : le frontend renvoie le contexte confirme. Le backend enfile un job qui genere et persiste `Course`, `Module` et les plans de `Lesson`. Les lessons ont un `module_id` cree par le backend et `content_markdown = null`.
   En cas d'echec sur `architecture_generation` ou `lesson_plan_generation`, `POST /api/generations/:requestID/structure/retry` peut relancer cette seule frontiere avec le meme payload confirme.
3. `POST /api/generations/lessons/:lessonID/content` ou `POST /api/generations/modules/:moduleID/contents` : le backend enfile les jobs de contenu. Le worker renouvelle son lease pendant OpenAI, applique une retry policy bornee et persiste les erreurs terminales.

`POST /api/generations` reste disponible comme mode automatique complet pour enchainer toute la pipeline sans confirmation intermediaire.

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

