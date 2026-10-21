package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/srini/url-shortener/internal/domain"
	"github.com/srini/url-shortener/internal/service"
)

// URLHandler handles HTTP requests for the URL Shortener API.
type URLHandler struct {
	service service.URLService
}

// NewURLHandler creates a new URLHandler instance.
func NewURLHandler(service service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

// CreateURL handles POST /api/v1/urls
func (h *URLHandler) CreateURL(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateURLInput
	if err := readJSON(w, r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	output, err := h.service.CreateURL(r.Context(), input)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/api/v1/urls/%s", output.Code))
	_ = writeJSON(w, http.StatusCreated, output, headers)
}

// ListURLs handles GET /api/v1/urls
func (h *URLHandler) ListURLs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if limitStr := q.Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0
	if offsetStr := q.Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	res, err := h.service.ListURLs(r.Context(), limit, offset)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, res, nil)
}

// GetURL handles GET /api/v1/urls/{code}
func (h *URLHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	output, err := h.service.GetURL(r.Context(), code)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, output, nil)
}

// UpdateURL handles PATCH /api/v1/urls/{code}
func (h *URLHandler) UpdateURL(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	var input domain.UpdateURLInput
	if err := readJSON(w, r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
		return
	}

	output, err := h.service.UpdateURL(r.Context(), code, input)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, output, nil)
}

// DeleteURL handles DELETE /api/v1/urls/{code}
func (h *URLHandler) DeleteURL(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if err := h.service.DeleteURL(r.Context(), code); err != nil {
		mapDomainError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStats handles GET /api/v1/urls/{code}/stats
func (h *URLHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	stats, err := h.service.GetStats(r.Context(), code)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	_ = writeJSON(w, http.StatusOK, stats, nil)
}

// Redirect handles GET /{code}
func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	originalURL, err := h.service.ResolveAndClick(r.Context(), code)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	// 307 Temporary Redirect ensures user agent does not change request method and
	// no-cache headers prevent client/CDN caching, keeping analytics real-time.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

// Healthz handles GET /healthz
func (h *URLHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	_ = writeJSON(w, http.StatusOK, resp, nil)
}

// Readyz handles GET /readyz
func (h *URLHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.service.CheckReadiness(r.Context()); err != nil {
		resp := map[string]string{
			"status":   "not ready",
			"database": "disconnected",
			"error":    err.Error(),
			"time":     time.Now().UTC().Format(time.RFC3339),
		}
		_ = writeJSON(w, http.StatusServiceUnavailable, resp, nil)
		return
	}

	resp := map[string]string{
		"status":   "ready",
		"database": "connected",
		"time":     time.Now().UTC().Format(time.RFC3339),
	}
	_ = writeJSON(w, http.StatusOK, resp, nil)
}
