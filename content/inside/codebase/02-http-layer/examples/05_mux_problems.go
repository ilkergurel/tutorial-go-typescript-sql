package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// LEARN(inside.codebase.http-layer): middleware.go's withProblemFallbacks in small.

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "detail": detail})
}

type fallbackWriter struct {
	http.ResponseWriter
	r        *http.Request
	replaced bool
}

func (f *fallbackWriter) WriteHeader(code int) {
	switch code {
	case http.StatusNotFound:
		f.replaced = true
		problem(f.ResponseWriter, code, "no route for "+f.r.Method+" "+f.r.URL.Path)
	case http.StatusMethodNotAllowed:
		f.replaced = true
		problem(f.ResponseWriter, code, f.r.Method+" is not allowed; allowed: "+f.Header().Get("Allow"))
	default:
		f.ResponseWriter.WriteHeader(code) // the mux's path-cleaning redirects
	}
}

func (f *fallbackWriter) Write(b []byte) (int, error) {
	if f.replaced {
		return len(b), nil // drop http.Error's text body
	}
	return f.ResponseWriter.Write(b)
}

func withProblemFallbacks(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern == "" { // no route matched
			w = &fallbackWriter{ResponseWriter: w, r: r}
		}
		mux.ServeHTTP(w, r) // not the handler Handler returned: only ServeHTTP fills r.PathValue
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/notes/{lessonId}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound) // a handler's own 404, in its own format
		fmt.Fprintln(w, "no note for", r.PathValue("lessonId"))
	})
	mux.HandleFunc("PUT /api/notes/{lessonId}", func(http.ResponseWriter, *http.Request) {})

	h := withProblemFallbacks(mux)
	for _, c := range [][2]string{
		{"GET", "/api/notes/go.basics.defer"},
		{"DELETE", "/api/notes/go.basics.defer"},
		{"GET", "/api/lessons/go.basics.defer"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c[0], c[1], nil))
		fmt.Printf("%s %s\n  -> %d [%s] allow=%q %s\n", c[0], c[1], rec.Code, rec.Header().Get("Content-Type"),
			rec.Header().Get("Allow"), strings.TrimSpace(rec.Body.String()))
	}
}
