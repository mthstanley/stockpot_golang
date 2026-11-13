//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	// collect table statistics so the query planner behaves as it would against
	// a long-lived database, rather than masking order-dependent queries
	if _, err := pool.Exec(ctx, "ANALYZE"); err != nil {
		t.Fatalf("failed to analyze db: %v", err)
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

type serverConfig struct {
	disableUserSignup bool
}

func setupHTTPServer(ctx context.Context, t *testing.T, config serverConfig) *httptest.Server {
	db := setupDB(ctx, t)

	userRepo := postgres.NewUserRepository(db)
	userService := user.NewDefaultService(userRepo)
	authRepo := postgres.NewAuthUserRepository(db)
	authService := auth.NewDefaultService(authRepo, userService, "secret")
	recipeRepo := postgres.NewRecipeRepository(db)
	recipeService := recipe.NewDefaultService(recipeRepo)

	router := stockpothttp.NewRouter(userService, authService, recipeService, "localhost", config.disableUserSignup)
	server := httptest.NewServer(*router)

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

func TestNonExistantRoute(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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

func TestDisableUserSignup(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: true})

	res, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	res, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"data":{"id":1,"name":"joe"}}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	res, err := server.Client().Get(server.URL + "/user/1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"data":null,"errors":[{"error":"Entity not found","details":"user entity with identifier 1 not found"}]}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
	expected := `{"data":{"id":1,"name":"joe"}}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	res, err := server.Client().Get(server.URL + "/user/foo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"data":null,"errors":[{"error":"Invalid path parameter","details":"failed to parse path value for path /user/{id}: strconv.ParseInt: parsing \"foo\": invalid syntax"}]}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	res, err := server.Client().Get(server.URL + "/user/auth")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"data":null,"errors":[{"error":"Unauthorized","details":"Request is missing auth headers"}]}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
	expected := `{"data":{"id":1,"name":"joe"}}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest("POST", server.URL+"/user/token", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.SetBasicAuth("test", "secret")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	data := map[string]map[string]string{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	token := data["data"]["token"]
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
	expected := `{"data":{"id":1,"name":"joe"}}`
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
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"yield_quantity": 1.5,
			"yield_units": "grams",
			"min_servings": 4,
			"max_servings": 6,
			"ingredients": [
				{
					"ingredient": "carrots",
					"quantity": 0.5,
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
	expected := `{"data": {
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
        "yield_quantity": 1.5,
        "yield_units": "grams",
        "min_servings": 4,
        "max_servings": 6,
        "ingredients": [
            {
                "id": 1,
                "ingredient": "carrots",
                "quantity": 0.5,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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
}

func TestUpdateRecipeAddNewIngredient(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"min_servings": 4,
			"max_servings": 6,
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
			"min_servings": 4,
			"max_servings": 6,
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
	expected := `{"data": {
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
        "min_servings": 4,
        "max_servings": 6,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeAddNewStep(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"min_servings": 4,
			"max_servings": 6,
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
			"min_servings": 4,
			"max_servings": 6,
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
	expected := `{"data": {
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
        "min_servings": 4,
        "max_servings": 6,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeChangeExistingFields(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"min_servings": 4,
			"max_servings": 6,
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
			"min_servings": 2,
			"max_servings": 3,
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
	expected := `{"data": {
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
        "min_servings": 2,
        "max_servings": 3,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestUpdateRecipeRemoveIngredientStep(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"min_servings": 4,
			"max_servings": 6,
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
			"min_servings": 4,
			"max_servings": 6,
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
	expected := `{"data": {
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
        "min_servings": 4,
        "max_servings": 6,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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
		t.Errorf("updated recipe doesn't match expected %s", diff)
	}

	expectedCode = 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestDeleteRecipe(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

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
			"min_servings": null,
			"max_servings": null,
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
	expected := `{"data": {
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
        "min_servings": null,
        "max_servings": null,
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
	}}`

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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
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

	req, err = http.NewRequest("GET", server.URL+"/recipe/1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ = io.ReadAll(res.Body)
	expected = `{"data":null,"errors":[{"error":"Entity not found","details":"recipe entity with identifier 1 not found"}]}`
	if string(actual) != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	expectedCode = 404
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

// Ingredient and unit names are looked up by joining against their tables, and
// once those tables have statistics (see setupDB) postgres prefers a hash join
// that returns rows in table order rather than input order. These names are
// deliberately ordered opposite to their table ids (butter/carrots already
// exist as 2/1, grams/cups are 12/5) so a lookup that doesn't preserve input
// order will mismatch them with their quantities.
func TestCreateRecipePreservesIngredientOrder(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
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
					"instruction": "Mix it all together"
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
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Carrot Butter",
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "butter",
					"quantity": 100,
					"units": "grams",
					"preparation": "softened"
				},
				{
					"ingredient": "carrots",
					"quantity": 2,
					"units": "cups",
					"preparation": "grated"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Mix it all together"
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
	expected := `{"data": {
		"id": 2,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Carrot Butter",
        "description": null,
        "prep_time": null,
        "cook_time": null,
        "inactive_time": null,
        "yield_quantity": 200,
        "yield_units": "grams",
        "min_servings": null,
        "max_servings": null,
        "ingredients": [
            {
                "id": 3,
                "ingredient": "butter",
                "quantity": 100,
                "units": "grams",
                "preparation": "softened"
            },
            {
                "id": 4,
                "ingredient": "carrots",
                "quantity": 2,
                "units": "cups",
                "preparation": "grated"
            }
        ],
        "steps": [
            {
                "id": 2,
                "ordinal": 1,
                "instruction": "Mix it all together"
            }
        ]
	}}`

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
}

func TestUpdateRecipePreservesIngredientOrder(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	req, err := http.NewRequest(
		"POST",
		server.URL+"/recipe",
		strings.NewReader(`{
			"title": "Buttered Carrots",
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
					"instruction": "Mix it all together"
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
			"yield_quantity": 200,
			"yield_units": "grams",
			"ingredients": [
				{
					"id": 1,
					"ingredient": "butter",
					"quantity": 100,
					"units": "grams",
					"preparation": "softened"
				},
				{
					"id": 2,
					"ingredient": "carrots",
					"quantity": 2,
					"units": "cups",
					"preparation": "grated"
				}
			],
			"steps": [
				{
					"id": 1,
					"ordinal": 1,
					"instruction": "Mix it all together"
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
	expected := `{"data": {
		"id": 1,
        "author": {
            "id": 1,
            "name": "joe"
        },
        "title": "Buttered Carrots",
        "description": null,
        "prep_time": null,
        "cook_time": null,
        "inactive_time": null,
        "yield_quantity": 200,
        "yield_units": "grams",
        "min_servings": null,
        "max_servings": null,
        "ingredients": [
            {
                "id": 1,
                "ingredient": "butter",
                "quantity": 100,
                "units": "grams",
                "preparation": "softened"
            },
            {
                "id": 2,
                "ingredient": "carrots",
                "quantity": 2,
                "units": "cups",
                "preparation": "grated"
            }
        ],
        "steps": [
            {
                "id": 1,
                "ordinal": 1,
                "instruction": "Mix it all together"
            }
        ]
	}}`

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
}

func TestGetRecipes(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, body := range []string{
		`{
			"title": "Buttered Carrots",
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
		}`,
		`{
			"title": "Boiled Potatoes",
			"yield_quantity": 100,
			"yield_units": "grams",
			"ingredients": [
				{
					"ingredient": "potatoes",
					"quantity": 100,
					"units": "grams",
					"preparation": "sliced"
				}
			],
			"steps": [
				{
					"ordinal": 1,
					"instruction": "Boil the potatoes"
				},
				{
					"ordinal": 2,
					"instruction": "Eat the potatoes"
				}
			]
		}`,
	} {
		req, err := http.NewRequest("POST", server.URL+"/recipe", strings.NewReader(body))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		req.SetBasicAuth("test", "secret")
		_, err = server.Client().Do(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	req, err := http.NewRequest("GET", server.URL+"/recipe", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	actual, _ := io.ReadAll(res.Body)
	expected := `{"data": [
		{
			"id": 1,
			"author": {
				"id": 1,
				"name": "joe"
			},
			"title": "Buttered Carrots",
			"description": null,
			"prep_time": null,
			"cook_time": null,
			"inactive_time": null,
			"yield_quantity": 200,
			"yield_units": "grams",
			"min_servings": null,
			"max_servings": null,
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
		},
		{
			"id": 2,
			"author": {
				"id": 1,
				"name": "joe"
			},
			"title": "Boiled Potatoes",
			"description": null,
			"prep_time": null,
			"cook_time": null,
			"inactive_time": null,
			"yield_quantity": 100,
			"yield_units": "grams",
			"min_servings": null,
			"max_servings": null,
			"ingredients": [
				{
					"id": 3,
					"ingredient": "potatoes",
					"quantity": 100,
					"units": "grams",
					"preparation": "sliced"
				}
			],
			"steps": [
				{
					"id": 2,
					"ordinal": 1,
					"instruction": "Boil the potatoes"
				},
				{
					"id": 3,
					"ordinal": 2,
					"instruction": "Eat the potatoes"
				}
			]
		}
	],
	"meta": {
		"page": 1,
		"size": 10,
		"pages": 1,
		"items": 2
	}}`

	diff, err := jsonEq(expected, string(actual))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff != "" {
		t.Errorf("recipes don't match expected %s", diff)
	}

	expectedCode := 200
	if res.StatusCode != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

type paginatedRecipes struct {
	Data []struct {
		ID int64 `json:"id"`
	} `json:"data"`
	Meta stockpothttp.ResponseMeta `json:"meta"`
}

func getRecipesPage(t *testing.T, server *httptest.Server, query string) (int, paginatedRecipes) {
	t.Helper()

	req, err := http.NewRequest("GET", server.URL+"/recipe"+query, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer res.Body.Close()

	var body paginatedRecipes
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	return res.StatusCode, body
}

func TestGetRecipesPagination(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for i := 1; i <= 5; i++ {
		body := fmt.Sprintf(`{
			"title": "Recipe %d",
			"yield_quantity": 1,
			"yield_units": "grams",
			"ingredients": [],
			"steps": []
		}`, i)
		req, err := http.NewRequest("POST", server.URL+"/recipe", strings.NewReader(body))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		req.SetBasicAuth("test", "secret")
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		res.Body.Close()
		if res.StatusCode != 201 {
			t.Fatalf("expected status code 201 creating recipe, got %d", res.StatusCode)
		}
	}

	tests := []struct {
		name        string
		query       string
		expectedIDs []int64
		expected    stockpothttp.ResponseMeta
	}{
		{"defaults", "", []int64{1, 2, 3, 4, 5}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 5}},
		{"first page", "?page=1&size=2", []int64{1, 2}, stockpothttp.ResponseMeta{Page: 1, Size: 2, Pages: 3, Items: 5}},
		{"middle page", "?page=2&size=2", []int64{3, 4}, stockpothttp.ResponseMeta{Page: 2, Size: 2, Pages: 3, Items: 5}},
		{"last partial page", "?page=3&size=2", []int64{5}, stockpothttp.ResponseMeta{Page: 3, Size: 2, Pages: 3, Items: 5}},
		{"past the end", "?page=4&size=2", []int64{}, stockpothttp.ResponseMeta{Page: 4, Size: 2, Pages: 3, Items: 5}},
		{"max size", "?size=100", []int64{1, 2, 3, 4, 5}, stockpothttp.ResponseMeta{Page: 1, Size: 100, Pages: 1, Items: 5}},
		// offset (page-1)*size would overflow int32
		{"max page", "?page=2147483647&size=100", []int64{}, stockpothttp.ResponseMeta{Page: 2147483647, Size: 100, Pages: 1, Items: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := getRecipesPage(t, server, tt.query)
			if status != 200 {
				t.Errorf("expected status code 200, got %d", status)
			}

			ids := []int64{}
			for _, r := range body.Data {
				ids = append(ids, r.ID)
			}
			if diff := cmp.Diff(tt.expectedIDs, ids); diff != "" {
				t.Errorf("recipe ids don't match expected (- wanted, + got): %s", diff)
			}
			if diff := cmp.Diff(tt.expected, body.Meta); diff != "" {
				t.Errorf("meta doesn't match expected (- wanted, + got): %s", diff)
			}
		})
	}
}

func TestGetRecipesEmpty(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	status, body := getRecipesPage(t, server, "")
	if status != 200 {
		t.Errorf("expected status code 200, got %d", status)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected no recipes, got %d", len(body.Data))
	}
	expected := stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 0}
	if diff := cmp.Diff(expected, body.Meta); diff != "" {
		t.Errorf("meta doesn't match expected (- wanted, + got): %s", diff)
	}
}

func TestGetRecipesInvalidPagination(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	tests := []struct {
		name    string
		query   string
		details string
	}{
		{"zero page", "?page=0", "failed to parse query value 0 for key page: page must be between 1 and 2147483647"},
		{"negative page", "?page=-1", "failed to parse query value -1 for key page: page must be between 1 and 2147483647"},
		{"non-numeric page", "?page=abc", `failed to parse query value abc for key page: strconv.ParseInt: parsing "abc": invalid syntax`},
		{"page out of int32 range", "?page=2147483648", `failed to parse query value 2147483648 for key page: strconv.ParseInt: parsing "2147483648": value out of range`},
		{"zero size", "?size=0", "failed to parse query value 0 for key size: size must be between 1 and 100"},
		{"size over max", "?size=101", "failed to parse query value 101 for key size: size must be between 1 and 100"},
		{"fractional size", "?size=1.5", `failed to parse query value 1.5 for key size: strconv.ParseInt: parsing "1.5": invalid syntax`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := server.Client().Get(server.URL + "/recipe" + tt.query)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			defer res.Body.Close()

			actual, _ := io.ReadAll(res.Body)
			expected, _ := json.Marshal(map[string]any{
				"data":   nil,
				"errors": []map[string]string{{"error": "Invalid query parameter", "details": tt.details}},
			})
			diff, err := jsonEq(string(expected), string(actual))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if diff != "" {
				t.Errorf("error response doesn't match expected %s", diff)
			}

			if res.StatusCode != 400 {
				t.Errorf("expected status code 400, got %d", res.StatusCode)
			}
		})
	}
}

func TestGetRecipesSearch(t *testing.T) {
	ctx := context.Background()
	server := setupHTTPServer(ctx, t, serverConfig{disableUserSignup: false})

	_, err := server.Client().Post(server.URL+"/user", "application/json", strings.NewReader(`{"username": "test", "password": "secret", "name": "joe"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	idToChange := 4
	recipes := []struct {
		id          *int
		title       string
		description string
		ingredients string
	}{
		{nil, "carrot", "foo", `[{"quantity": 1, "units":"grams", "ingredient":"potato"}]`},
		{nil, "potato", "bar", `[{"quantity": 1, "units":"grams", "ingredient":"potato"}]`},
		{nil, "butter", "bat", `[]`},
		{nil, "test", "baz", `[{"quantity": 1, "units":"grams", "ingredient":"potato"}]`},
		{&idToChange, "test", "baz", `[]`}, // remove ingredients from recipe to test that search vector is properly updated
	}
	for _, item := range recipes {
		body := fmt.Sprintf(`{
			"title": "%s",
			"description": "%s",
			"yield_quantity": 1,
			"yield_units": "grams",
			"ingredients": %s,
			"steps": []
		}`, item.title, item.description, item.ingredients)
		targetURL := server.URL + "/recipe"
		targetStatusCode := 201
		if item.id != nil {
			targetURL = targetURL + "/" + strconv.Itoa(*item.id)
			targetStatusCode = 200
		}
		req, err := http.NewRequest("POST", targetURL, strings.NewReader(body))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		req.SetBasicAuth("test", "secret")
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		res.Body.Close()
		if res.StatusCode != targetStatusCode {
			t.Fatalf("expected status code %d creating recipe, got %d", targetStatusCode, res.StatusCode)
		}
	}

	tests := []struct {
		name        string
		query       string
		expectedIDs []int64
		expected    stockpothttp.ResponseMeta
	}{
		{"defaults", "", []int64{1, 2, 3, 4}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 4}},
		{"single result", "?search=carrot", []int64{1}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 1}},
		{"multiple ranked results", "?search=potato", []int64{2, 1}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 2}},
		{"originally no ingredients", "?search=butter", []int64{3}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 1}},
		{"updated to no ingredients", "?search=test", []int64{4}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 1}},
		{"no matches", "?search=hello", []int64{}, stockpothttp.ResponseMeta{Page: 1, Size: 10, Pages: 1, Items: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := getRecipesPage(t, server, tt.query)
			if status != 200 {
				t.Errorf("expected status code 200, got %d", status)
			}

			ids := []int64{}
			for _, r := range body.Data {
				ids = append(ids, r.ID)
			}
			if diff := cmp.Diff(tt.expectedIDs, ids); diff != "" {
				t.Errorf("recipe ids don't match expected (- wanted, + got): %s", diff)
			}
			if diff := cmp.Diff(tt.expected, body.Meta); diff != "" {
				t.Errorf("meta doesn't match expected (- wanted, + got): %s", diff)
			}
		})
	}
}
