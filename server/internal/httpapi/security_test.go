package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The guards stand between any website the owner visits and code execution,
// so each attack gets its own case, next to the legitimate callers that must
// keep working: the browser through the Vite proxy, verify-lessons (a Node
// client without browser headers) and Compose health probes.
func TestRequestGuards(t *testing.T) {
	const runBody = `{"runtime":"go","files":[]}`
	tests := []struct {
		name        string
		method      string
		path        string
		host        string
		headers     map[string]string
		body        string
		wantStatus  int
		wantReached bool
	}{
		{
			name: "cross-site text/plain POST, the CSRF attack", method: "POST", path: "/api/run",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"Content-Type": "text/plain", "Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"},
			body:    runBody, wantStatus: http.StatusForbidden,
		},
		{
			name: "cross-site POST from a browser that sends Origin only", method: "POST", path: "/api/run",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"Content-Type": "text/plain", "Origin": "https://evil.example"},
			body:    runBody, wantStatus: http.StatusForbidden,
		},
		{
			name: "same-site page on another port", method: "POST", path: "/api/progress/x/completed",
			host:       "127.0.0.1:8080",
			headers:    map[string]string{"Origin": "http://127.0.0.1:3000", "Sec-Fetch-Site": "same-site"},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "text/plain body without browser headers", method: "POST", path: "/api/run",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"Content-Type": "text/plain"},
			body:    runBody, wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name: "DNS rebinding: same-origin to the browser, foreign Host", method: "POST", path: "/api/run",
			host:    "rebind.evil.example:8080",
			headers: map[string]string{"Content-Type": "application/json", "Origin": "http://rebind.evil.example:8080", "Sec-Fetch-Site": "same-origin"},
			body:    runBody, wantStatus: http.StatusForbidden,
		},
		{
			name: "DNS rebinding read of saved notes", method: "GET", path: "/api/notes/go.basics.pointers",
			host: "rebind.evil.example:8080", wantStatus: http.StatusForbidden,
		},
		{
			name: "the app through the Vite proxy", method: "POST", path: "/api/run",
			host:    "127.0.0.1:5173",
			headers: map[string]string{"Content-Type": "application/json", "Origin": "http://127.0.0.1:5173", "Sec-Fetch-Site": "same-origin"},
			body:    runBody, wantStatus: http.StatusOK, wantReached: true,
		},
		{
			name: "the app on localhost with a charset parameter", method: "PUT", path: "/api/notes/x",
			host:    "localhost:5173",
			headers: map[string]string{"Content-Type": "application/json; charset=utf-8", "Origin": "http://localhost:5173", "Sec-Fetch-Site": "same-origin"},
			body:    `{"body":"hi"}`, wantStatus: http.StatusOK, wantReached: true,
		},
		{
			name: "bodyless POST from the app", method: "POST", path: "/api/bookmarks/x",
			host:       "127.0.0.1:5173",
			headers:    map[string]string{"Origin": "http://127.0.0.1:5173", "Sec-Fetch-Site": "same-origin"},
			wantStatus: http.StatusOK, wantReached: true,
		},
		{
			name: "verify-lessons: Node fetch, no browser headers", method: "POST", path: "/api/run",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"Content-Type": "application/json"},
			body:    runBody, wantStatus: http.StatusOK, wantReached: true,
		},
		{
			name: "Compose health probe by container name", method: "GET", path: "/healthz",
			host: "backend:8080", wantStatus: http.StatusOK, wantReached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			h := withRequestGuards(slog.New(slog.DiscardHandler), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
			}))

			req := httptest.NewRequest(tt.method, "http://placeholder"+tt.path, strings.NewReader(tt.body))
			req.Host = tt.host
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus || reached != tt.wantReached {
				t.Fatalf("status = %d, handler reached = %v; want %d, %v (body %q)",
					rec.Code, reached, tt.wantStatus, tt.wantReached, rec.Body.String())
			}
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost":              true,
		"LOCALHOST:5173":         true,
		"localhost.:8080":        true,
		"127.0.0.1:8080":         true,
		"127.5.5.5":              true,
		"[::1]:8080":             true,
		"::1":                    true,
		"":                       false,
		"evil.example":           false,
		"localhost.evil.example": false,
		"127.0.0.1.nip.io:8080":  false,
		"192.168.1.10:8080":      false,
		"0.0.0.0:8080":           false,
	} {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}
