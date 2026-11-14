// LEARN(fullstack.tracelite.step-11-jobs): a job table as a queue: one-statement SKIP LOCKED claims, a lease that brings back a dead worker's job, a token that fences off a late finisher.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Job struct {
	ID, Lease uuid.UUID
	Name      string
	Attempts  int
}

var ErrLeaseLost = errors.New("lease lost: the job was claimed again")

type querier interface { // a pool, or a transaction
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// claim takes the job that has been due longest and leases it for two minutes.
func claim(ctx context.Context, db querier, now time.Time) (Job, error) {
	var j Job
	err := db.QueryRow(ctx, `UPDATE jobs SET state = 'running', attempts = attempts + 1, run_after = $2, lease_token = $3
		WHERE id = (SELECT id FROM jobs WHERE state IN ('queued', 'running') AND run_after <= $1
		            ORDER BY run_after, id LIMIT 1 FOR UPDATE SKIP LOCKED)
		RETURNING id, payload->>'name', attempts, lease_token`,
		now, now.Add(2*time.Minute), uuid.New()).Scan(&j.ID, &j.Name, &j.Attempts, &j.Lease)
	return j, err
}

// finish marks j done, unless its lease has passed to a later claim.
func finish(ctx context.Context, db *pgxpool.Pool, j Job) error {
	tag, err := db.Exec(ctx, "UPDATE jobs SET state = 'done', lease_token = NULL WHERE id = $1 AND lease_token = $2", j.ID, j.Lease)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrLeaseLost
	}
	return err
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	defer db.Close()
	_, err = db.Exec(ctx, `CREATE TABLE jobs (id uuid PRIMARY KEY DEFAULT uuidv7(), payload jsonb,
		  state text DEFAULT 'queued', attempts int DEFAULT 0, run_after timestamptz, lease_token uuid);
		INSERT INTO jobs (payload, run_after) VALUES ('{"name": "ci"}', '2026-11-13 06:02:30Z'), ('{"name": "chat"}', '2026-11-13 06:02:30Z')`)
	check(err)
	clock := time.Date(2026, 11, 13, 6, 2, 30, 0, time.UTC)
	tx, err := db.Begin(ctx)
	check(err)
	w1, err1 := claim(ctx, tx, clock) // w1's claim; its transaction is still open
	w2, err2 := claim(ctx, db, clock) // w2, on another connection, at the same moment
	check(errors.Join(err1, err2, tx.Commit(ctx)))
	fmt.Println("w1 claimed", w1.Name, "and w2 claimed", w2.Name, "without waiting for w1")
	check(finish(ctx, db, w2))
	clock = clock.Add(time.Minute) // w1 never reports back: killed, or frozen. Its lease ends at 06:04:30.
	_, err = claim(ctx, db, clock)
	fmt.Println(clock.Format("15:04:05"), "claim:", err)
	clock = clock.Add(time.Minute + time.Second)
	w3, err := claim(ctx, db, clock)
	check(err)
	fmt.Println(clock.Format("15:04:05"), "w3 claimed", w3.Name, "attempt", w3.Attempts)
	fmt.Println("w1 comes back, finishes:", finish(ctx, db, w1), "| w3 finishes:", finish(ctx, db, w3))
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
