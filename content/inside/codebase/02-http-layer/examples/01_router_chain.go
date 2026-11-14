package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

// LEARN(inside.codebase.http-layer): internal/httpapi/{router,middleware}.go in small.

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int)        { r.status = code; r.ResponseWriter.WriteHeader(code) }
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func withLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", rec.status)
		})
	}
}

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type": "about:blank", "title": http.StatusText(status), "status": status, "detail": detail,
	})
}

func NewRouter(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/notes/{lessonId}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"lesson": r.PathValue("lessonId")})
	})
	mux.HandleFunc("PUT /api/notes/{lessonId}", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Body string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
			problem(w, http.StatusBadRequest, "invalid request body")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return withLogging(logger)(mux) // the real chain adds recovery, guards and problem 404/405 (examples 2, 4, 5)
}

func main() {
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{} // the real logger keeps the time and also logs a duration
		}
		return a
	}
	h := NewRouter(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: noTime})))
	for _, c := range [][3]string{
		{"GET", "/api/notes/go.basics.defer", ""},
		{"PUT", "/api/notes/go.basics.defer", `{"body": `},
		{"DELETE", "/api/notes/go.basics.defer", ""},
		{"GET", "/api/lessons/go.basics.defer", ""},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c[0], c[1], strings.NewReader(c[2])))
		fmt.Printf("  -> %d [%s] allow=%q %s\n", rec.Code, rec.Header().Get("Content-Type"),
			rec.Header().Get("Allow"), strings.TrimSpace(rec.Body.String()))
	}
}
