package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is step 10's envelope.
type Event struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      any       `json:"data"`
}

// recordFailedRun stores a failed run and, in the same transaction, one delivery job per subscribed webhook.
func recordFailedRun(ctx context.Context, tx pgx.Tx, project, testCase uuid.UUID, at time.Time) error {
	_, err := tx.Exec(ctx, "INSERT INTO test_runs (test_case_id, result, finished_at) VALUES ($1, 'failed', $2)", testCase, at)
	if err != nil {
		return err
	}
	body, err := json.Marshal(Event{uuid.NewV7(), "test-run.failed", at, map[string]string{"testCase": "TC-3"}})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs (project_id, kind, payload)
		SELECT project_id, 'webhook', jsonb_build_object('webhookId', id, 'body', $2::text)
		FROM webhooks WHERE project_id = $1 AND 'test-run.failed' = ANY (events)`, project, string(body))
	return err
}

func main() {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err == nil {
		_, err = db.Exec(ctx, schema)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	project, testCase := uuid.MustParse("01937b6e-0000-7000-8000-00000000000a"), uuid.MustParse("01937b6e-0000-7000-8000-000000000003")
	at := time.Date(2026, 11, 13, 6, 2, 30, 0, time.UTC)
	err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error { return recordFailedRun(ctx, tx, project, testCase, at) })
	fmt.Println("run 1 committed:", err)
	err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		if err := recordFailedRun(ctx, tx, project, testCase, at); err != nil {
			return err
		}
		return errors.New("a later statement failed") // so the whole transaction rolls back
	})
	fmt.Println("run 2 rolled back:", err)
	var runs, jobs, events int
	if err := db.QueryRow(ctx, `SELECT (SELECT count(*) FROM test_runs), count(*), count(DISTINCT (payload->>'body')::jsonb->>'id')
		FROM jobs`).Scan(&runs, &jobs, &events); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("test runs: %d, delivery jobs: %d, event ids among them: %d\n", runs, jobs, events)
}

const schema = `CREATE TABLE projects (id uuid PRIMARY KEY);
CREATE TABLE test_cases (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects);
CREATE TABLE test_runs (id uuid PRIMARY KEY DEFAULT uuidv7(), test_case_id uuid NOT NULL REFERENCES test_cases,
  result text NOT NULL, finished_at timestamptz NOT NULL);
CREATE TABLE webhooks (id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL REFERENCES projects,
  url text NOT NULL, events text[] NOT NULL DEFAULT '{test-run.failed}');
CREATE TABLE jobs (id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL, kind text NOT NULL, payload jsonb NOT NULL);
INSERT INTO projects VALUES ('01937b6e-0000-7000-8000-00000000000a');
INSERT INTO test_cases VALUES ('01937b6e-0000-7000-8000-000000000003', '01937b6e-0000-7000-8000-00000000000a');
INSERT INTO webhooks (project_id, url) VALUES ('01937b6e-0000-7000-8000-00000000000a', 'https://ci.example/hooks'),
  ('01937b6e-0000-7000-8000-00000000000a', 'https://chat.example/hooks');`
