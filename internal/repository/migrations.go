package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/srini/url-shortener/migrations"
)

// RunMigrations executes embedded schema migrations sequentially on boot.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("executing database schema migrations...")

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute migration up SQL
	if _, err := pool.Exec(execCtx, migrations.Migration001Up); err != nil {
		return fmt.Errorf("migration 000001_create_urls_table failed: %w", err)
	}

	slog.Info("database schema migrations executed successfully")
	return nil
}
