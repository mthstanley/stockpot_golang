package http

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
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

func parseQueryInt32(r *http.Request, key string, def, lo, hi int32) (int32, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, nil
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, &QueryValueParseError{Key: key, Value: raw, Err: err}
	}
	if value < int64(lo) || value > int64(hi) {
		return 0, &QueryValueParseError{Key: key, Value: raw, Err: fmt.Errorf("%s must be between %d and %d", key, lo, hi)}
	}

	return int32(value), nil
}

func parsePagination(r *http.Request) (page int32, size int32, err error) {
	page, err = parseQueryInt32(r, "page", 1, 1, math.MaxInt32)
	if err != nil {
		return 0, 0, err
	}

	size, err = parseQueryInt32(r, "size", defaultPageSize, 1, maxPageSize)
	if err != nil {
		return 0, 0, err
	}

	return page, size, nil
}
