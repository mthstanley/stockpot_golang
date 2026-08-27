package http

import (
	"net/http"

	"github.com/mthstanley/stockpot/internal/core/auth"
	user "github.com/mthstanley/stockpot/internal/core/user"
)

func NewRouter(userService user.Service, authService auth.Service) *http.ServeMux {
	userHandler := NewUserHandler(userService, authService)
	validateAuth := ValidateAuth(authService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{id}", userHandler.HandleGetUser)
	mux.HandleFunc("POST /user", userHandler.HandleCreateUser)
	mux.Handle("GET /user/auth", validateAuth(http.HandlerFunc(userHandler.HandleGetAuthUser)))

	return mux
}
