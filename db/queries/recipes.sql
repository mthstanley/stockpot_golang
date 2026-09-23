-- name: InsertIngredients :many
INSERT INTO ingredient (name) 
SELECT i.name
FROM unnest(
    @names::text[]
) AS i(name)
ON CONFLICT (name) DO NOTHING RETURNING *;

-- name: InsertRecipe :one
INSERT INTO recipe (
    title,
    description,
    author,
    prep_time,
    cook_time,
    inactive_time,
    yield_quantity,
    yield_units
) VALUES ($1, $2, $3, $4, $5, $6, $7, (SELECT id FROM unit WHERE name = @yield_units_name))
RETURNING *;

-- name: InsertSteps :many
INSERT INTO step (recipe, ordinal, instruction) 
SELECT 
    unnest(@recipe_ids::bigint[]) as recipe_id,
    unnest(@ordinals::integer[]) as ordinal,
    unnest(@instructions::text[]) as instruction
RETURNING *;

-- name: InsertRecipeIngredients :many
INSERT INTO recipe_ingredient (recipe, ingredient, quantity, units, preparation) 
SELECT 
    unnest(@recipe_ids::bigint[]) as recipe_id,
    unnest(ARRAY(SELECT i.id FROM unnest(@ingredient_names::text[]) as n(ingredient_name) JOIN ingredient i ON n.ingredient_name = i.name)) as ingredient_id,
    unnest(@quantities::integer[]) as quantity,
    unnest(ARRAY(SELECT u.id FROM unnest(@unit_names::text[]) as n(unit_name) JOIN unit u ON n.unit_name = u.name)) as unit_id,
    unnest(@preparations::text[]) as preparation
RETURNING *;

-- name: UpsertRecipe :one
INSERT INTO recipe (
    id,
    title,
    description,
    author,
    prep_time,
    cook_time,
    inactive_time,
    yield_quantity,
    yield_units
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, (SELECT id FROM unit WHERE name = @yield_units_name))
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    author = EXCLUDED.author,
    prep_time = EXCLUDED.prep_time,
    cook_time = EXCLUDED.cook_time,
    inactive_time = EXCLUDED.inactive_time,
    yield_quantity = EXCLUDED.yield_quantity,
    yield_units = EXCLUDED.yield_units
RETURNING *;

-- name: UpsertSteps :many
INSERT INTO step (id, recipe, ordinal, instruction) 
SELECT 
    unnest(@ids::bigint[]) as id,
    unnest(@recipe_ids::bigint[]) as recipe_id,
    unnest(@ordinals::integer[]) as ordinal,
    unnest(@instructions::text[]) as instruction
ON CONFLICT (id) DO UPDATE SET
    recipe = EXCLUDED.recipe,
    ordinal = EXCLUDED.ordinal,
    instruction = EXCLUDED.instruction
RETURNING *;

-- name: DeleteSteps :exec
DELETE FROM step
WHERE recipe = $1 AND id != ALL(@step_ids::bigint[]);

-- name: UpsertRecipeIngredients :many
INSERT INTO recipe_ingredient (id, recipe, ingredient, quantity, units, preparation) 
SELECT 
    unnest(@ids::bigint[]) as id,
    unnest(@recipe_ids::bigint[]) as recipe_id,
    unnest(ARRAY(SELECT i.id FROM unnest(@ingredient_names::text[]) as n(ingredient_name) JOIN ingredient i ON n.ingredient_name = i.name)) as ingredient_id,
    unnest(@quantities::integer[]) as quantity,
    unnest(ARRAY(SELECT u.id FROM unnest(@unit_names::text[]) as n(unit_name) JOIN unit u ON n.unit_name = u.name)) as unit_id,
    unnest(@preparations::text[]) as preparation
ON CONFLICT (id) DO UPDATE SET
    recipe = EXCLUDED.recipe,
    ingredient = EXCLUDED.ingredient,
    quantity = EXCLUDED.quantity,
    units = EXCLUDED.units,
    preparation = EXCLUDED.preparation
RETURNING *;

-- name: DeleteRecipeIngredients :exec
DELETE FROM recipe_ingredient
WHERE recipe = $1 AND id != ALL(@ingredient_ids::bigint[]);

-- name: GetRecipe :one
SELECT
    r.id as id,
    r.title as title,
    r.description as description,
    sqlc.embed(au),
    EXTRACT(EPOCH FROM r.prep_time)::bigint_nullable as prep_time,
    EXTRACT(EPOCH FROM r.cook_time)::bigint_nullable as cook_time,
    EXTRACT(EPOCH FROM r.inactive_time)::bigint_nullable as inactive_time,
    r.yield_quantity as yield_quantity,
    sqlc.embed(ru)
FROM
    recipe AS r
    JOIN app_user au ON r.author = au.id
    JOIN unit ru ON r.yield_units = ru.id
WHERE r.id = $1;

-- name: GetRecipeSteps :many
SELECT
    s.id,
    s.recipe,
    s.ordinal,
    s.instruction
FROM step s
WHERE s.recipe = $1;

-- name: GetRecipeIngredients :many
SELECT
    ri.id,
    ri.recipe,
    sqlc.embed(i),
    ri.quantity,
    sqlc.embed(riu),
    ri.preparation
FROM recipe_ingredient ri
JOIN ingredient i ON i.id = ri.ingredient
JOIN unit riu ON ri.units = riu.id
WHERE ri.recipe = $1;

-- name: GetRecipes :many
SELECT
    r.id as id,
    r.title as title,
    r.description as description,
    sqlc.embed(au),
    EXTRACT(EPOCH FROM r.prep_time)::bigint_nullable as prep_time,
    EXTRACT(EPOCH FROM r.cook_time)::bigint_nullable as cook_time,
    EXTRACT(EPOCH FROM r.inactive_time)::bigint_nullable as inactive_time,
    r.yield_quantity as yield_quantity,
    sqlc.embed(ru)
FROM
    recipe AS r
    JOIN app_user au ON r.author = au.id
    JOIN unit ru ON r.yield_units = ru.id;

-- name: GetAllRecipeSteps :many
SELECT
    s.id,
    s.recipe,
    s.ordinal,
    s.instruction
FROM step s;

-- name: GetAllRecipeIngredients :many
SELECT
    ri.id,
    ri.recipe,
    sqlc.embed(i),
    ri.quantity,
    sqlc.embed(riu),
    ri.preparation
FROM recipe_ingredient ri
JOIN ingredient i ON i.id = ri.ingredient
JOIN unit riu ON ri.units = riu.id;

-- name: DeleteRecipe :exec
DELETE FROM recipe
WHERE id = $1;
