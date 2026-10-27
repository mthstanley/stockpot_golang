-- +goose Up
ALTER TABLE recipe
ALTER COLUMN yield_quantity TYPE DOUBLE PRECISION;

ALTER TABLE recipe_ingredient
ALTER COLUMN quantity TYPE DOUBLE PRECISION;

-- +goose Down
ALTER TABLE recipe_ingredient
ALTER COLUMN quantity TYPE INTEGER USING round(quantity)::INTEGER;

ALTER TABLE recipe
ALTER COLUMN yield_quantity TYPE INTEGER USING round(yield_quantity)::INTEGER;
