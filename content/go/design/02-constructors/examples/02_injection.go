package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Note struct {
	Lesson, Text string
	At           time.Time
}

// NoteStore is what Notes needs, declared where it is used.
type NoteStore interface {
	Save(ctx context.Context, n Note) error
}

// Notes receives every dependency through its constructor, the clock included.
type Notes struct {
	store NoteStore
	now   func() time.Time
}

func NewNotes(store NoteStore, now func() time.Time) *Notes {
	return &Notes{store: store, now: now}
}

var ErrEmpty = errors.New("note is empty")

func (n *Notes) Add(ctx context.Context, lesson, text string) (Note, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Note{}, ErrEmpty
	}
	note := Note{Lesson: lesson, Text: text, At: n.now()}
	if err := n.store.Save(ctx, note); err != nil {
		return Note{}, fmt.Errorf("save note for %s: %w", lesson, err)
	}
	return note, nil
}

// memStore and downStore are what a test passes; main would pass a pgx store.
type memStore struct{ saved []Note }

func (m *memStore) Save(_ context.Context, n Note) error {
	m.saved = append(m.saved, n)
	return nil
}

type downStore struct{}

func (downStore) Save(context.Context, Note) error { return errors.New("connection refused") }

func main() {
	ctx := context.Background()
	clock := func() time.Time { return time.Date(2026, 11, 13, 9, 30, 0, 0, time.UTC) }

	store := &memStore{}
	notes := NewNotes(store, clock)
	n, err := notes.Add(ctx, "go.design.constructors", "  options or a Config?  ")
	fmt.Printf("%q at %s, err=%v\n", n.Text, n.At.Format(time.DateTime), err)
	_, err = notes.Add(ctx, "go.design.constructors", "   ")
	fmt.Println(err, errors.Is(err, ErrEmpty), "saved:", len(store.saved))

	_, err = NewNotes(downStore{}, clock).Add(ctx, "go.basics.defer", "defer runs LIFO")
	fmt.Println(err)
}
