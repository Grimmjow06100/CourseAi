-- name: CreateUser :one
INSERT INTO users (
  id,
  username,
  password,
  created_at,
  updated_at
)
VALUES (
  @id,
  @username,
  @password,
  @created_at,
  @updated_at
)
RETURNING id, username, password, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, password, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserByUsername :one
SELECT id, username, password, created_at, updated_at
FROM users
WHERE username = @username;

-- name: DeleteUserByID :execrows
DELETE FROM users
WHERE id = @id;
