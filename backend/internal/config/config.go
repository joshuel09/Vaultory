// Package config reads Vaultory's runtime configuration from the environment.
//
// Every value comes from the environment and nothing secret has a default, so a misconfigured
// deployment fails at startup rather than at the first request (Constitution IV).
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL       string
	ListenAddr        string
	ImageStore        string
	ImageStorePath    string
	DevIdentity       bool
	SessionSecret     string
	IdempotencyWindow time.Duration
}

// Load reads the configuration, collecting every problem before returning so a misconfigured
// environment is reported in one go rather than one variable per restart.
func Load() (Config, error) {
	var problems []string

	cfg := Config{
		DatabaseURL:    os.Getenv("VAULTORY_DATABASE_URL"),
		ListenAddr:     envOr("VAULTORY_LISTEN_ADDR", "127.0.0.1:8080"),
		ImageStore:     envOr("VAULTORY_IMAGE_STORE", "filesystem"),
		ImageStorePath: envOr("VAULTORY_IMAGE_STORE_PATH", "./.imagestore"),
		SessionSecret:  os.Getenv("VAULTORY_SESSION_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		problems = append(problems, "VAULTORY_DATABASE_URL is required")
	}
	if cfg.SessionSecret == "" {
		problems = append(problems, "VAULTORY_SESSION_SECRET is required")
	}

	switch cfg.ImageStore {
	case "filesystem":
		if cfg.ImageStorePath == "" {
			problems = append(problems, "VAULTORY_IMAGE_STORE_PATH is required when VAULTORY_IMAGE_STORE is filesystem")
		}
	default:
		problems = append(problems, fmt.Sprintf("VAULTORY_IMAGE_STORE %q is not supported; only \"filesystem\" is implemented", cfg.ImageStore))
	}

	// Development-only collector resolution. Opt-in by exact value, so a stray or misspelled
	// setting cannot silently disable authentication (research.md Decision 1).
	cfg.DevIdentity = os.Getenv("VAULTORY_DEV_IDENTITY") == "enabled"

	window := envOr("VAULTORY_IDEMPOTENCY_WINDOW", "24h")
	d, err := time.ParseDuration(window)
	switch {
	case err != nil:
		problems = append(problems, fmt.Sprintf("VAULTORY_IDEMPOTENCY_WINDOW %q is not a duration", window))
	case d <= 0:
		problems = append(problems, "VAULTORY_IDEMPOTENCY_WINDOW must be positive")
	default:
		cfg.IdempotencyWindow = d
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("configuration invalid:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
