package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/srini/url-shortener/internal/domain"
)

type memoryRepository struct {
	mu   sync.RWMutex
	urls map[string]*domain.URL
}

// NewMemoryRepository constructs a thread-safe in-memory repository for testing.
func NewMemoryRepository() domain.URLRepository {
	return &memoryRepository{
		urls: make(map[string]*domain.URL),
	}
}

func (m *memoryRepository) Create(ctx context.Context, u *domain.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.urls[u.Code]; exists && existing.DeletedAt == nil {
		return domain.ErrConflict
	}

	copied := *u
	m.urls[u.Code] = &copied
	return nil
}

func (m *memoryRepository) GetByCode(ctx context.Context, code string) (*domain.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.urls[code]
	if !exists || u.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}

	copied := *u
	return &copied, nil
}

func (m *memoryRepository) List(ctx context.Context, limit, offset int) ([]*domain.URL, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*domain.URL
	for _, u := range m.urls {
		if u.DeletedAt == nil {
			copied := *u
			active = append(active, &copied)
		}
	}

	// Sort by CreatedAt descending
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt.After(active[j].CreatedAt)
	})

	total := len(active)
	if offset >= total {
		return []*domain.URL{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return active[offset:end], total, nil
}

func (m *memoryRepository) Update(ctx context.Context, code, newURL string) (*domain.URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[code]
	if !exists || u.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}

	u.OriginalURL = newURL
	u.UpdatedAt = time.Now().UTC()
	copied := *u
	return &copied, nil
}

func (m *memoryRepository) Delete(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[code]
	if !exists || u.DeletedAt != nil {
		return domain.ErrNotFound
	}

	now := time.Now().UTC()
	u.DeletedAt = &now
	return nil
}

func (m *memoryRepository) IncrementClicks(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[code]
	if !exists || u.DeletedAt != nil {
		return domain.ErrNotFound
	}

	u.Clicks++
	now := time.Now().UTC()
	u.LastAccessedAt = &now
	return nil
}

func (m *memoryRepository) Ping(ctx context.Context) error {
	return nil
}

func (m *memoryRepository) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.urls = make(map[string]*domain.URL)
}
