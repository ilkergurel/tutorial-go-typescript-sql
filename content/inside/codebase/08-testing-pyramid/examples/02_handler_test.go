// LEARN(inside.codebase.testing-pyramid): the missing layer: a handler test with a fake Runner, the seam httpapi.Runner was declared for.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type Request struct{ Runtime string }
type Result struct{ Stdout string }
type Refusal struct {
	Status int
	Detail string
}
type Runner interface {
	Run(ctx context.Context, req Request) (*Result, *Refusal, error)
}

func handleRun(runner Runner, record func(Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Runtime == "" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		res, refusal, err := runner.Run(r.Context(), req)
		switch {
		case err != nil:
			http.Error(w, "runner or playground database unavailable", http.StatusBadGateway)
		case refusal != nil:
			http.Error(w, refusal.Detail, refusal.Status)
		default:
			if r.Header.Get("X-Stacklab-No-History") == "" {
				record(req)
			}
			_ = json.NewEncoder(w).Encode(res)
		}
	}
}

type fakeRunner struct { // returns what it holds: no Go toolchain, no database
	res     *Result
	refusal *Refusal
	err     error
}

func (f fakeRunner) Run(context.Context, Request) (*Result, *Refusal, error) {
	return f.res, f.refusal, f.err
}

func TestHandleRun(t *testing.T) {
	tests := []struct {
		name, body, noHistory string
		runner                fakeRunner
		code, recorded        int
	}{
		{"ok, recorded", `{"Runtime":"go"}`, "", fakeRunner{res: &Result{}}, 200, 1},
		{"ok, verify run", `{"Runtime":"go"}`, "1", fakeRunner{res: &Result{}}, 200, 0},
		{"bad body", `{`, "", fakeRunner{}, 400, 0},
		{"both slots busy", `{"Runtime":"go"}`, "", fakeRunner{refusal: &Refusal{503, "server busy"}}, 503, 0},
		{"playground down", `{"Runtime":"sql"}`, "", fakeRunner{err: errors.New("dial tcp: refused")}, 502, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorded := 0
			req := httptest.NewRequest("POST", "/api/run", strings.NewReader(tc.body))
			req.Header.Set("X-Stacklab-No-History", tc.noHistory)
			rec := httptest.NewRecorder()
			handleRun(tc.runner, func(Request) { recorded++ })(rec, req)
			if rec.Code != tc.code || recorded != tc.recorded {
				t.Errorf("got %d, %d recorded; want %d, %d (body %q)", rec.Code, recorded, tc.code, tc.recorded, rec.Body)
			}
		})
	}
}
