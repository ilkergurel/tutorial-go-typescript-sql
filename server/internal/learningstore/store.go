// Package learningstore persists per-lesson learning state: progress status,
// notes, bookmarks, and quiz answers. Single-user: lesson_id is the primary key.
package learningstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Status is the lesson progress state machine: viewed (opened), ran (executed
// an example), completed (owner marks it done).
type Status string

const (
	StatusViewed    Status = "viewed"
	StatusRan       Status = "ran"
	StatusCompleted Status = "completed"
)

// mark is the shared upsert behind MarkViewed/MarkRan: status only moves
// forward (viewed < ran < completed), so a late 'viewed' never downgrades.
func (s *Store) mark(ctx context.Context, lessonID string, to Status) error {
	const q = `
		INSERT INTO progress (lesson_id, status) VALUES ($1, $2)
		ON CONFLICT (lesson_id) DO UPDATE SET
			status = CASE
				WHEN progress.status = 'completed' THEN 'completed'
				WHEN progress.status = 'ran' AND excluded.status = 'viewed' THEN 'ran'
				ELSE excluded.status
			END,
			completed_at = CASE
				WHEN excluded.status = 'completed' THEN now()
				ELSE progress.completed_at
			END`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, q, lessonID, string(to))
	if err != nil {
		return fmt.Errorf("learningstore: mark %s: %w", to, err)
	}
	return nil
}

func (s *Store) MarkViewed(ctx context.Context, lessonID string) error {
	return s.mark(ctx, lessonID, StatusViewed)
}

func (s *Store) MarkRan(ctx context.Context, lessonID string) error {
	return s.mark(ctx, lessonID, StatusRan)
}

func (s *Store) MarkCompleted(ctx context.Context, lessonID string) error {
	return s.mark(ctx, lessonID, StatusCompleted)
}

// ProgressRow is one lesson's progress for list views.
type ProgressRow struct {
	LessonID    string     `json:"lessonId"`
	Status      Status     `json:"status"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (s *Store) ListProgress(ctx context.Context) ([]ProgressRow, error) {
	const q = `SELECT lesson_id, status, completed_at FROM progress ORDER BY lesson_id`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("learningstore: list progress: %w", err)
	}
	defer rows.Close()
	var out []ProgressRow
	for rows.Next() {
		var r ProgressRow
		if err := rows.Scan(&r.LessonID, &r.Status, &r.CompletedAt); err != nil {
			return nil, fmt.Errorf("learningstore: scan progress: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetNote(ctx context.Context, lessonID string) (string, error) {
	const q = `SELECT body FROM notes WHERE lesson_id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var body string
	err := s.pool.QueryRow(ctx, q, lessonID).Scan(&body)
	if err != nil {
		if isNoRows(err) {
			return "", nil
		}
		return "", fmt.Errorf("learningstore: get note: %w", err)
	}
	return body, nil
}

func (s *Store) SaveNote(ctx context.Context, lessonID, body string) error {
	const q = `
		INSERT INTO notes (lesson_id, body, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (lesson_id) DO UPDATE SET body = excluded.body, updated_at = now()`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, q, lessonID, body)
	if err != nil {
		return fmt.Errorf("learningstore: save note: %w", err)
	}
	return nil
}

func (s *Store) ListBookmarks(ctx context.Context) ([]string, error) {
	const q = `SELECT lesson_id FROM bookmarks ORDER BY created_at DESC`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("learningstore: list bookmarks: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("learningstore: scan bookmark: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) AddBookmark(ctx context.Context, lessonID string) error {
	const q = `INSERT INTO bookmarks (lesson_id) VALUES ($1) ON CONFLICT DO NOTHING`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, q, lessonID)
	if err != nil {
		return fmt.Errorf("learningstore: add bookmark: %w", err)
	}
	return nil
}

func (s *Store) RemoveBookmark(ctx context.Context, lessonID string) error {
	const q = `DELETE FROM bookmarks WHERE lesson_id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, q, lessonID)
	if err != nil {
		return fmt.Errorf("learningstore: remove bookmark: %w", err)
	}
	return nil
}

// GetQuizAnswers returns {questionIndex: optionIndex}. Missing lesson = empty.
func (s *Store) GetQuizAnswers(ctx context.Context, lessonID string) (map[string]int, error) {
	const q = `SELECT answers FROM quiz_answers WHERE lesson_id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var raw []byte
	err := s.pool.QueryRow(ctx, q, lessonID).Scan(&raw)
	if err != nil {
		if isNoRows(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("learningstore: get quiz answers: %w", err)
	}
	out := map[string]int{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("learningstore: decode quiz answers: %w", err)
	}
	// A stored JSON null (from a PUT with body null, before SaveQuizAnswers
	// guarded against it) unmarshals into a nil map, which would go out as null.
	if out == nil {
		out = map[string]int{}
	}
	return out, nil
}

func (s *Store) SaveQuizAnswers(ctx context.Context, lessonID string, answers map[string]int) error {
	// A nil map marshals to null; the API promises an object.
	if answers == nil {
		answers = map[string]int{}
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return fmt.Errorf("learningstore: encode quiz answers: %w", err)
	}
	const q = `
		INSERT INTO quiz_answers (lesson_id, answers, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (lesson_id) DO UPDATE SET answers = excluded.answers, updated_at = now()`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err = s.pool.Exec(ctx, q, lessonID, raw)
	if err != nil {
		return fmt.Errorf("learningstore: save quiz answers: %w", err)
	}
	return nil
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// RunRow is one historical run for the history list.
type RunRow struct {
	ID        string    `json:"id"`
	LessonID  string    `json:"lessonId"`
	ExampleID string    `json:"exampleId"`
	Runtime   string    `json:"runtime"`
	CreatedAt time.Time `json:"createdAt"`
}

// History returns recent runs, newest first, optionally for one lesson.
func (s *Store) History(ctx context.Context, lessonID string) ([]RunRow, error) {
	const q = `SELECT id, lesson_id, example_id, runtime, created_at FROM runs
		WHERE ($1 = '' OR lesson_id = $1)
		ORDER BY created_at DESC LIMIT 100`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, q, lessonID)
	if err != nil {
		return nil, fmt.Errorf("learningstore: history: %w", err)
	}
	defer rows.Close()
	var out []RunRow
	for rows.Next() {
		var r RunRow
		if err := rows.Scan(&r.ID, &r.LessonID, &r.ExampleID, &r.Runtime, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("learningstore: scan history: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
