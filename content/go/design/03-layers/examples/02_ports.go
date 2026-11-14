package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// The core: rules, their errors and the port they need. No HTTP, no SQL.
var (
	ErrBadID = errors.New("lesson id must look like track.module.slug")
	ErrLimit = errors.New("bookmark limit reached")
)

const maxBookmarks = 2

type Store interface { // a port, declared by the core that uses it
	Add(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

type Bookmarks struct{ store Store }

func (b *Bookmarks) Add(ctx context.Context, id string) error {
	if strings.Count(id, ".") != 2 {
		return fmt.Errorf("%q: %w", id, ErrBadID)
	}
	n, err := b.store.Count(ctx)
	if err != nil {
		return fmt.Errorf("count bookmarks: %w", err)
	}
	if n >= maxBookmarks {
		return ErrLimit
	}
	return b.store.Add(ctx, id)
}

// memStore is a driven adapter: storage, pgx in production, a slice here.
type memStore struct{ ids []string }

func (m *memStore) Add(_ context.Context, id string) error { m.ids = append(m.ids, id); return nil }
func (m *memStore) Count(context.Context) (int, error)     { return len(m.ids), nil }

// handleAdd is a driving adapter: it maps the core's errors to HTTP statuses.
func handleAdd(b *Bookmarks) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch err := b.Add(r.Context(), r.PathValue("id")); {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, ErrBadID):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrLimit):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func main() {
	b := &Bookmarks{store: &memStore{}}

	// Another driving adapter, an import job: the same rules, no HTTP.
	for _, id := range []string{"go.basics.defer", "defer"} {
		fmt.Println("import", id+":", b.Add(context.Background(), id))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /bookmarks/{id}", handleAdd(b))
	for _, id := range []string{"sql.joins.lateral", "lateral", "ts.react.query"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("PUT", "/bookmarks/"+id, nil))
		fmt.Println(strings.TrimSpace(fmt.Sprintf("PUT %s -> %d %s", id, rec.Code, rec.Body)))
	}
}
