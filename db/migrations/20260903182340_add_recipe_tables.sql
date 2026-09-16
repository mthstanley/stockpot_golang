-- +goose Up
CREATE TABLE unit (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

INSERT INTO
  unit (name)
VALUES
  ('grams');

CREATE TABLE recipe (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  author BIGINT NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
  prep_time INTERVAL,
  cook_time INTERVAL,
  inactive_time INTERVAL,
  yield_quantity INTEGER NOT NULL,
  yield_units BIGINT NOT NULL REFERENCES unit (id),
  UNIQUE (title, author)
);

CREATE TABLE ingredient (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE recipe_ingredient (
  id BIGSERIAL PRIMARY KEY,
  recipe BIGINT NOT NULL REFERENCES recipe (id) ON DELETE CASCADE,
  ingredient BIGINT NOT NULL REFERENCES ingredient (id),
  quantity integer NOT NULL,
  units BIGINT NOT NULL REFERENCES unit (id),
  preparation TEXT NOT NULL
);

CREATE TABLE step (
  id BIGSERIAL PRIMARY KEY,
  recipe BIGINT NOT NULL REFERENCES recipe (id) ON DELETE CASCADE,
  ordinal integer NOT NULL,
  instruction TEXT NOT NULL,
  UNIQUE (recipe, ordinal)
);

-- +goose Down
DROP TABLE step;

DROP TABLE recipe_ingredient;

DROP TABLE ingredient;

DROP TABLE recipe;

DROP TABLE unit;
