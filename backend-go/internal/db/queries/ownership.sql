-- name: OwnsGenerationRequest :one
SELECT EXISTS (
  SELECT 1 FROM generation_requests
  WHERE id = @id AND clerk_user_id = @clerk_user_id
);

-- name: OwnsGenerationJob :one
SELECT EXISTS (
  SELECT 1
  FROM generation_jobs job
  JOIN generation_requests request ON request.id = job.request_id
  WHERE job.id = @id AND request.clerk_user_id = @clerk_user_id
);

-- name: OwnsCourse :one
SELECT EXISTS (
  SELECT 1
  FROM courses course
  WHERE course.id = @id AND course.clerk_user_id = @clerk_user_id
);

-- name: OwnsModule :one
SELECT EXISTS (
  SELECT 1
  FROM modules module
  JOIN courses course ON course.id = module.course_id
  WHERE module.id = @id AND course.clerk_user_id = @clerk_user_id
);

-- name: OwnsLesson :one
SELECT EXISTS (
  SELECT 1
  FROM lessons lesson
  JOIN modules module ON module.id = lesson.module_id
  JOIN courses course ON course.id = module.course_id
  WHERE lesson.id = @id AND course.clerk_user_id = @clerk_user_id
);
