package http

import (
	"encoding/json"
	"log"
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

func (h UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		log.Println("invalid path parameter:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	u, err := h.userService.Get(r.Context(), id)
	if err != nil {
		log.Println("error retrieving user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(convertToGetUser(u))
	if err != nil {
		log.Println("error json encoding response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Println("error writing result", err)
	}
}

func (h UserHandler) HandleGetAuthUser(w http.ResponseWriter, r *http.Request) {
	authUser, err := extractAuthUser(r)
	if err != nil {
		log.Println("auth user is missing for handler requiring auth:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(convertToGetUser(&authUser.User))
	if err != nil {
		log.Println("error json encoding response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Println("error writing result", err)
	}
}

func (h UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var createUser CreateUser
	err := json.NewDecoder(r.Body).Decode(&createUser)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	u, err := h.userService.Create(r.Context(), *createUser.convertToUser())
	if err != nil {
		log.Println("error creating user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	a, err := h.authService.CreateAuthUser(r.Context(), *u, *createUser.convertToCredentials())
	if err != nil {
		log.Println("error creating auth user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(convertToGetUser(&a.User))
	if err != nil {
		log.Println("error json encoding response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		log.Println("error writing result", err)
	}
}
