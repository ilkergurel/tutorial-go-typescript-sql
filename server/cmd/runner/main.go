// cmd/runner is the sandboxed code runner service (Docker mode only).
//
// Security boundary per ADR-0003: this container runs lesson code with no
// internet, a non-root user, read-only rootfs except workdirs, and CPU/mem/
// pids limits enforced by Compose. The backend is its only network peer.
//
// POST /run executes one request through runexec, the same code the backend
// runs in-process in local mode (ADR-0008); this file is only the HTTP shell.
package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"stacklab/server/internal/runapi"
	"stacklab/server/internal/runexec"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	executor := runexec.New(os.Getenv("STACKLAB_RUNNER_DATABASE_URL"), "") // no node_modules in the image: ts-test is local-only

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("POST /run", newRunHandler(logger, executor))

	// Same timeouts as the backend's http.Server: a stuck client can't hold a
	// worker forever.
	srv := &http.Server{
		Addr:              ":8090",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       45 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           mux,
	}
	logger.Info("runner listening", slog.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("runner exited", slog.Any("err", err))
		os.Exit(1)
	}
}

func newRunHandler(logger *slog.Logger, executor *runexec.Executor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req runapi.Request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
			problem(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := req.Validate(); err != nil {
			problem(w, http.StatusBadRequest, err.Error())
			return
		}

		res, refused, err := executor.Run(r.Context(), req)
		switch {
		case err != nil:
			logger.Error("run failed", slog.Any("err", err), slog.String("runtime", string(req.Runtime)))
			problem(w, http.StatusInternalServerError, "run failed")
		case refused != nil:
			problem(w, refused.Status, refused.Detail)
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}
	})
}

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(runapi.Error{Status: status, Title: "runner error", Detail: detail})
}
