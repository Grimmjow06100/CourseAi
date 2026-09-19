-- +goose Up
ALTER TYPE generation_pipeline_status ADD VALUE IF NOT EXISTS 'partial';
ALTER TYPE course_generation_status ADD VALUE IF NOT EXISTS 'partial';

-- +goose Down
-- PostgreSQL enum values are deliberately retained for rollback compatibility.
SELECT 1;
