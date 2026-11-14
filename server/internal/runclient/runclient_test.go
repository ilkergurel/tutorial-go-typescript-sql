package runclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stacklab/server/internal/runapi"
)

// fakeRunner answers POST /run with a fixed status, content type and body,
// the way cmd/runner (or something standing in front of it) would.
func fakeRunner(t *testing.T, status int, contentType, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/run" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("runner got %s %s (Content-Type %q), want POST /run as JSON",
				r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		var req runapi.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Runtime != runapi.RuntimeGo {
			t.Errorf("runner got request %+v, err %v", req, err)
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func run(c *Client) (*runapi.Result, *runapi.Error, error) {
	return c.Run(context.Background(), runapi.Request{
		Runtime: runapi.RuntimeGo,
		Files:   map[string]string{"main.go": "package main"},
	})
}

// problemJSON is what cmd/runner's problem() writes.
func problemJSON(status int, detail string) string {
	b, _ := json.Marshal(runapi.Error{Status: status, Title: "runner error", Detail: detail})
	return string(b)
}

func TestRunPassesResultThrough(t *testing.T) {
	c := fakeRunner(t, http.StatusOK, "application/json", `{"stdout":"hi\n","exitCode":3,"diagnostics":[],"files":[]}`)
	res, refusal, err := run(c)
	if err != nil || refusal != nil {
		t.Fatalf("err = %v, refusal = %+v; want a result", err, refusal)
	}
	if res.Stdout != "hi\n" || res.ExitCode != 3 {
		t.Fatalf("result = %+v", res)
	}
}

// The runner's refusals reach the handler with their status and detail; the
// detail is what the browser shows, so "server busy" must not turn into a
// bare "Service Unavailable".
func TestRunKeepsTheRunnersRefusalDetail(t *testing.T) {
	const tsTestOnly = "ts-test lessons run only in local mode (npm start), where the app's node_modules are available"
	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		wantDetail  string
	}{
		{"busy", http.StatusServiceUnavailable, "application/problem+json", problemJSON(503, "server busy"), "server busy"},
		{"runtime not available", http.StatusNotImplemented, "application/problem+json", problemJSON(501, tsTestOnly), tsTestOnly},
		{"request the runner rejects", http.StatusBadRequest, "application/problem+json", problemJSON(400, "too many files"), "too many files"},
		{"plain-text reply", http.StatusNotFound, "text/plain; charset=utf-8", "404 page not found\n", "404 page not found"},
		{"empty reply", http.StatusBadGateway, "", "", "Bad Gateway"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, refusal, err := run(fakeRunner(t, tt.status, tt.contentType, tt.body))
			if err != nil || res != nil {
				t.Fatalf("res = %+v, err = %v; want a refusal", res, err)
			}
			want := runapi.Error{Status: tt.status, Title: "runner error", Detail: tt.wantDetail}
			if refusal == nil || *refusal != want {
				t.Fatalf("refusal = %+v, want %+v", refusal, want)
			}
		})
	}
}

// A 500 is cmd/runner's "run failed": the machinery broke, which local mode
// reports as an error (and the handler as 502), so Docker mode must too.
func TestRunTurnsRunnerFailureIntoError(t *testing.T) {
	res, refusal, err := run(fakeRunner(t, http.StatusInternalServerError, "application/problem+json", problemJSON(500, "run failed")))
	if err == nil || res != nil || refusal != nil {
		t.Fatalf("res = %+v, refusal = %+v, err = %v; want an error", res, refusal, err)
	}
	if !strings.Contains(err.Error(), "run failed") {
		t.Fatalf("err = %v, want the runner's detail in it", err)
	}
}

func TestRunReportsBrokenAndMissingRunner(t *testing.T) {
	t.Run("200 with a body that isn't a result", func(t *testing.T) {
		if _, refusal, err := run(fakeRunner(t, http.StatusOK, "application/json", "<html>")); err == nil || refusal != nil {
			t.Fatalf("refusal = %+v, err = %v; want a decode error", refusal, err)
		}
	})
	t.Run("nothing listening", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()
		if _, refusal, err := run(New(srv.URL)); err == nil || refusal != nil {
			t.Fatalf("refusal = %+v, err = %v; want runner unreachable", refusal, err)
		}
	})
}
