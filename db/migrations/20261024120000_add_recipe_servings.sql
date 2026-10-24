-- +goose Up
ALTER TABLE recipe
ADD COLUMN min_servings INTEGER CHECK (min_servings > 0),
ADD COLUMN max_servings INTEGER CHECK (max_servings > 0),
ADD CONSTRAINT recipe_servings_range CHECK (min_servings <= max_servings);

-- +goose Down
ALTER TABLE recipe
DROP CONSTRAINT recipe_servings_range,
DROP COLUMN max_servings,
DROP COLUMN min_servings;
