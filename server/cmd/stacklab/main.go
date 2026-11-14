// cmd/stacklab is the StackLab backend binary.
//
// main.go only wires dependencies: config -> logger -> pool -> router -> server.
// All logic lives in internal/ so it is importable and testable; this file stays
// a readable dependency graph (an Inside StackLab lesson points here).
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"stacklab/server/internal/config"
	"stacklab/server/internal/httpapi"
	"stacklab/server/internal/platform"
	"stacklab/server/internal/runclient"
	"stacklab/server/internal/runexec"
)

// embeddedMigrations ships in the binary so the distroless image needs no
// extra volume.
//
//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// migrationsFS strips the "migrations/" prefix that go:embed keeps in every
// path: goose only reads the root of the FS it is given, so passing the embed
// FS directly makes startup fail with "no migrations found".
func migrationsFS() (fs.FS, error) {
	return fs.Sub(embeddedMigrations, "migrations")
}

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := platform.NewLogger(cfg.LogLevel, cfg.LogJSON)
	slog.SetDefault(logger)

	// signal.NotifyContext cancels this context on SIGINT/SIGTERM: one context
	// drives both graceful shutdown and every I/O call below (Windows sends
	// SIGINT on Ctrl+C, which matters since the owner develops on Windows).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("starting stacklab backend",
		slog.String("addr", cfg.Addr()),
		slog.String("db", maskDSN(cfg.DatabaseURL)),
	)

	pool, err := platform.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns, cfg.DatabaseMaxConnTTL)
	if err != nil {
		return err
	}
	defer pool.Close()
	db := platform.NewDB(pool)

	// Self-migrate before serving: schema is ready when the first request
	// arrives, and a fresh app-db volume comes up correct on `docker compose up`.
	migrations, err := migrationsFS()
	if err != nil {
		return fmt.Errorf("migrations fs: %w", err)
	}
	if err := platform.Migrate(ctx, db, migrations); err != nil {
		return err
	}
	logger.Info("migrations applied")

	// Local mode (no STACKLAB_RUNNER_URL) runs lessons in this process; Docker
	// mode sets it and proxies to the sandboxed runner container (ADR-0008).
	var runner httpapi.Runner = runexec.New(cfg.PlaygroundDatabaseURL, cfg.NodeProjectDir)
	runMode := "in-process"
	if cfg.RunnerBaseURL != "" {
		runner = runclient.New(cfg.RunnerBaseURL)
		runMode = cfg.RunnerBaseURL
	}
	logger.Info("lesson runner ready", slog.String("mode", runMode))
	if cfg.NodeProjectDir == "" {
		logger.Info("ts-test lessons disabled: no node_modules with vitest in the working directory (set STACKLAB_NODE_PROJECT)")
	}

	srv := &http.Server{
		Addr: cfg.Addr(),
		// Server timeouts: Go's defaults are zero (no deadline), which lets a
		// stuck client hold a connection forever. ReadHeaderTimeout especially
		// mitigates slowloris-style hangs.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.HTTPTimeout,
		WriteTimeout:      cfg.HTTPTimeout,
		IdleTimeout:       60 * time.Second,
		Handler:           httpapi.NewRouter(logger, db, runner),
	}

	// Serve in a goroutine so the main flow reaches the shutdown select below.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining")
	}

	// Drain: give in-flight requests 10s, then force. Connections are closed
	// abruptly by Close if the deadline passes.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown timed out", slog.Any("err", err))
	}
	logger.Info("stopped")
	return nil
}

// maskDSN hides the password before a DSN reaches the logs. The loop this
// replaced kept everything before '@', which is exactly where a URL DSN keeps
// user:password. URL DSNs go through url.URL.Redacted, plus the password query
// parameter that libpq also accepts; keyword/value DSNs get password=...
// masked. An unparsable URL isn't logged at all, since it may hold a password.
func maskDSN(dsn string) string {
	if !strings.Contains(dsn, "://") {
		return keywordPassword.ReplaceAllString(dsn, "${1}xxxxx")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "(unparsable database URL)"
	}
	if q := u.Query(); q.Has("password") {
		q.Set("password", "xxxxx")
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}

// keywordPassword matches password=value in a keyword/value DSN, where the
// value is single-quoted (with backslash escapes) or runs to the next space.
var keywordPassword = regexp.MustCompile(`(password\s*=\s*)('(?:[^'\\]|\\.)*'|\S*)`)
