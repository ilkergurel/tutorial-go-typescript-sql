package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// route reports which pattern matched (Request.Pattern, Go 1.23) and the
// wildcard values, so the output shows every decision ServeMux made.
func route(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, r.Pattern)
	if id := r.PathValue("id"); id != "" {
		fmt.Fprintf(w, " id=%s", id)
	}
	if rest := r.PathValue("rest"); rest != "" {
		fmt.Fprintf(w, " rest=%s", rest)
	}
}

func main() {
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"GET /posts/{id}",      // exactly one segment after /posts/
		"GET /posts/latest",    // a literal beats a wildcard
		"GET /files/{rest...}", // the remainder, slashes included
		"GET /docs/",           // trailing slash: the whole subtree
		"GET /{$}",             // only "/" itself
		"/",                    // everything else, any method
	} {
		mux.HandleFunc(pattern, route)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{
		"/posts/42", "/posts/latest", "/posts/42/comments",
		"/files/2026/rapor.pdf", "/docs/http/intro", "/docs", "/", "/nope",
	} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%-22s -> %s", path, body)
		if final := resp.Request.URL.Path; final != path {
			fmt.Printf(" (redirected to %s)", final)
		}
		fmt.Println()
	}
}
