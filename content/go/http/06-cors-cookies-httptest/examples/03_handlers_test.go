package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The code under test: one JSON route.
func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "1" {
			http.Error(w, "no such note", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":1,"text":"Merhaba"}`)
	})
	return mux
}

// No server, no client, no network: ServeHTTP is called directly.
func TestRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, method, target string
		wantStatus           int
		wantType             string
	}{
		{"found", "GET", "/notes/1", http.StatusOK, "application/json"},
		{"missing", "GET", "/notes/2", http.StatusNotFound, "text/plain; charset=utf-8"},
		{"wrong method", "DELETE", "/notes/1", http.StatusMethodNotAllowed, "text/plain; charset=utf-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newRouter().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.target, nil))
			res := rec.Result() // the response as sent, not the live header map
			if res.StatusCode != tc.wantStatus || res.Header.Get("Content-Type") != tc.wantType {
				t.Errorf("%s %s = %d %q, want %d %q", tc.method, tc.target,
					res.StatusCode, res.Header.Get("Content-Type"), tc.wantStatus, tc.wantType)
			}
		})
	}
}

// A real client and server, still without a network: NewTestServer
// (Go 1.27) uses an in-memory network and shuts down when the test ends.
func TestEndToEnd(t *testing.T) {
	srv := httptest.NewTestServer(t, newRouter())
	resp, err := srv.Client().Get("http://notes.example/notes/1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	t.Logf("GET http://notes.example/notes/1 -> %d %s", resp.StatusCode, body)
}
