-- AlterEnum
BEGIN;
CREATE TYPE "course_generation_status_new" AS ENUM ('analysis_pending', 'needs_clarification', 'analysis_completed', 'architecture_generating', 'structure_generated', 'lessons_generating', 'lessons_generated', 'content_generating', 'completed', 'failed');
ALTER TABLE "public"."courses" ALTER COLUMN "status" DROP DEFAULT;
ALTER TABLE "courses" ALTER COLUMN "status" TYPE "course_generation_status_new" USING ("status"::text::"course_generation_status_new");
ALTER TYPE "course_generation_status" RENAME TO "course_generation_status_old";
ALTER TYPE "course_generation_status_new" RENAME TO "course_generation_status";
DROP TYPE "public"."course_generation_status_old";
ALTER TABLE "courses" ALTER COLUMN "status" SET DEFAULT 'analysis_completed';
COMMIT;

-- AlterEnum
ALTER TYPE "level" ADD VALUE 'unknown';

-- AlterTable
ALTER TABLE "courses" ADD COLUMN     "final_project_constraints" JSONB NOT NULL DEFAULT '[]',
ADD COLUMN     "final_project_description" TEXT,
ADD COLUMN     "final_project_title" TEXT,
ALTER COLUMN "status" SET DEFAULT 'analysis_completed';

-- AlterTable
ALTER TABLE "modules" ADD COLUMN     "raw_lessons_plan_output" JSONB;

-- CreateTable
CREATE TABLE "generation_requests" (
    "id" UUID NOT NULL,
    "initial_user_prompt" TEXT NOT NULL,
    "is_out_of_scope" BOOLEAN NOT NULL DEFAULT false,
    "error_message" TEXT,
    "warning_message" TEXT,
    "suggested_title" TEXT,
    "short_synopsis" TEXT,
    "detected_current_level" "level",
    "detected_target_level" "level",
    "detected_goal" TEXT,
    "detected_language" "course_language",
    "clarification_questions" JSONB NOT NULL DEFAULT '[]',
    "raw_analysis_output" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "generation_requests_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "courses" ADD CONSTRAINT "courses_request_id_fkey" FOREIGN KEY ("request_id") REFERENCES "generation_requests"("id") ON DELETE CASCADE ON UPDATE CASCADE;
