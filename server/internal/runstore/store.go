// Package runstore records every executed example in the app database's run
// history (the runs table). GET /api/history (learningstore.History) reads it
// back; no screen calls that endpoint yet.
//
// The history is bounded: each insert also deletes everything but the newest
// rows, in the same transaction. Every row holds the code and the full result
// JSON, so an unbounded table would grow with every Run press for as long as
// the app is used. Rejected alternatives: a periodic cleanup job (a goroutine
// and a schedule to own, for one DELETE) and age-based expiry (a burst of
// runs on one day could still grow the table without limit).
package runstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stacklab/server/internal/runapi"
)

// DefaultKeep is how many runs the history keeps: ten times what GET
// /api/history returns. A typical row (an example of at most 80 lines plus its
// output) is a few kilobytes, so the table stays at a few megabytes.
const DefaultKeep = 1000

// Store persists run history in app-db.
type Store struct {
	pool *pgxpool.Pool
	keep int
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool, keep: DefaultKeep} }

// Record inserts one run and trims the history to the newest s.keep rows.
// History failures must never fail the run itself, so callers log the error
// and continue; a failed trim rolls the insert back with it.
func (s *Store) Record(ctx context.Context, req runapi.Request, res *runapi.Result) error {
	resultJSON, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("runstore: marshal result: %w", err)
	}
	code := ""
	for _, body := range req.Files {
		code = body
		break
	}
	const insert = `
		INSERT INTO runs (lesson_id, example_id, runtime, code, result)
		VALUES ($1, $2, $3, $4, $5)`
	// runs_created_at_idx serves the ORDER BY; with at most keep+1 rows in
	// the table, the subquery reads at most keep+1 index entries.
	const trim = `
		DELETE FROM runs
		WHERE id IN (SELECT id FROM runs ORDER BY created_at DESC OFFSET $1)`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, insert, req.LessonID, req.Example, string(req.Runtime), code, resultJSON); err != nil {
			return fmt.Errorf("runstore: insert: %w", err)
		}
		if _, err := tx.Exec(ctx, trim, s.keep); err != nil {
			return fmt.Errorf("runstore: trim: %w", err)
		}
		return nil
	})
}
