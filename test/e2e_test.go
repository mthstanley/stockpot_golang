//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	migrations "github.com/mthstanley/stockpot"
	"github.com/mthstanley/stockpot/internal/adapters/postgres"
	"github.com/mthstanley/stockpot/internal/core/auth"
	"github.com/mthstanley/stockpot/internal/core/recipe"
	"github.com/mthstanley/stockpot/internal/core/user"
	"github.com/pressly/goose/v3"
)

func loadFixture(t *testing.T, filename string) []byte {
	t.Helper() // Corrects line numbers in error logs

	// Path is relative to the package directory
	path := filepath.Join("fixtures", filename)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", filename, err)
	}

	return data
}

func setupDB(ctx context.Context, t *testing.T) *pgxpool.Pool {
	dbConnString := "postgresql://postgres:postgres@localhost:5432/stockpot"

	pool, err := pgxpool.New(ctx, dbConnString)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrations.Embedded)

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	if err := goose.Up(db, "db/migrations"); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	t.Cleanup(func() {
		if err := goose.DownTo(db, "db/migrations", 0); err != nil {
			t.Errorf("failed to tear down migrations: %v", err)
		}
		db.Close()
		pool.Close()
	})

	return pool
}

func TestRecipeRepo(t *testing.T) {
	ctx := context.Background()
	db := setupDB(ctx, t)
	userRepo := postgres.NewUserRepository(db)
	userService := user.NewDefaultService(userRepo)
	authRepo := postgres.NewAuthUserRepository(db)
	authService := auth.NewDefaultService(authRepo, userService, "secret")
	recipeRepo := postgres.NewRecipeRepository(db)

	u, err := userService.Create(ctx, user.User{Name: "matt"})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = authService.CreateAuthUser(ctx, *u, auth.UsernameAndPassword{Username: "test", Password: "test"})
	if err != nil {
		t.Fatalf("failed to create auth user: %v", err)
	}

	var recipeToCreate recipe.Recipe
	f := loadFixture(t, "create_recipe.json")
	err = json.Unmarshal(f, &recipeToCreate)
	if err != nil {
		t.Fatalf("failed to read recipe fixture: %v", err)
	}
	recipeToCreate.Author.ID = u.ID
	recipeToCreate.Author.Name = u.Name

	createdRecipe, err := recipeRepo.Create(ctx, recipeToCreate)
	if err != nil {
		t.Fatalf("failed to insert recipe: %v", err)
	}
	var expectedCreatedRecipe recipe.Recipe
	f = loadFixture(t, "created_recipe.json")
	err = json.Unmarshal(f, &expectedCreatedRecipe)
	if diff := cmp.Diff(&expectedCreatedRecipe, createdRecipe); diff != "" {
		t.Errorf("create recipe mismatch (-want +got): %s\n", diff)
	}

	var updateRecipe recipe.Recipe
	f = loadFixture(t, "update_recipe.json")
	err = json.Unmarshal(f, &updateRecipe)
	if err != nil {
		t.Fatalf("failed to read recipe fixture: %v", err)
	}
	updateRecipe.Author.ID = u.ID
	updateRecipe.Author.Name = u.Name
	updatedRecipe, err := recipeRepo.Update(ctx, updateRecipe)
	if err != nil {
		t.Fatalf("failed to update recipe: %v", err)
	}
	var expectedUpdatedRecipe recipe.Recipe
	f = loadFixture(t, "updated_recipe.json")
	err = json.Unmarshal(f, &expectedUpdatedRecipe)
	if diff := cmp.Diff(&expectedUpdatedRecipe, updatedRecipe); diff != "" {
		t.Errorf("update recipe mismatch (-want +got): %s\n", diff)
	}
}
