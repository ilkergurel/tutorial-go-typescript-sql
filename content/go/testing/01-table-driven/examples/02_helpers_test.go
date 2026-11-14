package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Journal appends lines to a file. Each test needs a fresh one.
type Journal struct{ f *os.File }

func OpenJournal(path string) (*Journal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	return &Journal{f: f}, err
}

func (j *Journal) Add(ctx context.Context, line string) error {
	if err := ctx.Err(); err != nil {
		return err // a canceled caller gets no write
	}
	_, err := fmt.Fprintln(j.f, line)
	return err
}

func (j *Journal) Close() error { return j.f.Close() }

// newJournal fails the test itself and arranges its own cleanup, so a
// test needs one line to get a journal.
func newJournal(t *testing.T) *Journal {
	t.Helper() // failures and logs below point at the caller's line
	j, err := OpenJournal(filepath.Join(t.TempDir(), "journal.txt"))
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	t.Cleanup(func() { // registered after TempDir's cleanup, so it runs before it
		t.Logf("cleanup: closing journal; t.Context().Err() = %v", t.Context().Err())
		if err := j.Close(); err != nil {
			t.Error(err)
		}
	})
	t.Log("journal opened")
	return j
}

func logWithoutHelper(t *testing.T) {
	t.Log("logged from a function without t.Helper")
}

func TestJournal(t *testing.T) {
	j := newJournal(t)
	logWithoutHelper(t)
	t.Cleanup(func() { t.Log("cleanup: registered last, runs first") })

	if err := j.Add(t.Context(), "ödeme alındı"); err != nil {
		t.Fatal(err)
	}
	t.Log("test body done")
}
