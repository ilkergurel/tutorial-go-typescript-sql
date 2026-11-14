package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
)

// Store is the whole data layer. In a real service: one SQL statement per method.
type Store struct {
	mu  sync.Mutex
	ids map[string]bool
}

func (s *Store) Add(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ids == nil {
		s.ids = map[string]bool{}
	}
	s.ids[id] = true
}

func (s *Store) List() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Sorted(maps.Keys(s.ids))
}

// routes is the whole application: decode the request, call the store, encode.
func routes(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /bookmarks/{id}", func(w http.ResponseWriter, r *http.Request) {
		s.Add(r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /bookmarks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.List())
	})
	return mux
}

func main() {
	h := routes(&Store{})
	for _, req := range []string{
		"PUT /bookmarks/sql.joins.lateral",
		"PUT /bookmarks/go.basics.defer",
		"PUT /bookmarks/go.basics.defer",
		"GET /bookmarks",
	} {
		method, path, _ := strings.Cut(req, " ")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		fmt.Println(strings.TrimSpace(fmt.Sprintf("%s -> %d %s", req, rec.Code, rec.Body)))
	}
}
