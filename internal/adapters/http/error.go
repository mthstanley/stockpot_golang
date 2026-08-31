package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mthstanley/stockpot/internal/core"
)

type PathValueParseError struct {
	Path string
	Err  error
}

func (e *PathValueParseError) Error() string {
	return fmt.Sprintf("failed to parse path value for path %s: %v", e.Path, e.Err)
}

func (e *PathValueParseError) Unwrap() error {
	return e.Err
}

var RequestBodyDecodeError = errors.New("invalid request body")

type ErrorResponse struct {
	Error   string
	Details string
}

func handleRequestBodyDecodeError(err error) (int, ErrorResponse) {
	status := http.StatusBadRequest
	var errResponse ErrorResponse
	if e, ok := errors.AsType[*json.SyntaxError](err); ok {
		errResponse = ErrorResponse{
			Error:   "Invalid JSON body syntax",
			Details: fmt.Sprintf("Character %d: %s", e.Offset, e.Error()),
		}
	} else if e, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		errResponse = ErrorResponse{
			Error:   "Invalid JSON body field type",
			Details: fmt.Sprintf("Field '%s' expects type %s, but received %s", e.Field, e.Type, e.Value),
		}
	} else if errors.Is(err, io.EOF) {
		errResponse = ErrorResponse{
			Error:   "Empty JSON body",
			Details: "The request body cannot be empty",
		}
	} else {
		errResponse = ErrorResponse{
			Error: "Invalid JSON body",
		}

	}

	return status, errResponse
}

func HandleErrors(next HandlerFuncWithError) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := next(w, r)
		if err == nil {
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		var errResponse ErrorResponse
		var status int
		if perr, ok := errors.AsType[*PathValueParseError](err); ok {
			status = http.StatusBadRequest
			errResponse = ErrorResponse{
				Error:   "Invalid path parameter",
				Details: perr.Error(),
			}
		} else if errors.Is(err, RequestBodyDecodeError) {
			status, errResponse = handleRequestBodyDecodeError(err)
		} else if enterr, ok := errors.AsType[*core.EntityNotFound](err); ok {
			status = http.StatusNotFound
			errResponse = ErrorResponse{
				Error:   "Entity not found",
				Details: fmt.Sprintf("%s entity with identifier %s not found", enterr.Type, enterr.Ident),
			}
		} else {
			status = http.StatusInternalServerError
			errResponse = ErrorResponse{
				Error: "Internal server error",
			}
			log.Printf("error handling request: %v", err)
		}

		data, err := json.Marshal(&errResponse)
		if err != nil {
			log.Println("error json encoding response:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := w.Write(data); err != nil {
			log.Println("error writing result", err)
		}
	})
}
