package http

import (
	"net/http"

	"github.com/mthstanley/stockpot/internal/core/auth"
	user "github.com/mthstanley/stockpot/internal/core/user"
)

type HandlerFuncWithError = func(w http.ResponseWriter, r *http.Request) error

func NewRouter(userService user.Service, authService auth.Service) *http.ServeMux {
	userHandler := NewUserHandler(userService, authService)
	validateAuth := ValidateAuth(authService)

	mux := http.NewServeMux()
	mux.Handle("GET /user/{id}", HandleErrors(userHandler.HandleGetUser))
	mux.Handle("POST /user", HandleErrors(userHandler.HandleCreateUser))
	mux.Handle("GET /user/auth", validateAuth(HandleErrors(userHandler.HandleGetAuthUser)))
	mux.Handle("GET /user/token", validateAuth(HandleErrors(userHandler.HandleGetToken)))

	return mux
}
