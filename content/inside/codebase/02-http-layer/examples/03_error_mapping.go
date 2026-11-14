package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

// Request, Result and RunError are cut-down internal/runapi types.
type Request struct{ Files map[string]string }
type Result struct {
	ExitCode int `json:"exitCode"`
}
type RunError struct {
	Status int
	Detail string
}
type Runner interface {
	Run(ctx context.Context, req Request) (*Result, *RunError, error)
}

type fakeRunner struct{} // its answer depends on main.go's content

func (fakeRunner) Run(_ context.Context, req Request) (*Result, *RunError, error) {
	switch req.Files["main.go"] {
	case "busy":
		return nil, &RunError{http.StatusServiceUnavailable, "server busy"}, nil
	case "db down":
		return nil, nil, errors.New("sqlrun: connect: connection refused")
	}
	return &Result{ExitCode: 1}, nil, nil // a compile error is a result
}

// handleRun is run.go's handler in small: no write deadline, Validate or history.
func handleRun(runner Runner, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest) // real: problem()
			return
		}
		res, refused, err := runner.Run(r.Context(), req)
		switch {
		case err != nil:
			logger.Error("run failed", "err", err)
			http.Error(w, "run failed: runner or playground database unavailable", http.StatusBadGateway)
		case refused != nil:
			http.Error(w, refused.Detail, refused.Status)
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
		}
	}
}

func main() {
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	h := handleRun(fakeRunner{}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: noTime})))
	for _, body := range []string{
		`{"runtime": "go", "files": {"main.go": "does not compile"}}`,
		`{"runtime": "go", "files": `,
		`{"runtime": "go", "files": {"main.go": "busy"}}`,
		`{"runtime": "sql", "files": {"main.go": "db down"}}`,
	} {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest("POST", "/api/run", strings.NewReader(body)))
		fmt.Println(rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}
