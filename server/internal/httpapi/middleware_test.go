package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stacklab/server/internal/platform"
)

// A handler behind the middleware must still be able to extend its own write
// deadline: the run route relies on it to outlive the server's WriteTimeout.
// Without Unwrap on statusRecorder (and on writeTracker) the reply below is
// cut off and the client sees EOF instead of "done".
func TestMiddlewareKeepsResponseControllerWorking(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	for name, wrap := range map[string]func(http.Handler) http.Handler{
		"withLogging":  withLogging(logger),
		"withRecovery": withRecovery(logger),
	} {
		t.Run(name, func(t *testing.T) {
			const serverWriteTimeout = 100 * time.Millisecond
			slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					t.Errorf("SetWriteDeadline through %s: %v", name, err)
				}
				time.Sleep(3 * serverWriteTimeout) // outlives the server-wide deadline
				_, _ = io.WriteString(w, "done")
			})

			srv := httptest.NewUnstartedServer(wrap(slow))
			srv.Config.WriteTimeout = serverWriteTimeout
			srv.Start()
			defer srv.Close()

			resp, err := http.Get(srv.URL)
			if err != nil {
				t.Fatalf("GET: %v (reply cut off by the server WriteTimeout)", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil || string(body) != "done" {
				t.Fatalf("body = %q, err = %v; want \"done\"", body, err)
			}
		})
	}
}

type problemBody struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// decodeProblem fails the test unless rec holds a problem+json reply with
// the given status.
func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder, status int) problemBody {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json (body %q)", ct, rec.Body.String())
	}
	var p problemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not one JSON object: %v (%q)", err, rec.Body.String())
	}
	if p.Status != status || p.Title != http.StatusText(status) || p.Type != "about:blank" {
		t.Fatalf("problem = %+v, want status %d", p, status)
	}
	return p
}

// A panicking handler gets a 500 problem reply instead of a dropped
// connection, and the log holds the panic value, the stack, and the access
// line with status 500 (recovery sits inside logging, as in NewRouter).
func TestRecoveryAnswersPanicWithProblem(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	panicky := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	withLogging(logger)(withRecovery(logger)(panicky)).ServeHTTP(rec, httptest.NewRequest("GET", "/api/progress", nil))

	decodeProblem(t, rec, http.StatusInternalServerError)
	for _, want := range []string{
		`msg="handler panicked"`, "panic=boom", "goroutine ", "TestRecoveryAnswersPanicWithProblem",
		`msg="http request" method=GET path=/api/progress status=500`,
	} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, logs.String())
		}
	}
}

// The cases recovery must not answer: http.ErrAbortHandler passes through
// unlogged (net/http aborts quietly on it), and a panic after the handler has
// written turns into ErrAbortHandler, so the client sees a broken reply
// rather than a half-written one that looks complete.
func TestRecoveryAbortsWhatItCannotAnswer(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantLogged bool
	}{
		{
			name:    "handler aborts with ErrAbortHandler",
			handler: func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) },
		},
		{
			name: "panic after the body started",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"stdout":`)
				panic("boom")
			},
			wantLogged: true,
		},
		{
			name: "panic after a flush",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_ = http.NewResponseController(w).Flush()
				panic("boom")
			},
			wantLogged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := withRecovery(slog.New(slog.NewTextHandler(&logs, nil)))(tt.handler)

			var got any
			func() {
				defer func() { got = recover() }()
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
			}()

			if got != http.ErrAbortHandler {
				t.Fatalf("recovered %v, want http.ErrAbortHandler", got)
			}
			if logged := strings.Contains(logs.String(), "handler panicked"); logged != tt.wantLogged {
				t.Fatalf("panic logged = %v, want %v:\n%s", logged, tt.wantLogged, logs.String())
			}
		})
	}
}

// The mux answers unmatched requests itself, with http.Error's text/plain.
// Through the real router those become problem JSON, the 405 keeps its Allow
// header, and replies the mux doesn't produce as errors pass untouched.
func TestRouterAnswersUnmatchedRequestsWithProblems(t *testing.T) {
	// No database: these requests never reach a store.
	h := NewRouter(slog.New(slog.DiscardHandler), platform.NewDB(nil), nil)
	serve := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Host = "127.0.0.1:8080" // past the loopback-Host guard
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	t.Run("404 for an unknown path", func(t *testing.T) {
		p := decodeProblem(t, serve("GET", "/api/lessons/go.basics.defer"), http.StatusNotFound)
		if p.Detail != "no route for GET /api/lessons/go.basics.defer" {
			t.Errorf("detail = %q", p.Detail)
		}
	})

	t.Run("405 keeps Allow", func(t *testing.T) {
		for path, allow := range map[string]string{
			"/api/run":     "POST",
			"/api/notes/x": "GET, HEAD, PUT",
		} {
			method := "GET"
			if path == "/api/notes/x" {
				method = "DELETE"
			}
			rec := serve(method, path)
			p := decodeProblem(t, rec, http.StatusMethodNotAllowed)
			if got := rec.Header().Get("Allow"); got != allow {
				t.Errorf("%s %s: Allow = %q, want %q", method, path, got, allow)
			}
			if !strings.HasSuffix(p.Detail, "allowed: "+allow) {
				t.Errorf("%s %s: detail = %q", method, path, p.Detail)
			}
		}
	})

	t.Run("a matched route is untouched", func(t *testing.T) {
		rec := serve("GET", "/healthz")
		if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
			t.Fatalf("GET /healthz = %d %q, want 200 \"ok\"", rec.Code, rec.Body.String())
		}
	})

	// GET matches no route here, so this reply also comes from the mux with
	// an empty pattern and goes through fallbackWriter.
	t.Run("the mux's path-cleaning redirect is untouched", func(t *testing.T) {
		rec := serve("GET", "/api//run")
		if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "/api/run" {
			t.Fatalf("GET /api//run = %d, Location %q; want 307 to /api/run",
				rec.Code, rec.Header().Get("Location"))
		}
	})
}
