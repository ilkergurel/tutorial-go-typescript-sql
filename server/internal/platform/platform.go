// Package platform groups infrastructure adapters shared by features: the
// database pool and its DB handle, startup migrations, and logging setup.
//
// Why not a top-level "db" package: platform marks code that is infrastructure
// rather than a feature; features depend on it, it never depends on features.
package platform

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewLogger builds the slog default logger: text handler for dev readability,
// JSON when logs are collected (prod). Level comes from config without a level
// library — slog ships the four levels StackLab needs.
func NewLogger(level string, json bool) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// NewPool opens the pgx connection pool. Pool sizing comes from config; the
// context bounds only the initial connect — the pool outlives it.
func NewPool(ctx context.Context, url string, maxConns, minConns int32, connTTL time.Duration) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("platform: parse database url: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = connTTL

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("platform: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("platform: ping database: %w", err)
	}
	return pool, nil
}
