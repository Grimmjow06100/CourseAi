/*
  Warnings:

  - You are about to drop the `User` table. If the table is not empty, all the data it contains will be lost.

*/
-- CreateEnum
CREATE TYPE "course_generation_status" AS ENUM ('draft', 'structure_generated', 'lessons_generating', 'completed', 'failed');

-- CreateEnum
CREATE TYPE "course_language" AS ENUM ('fr', 'en');

-- CreateEnum
CREATE TYPE "lesson_type" AS ENUM ('theory', 'practice', 'mixed', 'quiz');

-- CreateEnum
CREATE TYPE "level" AS ENUM ('beginner', 'intermediate', 'advanced', 'expert');

-- DropTable
DROP TABLE "User";

-- CreateTable
CREATE TABLE "users" (
    "id" UUID NOT NULL,
    "username" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "courses" (
    "id" UUID NOT NULL,
    "request_id" UUID NOT NULL,
    "language" "course_language" NOT NULL,
    "status" "course_generation_status" NOT NULL DEFAULT 'draft',
    "initial_user_prompt" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "synopsis" TEXT NOT NULL,
    "target_audience" TEXT,
    "current_level" "level" NOT NULL,
    "target_level" "level" NOT NULL,
    "prerequisites" JSONB NOT NULL DEFAULT '[]',
    "goals" JSONB NOT NULL DEFAULT '[]',
    "acquired_skills" JSONB NOT NULL DEFAULT '[]',
    "generation_payload" JSONB,
    "raw_architecture_output" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "courses_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "modules" (
    "id" UUID NOT NULL,
    "course_id" UUID NOT NULL,
    "module_order" INTEGER NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "key_learning_points" JSONB NOT NULL DEFAULT '[]',
    "raw_module_output" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "modules_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "lessons" (
    "id" UUID NOT NULL,
    "module_id" UUID NOT NULL,
    "lesson_order" INTEGER NOT NULL,
    "title" TEXT NOT NULL,
    "type" "lesson_type" NOT NULL,
    "estimated_duration_minutes" INTEGER NOT NULL,
    "learning_goal" TEXT NOT NULL,
    "requires_diagram" BOOLEAN NOT NULL DEFAULT false,
    "technical_keywords" JSONB NOT NULL DEFAULT '[]',
    "content_markdown" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "lessons_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "users_username_key" ON "users"("username");

-- CreateIndex
CREATE UNIQUE INDEX "courses_request_id_key" ON "courses"("request_id");

-- CreateIndex
CREATE INDEX "courses_status_idx" ON "courses"("status");

-- CreateIndex
CREATE INDEX "courses_language_idx" ON "courses"("language");

-- CreateIndex
CREATE INDEX "modules_course_id_idx" ON "modules"("course_id");

-- CreateIndex
CREATE UNIQUE INDEX "modules_course_order_unique" ON "modules"("course_id", "module_order");

-- CreateIndex
CREATE INDEX "lessons_module_id_idx" ON "lessons"("module_id");

-- CreateIndex
CREATE UNIQUE INDEX "lessons_module_order_unique" ON "lessons"("module_id", "lesson_order");

-- AddForeignKey
ALTER TABLE "modules" ADD CONSTRAINT "modules_course_id_fkey" FOREIGN KEY ("course_id") REFERENCES "courses"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "lessons" ADD CONSTRAINT "lessons_module_id_fkey" FOREIGN KEY ("module_id") REFERENCES "modules"("id") ON DELETE CASCADE ON UPDATE CASCADE;
