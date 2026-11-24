// Package config reads the application's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const minCSRFKeyLen = 32

// Config holds the settings parsed from the environment.
type Config struct {
	DatabasePath   string
	CSRFKey        []byte
	SessionIdle    time.Duration
	SessionMax     time.Duration
	TrustedProxies []string
}

// FromEnv builds a Config from getenv, applying defaults and validating values.
func FromEnv(getenv func(string) string) (Config, error) {
	key := getenv("CSRF_KEY")
	if key == "" {
		return Config{}, errors.New("CSRF_KEY is required")
	}
	if len(key) < minCSRFKeyLen {
		return Config{}, fmt.Errorf("CSRF_KEY must be at least %d bytes, got %d", minCSRFKeyLen, len(key))
	}

	cfg := Config{
		DatabasePath: "./data/passenger.db",
		CSRFKey:      []byte(key),
	}
	if p := getenv("DATABASE_PATH"); p != "" {
		cfg.DatabasePath = p
	}

	var err error
	if cfg.SessionIdle, err = duration(getenv, "SESSION_IDLE", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.SessionMax, err = duration(getenv, "SESSION_MAX", 12*time.Hour); err != nil {
		return Config{}, err
	}

	for _, p := range strings.Split(getenv("TRUSTED_PROXIES"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.TrustedProxies = append(cfg.TrustedProxies, p)
		}
	}
	return cfg, nil
}

func duration(getenv func(string) string, name string, def time.Duration) (time.Duration, error) {
	raw := getenv(name)
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %q", name, raw)
	}
	return d, nil
}
