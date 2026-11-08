package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ResponseMeta struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Pages int `json:"pages"`
	Items int `json:"items"`
}

type APIResponse[T any] struct {
	Data   T               `json:"data"`
	Errors []ErrorResponse `json:"errors,omitempty"`
	Meta   ResponseMeta    `json:"meta,omitzero"`
}

func writeJSON[T any](w http.ResponseWriter, status int, body APIResponse[T]) error {
	data, err := json.Marshal(&body)
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}
