package main

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
)

// LEARN(inside.codebase.http-layer): internal/httpapi/security.go in small, without its logging.

// withRequestGuards wraps every route in three independent checks (ADR-0011).
func withRequestGuards(h http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
	}))
	return cop.Handler(requireLoopbackHost(requireJSONBody(h)))
}

func requireLoopbackHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !isLoopbackHost(r.Host) {
			http.Error(w, "this server only answers requests addressed to localhost", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopbackHost, simplified: the real one also accepts "[::1]" and "localhost.".
func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport // no port
	}
	addr, err := netip.ParseAddr(host)
	return strings.EqualFold(host, "localhost") || (err == nil && addr.IsLoopback())
}

func requireJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		safe := r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions
		if !safe && r.ContentLength != 0 {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				http.Error(w, "request body must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/run", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ran") })
	h := withRequestGuards(mux) // the real router wraps this in withLogging and withRecovery

	post := func(who, host, contentType, fetchSite string) {
		req := httptest.NewRequest("POST", "/api/run", strings.NewReader(`{"runtime":"go"}`))
		req.Host = host
		req.Header.Set("Content-Type", contentType)
		if fetchSite != "" {
			req.Header.Set("Sec-Fetch-Site", fetchSite)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		fmt.Printf("%-24s %d %s\n", who, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	post("the app, via Vite", "localhost:5173", "application/json", "same-origin")
	post("another site's form", "127.0.0.1:8080", "text/plain", "cross-site")
	post("a client without headers", "127.0.0.1:8080", "text/plain", "")
	post("DNS rebinding", "evil.example:8080", "application/json", "same-origin")
}
