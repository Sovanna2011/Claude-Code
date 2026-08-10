// Package config reads the service's settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       string
	TokenTTL        time.Duration
	AllowedOrigins  []string
	MigrateOnStart  bool
	MigrationsDir   string
	ShutdownTimeout time.Duration
}

// Load reads the environment and refuses to start on anything that would only fail later, in
// front of a user. A missing database URL is a start-up error, not a 500 on the first request.
func Load() (Config, error) {
	cfg := Config{
		Addr:            env("FARMAREA_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("FARMAREA_DATABASE_URL"),
		JWTSecret:       os.Getenv("FARMAREA_JWT_SECRET"),
		TokenTTL:        time.Duration(envInt("FARMAREA_TOKEN_TTL_MINUTES", 480)) * time.Minute,
		AllowedOrigins:  splitAndTrim(env("FARMAREA_CORS_ORIGINS", "http://localhost:8081")),
		MigrateOnStart:  envBool("FARMAREA_MIGRATE_ON_START", true),
		MigrationsDir:   env("FARMAREA_MIGRATIONS_DIR", "../db/migrations"),
		ShutdownTimeout: 15 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("FARMAREA_DATABASE_URL is required")
	}
	// A signing key short enough to brute-force is worse than no authentication at all, because it
	// looks like authentication.
	if len(cfg.JWTSecret) < 32 {
		return cfg, fmt.Errorf("FARMAREA_JWT_SECRET must be at least 32 characters")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
