package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

// ignoresContext sleeps through the deadline, like a handler that calls a
// slow API without passing r.Context() along.
func ignoresContext(report chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, err := io.WriteString(w, "report\n")
		report <- fmt.Sprintf("ignoresContext finished anyway; its write: %v", err)
	}
}

// watchesContext gives up as soon as the deadline passes.
func watchesContext(report chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(300 * time.Millisecond): // the slow work
			io.WriteString(w, "report\n")
		case <-r.Context().Done():
			report <- fmt.Sprintf("watchesContext stopped: %v", r.Context().Err())
		}
	}
}

func main() {
	report := make(chan string)
	limit := func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, 100*time.Millisecond, "report took too long\n")
	}
	mux := http.NewServeMux()
	mux.Handle("GET /quick", limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "quick answer\n")
	})))
	mux.Handle("GET /ignores", limit(ignoresContext(report)))
	mux.Handle("GET /watches", limit(watchesContext(report)))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/quick", "/ignores", "/watches"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%-8s %d %s", path, resp.StatusCode, body)
		if path != "/quick" {
			fmt.Println("         server:", <-report) // wait for the handler's own story
		}
	}
}
