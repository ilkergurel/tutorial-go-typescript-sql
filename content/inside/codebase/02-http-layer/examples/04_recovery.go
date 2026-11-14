package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// LEARN(inside.codebase.http-layer): middleware.go's withRecovery in small.

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "detail": detail})
}

type writeTracker struct {
	http.ResponseWriter
	wrote bool
}

func (t *writeTracker) WriteHeader(code int)        { t.wrote = true; t.ResponseWriter.WriteHeader(code) }
func (t *writeTracker) Write(b []byte) (int, error) { t.wrote = true; return t.ResponseWriter.Write(b) }

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tw := &writeTracker{ResponseWriter: w}
		defer func() {
			switch v := recover(); {
			case v == nil:
			case v == http.ErrAbortHandler:
				panic(v) // a deliberate abort: net/http drops the connection quietly
			default:
				fmt.Println("  log: handler panicked:", v) // the real one uses slog, plus the stack
				if tw.wrote {
					panic(http.ErrAbortHandler) // too late for a 500: abort the half-sent reply
				}
				problem(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(tw, r)
	})
}

func main() {
	for _, c := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"panic before writing", func(http.ResponseWriter, *http.Request) {
			var notes map[string]string
			notes["go.basics.defer"] = "draft" // a nil map: this line panics
		}},
		{"panic after writing", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"stdout":"hi`)
			panic("encoder bug")
		}},
		{"deliberate abort", func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }},
	} {
		fmt.Println(c.name)
		func() {
			defer func() { // stands in for net/http's own recover in its connection goroutine
				if v := recover(); v != nil {
					fmt.Printf("  net/http gets panic(%v) and drops the connection\n", v)
				}
			}()
			rec := httptest.NewRecorder()
			withRecovery(c.handler).ServeHTTP(rec, httptest.NewRequest("GET", "/api/notes/x", nil))
			fmt.Printf("  -> %d [%s] %s\n", rec.Code, rec.Header().Get("Content-Type"), strings.TrimSpace(rec.Body.String()))
		}()
	}
}
