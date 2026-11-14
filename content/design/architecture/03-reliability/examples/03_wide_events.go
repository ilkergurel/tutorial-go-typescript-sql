// LEARN(design.architecture.reliability): one structured event per request; liveness vs readiness.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"
)

type eventKey struct{}

// annotate adds fields to the request's event; the middleware writes it once.
func annotate(ctx context.Context, attrs ...slog.Attr) {
	if ev, ok := ctx.Value(eventKey{}).(*[]slog.Attr); ok {
		*ev = append(*ev, attrs...)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }

func observe(log *slog.Logger, now func() time.Time, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, ev := now(), &[]slog.Attr{}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		r = r.WithContext(context.WithValue(r.Context(), eventKey{}, ev))
		next.ServeHTTP(rec, r) // the mux sets r.Pattern on this request, not the original
		log.LogAttrs(r.Context(), slog.LevelInfo, "request", append([]slog.Attr{
			slog.String("route", r.Pattern), // low cardinality, unlike r.URL.Path
			slog.Int("status", rec.status),
			slog.Duration("dur", now().Sub(start)),
		}, *ev...)...)
	})
}

func main() {
	var elapsed atomic.Int64 // a fake clock, so the durations below are stable
	now := func() time.Time { return time.Unix(0, elapsed.Load()) }
	var dbDown atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(http.ResponseWriter, *http.Request) {}) // alive: don't restart me
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if dbDown.Load() { // ready: send me traffic only while my dependencies work
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	mux.HandleFunc("GET /notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		if dbDown.Load() {
			elapsed.Add(int64(2 * time.Second)) // the query ran into its own deadline
			annotate(r.Context(), slog.String("db", "timeout"), slog.String("note", r.PathValue("id")))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		elapsed.Add(int64(12 * time.Millisecond))
		annotate(r.Context(), slog.String("db", "ok"))
	})
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	h := observe(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: noTime})), now, mux)
	get := func(path string) { h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil)) }
	get("/notes/go.basics.defer")
	fmt.Println("-- the database goes down")
	dbDown.Store(true)
	get("/healthz")
	get("/readyz")
	get("/notes/sql.joins.inner")
}
