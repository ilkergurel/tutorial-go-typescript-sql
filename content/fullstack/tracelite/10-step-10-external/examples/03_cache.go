package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrUnknownClause = errors.New("no such clause") // example 1's answer to a 404

// Clause is the answer of GET /api/standards/{ref}.
type Clause struct {
	Ref   string `json:"ref"`
	Title string `json:"title"`
	Stale bool   `json:"stale"` // from the cache, because the catalogue didn't answer
}

type entry struct {
	clause  Clause
	err     error // ErrUnknownClause: a remembered 404
	expires time.Time
}

type Standards struct {
	fetch func(ctx context.Context, ref string) (string, error) // example 2's fetchTitle
	now   func() time.Time
	mu    sync.Mutex // requests look up concurrently
	cache map[string]entry
}

func (s *Standards) Lookup(ctx context.Context, ref string) (Clause, error) {
	s.mu.Lock()
	e, cached := s.cache[ref]
	s.mu.Unlock()
	if cached && s.now().Before(e.expires) {
		return e.clause, e.err
	}
	title, err := s.fetch(ctx, ref) // not under the lock: it may take the whole budget
	switch {
	case err == nil:
		e = entry{clause: Clause{Ref: ref, Title: title}, expires: s.now().Add(24 * time.Hour)}
	case errors.Is(err, ErrUnknownClause):
		e = entry{err: err, expires: s.now().Add(10 * time.Minute)}
	case cached && e.err == nil:
		e.clause.Stale = true // an old title beats an error page
		return e.clause, nil
	default:
		return Clause{}, err
	}
	s.mu.Lock()
	s.cache[ref] = e
	s.mu.Unlock()
	return e.clause, e.err
}

func main() {
	clock := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	s := &Standards{now: func() time.Time { return clock }, cache: map[string]entry{}}
	s.fetch = func(ctx context.Context, ref string) (string, error) {
		fmt.Println("  catalogue asked for", ref)
		if ref == "ISO27001:Z.9" {
			return "", ErrUnknownClause
		}
		return "Authentication information", nil
	}
	lookup := func(ref string) {
		c, err := s.Lookup(context.Background(), ref)
		fmt.Printf("%s %s: %q, stale: %v, err: %v\n", clock.Format("Jan 2 15:04"), ref, c.Title, c.Stale, err)
	}
	for _, ref := range []string{"ISO27001:A.5.17", "ISO27001:A.5.17", "ISO27001:Z.9", "ISO27001:Z.9"} {
		lookup(ref) // each second lookup is answered from the cache, the 404 too
	}
	clock = clock.Add(11 * time.Minute)
	lookup("ISO27001:Z.9") // a 404 is kept for 10 minutes only
	clock = clock.Add(25 * time.Hour)
	s.fetch = func(context.Context, string) (string, error) { return "", errors.New("gave up: 503") } // down
	lookup("ISO27001:A.5.17")
}
