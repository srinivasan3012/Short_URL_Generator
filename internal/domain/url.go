package domain

import (
	"context"
	"errors"
	"time"
)

// Standard Domain Errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrConflict       = errors.New("short code already exists")
	ErrInvalidInput   = errors.New("invalid input data")
	ErrReservedCode   = errors.New("short code is a reserved system path")
	ErrRateLimited    = errors.New("rate limit exceeded")
	ErrDatabaseFailed = errors.New("database operation failed")
)

// URL represents the core URL Shortener domain entity.
type URL struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	OriginalURL    string     `json:"original_url"`
	Clicks         int64      `json:"clicks"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// CreateURLInput represents input payload for shortening a URL.
type CreateURLInput struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
}

// UpdateURLInput represents input payload for modifying target URL.
type UpdateURLInput struct {
	URL string `json:"url"`
}

// URLOutput formats the URL response for API consumers with the calculated ShortURL.
type URLOutput struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	OriginalURL    string     `json:"original_url"`
	ShortURL       string     `json:"short_url"`
	Clicks         int64      `json:"clicks"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// URLStats represents analytics for a short URL.
type URLStats struct {
	Code           string     `json:"code"`
	OriginalURL    string     `json:"original_url"`
	Clicks         int64      `json:"clicks"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PaginationMeta describes the current pagination window.
type PaginationMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// PaginatedURLResponse represents a paginated list of short URLs.
type PaginatedURLResponse struct {
	Data       []URLOutput    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// URLRepository defines the persistence contract for URL operations.
type URLRepository interface {
	Create(ctx context.Context, u *URL) error
	GetByCode(ctx context.Context, code string) (*URL, error)
	List(ctx context.Context, limit, offset int) ([]*URL, int, error)
	Update(ctx context.Context, code, newURL string) (*URL, error)
	Delete(ctx context.Context, code string) error
	IncrementClicks(ctx context.Context, code string) error
	Ping(ctx context.Context) error
	Close()
}
