package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srini/url-shortener/internal/domain"
)

const maxCollisionRetries = 5

// URLService exposes operations for managing short URLs.
type URLService interface {
	CreateURL(ctx context.Context, input domain.CreateURLInput) (*domain.URLOutput, error)
	GetURL(ctx context.Context, code string) (*domain.URLOutput, error)
	ListURLs(ctx context.Context, limit, offset int) (*domain.PaginatedURLResponse, error)
	UpdateURL(ctx context.Context, code string, input domain.UpdateURLInput) (*domain.URLOutput, error)
	DeleteURL(ctx context.Context, code string) error
	GetStats(ctx context.Context, code string) (*domain.URLStats, error)
	ResolveAndClick(ctx context.Context, code string) (string, error)
	CheckReadiness(ctx context.Context) error
}

type urlService struct {
	repo       domain.URLRepository
	generator  CodeGenerator
	validator  URLValidator
	baseURL    string
	codeLength int
}

// NewURLService constructs a new URLService instance.
func NewURLService(
	repo domain.URLRepository,
	generator CodeGenerator,
	validator URLValidator,
	baseURL string,
	codeLength int,
) URLService {
	// Clean up baseURL: strip trailing slashes
	baseURL = strings.TrimRight(baseURL, "/")

	return &urlService{
		repo:       repo,
		generator:  generator,
		validator:  validator,
		baseURL:    baseURL,
		codeLength: codeLength,
	}
}

func (s *urlService) CreateURL(ctx context.Context, input domain.CreateURLInput) (*domain.URLOutput, error) {
	validatedURL, err := s.validator.Validate(input.URL)
	if err != nil {
		return nil, err
	}

	customCode := strings.TrimSpace(input.CustomCode)
	if customCode != "" {
		if err := s.generator.ValidateCustom(customCode); err != nil {
			return nil, err
		}

		urlEntity := &domain.URL{
			ID:          uuid.New().String(),
			Code:        customCode,
			OriginalURL: validatedURL,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err := s.repo.Create(ctx, urlEntity); err != nil {
			return nil, err
		}

		return s.formatOutput(urlEntity), nil
	}

	// Automatic code generation with collision retry
	for attempt := 1; attempt <= maxCollisionRetries; attempt++ {
		code, err := s.generator.Generate(s.codeLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}

		if s.generator.IsReserved(code) {
			continue
		}

		urlEntity := &domain.URL{
			ID:          uuid.New().String(),
			Code:        code,
			OriginalURL: validatedURL,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		err = s.repo.Create(ctx, urlEntity)
		if err == nil {
			return s.formatOutput(urlEntity), nil
		}

		if !errors.Is(err, domain.ErrConflict) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%w: failed to generate unique code after %d attempts", domain.ErrConflict, maxCollisionRetries)
}

func (s *urlService) GetURL(ctx context.Context, code string) (*domain.URLOutput, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, domain.ErrInvalidInput
	}

	entity, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return s.formatOutput(entity), nil
}

func (s *urlService) ListURLs(ctx context.Context, limit, offset int) (*domain.PaginatedURLResponse, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	entities, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	outputs := make([]domain.URLOutput, len(entities))
	for i, u := range entities {
		outputs[i] = *s.formatOutput(u)
	}

	return &domain.PaginatedURLResponse{
		Data: outputs,
		Pagination: domain.PaginationMeta{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	}, nil
}

func (s *urlService) UpdateURL(ctx context.Context, code string, input domain.UpdateURLInput) (*domain.URLOutput, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, domain.ErrInvalidInput
	}

	validatedURL, err := s.validator.Validate(input.URL)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, code, validatedURL)
	if err != nil {
		return nil, err
	}

	return s.formatOutput(updated), nil
}

func (s *urlService) DeleteURL(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return domain.ErrInvalidInput
	}

	return s.repo.Delete(ctx, code)
}

func (s *urlService) GetStats(ctx context.Context, code string) (*domain.URLStats, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, domain.ErrInvalidInput
	}

	entity, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return &domain.URLStats{
		Code:           entity.Code,
		OriginalURL:    entity.OriginalURL,
		Clicks:         entity.Clicks,
		LastAccessedAt: entity.LastAccessedAt,
		CreatedAt:      entity.CreatedAt,
	}, nil
}

func (s *urlService) ResolveAndClick(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", domain.ErrInvalidInput
	}

	entity, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	// Increment clicks asynchronously with a detached context so background click logging
	// is not prematurely canceled if the client terminates their HTTP request connection early.
	go func(targetCode string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.repo.IncrementClicks(bgCtx, targetCode); err != nil {
			slog.Error("failed to record click analytics", "code", targetCode, "error", err)
		}
	}(code)

	return entity.OriginalURL, nil
}

func (s *urlService) CheckReadiness(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *urlService) formatOutput(u *domain.URL) *domain.URLOutput {
	return &domain.URLOutput{
		ID:             u.ID,
		Code:           u.Code,
		OriginalURL:    u.OriginalURL,
		ShortURL:       fmt.Sprintf("%s/%s", s.baseURL, u.Code),
		Clicks:         u.Clicks,
		LastAccessedAt: u.LastAccessedAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
