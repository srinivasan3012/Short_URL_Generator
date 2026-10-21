package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/srini/url-shortener/internal/domain"
)

func TestURLValidator_Valid(t *testing.T) {
	val := NewURLValidator()

	validURLs := []string{
		"https://golang.org",
		"http://example.com/some/path?param=value#anchor",
		"https://sub.domain.co.uk:8080/path",
		"http://192.168.1.1:8080/path",
	}

	for _, u := range validURLs {
		res, err := val.Validate(u)
		if err != nil {
			t.Errorf("expected %q to be valid, got: %v", u, err)
		}
		if res == "" {
			t.Errorf("expected non-empty validated URL for %q", u)
		}
	}
}

func TestURLValidator_Invalid(t *testing.T) {
	val := NewURLValidator()

	invalidURLs := []string{
		"",                                     // empty
		"   ",                                  // whitespace only
		"ftp://example.com/file.txt",           // unsupported scheme
		"javascript:alert(1)",                  // malicious XSS scheme
		"file:///etc/passwd",                   // file scheme
		"data:text/plain;base64,SGVsbG8=",      // data URI
		"not-a-url",                            // missing scheme & host
		"http://",                              // missing host
		"https://" + strings.Repeat("a", 2050), // exceeds max length
	}

	for _, u := range invalidURLs {
		_, err := val.Validate(u)
		if err == nil {
			t.Errorf("expected %q to fail validation, but it passed", u)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected domain.ErrInvalidInput for %q, got: %v", u, err)
		}
	}
}
