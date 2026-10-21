package service

import (
	"errors"
	"testing"

	"github.com/srini/url-shortener/internal/domain"
)

func TestBase62Generator_Length(t *testing.T) {
	gen := NewBase62Generator()
	lengths := []int{4, 6, 7, 10, 16}

	for _, l := range lengths {
		code, err := gen.Generate(l)
		if err != nil {
			t.Fatalf("unexpected error generating code: %v", err)
		}
		if len(code) != l {
			t.Errorf("expected code length %d, got %d (%s)", l, len(code), code)
		}
	}
}

func TestBase62Generator_ValidateCustom(t *testing.T) {
	gen := NewBase62Generator()

	validCodes := []string{"custom-code", "my_promo_1", "ABC123xyz", "1234"}
	for _, code := range validCodes {
		if err := gen.ValidateCustom(code); err != nil {
			t.Errorf("expected code %q to be valid, got: %v", code, err)
		}
	}

	invalidCodes := []string{
		"abc", // too short (<4)
		"this-code-is-way-too-long-to-be-allowed-because-it-exceeds-thirty-two-chars", // too long
		"has space",    // whitespace
		"has$special!", // invalid chars
		"api",          // reserved
		"healthz",      // reserved
		"readyz",       // reserved
	}

	for _, code := range invalidCodes {
		err := gen.ValidateCustom(code)
		if err == nil {
			t.Errorf("expected code %q to fail validation, but it passed", code)
		}
		if gen.IsReserved(code) && !errors.Is(err, domain.ErrReservedCode) {
			t.Errorf("expected ErrReservedCode for %q, got: %v", code, err)
		}
	}
}
