package http

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/mthstanley/stockpot/internal/core/auth"
)

type contextKey string

const authUserKey contextKey = "authenticatedUser"

func ValidateAuth(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			authUser, err := authService.Validate(
				r.Context(),
				auth.UsernameAndPassword{Username: username, Password: password},
			)
			if err != nil {
				log.Println("error validating user auth:", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey, authUser)
			authenticatedRequest := r.WithContext(ctx)

			next.ServeHTTP(w, authenticatedRequest)
		})
	}
}

func extractAuthUser(r *http.Request) (*auth.AuthUser, error) {
	val := r.Context().Value(authUserKey)
	if val == nil {
		return nil, errors.New("auth user not defined in request context")
	}

	authUser, ok := val.(*auth.AuthUser)
	if !ok {
		return nil, errors.New("auth user request context object is the wrong type")
	}
	return authUser, nil
}
