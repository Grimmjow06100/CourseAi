# Course AI

Course AI is an experimental application for generating IT training courses with AI.

The current backend is a NestJS API that can:

- analyze a raw user learning request;
- generate a course architecture;
- generate lesson plans for each module;
- generate full Markdown content for each lesson;
- persist generation outputs into PostgreSQL;
- expose CRUD routes for courses, modules, and lessons.

## Local Requirements

- Node.js
- npm
- Docker Desktop
- PostgreSQL through the root `docker-compose.yml`

## Start PostgreSQL

```powershell
docker compose up -d
```

The local database is exposed on:

```txt
localhost:5433
```

## Backend Setup

```powershell
cd backend
npm install
npx prisma migrate deploy
npx prisma generate
npm run start:dev
```

The API starts on:

```txt
http://localhost:3000
```

Swagger documentation is available at:

```txt
http://localhost:3000/docs
```

## Generation Pipeline

### 1. Analyze User Request

```http
POST /course/generator/analysis
```

Persists a `GenerationRequest` and returns:

- `requestId`
- validated analysis payload

### 2. Generate Course Architecture

```http
POST /course/generator/architecture
```

Requires the `requestId` from step 1.

Persists:

- `Course`
- `CourseModule[]`
- raw architecture output

Returns:

- `courseId`
- validated architecture payload

### 3. Generate Module Lesson Plan

```http
POST /course/generator/lesson
```

Requires the `courseId` from step 2 and the module context.

Persists:

- `Lesson[]`
- raw lessons plan output on the related module
- course generation status

### 4. Generate Lesson Content

```http
POST /course/generator/lesson-content
```

Requires the `courseId`, module context, and the persisted `lessonId`.

Persists:

- `Lesson.contentMarkdown`
- raw lesson content output
- course generation status

## CRUD Routes

Courses:

```http
GET    /course
POST   /course
GET    /course/:id
PATCH  /course/:id
DELETE /course/:id
```

Modules:

```http
GET    /course/:id/modules
GET    /course/modules/:moduleId
PATCH  /course/modules/:moduleId
DELETE /course/modules/:moduleId
```

Lessons:

```http
GET    /course/modules/:moduleId/lessons
GET    /course/lessons/:lessonId
PATCH  /course/lessons/:lessonId
DELETE /course/lessons/:lessonId
```

## Verification

```powershell
cd backend
npm run build
npx eslint "src/**/*.ts"
npm test
```
