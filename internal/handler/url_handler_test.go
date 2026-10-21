package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srini/url-shortener/internal/domain"
	"github.com/srini/url-shortener/internal/repository"
	"github.com/srini/url-shortener/internal/service"
)

func setupTestRouter() http.Handler {
	repo := repository.NewMemoryRepository()
	generator := service.NewBase62Generator()
	validator := service.NewURLValidator()
	svc := service.NewURLService(repo, generator, validator, "http://localhost:8080", 7)
	h := NewURLHandler(svc)
	return NewRouter(h, RouterConfig{
		RateLimitRPS:   1000,
		RateLimitBurst: 2000,
	})
}

func TestHandler_HealthzAndReadyz(t *testing.T) {
	router := setupTestRouter()

	// 1. Healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz, got: %d", rr.Code)
	}

	// 2. Readyz
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /readyz, got: %d", rr.Code)
	}
}

func TestHandler_CreateAndGetURL(t *testing.T) {
	router := setupTestRouter()

	// 1. Create short URL
	createPayload := []byte(`{"url":"https://go.dev/doc/tutorial"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(createPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got: %d, body: %s", rr.Code, rr.Body.String())
	}

	var created domain.URLOutput
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created.Code == "" {
		t.Fatal("expected non-empty short code")
	}
	if created.OriginalURL != "https://go.dev/doc/tutorial" {
		t.Fatalf("expected original URL, got: %s", created.OriginalURL)
	}

	// 2. Get short URL details
	req = httptest.NewRequest(http.MethodGet, "/api/v1/urls/"+created.Code, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rr.Code)
	}
}

func TestHandler_CreateCustomCodeAndConflict(t *testing.T) {
	router := setupTestRouter()

	// 1. Create with custom code
	payload := []byte(`{"url":"https://example.com","custom_code":"custom-123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got: %d, body: %s", rr.Code, rr.Body.String())
	}

	// 2. Duplicate custom code -> Conflict 409
	req = httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate custom code, got: %d", rr.Code)
	}
}

func TestHandler_ValidationErrors(t *testing.T) {
	router := setupTestRouter()

	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "Invalid scheme javascript:",
			body:     `{"url":"javascript:alert(1)"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Invalid custom code with spaces",
			body:     `{"url":"https://example.com","custom_code":"bad code"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Reserved custom code",
			body:     `{"url":"https://example.com","custom_code":"healthz"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Malformed JSON",
			body:     `{"url": bad json}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Empty body",
			body:     ``,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tc.wantCode {
				t.Errorf("expected %d, got %d, body: %s", tc.wantCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandler_RedirectAndStats(t *testing.T) {
	router := setupTestRouter()

	// 1. Create a short URL
	createPayload := []byte(`{"url":"https://example.com/target","custom_code":"redirect-test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(createPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create: %d", rr.Code)
	}

	// 2. Perform redirect GET /{code}
	req = httptest.NewRequest(http.MethodGet, "/redirect-test", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 Temporary Redirect, got: %d", rr.Code)
	}

	loc := rr.Header().Get("Location")
	if loc != "https://example.com/target" {
		t.Errorf("expected Location header https://example.com/target, got %s", loc)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl != "no-cache, no-store, must-revalidate" {
		t.Errorf("expected no-cache Cache-Control, got: %s", cacheControl)
	}

	// 3. Check stats
	req = httptest.NewRequest(http.MethodGet, "/api/v1/urls/redirect-test/stats", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for stats, got: %d", rr.Code)
	}

	var stats domain.URLStats
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}
	if stats.Code != "redirect-test" {
		t.Errorf("expected code redirect-test, got %s", stats.Code)
	}
}

func TestHandler_UpdateAndDelete(t *testing.T) {
	router := setupTestRouter()

	// 1. Create URL
	createPayload := []byte(`{"url":"https://example.com/old","custom_code":"update-test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(createPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("creation failed: %d", rr.Code)
	}

	// 2. Update destination URL PATCH
	updatePayload := []byte(`{"url":"https://example.com/new"}`)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/urls/update-test", bytes.NewBuffer(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for update, got: %d", rr.Code)
	}

	var updated domain.URLOutput
	_ = json.NewDecoder(rr.Body).Decode(&updated)
	if updated.OriginalURL != "https://example.com/new" {
		t.Errorf("expected updated URL, got: %s", updated.OriginalURL)
	}

	// 3. Delete URL DELETE
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/urls/update-test", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got: %d", rr.Code)
	}

	// 4. Try GET after delete -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/urls/update-test", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found after delete, got: %d", rr.Code)
	}
}
