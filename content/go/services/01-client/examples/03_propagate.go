package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func main() {
	stopped := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("fast") {
			fmt.Fprint(w, "42")
			return
		}
		<-r.Context().Done() // pretend to work until nobody is waiting
		stopped <- "upstream stopped: " + r.Context().Err().Error()
	}))
	defer upstream.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Our budget for the call, nested inside whatever our own caller allows.
		ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, upstream.URL+"?"+r.URL.RawQuery, nil)
		resp, err := client.Do(req)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			http.Error(w, "upstream timed out", http.StatusGatewayTimeout)
			return
		case err != nil:
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}))
	defer gateway.Close()

	call := func(ctx context.Context, query string) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gateway.URL+"?"+query, nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("caller gave up, cancelled:", errors.Is(err, context.Canceled))
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("%d %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	call(context.Background(), "fast")
	call(context.Background(), "slow")
	fmt.Println("  " + <-stopped)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel) // the browser tab was closed
	call(ctx, "slow")
	fmt.Println("  " + <-stopped)
}
