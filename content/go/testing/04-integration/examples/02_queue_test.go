package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Claim takes the oldest unclaimed job for worker (pgx.ErrNoRows: none left).
func Claim(ctx context.Context, db *pgxpool.Pool, worker string) (int64, error) {
	var id int64
	err := db.QueryRow(ctx, `UPDATE jobs SET worker = $1 WHERE id = (
		SELECT id FROM jobs WHERE worker IS NULL
		ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING id`, worker).Scan(&id)
	return id, err
}

var pool *pgxpool.Pool // nil when no database is configured

func TestMain(m *testing.M) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), url)
		if err == nil { // once per test binary: stands in for the migrations
			_, err = pool.Exec(context.Background(),
				"CREATE TABLE jobs (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, worker text)")
		}
		if err != nil {
			log.Fatal(err)
		}
		defer pool.Close()
	}
	m.Run()
}

func TestClaimHandsEachJobOutOnce(t *testing.T) {
	if pool == nil {
		t.Skip("DATABASE_URL not set")
	}
	// Clear committed rows before each test, not after: a failure's rows stay to inspect.
	if _, err := pool.Exec(t.Context(), "TRUNCATE jobs RESTART IDENTITY"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), "INSERT INTO jobs (worker) SELECT NULL FROM generate_series(1, 50)"); err != nil {
		t.Fatal(err)
	}
	claimed := make([][]int64, 4) // one slice per worker, so no shared writes
	var wg sync.WaitGroup
	for w := range claimed { // four workers, four pool connections, really concurrent
		wg.Go(func() {
			for {
				id, err := Claim(t.Context(), pool, fmt.Sprint("w", w))
				if err != nil {
					if !errors.Is(err, pgx.ErrNoRows) {
						t.Error(err)
					}
					return
				}
				claimed[w] = append(claimed[w], id)
			}
		})
	}
	wg.Wait()
	ids := slices.Concat(claimed...)
	total := len(ids)
	slices.Sort(ids)
	if distinct := len(slices.Compact(ids)); total != 50 || distinct != 50 {
		t.Fatalf("claimed %d jobs, %d distinct; want 50 and 50", total, distinct)
	}
	t.Logf("50 jobs, 4 workers: every job claimed exactly once")
}
