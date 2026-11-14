// Package httpapi — run.go holds the POST /api/run handler.
//
// Flow: validate → execute via the Runner → record history (best effort) →
// respond. Where lesson code actually runs is the Runner's business: in-process
// in local mode (runexec, ADR-0008) or in the sandboxed runner container in
// Docker mode (runclient, ADR-0003).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"stacklab/server/internal/runapi"
	"stacklab/server/internal/runstore"
)

// replyWriteBudget is how long writing the reply may take once the run is
// over. While the run executes there is no write deadline: nothing is written
// then, and how long a run takes (a wait for a free run slot, then the run
// itself) is bounded by the Runner, not by this handler.
const replyWriteBudget = 10 * time.Second

// noHistoryHeader marks a run that must not appear in run history.
const noHistoryHeader = "X-Stacklab-No-History"

// Runner executes one validated run request. A *runapi.Error is a refusal to
// pass through (e.g. busy); an error means the run machinery itself failed.
// Declared here, by its consumer: runexec.Executor and runclient.Client both
// satisfy it without knowing about each other.
type Runner interface {
	Run(ctx context.Context, req runapi.Request) (*runapi.Result, *runapi.Error, error)
}

// runService bundles the run path's dependencies for the handler.
type runService struct {
	runner Runner
	store  *runstore.Store
	logger *slog.Logger
}

// handleRun validates, executes, records. Run history failures log but don't
// fail the response: a lost history row is not the learner's problem.
func (s runService) handleRun(w http.ResponseWriter, r *http.Request) {
	var req runapi.Request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		problem(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		problem(w, http.StatusBadRequest, err.Error())
		return
	}

	// The server-wide WriteTimeout (15s) counts from the request, and a run
	// can take longer: up to 10s waiting for a free run slot, then up to 40s
	// (runexec). A reply cut off there would lose a "timed out" result. So the
	// deadline is lifted for the run (under HTTP/2 an expired one resets the
	// stream even before anything is written) and restarted for the reply.
	// A deadline guessed up front once left out the wait for a slot.
	rc := http.NewResponseController(w)
	liftErr := rc.SetWriteDeadline(time.Time{})

	res, runErr, err := s.runner.Run(r.Context(), req)

	// Log a failure instead of ignoring it: a wrapper without Unwrap once made
	// this fail silently and slow runs ended in "socket hang up".
	if dErr := errors.Join(liftErr, rc.SetWriteDeadline(time.Now().Add(replyWriteBudget))); dErr != nil {
		s.logger.Warn("run: cannot extend write deadline; slow runs may be cut off", slog.Any("err", dErr))
	}

	if err != nil {
		s.logger.Error("run failed", slog.Any("err", err), slog.String("runtime", string(req.Runtime)))
		// 502: an upstream dependency failed (runner container, or the
		// playground database in local mode); the details are in the log.
		problem(w, http.StatusBadGateway, "run failed: runner or playground database unavailable")
		return
	}
	if runErr != nil {
		problem(w, runErr.Status, runErr.Detail)
		return
	}

	// scripts/verify-lessons.mjs replays every lesson example; those runs are
	// checks, not the learner's history.
	if r.Header.Get(noHistoryHeader) == "" {
		if err := s.store.Record(r.Context(), req, res); err != nil {
			s.logger.Warn("run history not recorded", slog.Any("err", err))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
