-- name: GetAuthUser :one
SELECT * FROM auth_user WHERE username = $1 LIMIT 1;

-- name: CreateAuthUser :one
INSERT INTO auth_user (username, password_hash, app_user) values ($1, $2, $3) RETURNING *;
