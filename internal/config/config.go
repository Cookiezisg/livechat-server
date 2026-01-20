package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr     string
	PublicURL    string
	DatabaseURL  string
	RedisURL     string
	JWTSecret    string
	TokenTTL     time.Duration
	CORSOrigins  []string
	UploadDir    string
	MaxUploadMB  int64
	Environment  string
	AllowOrigins string
}

func Load() Config {
	cfg := Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		PublicURL:   getenv("PUBLIC_URL", "http://localhost:8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://livechat:livechat@localhost:5432/livechat?sslmode=disable"),
		RedisURL:    getenv("REDIS_URL", ""),
		JWTSecret:   getenv("JWT_SECRET", "dev-secret-change-me"),
		TokenTTL:    getenvDuration("TOKEN_TTL", 24*time.Hour),
		UploadDir:   getenv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB: getenvInt64("MAX_UPLOAD_MB", 10),
		Environment: getenv("ENV", "development"),
	}
	cfg.AllowOrigins = getenv("CORS_ORIGINS", "http://localhost:5173")
	cfg.CORSOrigins = splitAndTrim(cfg.AllowOrigins)
	return cfg
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func getenvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
