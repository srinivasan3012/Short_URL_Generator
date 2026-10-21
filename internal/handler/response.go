package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/srini/url-shortener/internal/domain"
	"github.com/srini/url-shortener/internal/middleware"
)

const maxBodyBytes = 1048576 // 1 MB

type FieldIssue struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

type ErrorEnvelope struct {
	Error ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Details   []FieldIssue `json:"details,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
}

func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBodyBytes)
		default:
			return err
		}
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(js)
	return err
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details []FieldIssue) {
	reqID := middleware.GetRequestID(r.Context())
	env := ErrorEnvelope{
		Error: ErrorDetails{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: reqID,
		},
	}
	_ = writeJSON(w, status, env, nil)
}

func mapDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "The requested short URL was not found", nil)
	case errors.Is(err, domain.ErrConflict):
		writeError(w, r, http.StatusConflict, "CONFLICT", "The requested short code is already in use", nil)
	case errors.Is(err, domain.ErrReservedCode):
		writeError(w, r, http.StatusBadRequest, "RESERVED_CODE", "The custom code matches a reserved system endpoint", nil)
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected internal error occurred", nil)
	}
}
