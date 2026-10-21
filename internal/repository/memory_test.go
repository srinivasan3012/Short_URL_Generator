package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srini/url-shortener/internal/domain"
)

func TestMemoryRepository_CRUD(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	url1 := &domain.URL{
		ID:          uuid.New().String(),
		Code:        "test1234",
		OriginalURL: "https://example.com/one",
		Clicks:      0,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// 1. Create
	if err := repo.Create(ctx, url1); err != nil {
		t.Fatalf("expected successful creation, got: %v", err)
	}

	// 2. Duplicate code -> Conflict
	if err := repo.Create(ctx, url1); err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}

	// 3. GetByCode
	found, err := repo.GetByCode(ctx, "test1234")
	if err != nil {
		t.Fatalf("expected found url, got: %v", err)
	}
	if found.OriginalURL != url1.OriginalURL {
		t.Errorf("expected %s, got %s", url1.OriginalURL, found.OriginalURL)
	}

	// 4. Update
	updated, err := repo.Update(ctx, "test1234", "https://example.com/updated")
	if err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}
	if updated.OriginalURL != "https://example.com/updated" {
		t.Errorf("expected updated URL, got %s", updated.OriginalURL)
	}

	// 5. Delete
	if err := repo.Delete(ctx, "test1234"); err != nil {
		t.Fatalf("expected delete to succeed, got: %v", err)
	}

	// 6. Get after delete -> Not Found
	_, err = repo.GetByCode(ctx, "test1234")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got: %v", err)
	}
}

func TestMemoryRepository_ConcurrentClicks(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	url := &domain.URL{
		ID:          uuid.New().String(),
		Code:        "concurrent",
		OriginalURL: "https://example.com/concurrent",
		Clicks:      0,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.Create(ctx, url); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	const goroutines = 50
	const clicksPerGoroutine = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < clicksPerGoroutine; j++ {
				_ = repo.IncrementClicks(ctx, "concurrent")
			}
		}()
	}

	wg.Wait()

	found, err := repo.GetByCode(ctx, "concurrent")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	expectedClicks := int64(goroutines * clicksPerGoroutine)
	if found.Clicks != expectedClicks {
		t.Errorf("expected %d clicks, got %d", expectedClicks, found.Clicks)
	}
}
