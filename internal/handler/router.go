package handler

import (
	"net/http"

	"github.com/srini/url-shortener/internal/middleware"
	"github.com/srini/url-shortener/internal/shortener"
)

// RouterConfig holds settings for assembling the HTTP handler stack.
type RouterConfig struct {
	RateLimitRPS     float64
	RateLimitBurst   int
	ShortenerHandler *shortener.Handler
}

// NewRouter constructs and registers all API routes with their middleware stack.
func NewRouter(h *URLHandler, cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Probes
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /readyz", h.Readyz)

	// API v1 Resource Routes
	mux.HandleFunc("POST /api/v1/urls", h.CreateURL)
	mux.HandleFunc("GET /api/v1/urls", h.ListURLs)
	mux.HandleFunc("GET /api/v1/urls/{code}", h.GetURL)
	mux.HandleFunc("PATCH /api/v1/urls/{code}", h.UpdateURL)
	mux.HandleFunc("DELETE /api/v1/urls/{code}", h.DeleteURL)
	mux.HandleFunc("GET /api/v1/urls/{code}/stats", h.GetStats)

	// In-Memory Challenge 5 Endpoints
	shortenerHandler := cfg.ShortenerHandler
	if shortenerHandler == nil {
		shortenerHandler = shortener.NewHandler(shortener.NewMemoryStore())
	}
	mux.HandleFunc("POST /shorten", shortenerHandler.Shorten)
	mux.HandleFunc("GET /shorten", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("PUT /shorten", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("PATCH /shorten", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("DELETE /shorten", shortenerHandler.MethodNotAllowed)

	mux.HandleFunc("GET /resolve/{code}", shortenerHandler.Resolve)
	mux.HandleFunc("GET /resolve/", shortenerHandler.Resolve)
	mux.HandleFunc("GET /resolve", shortenerHandler.Resolve)
	mux.HandleFunc("POST /resolve/{code}", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("PUT /resolve/{code}", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("PATCH /resolve/{code}", shortenerHandler.MethodNotAllowed)
	mux.HandleFunc("DELETE /resolve/{code}", shortenerHandler.MethodNotAllowed)

	// Public Web UI and Redirect Routes
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /{code}", h.Redirect)

	// Build middleware pipeline: RequestID -> Logger -> Recoverer -> RateLimiter -> CORS
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	var handler http.Handler = mux
	handler = rateLimiter.Middleware(handler)
	handler = middleware.CORS(handler)
	handler = middleware.Recoverer(handler)
	handler = middleware.Logger(handler)
	handler = middleware.RequestID(handler)

	return handler
}
