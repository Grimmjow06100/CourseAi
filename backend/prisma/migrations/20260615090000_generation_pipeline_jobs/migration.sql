CREATE TYPE "generation_pipeline_status" AS ENUM ('queued', 'running', 'completed', 'failed');

ALTER TABLE "generation_requests"
  ADD COLUMN "pipeline_status" "generation_pipeline_status" NOT NULL DEFAULT 'queued',
  ADD COLUMN "current_step" TEXT,
  ADD COLUMN "progress_percent" INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN "failure_message" TEXT,
  ADD COLUMN "started_at" TIMESTAMP(3),
  ADD COLUMN "completed_at" TIMESTAMP(3);

CREATE INDEX "generation_requests_pipeline_status_idx" ON "generation_requests"("pipeline_status");
