-- name: GetAppUser :one
SELECT * FROM app_user WHERE id = $1 LIMIT 1;

-- name: CreateAppUser :one
INSERT INTO app_user (name) values ($1) RETURNING *;
