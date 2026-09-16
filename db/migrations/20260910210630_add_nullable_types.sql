-- +goose Up
CREATE DOMAIN bigint_nullable AS bigint;

-- +goose Down
DROP DOMAIN bigint_nullable;
