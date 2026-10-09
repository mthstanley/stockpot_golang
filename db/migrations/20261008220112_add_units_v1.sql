-- +goose Up
INSERT INTO
  unit (name)
VALUES
  ('teaspoons'),
  ('tablespoons'),
  ('fluid ounces'),
  ('cups'),
  ('pints'),
  ('quarts'),
  ('gallons'),
  ('pounds'),
  ('ounces'),
  ('inches'),
  ('grams'),
  ('milliliters'),
  ('liters'),
  ('deciliters'),
  ('sprigs'),
  ('cloves'),
  ('to taste'),
  ('whole'),
  ('small'),
  ('medium'),
  ('large');

-- +goose Down
-- TODO: finish this down migration
UPDATE recipe AS r
SET
  yield_units = (
    SELECT
      id
    FROM
      unit
    WHERE
      name = 'unknown'
  )
FROM
  unit u
WHERE
  r.yield_units = u.id
  AND u.name in (
    'teaspoons',
    'tablespoons',
    'fluid ounces',
    'cups',
    'pints',
    'quarts',
    'gallons',
    'pounds',
    'ounces',
    'inches',
    'grams',
    'milliliters',
    'liters',
    'deciliters',
    'sprigs',
    'cloves',
    'to taste',
    'whole',
    'small',
    'medium',
    'large'
  );

UPDATE recipe_ingredient AS r
SET
  units = (
    SELECT
      id
    FROM
      unit
    WHERE
      name = 'unknown'
  )
FROM
  unit u
WHERE
  r.units = u.id
  AND u.name in (
    'teaspoons',
    'tablespoons',
    'fluid ounces',
    'cups',
    'pints',
    'quarts',
    'gallons',
    'pounds',
    'ounces',
    'inches',
    'grams',
    'milliliters',
    'liters',
    'deciliters',
    'sprigs',
    'cloves',
    'to taste',
    'whole',
    'small',
    'medium',
    'large'
  );

DELETE FROM unit
WHERE
  name in (
    'teaspoons',
    'tablespoons',
    'fluid ounces',
    'cups',
    'pints',
    'quarts',
    'gallons',
    'pounds',
    'ounces',
    'inches',
    'grams',
    'milliliters',
    'liters',
    'deciliters',
    'sprigs',
    'cloves',
    'to taste',
    'whole',
    'small',
    'medium',
    'large'
  );
