package http

import (
	"net/http"

	user "github.com/mthstanley/stockpot/internal/core"
)

func NewRouter(userService user.Service) *http.ServeMux {
	userHandler := UserHandler{
		userService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /user", userHandler.GetUser)

	return mux
}
