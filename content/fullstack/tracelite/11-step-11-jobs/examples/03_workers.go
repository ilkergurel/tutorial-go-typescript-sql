package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

func claim(ctx context.Context, db *pgxpool.Pool) (id, lease uuid.UUID, err error) {
	err = db.QueryRow(ctx, `UPDATE jobs SET state = 'running', attempts = attempts + 1, -- example 1's claim, on Go's clock
		  run_after = $1::timestamptz + interval '2 minutes', lease_token = $2
		WHERE id = (SELECT id FROM jobs WHERE state IN ('queued', 'running') AND run_after <= $1
		            ORDER BY run_after, id LIMIT 1 FOR UPDATE SKIP LOCKED)
		RETURNING id, lease_token`, time.Now(), uuid.New()).Scan(&id, &lease)
	return id, lease, err
}

func runPool(stop context.Context, db *pgxpool.Pool, workers int, drain time.Duration) {
	jobs, cancelJobs := context.WithCancel(context.WithoutCancel(stop))
	defer cancelJobs()
	context.AfterFunc(stop, func() { time.AfterFunc(drain, cancelJobs) })
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for stop.Err() == nil {
				if id, lease, err := claim(jobs, db); err == nil {
					run(jobs, db, id, lease)
				} else {
					time.Sleep(50 * time.Millisecond) // nothing due, or the database is away
				}
			}
		})
	}
	wg.Wait()
}

func run(ctx context.Context, db *pgxpool.Pool, id, lease uuid.UUID) {
	// pg_sleep stands in for step 8's report query; only shutdown makes it fail here.
	_, err := db.Exec(ctx, "SELECT pg_sleep((payload->>'seconds')::float8) FROM jobs WHERE id = $1", id)
	sql := "UPDATE jobs SET state = 'done', finished_at = $3, lease_token = NULL WHERE id = $1 AND lease_token = $2"
	if err != nil { // interrupted by shutdown, which is not the job's fault: its attempt is given back
		sql = `UPDATE jobs SET state = 'queued', attempts = attempts - 1, run_after = $3, lease_token = NULL,
			last_error = 'interrupted by shutdown' WHERE id = $1 AND lease_token = $2`
	}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second) // ctx may be canceled by now
	defer cancel()
	if _, err := db.Exec(bg, sql, id, lease, time.Now()); err != nil {
		log.Print(err)
	}
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = db.Exec(ctx, `CREATE TABLE jobs (id uuid PRIMARY KEY DEFAULT uuidv7(), payload jsonb, state text DEFAULT 'queued',
			  attempts int DEFAULT 0, run_after timestamptz DEFAULT now(), lease_token uuid, last_error text DEFAULT '', finished_at timestamptz);
			INSERT INTO jobs (payload) VALUES ('{"project": "Ödeme", "seconds": 1.5}'), ('{"project": "Raporlama", "seconds": 0.3}'),
			  ('{"project": "Arşiv", "seconds": 0.1}')`)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	sigterm, cancel := context.WithTimeout(ctx, 100*time.Millisecond) // signal.NotifyContext(ctx, syscall.SIGTERM) in main
	defer cancel()
	runPool(sigterm, db, 2, 700*time.Millisecond)
	var table string
	if err := db.QueryRow(ctx, `SELECT string_agg(format('%-9s %-6s attempts %s %L', payload->>'project', state, attempts,
		last_error), E'\n' ORDER BY id) FROM jobs`).Scan(&table); err != nil {
		log.Fatal(err)
	}
	fmt.Println(table)
}
