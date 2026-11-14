// cmd/mocksvc provides deterministic fake external services for lessons.
// Phase 1 ships the health endpoint only; the full endpoint set (weather, slow,
// flaky, ratelimited, echo, suppliers, sse, ws, webhook, standards) lands in
// Phase 6. Determinism matters: lesson checks compare exact outputs, so "flaky"
// must fail predictably per seed.
package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	_ = http.ListenAndServe("127.0.0.1:8091", mux)
}
