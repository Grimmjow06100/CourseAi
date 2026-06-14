-- Add raw AI output storage for the lesson content generation step.
ALTER TABLE "lessons" ADD COLUMN "raw_content_output" JSONB;
