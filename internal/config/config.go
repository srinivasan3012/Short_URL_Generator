package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the URL shortener application.
type Config struct {
	// Server
	Port        string
	Environment string
	BaseURL     string

	// Timeouts
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// Database
	DatabaseURL         string
	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxLifetime   time.Duration
	DBConnMaxIdleTime   time.Duration
	RunMigrationsOnBoot bool

	// Application Settings
	CodeLength     int
	RateLimitRPS   float64
	RateLimitBurst int
}

// Load reads configuration from environment variables, falling back to sensible defaults.
func Load() (*Config, error) {
	loadDotEnv()

	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		dbURL = getEnv("POSTGRES_URL", "")
	}
	if dbURL == "" {
		dbURL = getEnv("NEON_DATABASE_URL", "")
	}

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		BaseURL:             getEnv("BASE_URL", "http://localhost:8080"),
		ReadTimeout:         getEnvDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:        getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:         getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout:     getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseURL:         dbURL,
		DBMaxOpenConns:      getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:      getEnvInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime:   getEnvDuration("DB_CONN_MAX_LIFETIME", 15*time.Minute),
		DBConnMaxIdleTime:   getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		RunMigrationsOnBoot: getEnvBool("RUN_MIGRATIONS_ON_BOOT", true),
		CodeLength:          getEnvInt("CODE_LENGTH", 7),
		RateLimitRPS:        getEnvFloat("RATE_LIMIT_RPS", 20.0),
		RateLimitBurst:      getEnvInt("RATE_LIMIT_BURST", 40),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate ensures all critical configuration parameters are within acceptable bounds.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT must not be empty")
	}

	if _, err := url.ParseRequestURI(c.BaseURL); err != nil {
		return fmt.Errorf("BASE_URL is invalid: %w", err)
	}

	if c.CodeLength < 4 || c.CodeLength > 32 {
		return fmt.Errorf("CODE_LENGTH must be between 4 and 32, got %d", c.CodeLength)
	}

	if c.RateLimitRPS <= 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be positive, got %f", c.RateLimitRPS)
	}

	if c.RateLimitBurst <= 0 {
		return fmt.Errorf("RATE_LIMIT_BURST must be positive, got %d", c.RateLimitBurst)
	}

	// In production, DATABASE_URL must be specified
	if strings.ToLower(c.Environment) == "production" && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL must be set in production environment")
	}

	return nil
}

// Helper functions for reading typed environment variables

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvFloat(key string, defaultVal float64) float64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func loadDotEnv() {
	for _, filename := range []string{".env", ".env.local"} {
		content, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				if _, exists := os.LookupEnv(key); !exists {
					_ = os.Setenv(key, val)
				}
			}
		}
	}
}
