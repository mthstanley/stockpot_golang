package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/mthstanley/stockpot/internal/core/auth"
)

type contextKey string

const authUserKey contextKey = "authenticatedUser"

func extractJWT(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}

	return strings.TrimPrefix(authHeader, "Bearer "), true
}

func ValidateAuth(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var credential auth.UserCredentials
			if username, password, ok := r.BasicAuth(); ok {
				credential = auth.UsernameAndPassword{Username: username, Password: password}
			} else if tokenString, ok := extractJWT(r); ok {
				credential = auth.JWT{Token: tokenString}
			} else {
				errResponse := ErrorResponse{
					Error:   "Unauthorized",
					Details: "Request is missing auth headers",
				}
				data, err := json.Marshal(&errResponse)
				if err != nil {
					log.Println("error json encoding response:", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if _, err := w.Write(data); err != nil {
					log.Println("error writing result", err)
				}
				return
			}

			authUser, err := authService.Validate(
				r.Context(),
				credential,
			)
			if err != nil {
				if !errors.Is(err, auth.InvalidCredentialsError) {
					// if we got an error that would not be part of the normal
					// flow for invalid credentials then log it
					log.Println("error validating user auth:", err)
				}
				errResponse := ErrorResponse{
					Error:   "Unauthorized",
					Details: "Request with invalid credentials",
				}
				data, err := json.Marshal(&errResponse)
				if err != nil {
					log.Println("error json encoding response:", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if _, err := w.Write(data); err != nil {
					log.Println("error writing result", err)
				}
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
