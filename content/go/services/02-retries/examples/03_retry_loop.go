package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"time"
)

// get retries 429 and 503 within ctx's deadline. Example 2 parses Retry-After fully.
func get(ctx context.Context, c *http.Client, r *rand.Rand, url string) error {
	for attempt := range 4 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusTooManyRequests &&
			resp.StatusCode != http.StatusServiceUnavailable {
			return nil
		}
		wait := r.N(min(100*time.Millisecond<<attempt, 2*time.Second)) // full jitter
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs >= 0 {
			wait = time.Duration(secs) * time.Second // the server's hint wins
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < wait {
			return fmt.Errorf("%d: server asks for %v, more than our budget", resp.StatusCode, wait)
		}
		fmt.Printf("%d, waiting %v\n", resp.StatusCode, wait.Round(time.Millisecond))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("giving up after 4 attempts")
}

func main() {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1: // no hint: we back off on our own
		case 2:
			w.Header().Set("Retry-After", "1")
		default:
			w.Header().Set("Retry-After", "3600") // come back in an hour
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	r := rand.New(rand.NewPCG(1, 2))
	fmt.Println("err:", get(ctx, &http.Client{Timeout: 5 * time.Second}, r, srv.URL))
}
