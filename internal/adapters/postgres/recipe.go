package postgres

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/mthstanley/stockpot/db/sqlc"
	"github.com/mthstanley/stockpot/internal/core"
	"github.com/mthstanley/stockpot/internal/core/recipe"
)

type RecipeRepository struct {
	db *pgxpool.Pool
}

func NewRecipeRepository(db *pgxpool.Pool) *RecipeRepository {
	return &RecipeRepository{db}
}

func (r RecipeRepository) Get(ctx context.Context) ([]recipe.Recipe, error) {
	queries := db.New(r.db)
	recipes := make(map[int64]recipe.Recipe)

	dbRecipes, err := queries.GetRecipes(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []recipe.Recipe{}, nil
		}
		return nil, fmt.Errorf("failed to retrieve recipe: %w", err)
	}

	for _, r := range dbRecipes {
		recipes[r.ID] = recipe.Recipe{
			ID:            &r.ID,
			Title:         r.Title,
			Description:   convertText(r.Description),
			Author:        *convertToUserDomainModel(r.AppUser),
			PrepTime:      convertEpochSeconds(r.PrepTime),
			CookTime:      convertEpochSeconds(r.CookTime),
			InactiveTime:  convertEpochSeconds(r.InactiveTime),
			YieldQuantity: int(r.YieldQuantity),
			YieldUnits:    recipe.Unit{ID: &r.Unit.ID, Name: r.Unit.Name},
			Ingredients:   []recipe.RecipeIngredient{},
			Steps:         []recipe.Step{},
		}
	}

	dbRecipeSteps, err := queries.GetAllRecipeSteps(ctx)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to retrieve recipe steps: %w", err)
		}
	}

	for _, s := range dbRecipeSteps {
		if val, ok := recipes[s.Recipe]; ok {
			val.Steps = append(val.Steps, recipe.Step{
				ID:          &s.ID,
				RecipeID:    &s.Recipe,
				Ordinal:     int(s.Ordinal),
				Instruction: s.Instruction,
			})
		} else {
			return nil, errors.New("missing recipe for step")
		}
	}

	dbRecipeIngredients, err := queries.GetAllRecipeIngredients(ctx)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to retrieve recipe ingredients: %w", err)
		}
	}

	for _, i := range dbRecipeIngredients {
		if val, ok := recipes[i.Recipe]; ok {
			val.Ingredients = append(val.Ingredients, recipe.RecipeIngredient{
				ID:          &i.ID,
				RecipeID:    &i.Recipe,
				Ingredient:  recipe.Ingredient{ID: &i.Ingredient.ID, Name: i.Ingredient.Name},
				Quantity:    int(i.Quantity),
				Units:       recipe.Unit{ID: &i.Unit.ID, Name: i.Unit.Name},
				Preparation: i.Preparation,
			})
		} else {
			return nil, errors.New("missing recipe for recipe ingredient")
		}
	}

	return slices.Collect(maps.Values(recipes)), nil
}

func (r RecipeRepository) GetByID(ctx context.Context, id int64) (*recipe.Recipe, error) {
	queries := db.New(r.db)
	var result recipe.Recipe

	dbRecipe, err := queries.GetRecipe(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &core.EntityNotFound{Type: recipe.EntityType, Ident: strconv.FormatInt(id, 10), Err: err}
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	result = recipe.Recipe{
		ID:            &dbRecipe.ID,
		Title:         dbRecipe.Title,
		Description:   convertText(dbRecipe.Description),
		Author:        *convertToUserDomainModel(dbRecipe.AppUser),
		PrepTime:      convertEpochSeconds(dbRecipe.PrepTime),
		CookTime:      convertEpochSeconds(dbRecipe.CookTime),
		InactiveTime:  convertEpochSeconds(dbRecipe.InactiveTime),
		YieldQuantity: int(dbRecipe.YieldQuantity),
		YieldUnits:    recipe.Unit{ID: &dbRecipe.Unit.ID, Name: dbRecipe.Unit.Name},
		Ingredients:   []recipe.RecipeIngredient{},
		Steps:         []recipe.Step{},
	}

	dbRecipeSteps, err := queries.GetRecipeSteps(ctx, id)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to retrieve recipe steps: %w", err)
		}
	}

	for _, s := range dbRecipeSteps {
		result.Steps = append(result.Steps, recipe.Step{
			ID:          &s.ID,
			RecipeID:    &s.Recipe,
			Ordinal:     int(s.Ordinal),
			Instruction: s.Instruction,
		})
	}

	dbRecipeIngredients, err := queries.GetRecipeIngredients(ctx, id)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to retrieve recipe ingredients: %w", err)
		}
	}

	for _, i := range dbRecipeIngredients {
		result.Ingredients = append(result.Ingredients, recipe.RecipeIngredient{
			ID:          &i.ID,
			RecipeID:    &i.Recipe,
			Ingredient:  recipe.Ingredient{ID: &i.Ingredient.ID, Name: i.Ingredient.Name},
			Quantity:    int(i.Quantity),
			Units:       recipe.Unit{ID: &i.Unit.ID, Name: i.Unit.Name},
			Preparation: i.Preparation,
		})
	}

	return &result, nil
}

func (r RecipeRepository) Create(ctx context.Context, rec recipe.Recipe) (*recipe.Recipe, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create db transaction: %w", err)
	}

	defer tx.Rollback(ctx)
	qtx := db.New(r.db).WithTx(tx)

	var ingredientNames []string
	for _, i := range rec.Ingredients {
		ingredientNames = append(ingredientNames, i.Ingredient.Name)
	}
	_, err = qtx.InsertIngredients(ctx, ingredientNames)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new ingredients: %w", err)
	}

	recipeResult, err := qtx.InsertRecipe(ctx, db.InsertRecipeParams{
		Title:          rec.Title,
		Description:    convertToText(rec.Description),
		Author:         *rec.Author.ID,
		PrepTime:       convertToInterval(rec.PrepTime),
		CookTime:       convertToInterval(rec.CookTime),
		InactiveTime:   convertToInterval(rec.InactiveTime),
		YieldQuantity:  int32(rec.YieldQuantity),
		YieldUnitsName: rec.YieldUnits.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe: %w", err)
	}

	var steps db.InsertStepsParams
	for _, s := range rec.Steps {
		steps.RecipeIds = append(steps.RecipeIds, recipeResult.ID)
		steps.Ordinals = append(steps.Ordinals, int32(s.Ordinal))
		steps.Instructions = append(steps.Instructions, s.Instruction)
	}
	_, err = qtx.InsertSteps(ctx, steps)
	if err != nil {
		return nil, fmt.Errorf("failed to insert steps: %w", err)
	}

	var ingredients db.InsertRecipeIngredientsParams
	for _, i := range rec.Ingredients {
		ingredients.RecipeIds = append(ingredients.RecipeIds, recipeResult.ID)
		ingredients.IngredientNames = append(ingredients.IngredientNames, i.Ingredient.Name)
		ingredients.Preparations = append(ingredients.Preparations, i.Preparation)
		ingredients.Quantities = append(ingredients.Quantities, int32(i.Quantity))
		ingredients.UnitNames = append(ingredients.UnitNames, i.Units.Name)
	}
	_, err = qtx.InsertRecipeIngredients(ctx, ingredients)
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe ingredients: %w", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, recipeResult.ID)
}

func (r RecipeRepository) Update(ctx context.Context, rec recipe.Recipe) (*recipe.Recipe, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create db transaction: %w", err)
	}

	defer tx.Rollback(ctx)
	qtx := db.New(r.db).WithTx(tx)

	var ingredientNames []string
	for _, i := range rec.Ingredients {
		ingredientNames = append(ingredientNames, i.Ingredient.Name)
	}
	_, err = qtx.InsertIngredients(ctx, ingredientNames)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new ingredients: %w", err)
	}

	recipeResult, err := qtx.UpsertRecipe(ctx, db.UpsertRecipeParams{
		ID:             *rec.ID,
		Title:          rec.Title,
		Description:    convertToText(rec.Description),
		Author:         *rec.Author.ID,
		PrepTime:       convertToInterval(rec.PrepTime),
		CookTime:       convertToInterval(rec.CookTime),
		InactiveTime:   convertToInterval(rec.InactiveTime),
		YieldQuantity:  int32(rec.YieldQuantity),
		YieldUnitsName: rec.YieldUnits.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe: %w", err)
	}

	var usteps db.UpsertStepsParams
	var isteps db.InsertStepsParams
	for _, s := range rec.Steps {
		if s.ID != nil {
			usteps.Ids = append(usteps.Ids, *s.ID)
			usteps.RecipeIds = append(usteps.RecipeIds, recipeResult.ID)
			usteps.Ordinals = append(usteps.Ordinals, int32(s.Ordinal))
			usteps.Instructions = append(usteps.Instructions, s.Instruction)
		} else {
			isteps.RecipeIds = append(isteps.RecipeIds, recipeResult.ID)
			isteps.Ordinals = append(isteps.Ordinals, int32(s.Ordinal))
			isteps.Instructions = append(isteps.Instructions, s.Instruction)
		}
	}
	upsertedSteps, err := qtx.UpsertSteps(ctx, usteps)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert steps: %w", err)
	}
	insertedSteps, err := qtx.InsertSteps(ctx, isteps)
	if err != nil {
		return nil, fmt.Errorf("failed to insert steps: %w", err)
	}

	deleteSteps := db.DeleteStepsParams{
		Recipe: recipeResult.ID,
	}
	for _, s := range upsertedSteps {
		deleteSteps.StepIds = append(deleteSteps.StepIds, s.ID)
	}
	for _, s := range insertedSteps {
		deleteSteps.StepIds = append(deleteSteps.StepIds, s.ID)
	}
	err = qtx.DeleteSteps(ctx, deleteSteps)
	if err != nil {
		return nil, fmt.Errorf("failed to delete removed steps: %w", err)
	}

	var uingredients db.UpsertRecipeIngredientsParams
	var iingredients db.InsertRecipeIngredientsParams
	for _, i := range rec.Ingredients {
		if i.ID != nil {
			uingredients.Ids = append(uingredients.Ids, *i.ID)
			uingredients.RecipeIds = append(uingredients.RecipeIds, recipeResult.ID)
			uingredients.IngredientNames = append(uingredients.IngredientNames, i.Ingredient.Name)
			uingredients.Preparations = append(uingredients.Preparations, i.Preparation)
			uingredients.Quantities = append(uingredients.Quantities, int32(i.Quantity))
			uingredients.UnitNames = append(uingredients.UnitNames, i.Units.Name)
		} else {
			iingredients.RecipeIds = append(iingredients.RecipeIds, recipeResult.ID)
			iingredients.IngredientNames = append(iingredients.IngredientNames, i.Ingredient.Name)
			iingredients.Preparations = append(iingredients.Preparations, i.Preparation)
			iingredients.Quantities = append(iingredients.Quantities, int32(i.Quantity))
			iingredients.UnitNames = append(iingredients.UnitNames, i.Units.Name)
		}
	}
	upsertedIngredients, err := qtx.UpsertRecipeIngredients(ctx, uingredients)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert recipe ingredients: %w", err)
	}
	insertedIngredients, err := qtx.InsertRecipeIngredients(ctx, iingredients)
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe ingredients: %w", err)
	}

	deleteIngredients := db.DeleteRecipeIngredientsParams{
		Recipe: recipeResult.ID,
	}
	for _, i := range upsertedIngredients {
		deleteIngredients.IngredientIds = append(deleteIngredients.IngredientIds, i.ID)
	}
	for _, i := range insertedIngredients {
		deleteIngredients.IngredientIds = append(deleteIngredients.IngredientIds, i.ID)
	}
	err = qtx.DeleteRecipeIngredients(ctx, deleteIngredients)
	if err != nil {
		return nil, fmt.Errorf("failed to delete removed ingredients: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, recipeResult.ID)
}

func (r RecipeRepository) DeleteByID(ctx context.Context, id int64) (*recipe.Recipe, error) {
	result, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe before deleting: %w", err)
	}

	queries := db.New(r.db)
	err = queries.DeleteRecipe(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete recipe: %w", err)
	}

	return result, nil
}
