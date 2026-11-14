// LEARN(design.architecture.styles): REST conditional requests: ETag and If-Match turn a lost update into 412.
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// note is one editable resource; its version goes out as a strong ETag.
type note struct {
	mu      sync.Mutex
	body    string
	version int
}

func (n *note) get(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("ETag", fmt.Sprintf(`"%d"`, n.version))
	io.WriteString(w, n.body)
}

// put replaces the note only if the client's copy is still current.
func (n *note) put(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock() // check and write as one step; example 3 does it in one UPDATE
	defer n.mu.Unlock()
	switch r.Header.Get("If-Match") {
	case "":
		http.Error(w, "send If-Match with the ETag you read", http.StatusPreconditionRequired)
	case fmt.Sprintf(`"%d"`, n.version):
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
		n.body, n.version = string(body), n.version+1
		w.Header().Set("ETag", fmt.Sprintf(`"%d"`, n.version))
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "the note changed since you read it", http.StatusPreconditionFailed)
	}
}

func main() {
	n := &note{body: "defer runs LIFO", version: 1}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /notes/go.basics.defer", n.get)
	mux.HandleFunc("PUT /notes/go.basics.defer", n.put)

	call := func(tab, method, etag, body string) string {
		req := httptest.NewRequest(method, "/notes/go.basics.defer", strings.NewReader(body))
		if etag != "" {
			req.Header.Set("If-Match", etag)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		fmt.Printf("%s %-3s If-Match=%-4s -> %d %s ETag=%s\n", tab, method, etag,
			rec.Code, http.StatusText(rec.Code), rec.Header().Get("ETag"))
		return rec.Header().Get("ETag")
	}
	a := call("tab A", "GET", "", "")
	b := call("tab B", "GET", "", "")
	call("tab A", "PUT", a, "defer runs LIFO, at function exit")
	call("tab B", "PUT", b, "defer evaluates arguments early") // B's copy is stale
	call("tab B", "PUT", "", "just overwrite it")
	b = call("tab B", "GET", "", "")
	call("tab B", "PUT", b, "defer runs LIFO, at function exit; arguments early")
}
