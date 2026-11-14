package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// register reports a registration panic as an error, for display only: in a
// real program the panic at startup is exactly what you want.
func register(mux *http.ServeMux, pattern string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// A conflict message names source files; keep its explanation.
			msg := fmt.Sprint(r)
			if _, why, ok := strings.Cut(msg, ":\n"); ok {
				msg = why
			}
			err = fmt.Errorf("%s", msg)
		}
	}()
	mux.HandleFunc(pattern, func(http.ResponseWriter, *http.Request) {})
	return nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "item "+r.PathValue("id"))
	})
	mux.HandleFunc("DELETE /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, method := range []string{"GET", "HEAD", "DELETE", "PUT", "get"} {
		req, _ := http.NewRequest(method, srv.URL+"/items/1", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%-6s %d %q", method, resp.StatusCode, body)
		if allow := resp.Header.Get("Allow"); allow != "" {
			fmt.Printf(" Allow: %s", allow)
		}
		fmt.Println()
	}

	fmt.Println("--- registering on a fresh mux")
	fresh := http.NewServeMux()
	for _, p := range []string{"GET /", "/index.html", "GET /u/{id}", "GET /u/{name}", "/u_{id}"} {
		fmt.Printf("%-14s %v\n", p, register(fresh, p))
	}
}
