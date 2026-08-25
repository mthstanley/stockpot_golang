-- +goose Up
CREATE TABLE app_user (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

-- +goose Down
DROP TABLE app_user;
