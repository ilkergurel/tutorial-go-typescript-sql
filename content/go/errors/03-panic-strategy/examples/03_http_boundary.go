package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
)

var ErrNotFound = errors.New("not found")

// appHandler returns its error instead of writing it, so one place maps
// errors to status codes, logs, and recovers.
type appHandler func(w http.ResponseWriter, r *http.Request) error

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p := recover(); p != nil {
			if p == http.ErrAbortHandler {
				panic(p) // a deliberate abort: let net/http handle it
			}
			log.Printf("panic: %s %s: %v", r.Method, r.URL.Path, p) // + debug.Stack()
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}()
	switch err := h(w, r); {
	case err == nil:
	case errors.Is(err, ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func user(w http.ResponseWriter, r *http.Request) error {
	if id := r.PathValue("id"); id != "0" {
		return fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	var tags []string
	_, err := io.WriteString(w, tags[0]) // a bug: index out of range
	return err
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /raw", func(http.ResponseWriter, *http.Request) { panic("boom") })
	mux.Handle("GET /users/{id}", appHandler(user))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/raw", "/users/7", "/users/0"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			fmt.Printf("%-9s no response: net/http recovered and dropped the connection\n", path)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%-9s %d %s", path, resp.StatusCode, body)
	}
}
