// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration.
type Config struct {
	AppEnv          string
	AppPort         string
	AppBaseURL      string
	FrontendBaseURL string

	DatabaseURL string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	BcryptCost    int

	S3Endpoint        string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	S3Bucket          string
	S3PublicBaseURL   string
	S3ForcePathStyle  bool

	CORSAllowedOrigins []string

	RateLimitRPS   int
	RateLimitBurst int

	LogLevel  string
	LogFormat string
}

// Load reads configuration from environment (.env if present) and returns it.
// It returns an error only when a required variable is missing or invalid.
func Load() (*Config, error) {
	// .env is best-effort; missing file is not fatal.
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		AppPort:         getEnv("APP_PORT", "8080"),
		AppBaseURL:      getEnv("APP_BASE_URL", "http://localhost:8080"),
		FrontendBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:5173"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:        getEnv("JWT_SECRET", ""),
		S3Endpoint:       getEnv("S3_ENDPOINT", ""),
		S3Region:         getEnv("S3_REGION", "auto"),
		S3AccessKey:      getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getEnv("S3_SECRET_KEY", ""),
		S3Bucket:         getEnv("S3_BUCKET", ""),
		S3PublicBaseURL:  getEnv("S3_PUBLIC_BASE_URL", ""),
		S3ForcePathStyle: getEnvBool("S3_FORCE_PATH_STYLE", true),

		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),

		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 20),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 40),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	var err error
	if cfg.JWTAccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m")); err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	if cfg.JWTRefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h")); err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	cfg.BcryptCost = getEnvInt("BCRYPT_COST", 12)

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
