//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	migrations "github.com/mthstanley/stockpot"
	stockpothttp "github.com/mthstanley/stockpot/internal/adapters/http"
	"github.com/mthstanley/stockpot/internal/adapters/postgres"
	"github.com/mthstanley/stockpot/internal/core/auth"
	"github.com/mthstanley/stockpot/internal/core/recipe"
	"github.com/mthstanley/stockpot/internal/core/user"
	"github.com/pressly/goose/v3"
)

func jsonEq(expectedJSON, actualJSON string) (string, error) {
	var obj1, obj2 any
	if err := json.Unmarshal([]byte(expectedJSON), &obj1); err != nil {
		return "", fmt.Errorf("error parsing first JSON: %w", err)
	}

	if err := json.Unmarshal([]byte(actualJSON), &obj2); err != nil {
		return "", fmt.Errorf("error parsing second JSON: %w", err)
	}

	diff := cmp.Diff(obj1, obj2)
	if diff != "" {
		return fmt.Sprintf("- wanted, + got: %s", diff), nil
	}
	return "", nil
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

func setupHTTPServer(ctx context.Context, t *testing.T) *httptest.Server {
	db := setupDB(ctx, t)

	userRepo := postgres.NewUserRepository(db)
	userService := user.NewDefaultService(userRepo)
	authRepo := postgres.NewAuthUserRepository(db)
	authService := auth.NewDefaultService(authRepo, userService, "secret")
	recipeRepo := postgres.NewRecipeRepository(db)
	recipeService := recipe.NewDefaultService(recipeRepo)

	router := stockpothttp.NewRouter(userService, authService, recipeService)
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

func TestNonExistantRoute(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	res, err := server.Client().Get(server.URL + "/undefined")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := "404 page not found\n"
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 404
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	res, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"id":1,"name":"joe"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 201
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestGetNonExistantUser(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	res, err := server.Client().Get(server.URL + "/user/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"Error":"Entity not found","Details":"user entity with identifier 1 not found"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 404
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestGetExistingUser(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	res, err := server.Client().Get(server.URL + "/user/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"id":1,"name":"joe"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestGetInvalidUserID(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	res, err := server.Client().Get(server.URL + "/user/foo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"Error":"Invalid path parameter","Details":"failed to parse path value for path /user/{id}: strconv.ParseInt: parsing \"foo\": invalid syntax"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 400
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestErrorMissingCredentials(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	res, err := server.Client().Get(server.URL + "/user/auth")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"Error":"Unauthorized","Details":"Request is missing auth headers"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 401
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestSuccessfulAuthentication(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest("GET", server.URL+"/user/auth", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"id":1,"name":"joe"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestTokenAuthenticationFlow(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest("GET", server.URL+"/user/token", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	data := map[string]string{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	token := data["token"]
	req, err = http.NewRequest("GET", server.URL+"/user/auth", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	actual, _ := io.ReadAll(res.Body)
	expected := `{"id":1,"name":"joe"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestCreateRecipe(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				},
				{
					"ingredient": "butter",
					"quantity": 200,
					"units": "grams",
					"preparation": "melted"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Buttered Carrots",
        "description": "Buttery carrots in a butter sauce",
        "prep_time": 360,
        "cook_time": 400,
        "inactive_time": 8600,
        "yield_quantity": 200,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 200,
                "units": "grams",
                "preparation": "diced"
            },
            {
                "id": 2,
                "ingredient": "butter",
                "quantity": 200,
                "units": "grams",
                "preparation": "melted"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Saute the carrots in the butter"
            }
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("created recipe doesn't match expected %s", diff)
	}

	expectedCode := 201
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("created recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeAddNewIngredient(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	_, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err = http.NewRequest(
		"POST",
		server.URL+"/recipe/1",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"id": 1,
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				},
				{
					"ingredient": "butter",
					"quantity": 400,
					"units": "grams",
					"preparation": "melted"
				}
			],
			"steps": [
				{
					"id": 1,
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Buttered Carrots",
        "description": "Buttery carrots in a butter sauce",
        "prep_time": 360,
        "cook_time": 400,
        "inactive_time": 8600,
        "yield_quantity": 200,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 200,
                "units": "grams",
                "preparation": "diced"
            },
            {
                "id": 2,
                "ingredient": "butter",
                "quantity": 400,
                "units": "grams",
                "preparation": "melted"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Saute the carrots in the butter"
            }
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeAddNewStep(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	_, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err = http.NewRequest(
		"POST",
		server.URL+"/recipe/1",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"id": 1,
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				}
			],
			"steps": [
				{
					"id": 1,
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				},
				{
					"ordinal": 2,
					"instruction": "Eat the carrots"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Buttered Carrots",
        "description": "Buttery carrots in a butter sauce",
        "prep_time": 360,
        "cook_time": 400,
        "inactive_time": 8600,
        "yield_quantity": 200,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 200,
                "units": "grams",
                "preparation": "diced"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Saute the carrots in the butter"
            },
			{
                "id": 2,
				"ordinal": 2,
				"instruction": "Eat the carrots"
			}
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeChangeExistingFields(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	_, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err = http.NewRequest(
		"POST",
		server.URL+"/recipe/1",
		strings.NewReader(`{
			"id": 1,
			"title": "Boiled Potatoes",
			"description": "Bolied potatoes, that's it.",
			"prep_time": 200,
			"cook_time": 200,
			"inactive_time": 200,
			"yield_quantity": 100,
			"yield_units": "grams",
			"ingredients": [
				{
					"id": 1,
					"ingredient": "potatoes",
					"quantity": 100,
					"units": "grams",
					"preparation": "sliced"
				}
			],
			"steps": [
				{
					"id": 1,
					"ordinal": 1,
					"instruction": "Boil the potatoes"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
		"title": "Boiled Potatoes",
        "description": "Bolied potatoes, that's it.",
        "prep_time": 200,
        "cook_time": 200,
        "inactive_time": 200,
        "yield_quantity": 100,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "potatoes",
                "quantity": 100,
                "units": "grams",
                "preparation": "sliced"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Boil the potatoes"
            }
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeRemoveIngredientStep(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				},
				{
					"ingredient": "butter",
					"quantity": 200,
					"units": "grams",
					"preparation": "melted"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				},
				{
					"ordinal": 2,
					"instruction": "Eat the carrots"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	_, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err = http.NewRequest(
		"POST",
		server.URL+"/recipe/1",
		strings.NewReader(`{
			"id": 1,
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"id": 1,
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				}
			],
			"steps": [
				{
					"id": 1,
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
		"title": "Buttered Carrots",
        "description": "Buttery carrots in a butter sauce",
        "prep_time": 360,
        "cook_time": 400,
        "inactive_time": 8600,
        "yield_quantity": 200,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 200,
                "units": "grams",
                "preparation": "diced"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Saute the carrots in the butter"
            }
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestDeleteRecipe(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t)

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
			"description": "Buttery carrots in a butter sauce",
			"prep_time": 360,
			"cook_time": 400,
			"inactive_time": 8600,
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 200,
					"units": "grams",
					"preparation": "diced"
				},
				{
					"ingredient": "butter",
					"quantity": 200,
					"units": "grams",
					"preparation": "melted"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Saute the carrots in the butter"
				}
			]
		}`),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()
	actual, _ := io.ReadAll(res.Body)
	expected := `{
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Buttered Carrots",
        "description": "Buttery carrots in a butter sauce",
        "prep_time": 360,
        "cook_time": 400,
        "inactive_time": 8600,
        "yield_quantity": 200,
        "yield_units": "grams",
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 200,
                "units": "grams",
                "preparation": "diced"
            },
            {
                "id": 2,
                "ingredient": "butter",
                "quantity": 200,
                "units": "grams",
                "preparation": "melted"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Saute the carrots in the butter"
            }
        ]
	}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("created recipe doesn't match expected %s", diff)
	}

	expectedCode := 201
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("created recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	req, err = http.NewRequest("DELETE", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	diff, err = jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("created recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}

	res, err = server.Client().Get(server.URL + "/recipe/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	expected = `{"Error":"Entity not found","Details":"recipe entity with identifier 1 not found"}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode = 404
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}
