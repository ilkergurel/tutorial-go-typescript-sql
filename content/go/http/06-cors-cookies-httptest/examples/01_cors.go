package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
)

// cors labels responses for allowed origins and answers preflights. It never
// blocks anything: the browser enforces CORS, by hiding responses from pages.
func cors(allowed map[string]bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin") // the answer depends on Origin: tell caches
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if allowed[origin] { // a preflight: may this origin send that request?
				w.Header().Set("Access-Control-Allow-Methods", "GET, PUT")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	var ran atomic.Int64 // how often the real handler executed
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		ran.Add(1)
		io.WriteString(w, "saved")
	})
	allowed := map[string]bool{"http://127.0.0.1:5173": true}
	srv := httptest.NewServer(cors(allowed, mux))
	defer srv.Close()

	for _, c := range []struct{ method, origin, note string }{
		{"OPTIONS", "http://127.0.0.1:5173", "preflight from the app"},
		{"PUT", "http://127.0.0.1:5173", "the real request"},
		{"OPTIONS", "https://evil.example", "preflight from another site"},
		{"PUT", "https://evil.example", "sent anyway, e.g. by curl"},
		{"PUT", "null", "from a sandboxed iframe"},
	} {
		req, _ := http.NewRequest(c.method, srv.URL+"/notes/1", nil)
		req.Header.Set("Origin", c.origin)
		if c.method == "OPTIONS" {
			req.Header.Set("Access-Control-Request-Method", "PUT")
			req.Header.Set("Access-Control-Request-Headers", "content-type")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		resp.Body.Close()
		h := resp.Header
		fmt.Printf("%-7s %-21s %d allow-origin=%q methods=%q  (%s)\n", c.method, c.origin,
			resp.StatusCode, h.Get("Access-Control-Allow-Origin"), h.Get("Access-Control-Allow-Methods"), c.note)
	}
	fmt.Println("the PUT handler ran", ran.Load(), "times")
}
