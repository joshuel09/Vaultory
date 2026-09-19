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

// MinSessionSecretLength is the shortest session secret the service will start with (FR-019a).
const MinSessionSecretLength = 32

type Config struct {
	DatabaseURL       string
	ListenAddr        string
	ImageStore        string
	ImageStorePath    string
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
	// FR-019 and FR-019a. This secret is the only input to the signature the session resolver
	// verifies, so a weak one makes every session forgeable — and unlike a wrong password,
	// nothing about the running system would look wrong. Refusing to start is the whole
	// protection, so it is enforced here rather than left to documentation.
	switch {
	case cfg.SessionSecret == "":
		problems = append(problems, "VAULTORY_SESSION_SECRET is required")
	case len(cfg.SessionSecret) < MinSessionSecretLength:
		problems = append(problems, fmt.Sprintf(
			"VAULTORY_SESSION_SECRET must be at least %d characters (got %d): it is the only "+
				"input to the session signature, so a short one is forgeable",
			MinSessionSecretLength, len(cfg.SessionSecret)))
	}

	switch cfg.ImageStore {
	case "filesystem":
		if cfg.ImageStorePath == "" {
			problems = append(problems, "VAULTORY_IMAGE_STORE_PATH is required when VAULTORY_IMAGE_STORE is filesystem")
		}
	default:
		problems = append(problems, fmt.Sprintf("VAULTORY_IMAGE_STORE %q is not supported; only \"filesystem\" is implemented", cfg.ImageStore))
	}

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
