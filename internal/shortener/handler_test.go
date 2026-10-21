package shortener

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func newTestMux(store Store) http.Handler {
	h := NewHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", h.Shorten)
	mux.HandleFunc("/resolve/{code}", h.Resolve)
	return mux
}

// 1. Creating a new code returns 201 Created.
func TestShorten_CreateNewCode(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	body := `{"code":"abc123","url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp ShortenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "abc123" || resp.URL != "https://example.com" {
		t.Errorf("unexpected response content: %+v", resp)
	}
}

// 2. Resolving the newly created code returns the original URL (200 OK).
func TestResolve_ExistingCode(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Save("mycode", "https://golang.org")
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/resolve/mycode", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp ShortenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "mycode" || resp.URL != "https://golang.org" {
		t.Errorf("unexpected response content: %+v", resp)
	}
}

// 3 & 4. Creating the same code a second time returns 409 Conflict and preserves original mapping.
func TestShorten_DuplicateCodeConflictAndPreservesOriginal(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	// Initial creation
	initialBody := `{"code":"dup123","url":"https://first-destination.com"}`
	req1 := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(initialBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("initial creation failed: status %d", w1.Code)
	}

	// Duplicate attempt with different destination URL
	dupBody := `{"code":"dup123","url":"https://malicious-overwrite.com"}`
	req2 := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(dupBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict on duplicate, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "short code already exists" {
		t.Errorf("expected error message 'short code already exists', got: %q", errResp.Error)
	}

	// Verify original URL was NOT overwritten
	resolvedURL, err := store.Get("dup123")
	if err != nil {
		t.Fatalf("failed to retrieve stored code: %v", err)
	}
	if resolvedURL != "https://first-destination.com" {
		t.Fatalf("SECURITY VIOLATION: duplicate request overwritten original destination! Expected 'https://first-destination.com', got %q", resolvedURL)
	}
}

// 5. Resolving a missing code returns 404 Not Found.
func TestResolve_MissingCode(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/resolve/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "short code not found" {
		t.Errorf("expected error message 'short code not found', got %q", errResp.Error)
	}
}

// 6. Invalid JSON returns 400 Bad Request.
func TestShorten_InvalidJSON(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	invalidJSONs := []string{
		`{not valid json}`,
		`{"code": "abc", "url":}`,
		``,
	}

	for _, body := range invalidJSONs {
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("for body %q, expected status 400 Bad Request, got %d", body, w.Code)
		}
	}
}

// 7. Missing code or URL returns 400 Bad Request.
func TestShorten_MissingRequiredFields(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	testCases := []struct {
		name string
		body string
	}{
		{"missing code", `{"url":"https://example.com"}`},
		{"empty code", `{"code":"  ","url":"https://example.com"}`},
		{"missing url", `{"code":"test123"}`},
		{"empty url", `{"code":"test123","url":""}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// 8. Invalid destination URL returns 400 Bad Request.
func TestShorten_InvalidDestinationURL(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	invalidURLs := []string{
		`{"code":"c1","url":"ftp://example.com"}`,
		`{"code":"c2","url":"javascript:alert(1)"}`,
		`{"code":"c3","url":"file:///etc/passwd"}`,
		`{"code":"c4","url":"not-a-url"}`,
		`{"code":"c5","url":"http://"}`,
	}

	for _, body := range invalidURLs {
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("for body %q, expected status 400 Bad Request, got %d", body, w.Code)
		}
	}
}

// 9. Unsupported HTTP methods return 405 Method Not Allowed.
func TestMethods_NotAllowed(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store)

	// GET on /shorten via handler
	req1 := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	w1 := httptest.NewRecorder()
	h.Shorten(w1, req1)
	if w1.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 on GET /shorten, got %d", w1.Code)
	}

	// POST on /resolve/abc via handler
	req2 := httptest.NewRequest(http.MethodPost, "/resolve/abc", nil)
	w2 := httptest.NewRecorder()
	h.Resolve(w2, req2)
	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 on POST /resolve/abc, got %d", w2.Code)
	}

	// Verify MethodNotAllowed handler helper
	req3 := httptest.NewRequest(http.MethodPut, "/shorten", nil)
	w3 := httptest.NewRecorder()
	h.MethodNotAllowed(w3, req3)
	if w3.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 on MethodNotAllowed, got %d", w3.Code)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Error != "method not allowed" {
		t.Errorf("expected 'method not allowed', got %q", errResp.Error)
	}
}

// 10. Concurrent attempts to create the same code result in exactly one 201, with all others 409.
func TestShorten_ConcurrentDuplicateCreation(t *testing.T) {
	store := NewMemoryStore()
	mux := newTestMux(store)

	const concurrency = 25
	var wg sync.WaitGroup
	var successCount int64
	var conflictCount int64

	body := `{"code":"racecode","url":"https://winner.com"}`

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			switch w.Code {
			case http.StatusCreated:
				atomic.AddInt64(&successCount, 1)
			case http.StatusConflict:
				atomic.AddInt64(&conflictCount, 1)
			default:
				t.Errorf("unexpected status code during race: %d", w.Code)
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful creation (201), got %d", successCount)
	}
	if conflictCount != int64(concurrency-1) {
		t.Fatalf("expected exactly %d conflicts (409), got %d", concurrency-1, conflictCount)
	}

	// Confirm final stored value
	finalURL, err := store.Get("racecode")
	if err != nil {
		t.Fatalf("failed to retrieve stored code: %v", err)
	}
	if finalURL != "https://winner.com" {
		t.Fatalf("unexpected stored URL: %s", finalURL)
	}
}
