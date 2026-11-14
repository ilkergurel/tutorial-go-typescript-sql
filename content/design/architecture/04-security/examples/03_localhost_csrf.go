// LEARN(design.architecture.security): a JSON API on 127.0.0.1 is reachable from any page in the browser.
package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
)

// run has the shape of a code-runner endpoint: decode JSON, act on it.
// No cookies, no auth: being on 127.0.0.1 is its only credential.
func run(w http.ResponseWriter, r *http.Request) {
	var req struct{ Runtime, Code string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "ran %d bytes of %s", len(req.Code), req.Runtime)
}

// requireJSON refuses bodies a cross-site page could send without a CORS preflight.
func requireJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next(w, r)
	}
}

func main() {
	servers := []struct {
		name string
		h    http.Handler
	}{
		{"no defense", http.HandlerFunc(run)},
		{"requireJSON", requireJSON(run)},
		{"CrossOriginProtection", http.NewCrossOriginProtection().Handler(http.HandlerFunc(run))},
	}
	// fetch(url, {method: "POST", mode: "no-cors", body: JSON.stringify(...)}) from
	// another site sends exactly this: a "simple" request, no preflight.
	senders := []struct {
		name  string
		ctype string
		site  string
	}{
		{"evil.example page", "text/plain;charset=UTF-8", "cross-site"},
		{"the app's own page", "application/json", "same-origin"},
		{"curl", "application/json", ""},
	}
	for _, s := range servers {
		srv := httptest.NewServer(s.h)
		for _, from := range senders {
			body := strings.NewReader(`{"Runtime":"go","Code":"package main"}`)
			req, _ := http.NewRequest("POST", srv.URL+"/api/run", body)
			req.Header.Set("Content-Type", from.ctype)
			if from.site != "" { // browsers add Sec-Fetch-Site; page scripts can't forge it
				req.Header.Set("Sec-Fetch-Site", from.site)
			}
			if from.site == "cross-site" {
				req.Header.Set("Origin", "https://evil.example")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				continue
			}
			resp.Body.Close()
			fmt.Printf("%-22s %-19s -> %d\n", s.name, from.name, resp.StatusCode)
		}
		srv.Close()
	}
}
