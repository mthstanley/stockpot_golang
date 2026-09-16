-- +goose Up
CREATE TABLE auth_user (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  app_user BIGINT NOT NULL references app_user (id)
);

-- +goose Down
DROP TABLE auth_user;
