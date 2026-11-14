package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
)

// LEARN(inside.codebase.wiring): a miniature of server/cmd/stacklab/main.go.

type Config struct{ RunnerURL string } // internal/config: plain fields, defaults

func Load(getenv func(string) string) Config {
	return Config{RunnerURL: getenv("STACKLAB_RUNNER_URL")}
}

// Runner is declared where it is used (internal/httpapi/run.go).
type Runner interface {
	Run(ctx context.Context, code string) (string, error)
}

type Executor struct{ slots chan struct{} } // internal/runexec

func NewExecutor() *Executor { return &Executor{slots: make(chan struct{}, 2)} }
func (e *Executor) Run(_ context.Context, code string) (string, error) {
	return "ran " + code + " in this process", nil
}

type Client struct{ url string } // internal/runclient

func NewClient(url string) *Client { return &Client{url: url} }
func (c *Client) Run(_ context.Context, code string) (string, error) {
	return "POST " + c.url + "/run with " + code, nil
}

func NewRouter(logger *slog.Logger, runner Runner) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/run", func(w http.ResponseWriter, r *http.Request) {
		code, _ := io.ReadAll(r.Body)
		out, err := runner.Run(r.Context(), string(code))
		if err != nil {
			logger.Error("run failed", "err", err)
			http.Error(w, "run failed", http.StatusBadGateway)
			return
		}
		io.WriteString(w, out)
	})
	return mux
}

func wire(getenv func(string) string) {
	cfg := Load(getenv)
	logger := slog.New(slog.DiscardHandler)
	var runner Runner = NewExecutor()
	if cfg.RunnerURL != "" {
		runner = NewClient(cfg.RunnerURL)
	}
	router := NewRouter(logger, runner)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/api/run", strings.NewReader("main.go")))
	fmt.Printf("%-15T %s\n", runner, rec.Body)
}

func main() {
	local := map[string]string{} // npm start: no environment needed
	docker := map[string]string{"STACKLAB_RUNNER_URL": "http://runner:8090"}
	for _, env := range []map[string]string{local, docker} {
		wire(func(key string) string { return env[key] })
	}
}
