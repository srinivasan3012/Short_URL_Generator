package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/srini/url-shortener/internal/config"
	"github.com/srini/url-shortener/internal/domain"
)

const pgErrUniqueViolation = "23505"

type postgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository establishes a connection pool to PostgreSQL using pgxpool.
func NewPostgresRepository(ctx context.Context, cfg *config.Config) (domain.URLRepository, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database url: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.DBMaxOpenConns)
	poolConfig.MinConns = int32(cfg.DBMaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.DBConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.DBConnMaxIdleTime

	// Connection timeout
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	if cfg.RunMigrationsOnBoot {
		if err := RunMigrations(connectCtx, pool); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return &postgresRepository{pool: pool}, nil
}

func (r *postgresRepository) Create(ctx context.Context, u *domain.URL) error {
	query := `
		INSERT INTO urls (id, code, original_url, clicks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Code, u.OriginalURL, u.Clicks, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to insert url: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByCode(ctx context.Context, code string) (*domain.URL, error) {
	query := `
		SELECT id, code, original_url, clicks, last_accessed_at, created_at, updated_at, deleted_at
		FROM urls
		WHERE code = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, code)

	var u domain.URL
	err := row.Scan(
		&u.ID,
		&u.Code,
		&u.OriginalURL,
		&u.Clicks,
		&u.LastAccessedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query url by code: %w", err)
	}

	return &u, nil
}

func (r *postgresRepository) List(ctx context.Context, limit, offset int) ([]*domain.URL, int, error) {
	countQuery := `SELECT COUNT(*) FROM urls WHERE deleted_at IS NULL`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count urls: %w", err)
	}

	listQuery := `
		SELECT id, code, original_url, clicks, last_accessed_at, created_at, updated_at, deleted_at
		FROM urls
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list urls: %w", err)
	}
	defer rows.Close()

	urls := make([]*domain.URL, 0, limit)
	for rows.Next() {
		var u domain.URL
		if err := rows.Scan(
			&u.ID,
			&u.Code,
			&u.OriginalURL,
			&u.Clicks,
			&u.LastAccessedAt,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan url row: %w", err)
		}
		urls = append(urls, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return urls, total, nil
}

func (r *postgresRepository) Update(ctx context.Context, code, newURL string) (*domain.URL, error) {
	query := `
		UPDATE urls
		SET original_url = $2, updated_at = NOW()
		WHERE code = $1 AND deleted_at IS NULL
		RETURNING id, code, original_url, clicks, last_accessed_at, created_at, updated_at, deleted_at
	`
	row := r.pool.QueryRow(ctx, query, code, newURL)

	var u domain.URL
	err := row.Scan(
		&u.ID,
		&u.Code,
		&u.OriginalURL,
		&u.Clicks,
		&u.LastAccessedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update url: %w", err)
	}

	return &u, nil
}

func (r *postgresRepository) Delete(ctx context.Context, code string) error {
	query := `
		UPDATE urls
		SET deleted_at = NOW()
		WHERE code = $1 AND deleted_at IS NULL
	`
	cmdTag, err := r.pool.Exec(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to delete url: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *postgresRepository) IncrementClicks(ctx context.Context, code string) error {
	query := `
		UPDATE urls
		SET clicks = clicks + 1, last_accessed_at = NOW()
		WHERE code = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to increment clicks: %w", err)
	}
	return nil
}

func (r *postgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *postgresRepository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}
