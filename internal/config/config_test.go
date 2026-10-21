package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoad_Defaults(t *testing.T) {
	// Clear any overrides
	os.Unsetenv("PORT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("BASE_URL")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("CODE_LENGTH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with defaults, got: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default Environment development, got %s", cfg.Environment)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("expected default BaseURL http://localhost:8080, got %s", cfg.BaseURL)
	}
	if cfg.CodeLength != 7 {
		t.Errorf("expected default CodeLength 7, got %d", cfg.CodeLength)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected ReadTimeout 5s, got %v", cfg.ReadTimeout)
	}
}

func TestConfigValidation_InvalidBaseURL(t *testing.T) {
	os.Setenv("BASE_URL", "not-a-valid-url")
	defer os.Unsetenv("BASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid BASE_URL, got nil")
	}
}

func TestConfigValidation_InvalidCodeLength(t *testing.T) {
	os.Setenv("CODE_LENGTH", "2")
	defer os.Unsetenv("CODE_LENGTH")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for CODE_LENGTH < 4, got nil")
	}
}

func TestConfigValidation_ProductionRequiresDatabaseURL(t *testing.T) {
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("DATABASE_URL", "")
	defer os.Unsetenv("ENVIRONMENT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for production without DATABASE_URL, got nil")
	}
}
