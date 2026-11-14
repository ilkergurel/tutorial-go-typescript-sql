package main

import (
	"database/sql"
	"io/fs"
	"slices"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	"github.com/pressly/goose/v3"
)

// TestGooseSeesEveryEmbeddedMigration guards the go:embed/goose contract. When
// the embed FS was passed to goose unchanged, goose found zero migrations and
// the backend container exited at startup; this catches that without a
// database, because sql.Open never dials and NewProvider only collects files.
func TestGooseSeesEveryEmbeddedMigration(t *testing.T) {
	embedded, err := fs.Glob(embeddedMigrations, "migrations/*.sql")
	if err != nil || len(embedded) == 0 {
		t.Fatalf("no embedded migrations (err=%v)", err)
	}

	fsys, err := migrationsFS()
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", "postgres://unused@127.0.0.1:1/unused")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		t.Fatalf("goose.NewProvider: %v", err)
	}
	if got := len(provider.ListSources()); got != len(embedded) {
		t.Fatalf("goose sees %d migrations, %d are embedded", got, len(embedded))
	}
}

// TestMigrationsHaveGooseUpAnnotation catches the other startup failure: goose
// only parses a file when it is applied, so a file without "-- +goose Up"
// passes the test above and then kills the backend with "unexpected state 0".
// goose's own parser is internal, so this checks the one required line.
func TestMigrationsHaveGooseUpAnnotation(t *testing.T) {
	names, err := fs.Glob(embeddedMigrations, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		body, err := fs.ReadFile(embeddedMigrations, name)
		if err != nil {
			t.Fatal(err)
		}
		// TrimSpace per line: a CRLF checkout on Windows must not fail falsely.
		if !slices.ContainsFunc(strings.Split(string(body), "\n"), func(line string) bool {
			return strings.TrimSpace(line) == "-- +goose Up"
		}) {
			t.Errorf("%s: missing a \"-- +goose Up\" line", name)
		}
	}
}

// The startup log prints the DSN; the old masking kept everything before '@',
// which in a URL DSN is user:password.
func TestMaskDSNHidesPasswords(t *testing.T) {
	for dsn, want := range map[string]string{
		"postgres://stacklab:s3cret@127.0.0.1:5432/stacklab?sslmode=disable": "postgres://stacklab:xxxxx@127.0.0.1:5432/stacklab?sslmode=disable",
		"postgres://stacklab@127.0.0.1/stacklab?password=s3cret":             "postgres://stacklab@127.0.0.1/stacklab?password=xxxxx",
		"postgres://127.0.0.1/stacklab":                                      "postgres://127.0.0.1/stacklab",
		"host=127.0.0.1 user=stacklab password=s3cret dbname=stacklab":       "host=127.0.0.1 user=stacklab password=xxxxx dbname=stacklab",
		`host=127.0.0.1 password = 'it\'s s3cret' dbname=stacklab`:           "host=127.0.0.1 password = xxxxx dbname=stacklab",
		"postgres://u:s3cret@[bad":                                           "(unparsable database URL)",
	} {
		got := maskDSN(dsn)
		if got != want {
			t.Errorf("maskDSN(%q) = %q, want %q", dsn, got, want)
		}
		if strings.Contains(got, "s3cret") {
			t.Errorf("maskDSN(%q) leaks the password: %q", dsn, got)
		}
	}
}
