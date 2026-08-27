package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mthstanley/stockpot/internal/core/auth"
	user "github.com/mthstanley/stockpot/internal/core/user"
)

type UserHandler struct {
	userService user.Service
	authService auth.Service
}

func NewUserHandler(userService user.Service, authService auth.Service) *UserHandler {
	return &UserHandler{userService, authService}
}

type GetUser struct {
	ID   *int64 `json:"id"`
	Name string `json:"name"`
}

func convertToGetUser(d *user.User) GetUser {
	return GetUser{
		d.ID,
		d.Name,
	}
}

type CreateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (c CreateUser) convertToUser() *user.User {
	return &user.User{
		ID:   nil,
		Name: c.Name,
	}
}

func (c CreateUser) convertToCredentials() *auth.UsernameAndPassword {
	return &auth.UsernameAndPassword{
		Username: c.Username,
		Password: c.Password,
	}
}

func (h UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &PathValueParseError{Path: "/user/{id}", Err: err}
	}

	u, err := h.userService.Get(r.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	data, err := json.Marshal(convertToGetUser(u))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body", err)
	}

	return nil
}

func (h UserHandler) HandleGetAuthUser(w http.ResponseWriter, r *http.Request) error {
	authUser, err := extractAuthUser(r)
	if err != nil {
		return fmt.Errorf("auth user is missing for handler requiring auth: %w", err)
	}

	data, err := json.Marshal(convertToGetUser(&authUser.User))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body", err)
	}

	return nil
}

func (h UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) error {
	var createUser CreateUser
	err := json.NewDecoder(r.Body).Decode(&createUser)
	if err != nil {
		return errors.Join(RequestBodyDecodeError, err)
	}

	u, err := h.userService.Create(r.Context(), *createUser.convertToUser())
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	a, err := h.authService.CreateAuthUser(r.Context(), *u, *createUser.convertToCredentials())
	if err != nil {
		return fmt.Errorf("failed to create auth user: %w", err)
	}

	data, err := json.Marshal(convertToGetUser(&a.User))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body", err)
	}

	return nil
}
