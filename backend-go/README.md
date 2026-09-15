# Course AI Backend Go

Backend Go officiel de Course AI.

Ce service expose l'API HTTP de l'application, verifie les sessions Clerk, accede a PostgreSQL via `pgx`, et execute la pipeline de generation dans des workers durables adosses a PostgreSQL.

## Stack backend

- Go 1.27+
- Gin pour HTTP
- pgx/pgxpool pour PostgreSQL
- Goose pour les migrations SQL
- Clerk pour l'authentification bearer
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
make tools
make migrate-up
go run ./cmd/api
```

`make tools` installe les versions attendues de sqlc et Goose ainsi que `govulncheck`. Goose est compile avec le driver PostgreSQL et sans les drivers inutilises. Le fichier `.env` fournit deja `GOOSE_DRIVER`, `GOOSE_DBSTRING` et `GOOSE_MIGRATION_DIR` : utiliser alors `goose up` ou les targets `make migrate-*`. Ne pas ajouter une seconde fois `postgres DATABASE_URL` sur la meme commande, sinon Goose interprete `postgres` comme le nom de la commande.

Serveur local :

```txt
http://localhost:8080
```

## Documentation API

Une spécification OpenAPI 3 exhaustive est embarquée dans le binaire et servie par Gin :

```txt
Swagger UI :     http://localhost:8080/docs
OpenAPI JSON :   http://localhost:8080/docs/openapi.json
```

L'interface Swagger permet d'inspecter les schémas, les exemples, les codes d'erreur et d'exécuter les requêtes contre l'hôte courant. Utiliser le bouton `Authorize` avec un session token Clerk. Les routes de documentation ne sont pas enregistrees lorsque `APP_ENV=production`.

Le test `TestOpenAPIDocumentCoversEveryPublicRoute` compare les opérations documentées aux routes Gin. Il échoue lorsqu'une route métier est ajoutée ou supprimée sans mettre à jour `internal/infrastructure/http/apidocs/openapi.json`.

## Variables d'environnement

Voir `.env.example` pour la liste complete.

Variables principales :

```env
HTTP_ADDR=:8080
DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai?sslmode=disable
CLERK_SECRET_KEY=sk_test_xxx
CLERK_AUTHORIZED_PARTIES=http://localhost:5173
OPENAI_API_KEY=sk-your-api-key
OPENAI_MODEL=gpt-5.6
OPENAI_MAX_OUTPUT_TOKENS=12000
OPENAI_MAX_RETRIES=0
PROMPTS_DIR=./prompts
GENERATION_WORKER_ENABLED=true
GENERATION_WORKER_CONCURRENCY=1
HTTP_MAX_BODY_BYTES=65536
GENERATION_RATE_LIMIT_REQUESTS=10
GENERATION_MAX_ACTIVE_PER_USER=2
GENERATION_MAX_DAILY_PER_USER=10
GENERATION_MAX_PENDING_JOBS=500
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

`OPENAI_API_KEY` est requise pour les routes de generation IA. Le SDK OpenAI ne retente pas par defaut (`OPENAI_MAX_RETRIES=0`) : les retries durables des jobs restent la source de verite. Les bodies sont limites a 64 Kio, les prompts a 4 000 caracteres, puis les limites actives/journalieres et la profondeur globale de queue sont verifiees sous verrou transactionnel PostgreSQL.

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

La migration `00005_migrate_users_to_clerk.sql` a prepare la transition depuis l'ancienne authentification locale. La migration `00006_add_clerk_ownership_and_webhooks.sql` a introduit l'ownership initial. La migration `00007_remove_clerk_user_sync.sql` retire ensuite la projection `users` et les evenements webhook. La migration `00008_generation_operational_safety.sql` ajoute le suivi de reconciliation des echecs workers et les index d'admission/retention. La migration `00009_generation_history.sql` indexe l'historique chronologique par proprietaire Clerk.

## Routes

### Horloge et authentification

La verification des jetons Clerk accepte une tolerance horaire bornee a 15 secondes pour `iat`, `nbf` et `exp`. La signature, l'origine autorisee et le sujet utilisateur restent obligatoires. Un jeton expire ou emis dans le futur au-dela de cette tolerance est refuse.

Garder l'horloge de l'hote synchronisee (NTP). Un serveur en retard peut rejeter un jeton neuf comme pas encore valide. Sous Windows, `w32tm /query /status` permet de verifier le service de synchronisation ; corriger l'heure via les parametres Windows si le service est arrete. Une modification du middleware necessite le redemarrage de l'API Go.

Sante :

```http
GET /health
GET /health/live
GET /health/ready
```

`/health/live` ne teste que le processus. `/health` et `/health/ready` retournent `503` si PostgreSQL est indisponible ou si le worker attendu est desactive. Toutes les routes `/api` exigent `Authorization: Bearer <session-token-clerk>`. Une ressource appartenant a un autre utilisateur est retournee comme introuvable.

Catalogue :

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
GET  /api/generations/:requestID/jobs
GET  /api/generations/:requestID/result
POST /api/generations/:requestID/retry
DELETE /api/generations/:requestID
GET  /api/generation-jobs/:jobID
```

`POST /api/generations` persiste atomiquement une request et un job durable `analysis`. Apres l'analyse, la pipeline continue automatiquement uniquement si le brief est complet. Sinon, la request passe a `awaiting_clarification` et aucun job en aval n'est cree.
`POST /api/generations/:requestID/clarifications` valide les reponses face aux questions persistees, confirme le brief et cree le job `architecture` dans la meme transaction.
`POST /api/generations/:requestID/structure` enfile la generation de la formation, des modules et des plans de lessons a partir du contexte confirme.
`POST /api/generations/:requestID/structure/retry` nettoie atomiquement les donnees partielles et enfile une nouvelle tentative de structure.
`POST /api/generations/lessons/:lessonID/content` enfile la generation et la persistance du contenu d'une lesson.
`POST /api/generations/modules/:moduleID/contents` enfile le contenu manquant des lessons du module.
`GET /api/generations/:requestID/jobs` retourne les etats publics des jobs apres controle du proprietaire Clerk de la request. Le frontend peut ainsi reprendre le suivi apres rechargement et suivre les jobs enfants d'un module, sans exposer leurs payloads prives.
`DELETE /api/generations/:requestID` supprime la request, ses jobs et tout son graphe de cours par cascade. Supprimer un cours applique la meme politique. Un worker deja dans un appel OpenAI peut terminer cet appel, mais ses ecritures sont ensuite refusees car son claim a disparu.

`GET /api/courses` retourne uniquement des resumes et ne charge pas modules, lessons, contenus et activites. Le graphe complet reste reserve a `GET /api/courses/:courseID`. Les payloads ordinaires de cours/lecon ne contiennent ni corrections d'exercices, ni bonnes reponses de quiz ; la revelation explicite passe par `/api/lessons/:lessonID/solutions`.

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

La carte detaillee des dependances, des flux HTTP/jobs et les regles indiquant ou placer chaque type de code se trouvent dans [`docs/architecture.md`](docs/architecture.md).

```txt
cmd/api                         configuration, composition des dependances et cycle de vie
docs/architecture.md            guide d'architecture et parcours d'onboarding
internal/config                 chargement .env et variables d'environnement
internal/db                     ouverture du pool PostgreSQL et code sqlc genere
internal/domain                 entites et regles metier pures
internal/contract               interfaces entre couches
internal/service                use cases applicatifs
internal/shared                 helpers generiques sans dependance metier ou infrastructure
internal/infrastructure/auth    configuration de la verification des sessions Clerk
internal/infrastructure/clock   horloge systeme
internal/infrastructure/http    router, handlers, DTOs, middlewares et documentation OpenAPI Gin
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
make lint
sqlc vet
govulncheck ./...
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

Les prompts, payloads et contenus generes ne sont pas journalises. Toutes les erreurs HTTP et workers incluent une stack trace interne. Chaque reponse OpenAI journalise le modele, le prompt logique, le nombre de tokens d'entree, de cache, de raisonnement et de sortie, correles au job et a la request. L'evenement periodique `generation_queue_metrics` expose les volumes par statut, leases expires et echecs non reconcilies.

Le reconciler cherche periodiquement les jobs `failed` dont `failure_handled_at` est nul. Il rejoue de facon idempotente la transition terminale de la request/course, puis marque l'echec comme traite. La retention supprime par lots les jobs terminaux racines et met a `NULL` les sorties brutes IA apres 30 jours par defaut.

L'implementation concrete de l'IA est dans `internal/infrastructure/openai` et elle est injectee dans `cmd/api/main.go`. Les prompts sont charges depuis `PROMPTS_DIR` par `internal/infrastructure/prompts`.

## Deploiement Railway

Configurer le service avec une racine `backend-go` et le fichier `railway.toml`. Le `Dockerfile` construit l'API et une version PostgreSQL-only de Goose. Railway execute les migrations en pre-deploy, verifie `/health/ready`, puis lance l'API et le worker dans le meme processus.

Variables minimales :

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
APP_ENV=production
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=${{Postgres.DATABASE_URL}}
GOOSE_MIGRATION_DIR=/app/migrations
OPENAI_API_KEY=...
OPENAI_MODEL=...
OPENAI_MAX_OUTPUT_TOKENS=12000
OPENAI_MAX_RETRIES=0
CLERK_SECRET_KEY=...
CLERK_AUTHORIZED_PARTIES=https://votre-frontend.example
PROMPTS_DIR=/app/prompts
CORS_ALLOWED_ORIGINS=https://votre-frontend.example
GENERATION_WORKER_ENABLED=true
GENERATION_WORKER_CONCURRENCY=1
HTTP_MAX_BODY_BYTES=65536
GENERATION_RATE_LIMIT_REQUESTS=10
GENERATION_RATE_LIMIT_WINDOW=1m
GENERATION_MAX_ACTIVE_PER_USER=2
GENERATION_MAX_DAILY_PER_USER=10
GENERATION_MAX_PENDING_JOBS=500
GENERATION_JOB_RECONCILIATION_BATCH=100
GENERATION_RETENTION_ENABLED=true
GENERATION_RETENTION_PERIOD=720h
GENERATION_RETENTION_INTERVAL=6h
GENERATION_RETENTION_BATCH=100
GENERATION_METRICS_INTERVAL=1m
```

`GET /api/generations?status=&page=&pageSize=` retourne l'historique pagine de l'utilisateur authentifie, y compris les demandes sans cours ou en attente de clarification.

Railway fournit `PORT`; le backend ecoute automatiquement sur `:$PORT` lorsque `HTTP_ADDR` n'est pas defini. Garder une seule replica et une concurrence de `1` pour le premier deploiement, puis augmenter apres mesure des limites OpenAI et PostgreSQL.

En production, le demarrage est refuse si le worker est desactive, si sa concurrence differe de `1`, si CORS/Clerk contiennent une origine non HTTPS, ou si les bornes OpenAI/HTTP sont dangereuses. `X-Request-ID` est accepte lorsqu'il s'agit d'un UUID, sinon le backend en genere un et le retourne dans la reponse.

Configurer dans Railway ou dans le collecteur de logs des alertes sur :

- `/health/ready` en erreur ou indisponible ;
- `generation_queue_metrics.expired_leases > 0` ;
- `generation_queue_metrics.unreconciled_failures > 0` ;
- une profondeur `queued + retry_scheduled + running` proche de `GENERATION_MAX_PENDING_JOBS` ;
- les evenements `generation_job_failed`, `generation_job_claim_failed` et `generation_retention_*_failed` ;
- une hausse du taux de `generation_job_retry_scheduled`, de `duration_ms` ou des tokens OpenAI par `job_kind`.

Avant une promotion publique, executer en staging une generation complete avec un vrai utilisateur Clerk et le modele OpenAI configure. Verifier la pause pour clarification, la reprise, la formation finale, l'ownership, les jobs, les tokens, la duree, le cout, puis redemarrer le service pendant un job pour confirmer sa reprise.

## CI, backup et rollback

Le workflow `.github/workflows/backend-go-ci.yml` regenere sqlc, applique toutes les migrations sur PostgreSQL vide, execute vet, tests unitaires, integration, race detector, `govulncheck`, build Linux et build Docker.

Les scripts `scripts/backup-postgres.sh` et `scripts/restore-postgres.sh` utilisent le format custom de PostgreSQL. Un test de restauration doit etre execute en staging avant la premiere promotion :

```sh
DATABASE_URL="$STAGING_DATABASE_URL" sh scripts/backup-postgres.sh staging.dump
COURSE_AI_CONFIRM_RESTORE=yes DATABASE_URL="$RESTORE_DATABASE_URL" sh scripts/restore-postgres.sh staging.dump
```

Rollback applicatif : redeployer le SHA precedent. Rollback de schema : sauvegarder d'abord, verifier que la migration est reversible, puis executer `goose down-to <version>` uniquement si l'ancien binaire exige l'ancien schema. Les migrations additives compatibles peuvent rester en place pendant un rollback applicatif.

