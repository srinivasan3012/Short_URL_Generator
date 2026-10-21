package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/srini/url-shortener/internal/domain"
)

const maxURLLength = 2048

// URLValidator validates destination URLs for security and RFC compliance.
type URLValidator interface {
	Validate(rawURL string) (string, error)
}

type defaultValidator struct{}

// NewURLValidator creates a new URLValidator.
func NewURLValidator() URLValidator {
	return &defaultValidator{}
}

// Validate ensures rawURL is well-formed, uses http/https, and does not exceed maximum length.
func (v *defaultValidator) Validate(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("%w: url cannot be empty", domain.ErrInvalidInput)
	}

	if len(trimmed) > maxURLLength {
		return "", fmt.Errorf("%w: url exceeds maximum length of %d characters", domain.ErrInvalidInput, maxURLLength)
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: malformed url format", domain.ErrInvalidInput)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: unsupported scheme '%s'; only http and https are allowed", domain.ErrInvalidInput, parsed.Scheme)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("%w: url must have a valid host", domain.ErrInvalidInput)
	}

	return parsed.String(), nil
}
