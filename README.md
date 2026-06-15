# Course AI

Course AI est une API NestJS qui génère automatiquement des formations IT avec l'IA.

Le MVP backend sait aujourd'hui :

- analyser une demande utilisateur brute ;
- générer l'architecture pédagogique d'une formation ;
- générer le plan des lessons pour chaque module ;
- générer le contenu Markdown complet des lessons ;
- persister chaque étape dans PostgreSQL avec Prisma ;
- exposer une pipeline complète asynchrone avec suivi de statut ;
- exposer des routes CRUD pour les formations, modules et lessons ;
- documenter l'API avec Swagger.

## Stack

- Backend : NestJS, TypeScript, Prisma, PostgreSQL
- IA : OpenAI SDK avec sorties structurées Zod
- Validation : `class-validator`, `class-transformer`, validation d'environnement avec Zod
- Infra locale : Docker Compose pour PostgreSQL
- Déploiement backend prévu : Railway

## Structure du repo

```txt
.
├── backend/              # API NestJS principale
│   ├── prisma/           # schema Prisma et migrations
│   ├── prompts/          # prompts .prompt.md chargés au démarrage
│   ├── src/              # code source NestJS
│   ├── Dockerfile        # image backend production
│   ├── railway.json      # configuration Railway du service backend
│   └── .env.example      # variables backend attendues
├── backend-go/           # expérimentation Go, non utilisée par le MVP NestJS
├── docker-compose.yml    # PostgreSQL local
├── .env.example          # variables Docker Compose locales
└── README.md
```

## Prérequis

- Node.js 22+
- npm
- Docker Desktop
- Une clé API OpenAI valide

## Installation locale

Depuis la racine du projet :

```powershell
Copy-Item .env.example .env
docker compose up -d
```

Puis côté backend :

```powershell
cd backend
Copy-Item .env.example .env
npm install
npm run db:migrate:deploy
npm run prisma:generate
npm run start:dev
```

L'API démarre par défaut sur :

```txt
http://localhost:3000
```

Swagger :

```txt
http://localhost:3000/docs
```

Healthcheck :

```txt
http://localhost:3000/health
```

## Variables d'environnement

Racine du projet, pour PostgreSQL local :

```env
POSTGRES_USER=course_ai
POSTGRES_PASSWORD=course_ai_password
POSTGRES_DB=course_ai
POSTGRES_PORT=5433
```

Backend, dans `backend/.env` :

```env
DATABASE_URL=postgresql://course_ai:course_ai_password@localhost:5433/course_ai
OPENAI_API_KEY=sk-your-api-key
AI_MODEL=gpt-5.4
OPENAI_MAX_RETRIES=2
PORT=3000
CORS_ORIGIN=http://localhost:5173
THROTTLE_TTL=60000
THROTTLE_LIMIT=100
```

`CORS_ORIGIN` accepte `*` ou une liste séparée par des virgules :

```env
CORS_ORIGIN=https://course-ai-front.vercel.app,http://localhost:5173
```

## Pipeline de génération

La route principale démarre une génération complète en arrière-plan :

```http
POST /course/generator/full-course
```

Payload :

```json
{
  "prompt": "Je veux apprendre Docker pour déployer une API Node.js."
}
```

Réponse `202 Accepted` :

```json
{
  "requestId": "uuid",
  "status": "QUEUED",
  "statusUrl": "/course/generator/requests/uuid/status",
  "resultUrl": "/course/generator/requests/uuid/result"
}
```

Suivre le statut :

```http
GET /course/generator/requests/:requestId/status
```

Récupérer le résultat :

```http
GET /course/generator/requests/:requestId/result
```

Relancer une génération depuis le prompt initial :

```http
POST /course/generator/requests/:requestId/retry
```

La pipeline persiste :

1. `GenerationRequest`
2. analyse utilisateur
3. `Course`
4. `CourseModule[]`
5. `Lesson[]`
6. `Lesson.contentMarkdown`

## Routes étape par étape

Ces routes servent au debug ou à une interface guidée :

```http
POST /course/generator/analysis
POST /course/generator/architecture
POST /course/generator/lesson
POST /course/generator/lesson-content
```

## CRUD formations

Lister les formations avec pagination et filtres :

```http
GET /course?page=1&pageSize=20&language=FR&status=COMPLETED&search=docker
```

Routes courses :

```http
GET    /course
POST   /course
GET    /course/:id
PATCH  /course/:id
DELETE /course/:id
```

Routes modules :

```http
GET    /course/:id/modules
GET    /course/modules/:moduleId
PATCH  /course/modules/:moduleId
DELETE /course/modules/:moduleId
```

Routes lessons :

```http
GET    /course/modules/:moduleId/lessons
GET    /course/lessons/:lessonId
PATCH  /course/lessons/:lessonId
DELETE /course/lessons/:lessonId
```

## Commandes utiles

Depuis `backend/` :

```powershell
npm run build
npm run lint
npm test
npm run test:e2e
npm run db:migrate:deploy
npm run prisma:generate
```

## Déploiement Railway

Le backend est prêt pour un service Railway dédié avec Docker.

Dans Railway :

1. Créer un projet Railway.
2. Ajouter un service PostgreSQL.
3. Ajouter un service depuis le repo GitHub.
4. Dans les settings du service backend, définir le root directory sur :

```txt
/backend
```

5. Définir le chemin du fichier config Railway sur :

```txt
/backend/railway.json
```

6. Ajouter les variables du service backend :

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
OPENAI_API_KEY=sk-your-api-key
AI_MODEL=gpt-5.4
OPENAI_MAX_RETRIES=2
CORS_ORIGIN=https://your-frontend-domain.com
THROTTLE_TTL=60000
THROTTLE_LIMIT=100
```

Railway fournit automatiquement `PORT`. Le code l'utilise déjà au démarrage.

La configuration `backend/railway.json` :

- build avec `backend/Dockerfile` ;
- lance `npm run db:migrate:deploy` en `preDeployCommand` ;
- démarre avec `npm run start:prod` ;
- vérifie `/health` avant d'activer le nouveau déploiement ;
- redémarre le service en cas d'échec.

## Points d'attention

- Les prompts doivent rester dans `backend/prompts`, car ils sont lus au démarrage de l'application.
- Ne jamais commiter `.env`.
- Les enums Prisma exposées par l'API utilisent les valeurs TypeScript, par exemple `FR`, `EN`, `COMPLETED`, `BEGINNER`.
- Si PostgreSQL local occupe déjà `5432`, ce projet utilise `5433` côté hôte par défaut.
- Si `OPENAI_API_KEY` semble incorrecte malgré `backend/.env`, vérifier les variables globales du shell, de l'IDE et de Railway : elles peuvent surcharger le fichier `.env`.

## Vérification rapide

```powershell
cd backend
npm run build
npm run lint
npm test
```

Ensuite ouvrir :

```txt
http://localhost:3000/docs
```
