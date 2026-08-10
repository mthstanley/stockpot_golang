package http

import (
	"encoding/json"
	"log"
	"net/http"

	user "github.com/mthstanley/stockpot/internal/core"
)

type UserHandler struct {
	userService user.Service
}

type GetUser struct {
	ID   *int64 `json:"id"`
	Name string `json:"name"`
}

func convertToUserDTO(d *user.User) GetUser {
	return GetUser{
		d.ID,
		d.Name,
	}
}

func (h UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.userService.Get(1)
	if err != nil {
		log.Println("error retrieving user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(convertToUserDTO(u))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Println("error writing result", err)
	}
}
