package service

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"

	"github.com/srini/url-shortener/internal/domain"
)

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	base62Length    = big.NewInt(int64(len(base62Chars)))
	customCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,32}$`)

	reservedCodes = map[string]struct{}{
		"api":         {},
		"healthz":     {},
		"readyz":      {},
		"docs":        {},
		"metrics":     {},
		"favicon.ico": {},
		"static":      {},
		"swagger":     {},
		"admin":       {},
		"login":       {},
		"register":    {},
	}
)

// CodeGenerator defines methods for generating and validating short codes.
type CodeGenerator interface {
	Generate(length int) (string, error)
	ValidateCustom(code string) error
	IsReserved(code string) bool
}

type base62Generator struct{}

// NewBase62Generator creates a new cryptographically secure Base62 code generator.
func NewBase62Generator() CodeGenerator {
	return &base62Generator{}
}

// Generate creates a cryptographically secure random string of given length using Base62.
func (g *base62Generator) Generate(length int) (string, error) {
	if length <= 0 {
		length = 7
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, base62Length)
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[num.Int64()]
	}

	return string(result), nil
}

// ValidateCustom verifies that a user-specified custom code is valid and not reserved.
func (g *base62Generator) ValidateCustom(code string) error {
	trimmed := strings.TrimSpace(code)
	if g.IsReserved(trimmed) {
		return domain.ErrReservedCode
	}

	if !customCodeRegex.MatchString(trimmed) {
		return domain.ErrInvalidInput
	}

	return nil
}

// IsReserved returns true if the code conflicts with any system routes.
func (g *base62Generator) IsReserved(code string) bool {
	_, exists := reservedCodes[strings.ToLower(code)]
	return exists
}
