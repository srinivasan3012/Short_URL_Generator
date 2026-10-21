package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/srini/url-shortener/internal/config"
	"github.com/srini/url-shortener/internal/domain"
	"github.com/srini/url-shortener/internal/handler"
	"github.com/srini/url-shortener/internal/repository"
	"github.com/srini/url-shortener/internal/service"
	"github.com/srini/url-shortener/internal/shortener"
)

func main() {
	// 1. Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting URL Shortener service...")

	// 2. Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Initialize repository
	var repo domain.URLRepository
	if cfg.DatabaseURL != "" {
		slog.Info("connecting to PostgreSQL database...", "url", sanitizeDBURL(cfg.DatabaseURL))
		pgRepo, err := repository.NewPostgresRepository(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to connect to PostgreSQL", "error", err)
			os.Exit(1)
		}
		defer pgRepo.Close()
		repo = pgRepo
		slog.Info("PostgreSQL connection pool established successfully")
	} else {
		slog.Warn("DATABASE_URL is not set; running in local in-memory storage mode")
		repo = repository.NewMemoryRepository()
		defer repo.Close()
	}

	// 4. Construct service layer
	generator := service.NewBase62Generator()
	validator := service.NewURLValidator()
	urlService := service.NewURLService(repo, generator, validator, cfg.BaseURL, cfg.CodeLength)

	// 5. Construct HTTP router
	inMemoryStore := shortener.NewMemoryStore()
	shortenerHandler := shortener.NewHandler(inMemoryStore)

	urlHandler := handler.NewURLHandler(urlService)
	router := handler.NewRouter(urlHandler, handler.RouterConfig{
		RateLimitRPS:     cfg.RateLimitRPS,
		RateLimitBurst:   cfg.RateLimitBurst,
		ShortenerHandler: shortenerHandler,
	})

	// 6. Configure HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// 7. Start server asynchronously
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("listening and serving HTTP requests", "port", cfg.Port, "base_url", cfg.BaseURL)
		serverErrors <- server.ListenAndServe()
	}()

	// 8. Graceful shutdown handler
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("fatal HTTP server error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		slog.Info("shutdown signal received", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("server failed to shutdown gracefully; forcing termination", "error", err)
			_ = server.Close()
		}
		slog.Info("server shutdown complete")
	}
}

// sanitizeDBURL masks sensitive credentials in log output.
func sanitizeDBURL(raw string) string {
	if len(raw) < 15 {
		return "***"
	}
	return fmt.Sprintf("postgres://***@***:%s", raw[len(raw)-15:])
}
