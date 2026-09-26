package http

import (
	"net/http"

	"github.com/mthstanley/stockpot/internal/core/auth"
	"github.com/mthstanley/stockpot/internal/core/recipe"
	user "github.com/mthstanley/stockpot/internal/core/user"
	"github.com/rs/cors"
)

type HandlerFuncWithError = func(w http.ResponseWriter, r *http.Request) error

func NewRouter(userService user.Service, authService auth.Service, recipeService recipe.Service, apiDomain string) *http.Handler {
	userHandler := NewUserHandler(userService, authService)
	validateAuth := ValidateAuth(authService)
	recipeHandler := NewRecipeHandler(recipeService)

	mux := http.NewServeMux()

	mux.Handle("GET /user/{id}", HandleErrors(userHandler.HandleGetUser))
	mux.Handle("POST /user", HandleErrors(userHandler.HandleCreateUser))
	mux.Handle("GET /user/auth", validateAuth(HandleErrors(userHandler.HandleGetAuthUser)))
	mux.Handle("POST /user/token", validateAuth(HandleErrors(userHandler.HandleGetToken)))

	mux.Handle("GET /recipe", HandleErrors(recipeHandler.HandleGetRecipes))
	mux.Handle("POST /recipe", validateAuth(HandleErrors(recipeHandler.HandleCreateRecipe)))
	mux.Handle("GET /recipe/{id}", HandleErrors(recipeHandler.HandleGetRecipe))
	mux.Handle("POST /recipe/{id}", validateAuth(HandleErrors(recipeHandler.HandleUpdateRecipe)))
	mux.Handle("DELETE /recipe/{id}", validateAuth(HandleErrors(recipeHandler.HandleDeleteRecipe)))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{apiDomain},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            false,
	})
	router := c.Handler(mux)

	return &router
}
