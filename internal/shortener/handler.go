package shortener

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxRequestBodyBytes = 1048576 // 1 MB

// ShortenRequest represents the JSON request payload for creating a mapping.
type ShortenRequest struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// ShortenResponse represents the JSON response confirming creation or resolution.
type ShortenResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// ErrorResponse represents a standardized JSON error message.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Handler handles HTTP requests for in-memory URL shortening and resolution.
type Handler struct {
	store Store
}

// NewHandler constructs a new shortener Handler with the given Store.
func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

// Shorten handles POST /shorten
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()

	var req ShortenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			writeJSONError(w, http.StatusBadRequest, "request body cannot be empty")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "code is required")
		return
	}

	targetURL := strings.TrimSpace(req.URL)
	if targetURL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}

	// Validate URL syntax and require absolute HTTP/HTTPS scheme with valid host
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.TrimSpace(parsed.Host) == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid destination url: must be an absolute http or https url")
		return
	}

	// Atomically store mapping
	if err := h.store.Save(code, targetURL); err != nil {
		if errors.Is(err, ErrCodeExists) {
			writeJSONError(w, http.StatusConflict, "short code already exists")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to save short code")
		return
	}

	writeJSON(w, http.StatusCreated, ShortenResponse{
		Code: code,
		URL:  targetURL,
	})
}

// Resolve handles GET /resolve/{code}
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract code: prioritize r.PathValue("code"), fallback to URL path suffix
	code := r.PathValue("code")
	if code == "" {
		trimmed := strings.TrimPrefix(r.URL.Path, "/resolve/")
		code = strings.TrimSpace(trimmed)
	}

	if code == "" {
		writeJSONError(w, http.StatusNotFound, "short code not found")
		return
	}

	targetURL, err := h.store.Get(code)
	if err != nil {
		if errors.Is(err, ErrCodeNotFound) {
			writeJSONError(w, http.StatusNotFound, "short code not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve short code")
		return
	}

	writeJSON(w, http.StatusOK, ShortenResponse{
		Code: code,
		URL:  targetURL,
	})
}

// MethodNotAllowed writes a standardized 405 Method Not Allowed JSON response.
func (h *Handler) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
