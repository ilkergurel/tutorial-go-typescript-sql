package httpapi

// Request guards. In local mode (ADR-0008) POST /api/run executes code as the
// owner's OS user, so any request that reaches it is code execution. Binding to
// 127.0.0.1 keeps other machines out but not other websites: a page the owner
// visits can still send requests to 127.0.0.1 from the owner's own browser.
// Three independent checks, each closing a different door:
//
//   - Cross-origin protection (net/http, Go 1.25) rejects unsafe requests that
//     the browser marks as cross-site via Sec-Fetch-Site or Origin. This covers
//     the "simple" text/plain POST a hostile page can send without a CORS
//     preflight.
//   - JSON-only request bodies. A cross-site page can't send application/json
//     without a preflight, which this server never approves, so this stops the
//     same attack even in a browser that sends neither header.
//   - A loopback Host on /api/. DNS rebinding points a hostile name at
//     127.0.0.1; the browser then treats the attacker's page as same-origin and
//     the first two checks pass, but the Host header still carries the
//     attacker's name.
//
// Rejected alternative: CSRF tokens. They need a session and a token endpoint
// for a single-user app, and the browser headers already identify cross-site
// requests. Health probes stay outside the Host check because Compose probes
// them by container name.

import (
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// withRequestGuards wraps h in the three checks above, outermost first.
func withRequestGuards(logger *slog.Logger, h http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Warn("cross-origin request rejected",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("origin", r.Header.Get("Origin")),
			slog.String("sec_fetch_site", r.Header.Get("Sec-Fetch-Site")),
		)
		problem(w, http.StatusForbidden, "cross-origin request rejected")
	}))
	return cop.Handler(requireLoopbackHost(logger, requireJSONBody(h)))
}

// requireLoopbackHost refuses /api/ requests addressed to any name other than
// localhost or a loopback IP.
func requireLoopbackHost(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !isLoopbackHost(r.Host) {
			logger.Warn("request for a non-loopback host rejected",
				slog.String("host", r.Host), slog.String("path", r.URL.Path))
			problem(w, http.StatusForbidden, "this server only answers requests addressed to localhost")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopbackHost accepts "localhost", "127.0.0.1:8080", "[::1]:5173" and the
// like. The port is irrelevant: the Vite proxy forwards the browser's Host,
// which names port 5173, not 8080.
func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"), ".")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
}

// requireJSONBody refuses unsafe requests whose body isn't application/json.
// Bodyless POSTs (progress, bookmarks) carry Content-Length: 0 and pass.
func requireJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isSafeMethod(r.Method) && r.ContentLength != 0 {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				problem(w, http.StatusUnsupportedMediaType, "request body must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}
