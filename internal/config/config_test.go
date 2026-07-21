package config_test

import (
	"testing"
	"time"

	"github.com/nicolas2601/go-graphql-orders-api/internal/config"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg := config.Load(envFrom(map[string]string{}))

	if cfg.Port != "8080" {
		t.Errorf("Port default: got %q, want 8080", cfg.Port)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv default: got %q, want production (fail-safe)", cfg.AppEnv)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel default: got %q, want info", cfg.LogLevel)
	}
	if !cfg.GraphQLPlayground {
		t.Error("GraphQLPlayground default should be true")
	}
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Errorf("JWTAccessTTL default: got %v, want 15m", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL != 168*time.Hour {
		t.Errorf("JWTRefreshTTL default: got %v, want 168h", cfg.JWTRefreshTTL)
	}
	if !cfg.SeedProducts {
		t.Error("SeedProducts default should be true")
	}
	if cfg.IsDevelopment() {
		t.Error("default env must not be development (fail-safe)")
	}
}

func TestLoadCustomValues(t *testing.T) {
	cfg := config.Load(envFrom(map[string]string{
		"PORT":               "9090",
		"APP_ENV":            "development",
		"LOG_LEVEL":          "debug",
		"DATABASE_URL":       "postgres://u:p@localhost:5432/orders",
		"JWT_SECRET":         "s3cr3t",
		"JWT_ACCESS_TTL":     "5m",
		"JWT_REFRESH_TTL":    "48h",
		"GRAPHQL_PLAYGROUND": "false",
		"SEED_PRODUCTS":      "false",
	}))

	if cfg.Port != "9090" {
		t.Errorf("Port: got %q, want 9090", cfg.Port)
	}
	if !cfg.IsDevelopment() {
		t.Error("APP_ENV=development should be detected")
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/orders" {
		t.Errorf("DatabaseURL: got %q", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "s3cr3t" {
		t.Errorf("JWTSecret: got %q", cfg.JWTSecret)
	}
	if cfg.JWTAccessTTL != 5*time.Minute {
		t.Errorf("JWTAccessTTL: got %v, want 5m", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL != 48*time.Hour {
		t.Errorf("JWTRefreshTTL: got %v, want 48h", cfg.JWTRefreshTTL)
	}
	if cfg.GraphQLPlayground {
		t.Error("GRAPHQL_PLAYGROUND=false should disable playground")
	}
	if cfg.SeedProducts {
		t.Error("SEED_PRODUCTS=false should disable seeding")
	}
}

func TestLoadInvalidDurationFallsBackToDefault(t *testing.T) {
	cfg := config.Load(envFrom(map[string]string{"JWT_ACCESS_TTL": "not-a-duration"}))
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Errorf("invalid duration should fall back to 15m, got %v", cfg.JWTAccessTTL)
	}
}
