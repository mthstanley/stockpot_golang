-- +goose Up
ALTER TABLE recipe
ADD COLUMN search_vector tsvector;

UPDATE recipe r
SET
  search_vector = setweight (to_tsvector ('english', coalesce(title, '')), 'A') || setweight (
    to_tsvector ('english', coalesce(description, '')),
    'B'
  ) || setweight (
    to_tsvector (
      'english',
      coalesce(
        (
          SELECT
            string_agg (i.name, ' ')
          FROM
            recipe_ingredient AS ri
            JOIN ingredient AS i on ri.ingredient = i.id
          WHERE
            ri.recipe = r.id
        ),
        ''
      )
    ),
    'C'
  );

CREATE INDEX recipe_search_vector_idx ON recipe USING GIN (search_vector);

-- +goose Down
DROP INDEX recipe_search_vector_idx;

ALTER TABLE recipe
DROP COLUMN search_vector;
