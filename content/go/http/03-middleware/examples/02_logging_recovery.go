package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
)

// statusRecorder remembers the status for the access log. A handler that
// never calls WriteHeader sends an implicit 200, hence the default.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the real writer through us.
func (rec *statusRecorder) Unwrap() http.ResponseWriter { return rec.ResponseWriter }

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status)
	})
}

func recoverer(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "path", r.URL.Path, "value", p) // + debug.Stack() in real code
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // no timestamps: stable output
			}
			return a
		},
	}))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "fine\n") })
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) {
		var hits map[string]int
		hits[r.URL.Path]++ // a bug: nil map
	})
	mux.HandleFunc("GET /flush", func(w http.ResponseWriter, r *http.Request) {
		hidden := struct{ http.ResponseWriter }{w} // a wrapper without Unwrap
		fmt.Fprintln(w, "with Unwrap:   ", http.NewResponseController(w).Flush())
		fmt.Fprintln(w, "without Unwrap:", http.NewResponseController(hidden).Flush())
	})
	h := logging(log, recoverer(log, mux))

	for _, path := range []string{"/ok", "/missing", "/boom", "/flush"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		fmt.Printf("client got %d:\n%s", rec.Code, rec.Body)
	}
}
