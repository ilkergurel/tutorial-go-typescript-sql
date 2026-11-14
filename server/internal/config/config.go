// Package config loads StackLab's configuration from environment variables.
//
// No config library: the build prompt mandates env vars only, and a struct with
// a Load function keeps the source of every setting greppable. Defaults match
// local mode (ADR-0008: one PostgreSQL on 127.0.0.1:5432, lessons run
// in-process), so `npm start` needs no environment at all; docker-compose.yml
// overrides them for Docker mode.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config carries everything cmd/stacklab wires at startup. Fields are exported
// because consumers are other packages, not because callers may mutate them.
type Config struct {
	HTTPHost    string
	HTTPPort    int
	HTTPTimeout time.Duration

	DatabaseURL        string
	DatabaseMaxConns   int32
	DatabaseMinConns   int32
	DatabaseMaxConnTTL time.Duration

	// RunnerBaseURL selects the run mode: empty runs lessons in-process (local
	// mode); set, it proxies them to the sandboxed runner container (Docker).
	RunnerBaseURL string
	// PlaygroundDatabaseURL is where in-process SQL lessons run. It must be a
	// non-superuser role (scripts/setup-local-db.sql).
	PlaygroundDatabaseURL string
	// NodeProjectDir is the app's root (the one with node_modules), where
	// ts-test lessons find Vitest and Playwright. Empty disables ts-test.
	NodeProjectDir string

	LogLevel string // "debug" | "info" | "warn" | "error"
	LogJSON  bool   // JSON handler in prod, text in dev
}

// Load reads configuration from the environment, applying defaults. It returns
// an error only for values that are present but invalid; absent values default.
func Load() (Config, error) {
	cfg := Config{
		HTTPHost:           env("STACKLAB_HTTP_HOST", "127.0.0.1"),
		HTTPPort:           8080,
		HTTPTimeout:        15 * time.Second,
		DatabaseURL:        env("STACKLAB_DATABASE_URL", "postgres://stacklab:stacklab@127.0.0.1:5432/stacklab"),
		DatabaseMaxConns:   10,
		DatabaseMinConns:   2,
		DatabaseMaxConnTTL: 30 * time.Minute,
		RunnerBaseURL:      env("STACKLAB_RUNNER_URL", ""),
		PlaygroundDatabaseURL: env("STACKLAB_RUNNER_DATABASE_URL",
			"postgres://playground:playground@127.0.0.1:5432/playground"),
		NodeProjectDir: env("STACKLAB_NODE_PROJECT", defaultNodeProject()),
		LogLevel:       env("STACKLAB_LOG_LEVEL", "info"),
		LogJSON:        env("STACKLAB_LOG_FORMAT", "text") == "json",
	}

	var err error
	if cfg.HTTPPort, err = envInt("STACKLAB_HTTP_PORT", cfg.HTTPPort); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns, err = envInt32("STACKLAB_DB_MAX_CONNS", cfg.DatabaseMaxConns); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns, err = envInt32("STACKLAB_DB_MIN_CONNS", cfg.DatabaseMinConns); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// defaultNodeProject is the working directory when it holds the app's
// node_modules with Vitest: `npm start` runs the backend from the repo root.
func defaultNodeProject() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(filepath.Join(wd, "node_modules", "vitest", "vitest.mjs")); err != nil {
		return ""
	}
	return wd
}

// Addr returns the listen address for net/http.
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort) }

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s: %w", key, err)
	}
	return n, nil
}

func envInt32(key string, fallback int32) (int32, error) {
	n, err := envInt(key, int(fallback))
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}
