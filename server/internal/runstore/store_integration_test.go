//go:build integration

package runstore

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stacklab/server/internal/runapi"
)

// testStore returns a Store over a runs table in a schema of its own, created
// from the real migration and dropped afterwards. TEST_DATABASE_URL is the
// local playground database; its role may create schemas, as sqlrun does.
func testStore(t *testing.T, keep int) (*Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := "runstore_test_" + strings.ToLower(rand.Text())

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) })
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
		}
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	ddl, err := os.ReadFile("../../cmd/stacklab/migrations/002_runs.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(ddl)); err != nil {
		t.Fatalf("apply 002_runs.sql: %v", err)
	}
	return &Store{pool: pool, keep: keep}, pool
}

func examples(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT example_id FROM runs")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(ids)
	return ids
}

// Record keeps only the newest rows. The old rows get explicit timestamps an
// hour apart, so which ones survive doesn't depend on the clock's resolution.
func TestRecordKeepsOnlyTheNewestRuns(t *testing.T) {
	ctx := context.Background()
	store, pool := testStore(t, 3)
	for k := 1; k <= 5; k++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO runs (lesson_id, example_id, runtime, code, result, created_at)
			VALUES ('l', $1, 'go', '', '{}', now() - make_interval(hours => $2))`,
			fmt.Sprintf("old%d", k), k); err != nil {
			t.Fatal(err)
		}
	}

	record := func(example string) {
		t.Helper()
		req := runapi.Request{Runtime: runapi.RuntimeGo, Example: example, LessonID: "l", Files: map[string]string{"main.go": "package main"}}
		if err := store.Record(ctx, req, &runapi.Result{Stdout: "hi\n"}); err != nil {
			t.Fatalf("Record(%s): %v", example, err)
		}
	}

	record("new1")
	if got, want := examples(t, pool), []string{"new1", "old1", "old2"}; !slices.Equal(got, want) {
		t.Fatalf("after one run: %v, want %v", got, want)
	}
	record("new2")
	if got, want := examples(t, pool), []string{"new1", "new2", "old1"}; !slices.Equal(got, want) {
		t.Fatalf("after two runs: %v, want %v", got, want)
	}
}
