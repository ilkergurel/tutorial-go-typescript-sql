package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrLimit = errors.New("bookmark limit reached")

type memStore struct {
	mu   sync.Mutex
	ids  []string
	gate sync.WaitGroup // holds each request after counting until both have
}

// A CRUD-shaped port splits the rule across two calls: Count, then Add.
func (s *memStore) Count(context.Context) (int, error) {
	s.mu.Lock()
	n := len(s.ids)
	s.mu.Unlock()
	s.gate.Done()
	s.gate.Wait() // the other request runs here, as it can in production
	return n, nil
}

func (s *memStore) Add(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, id)
	return nil
}

// A use-case-shaped port makes check and insert one operation of the store.
func (s *memStore) AddWithin(_ context.Context, id string, limit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ids) >= limit {
		return ErrLimit
	}
	s.ids = append(s.ids, id)
	return nil
}

// addSplit is the service method from the previous example, rule and all.
func addSplit(ctx context.Context, s *memStore, id string, limit int) error {
	n, err := s.Count(ctx)
	if err != nil {
		return err
	}
	if n >= limit {
		return ErrLimit
	}
	return s.Add(ctx, id)
}

// run starts two requests at once against a store that already holds two.
func run(name string, add func(*memStore, string) error) {
	s := &memStore{ids: []string{"go.basics.defer", "sql.joins.lateral"}}
	s.gate.Add(2)
	errs := make([]error, 2) // one slot per request, so no lock is needed
	var wg sync.WaitGroup
	for i, id := range []string{"ts.react.query", "go.http.middleware"} {
		wg.Go(func() { errs[i] = add(s, id) })
	}
	wg.Wait()
	fmt.Printf("%-10s %d stored (limit 3), rejected: %v\n", name+":", len(s.ids), errors.Join(errs...))
}

func main() {
	ctx := context.Background()
	run("Count+Add", func(s *memStore, id string) error { return addSplit(ctx, s, id, 3) })
	run("AddWithin", func(s *memStore, id string) error { return s.AddWithin(ctx, id, 3) })
}
