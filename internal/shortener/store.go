package shortener

import (
	"errors"
	"sync"
)

var (
	// ErrCodeExists is returned when attempting to create a code that already exists.
	ErrCodeExists = errors.New("short code already exists")

	// ErrCodeNotFound is returned when attempting to resolve a non-existent code.
	ErrCodeNotFound = errors.New("short code not found")
)

// Store defines the contract for storing and retrieving short code mappings.
type Store interface {
	Save(code, targetURL string) error
	Get(code string) (string, error)
}

// MemoryStore provides thread-safe, in-memory URL storage using map[string]string and sync.RWMutex.
type MemoryStore struct {
	mu   sync.RWMutex
	urls map[string]string
}

// NewMemoryStore initializes a new, isolated in-memory URL store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls: make(map[string]string),
	}
}

// Save atomically checks for the existence of code and inserts the target URL.
// If the code already exists, it returns ErrCodeExists and guarantees the original mapping is never overwritten.
func (s *MemoryStore) Save(code, targetURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Atomic existence check under write lock
	if _, exists := s.urls[code]; exists {
		return ErrCodeExists
	}

	s.urls[code] = targetURL
	return nil
}

// Get retrieves the original destination URL for a given short code.
// If the code does not exist, it returns ErrCodeNotFound.
func (s *MemoryStore) Get(code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	targetURL, exists := s.urls[code]
	if !exists {
		return "", ErrCodeNotFound
	}

	return targetURL, nil
}
