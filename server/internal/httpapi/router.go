// Package httpapi owns the HTTP surface: routing, middleware, and the single
// place where errors map to responses.
//
// Standard-library ServeMux with Go 1.22+ method patterns, per ADR-0001: the
// router is itself curriculum, and a framework would hide the machinery the
// HTTP lessons teach.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"stacklab/server/internal/learningstore"
	"stacklab/server/internal/platform"
	"stacklab/server/internal/runstore"
)

// NewRouter builds the full handler chain. Dependencies arrive as explicit
// arguments (constructor injection, no globals) so handlers stay testable with
// fakes; main decides which Runner (in-process or HTTP) to pass.
func NewRouter(logger *slog.Logger, pool *platform.DB, runner Runner) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /readyz", handleReady(logger, pool))

	runSvc := runService{
		runner: runner,
		store:  runstore.NewStore(pool.Pool()),
		logger: logger,
	}
	mux.HandleFunc("POST /api/run", runSvc.handleRun)

	learn := learningHandlers{store: learningstore.NewStore(pool.Pool()), logger: logger}
	mux.HandleFunc("GET /api/progress", learn.progress)
	mux.HandleFunc("POST /api/progress/{lessonId}/viewed", learn.markViewed)
	mux.HandleFunc("POST /api/progress/{lessonId}/ran", learn.markRan)
	mux.HandleFunc("POST /api/progress/{lessonId}/completed", learn.markCompleted)
	mux.HandleFunc("GET /api/notes/{lessonId}", learn.getNote)
	mux.HandleFunc("PUT /api/notes/{lessonId}", learn.putNote)
	mux.HandleFunc("GET /api/bookmarks", learn.listBookmarks)
	mux.HandleFunc("POST /api/bookmarks/{lessonId}", learn.addBookmark)
	mux.HandleFunc("DELETE /api/bookmarks/{lessonId}", learn.removeBookmark)
	mux.HandleFunc("GET /api/quiz/{lessonId}", learn.getQuiz)
	mux.HandleFunc("PUT /api/quiz/{lessonId}", learn.putQuiz)
	mux.HandleFunc("GET /api/history", learn.history)

	// Outermost first: logging, so rejected and panicking requests still show
	// up in the log; recovery just inside it, so the log sees the 500; then the
	// guards; then the mux, whose own 404 and 405 become problem JSON.
	return withLogging(logger)(withRecovery(logger)(withRequestGuards(logger, withProblemFallbacks(mux))))
}

// handleHealth is liveness: the process is up, no dependencies checked.
// Kubernetes/Compose probes restart on failure here; readiness is separate.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleReady is readiness: the process can serve traffic, i.e. the database
// accepts queries. Failing this removes the instance from load without a restart.
func handleReady(logger *slog.Logger, db *platform.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			logger.Error("readiness check failed", slog.Any("err", err))
			problem(w, http.StatusServiceUnavailable, "database unreachable")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

// problem writes an RFC 9457 problem-details response. All error responses go
// through this one function, the mux's own 404/405 and recovered panics
// included, so the API's error shape stays uniform.
func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
