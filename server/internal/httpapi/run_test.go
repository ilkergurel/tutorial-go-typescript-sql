package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stacklab/server/internal/runapi"
)

// slowRunner answers after d, like a run that first waited for a free run
// slot and then ran.
type slowRunner struct{ d time.Duration }

func (r slowRunner) Run(ctx context.Context, _ runapi.Request) (*runapi.Result, *runapi.Error, error) {
	time.Sleep(r.d)
	return &runapi.Result{Stdout: "done"}, nil, nil
}

// A run may take longer than the server's WriteTimeout and any budget guessed
// before it starts (a fixed 45s once left out the queue wait): the reply must
// still arrive, through the real middleware, and no deadline warning logged.
func TestRunReplyOutlivesWriteTimeout(t *testing.T) {
	const serverWriteTimeout = 100 * time.Millisecond
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	svc := runService{runner: slowRunner{3 * serverWriteTimeout}, logger: logger}

	srv := httptest.NewUnstartedServer(withLogging(logger)(http.HandlerFunc(svc.handleRun)))
	srv.Config.WriteTimeout = serverWriteTimeout
	srv.Start()
	defer srv.Close()

	req, err := http.NewRequest("POST", srv.URL, strings.NewReader(`{"runtime":"go","files":{"main.go":"package main"}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(noHistoryHeader, "1") // no store in this test
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v (reply cut off by the write deadline)", err)
	}
	defer resp.Body.Close()
	var res runapi.Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || resp.StatusCode != http.StatusOK || res.Stdout != "done" {
		t.Fatalf("status %d, stdout %q, err %v; want 200 \"done\"", resp.StatusCode, res.Stdout, err)
	}
	if strings.Contains(logs.String(), "cannot extend write deadline") {
		t.Fatalf("deadline warning logged:\n%s", logs.String())
	}
}
