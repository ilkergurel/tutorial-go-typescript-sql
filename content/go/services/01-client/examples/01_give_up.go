package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
)

// upstream is the service we call: /ok answers, /missing is a 404,
// /hang never answers, /stall sends half a body and then goes quiet.
func upstream(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ok":
		fmt.Fprint(w, "pong")
		return
	case "/missing":
		http.NotFound(w, r)
		return
	case "/stall":
		fmt.Fprint(w, "half a bo")
		w.(http.Flusher).Flush()
	}
	<-r.Context().Done() // the server cancels it when the client hangs up
}

// get makes one call, bounded by ctx, and closes what it opens.
func get(ctx context.Context, c *http.Client, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err // no response, nothing to close
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(body), nil
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(upstream))
	defer srv.Close()
	fmt.Println("http.DefaultClient.Timeout:", http.DefaultClient.Timeout)

	client := &http.Client{Timeout: time.Second} // backstop: connect + headers + body
	show := func(ctx context.Context, path string) {
		body, err := get(ctx, client, srv.URL+path)
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err // the full text names the URL, and its port is random
		}
		fmt.Printf("%-8s body=%q timeout=%t err=%v\n",
			path, body, errors.Is(err, context.DeadlineExceeded), err)
	}
	show(context.Background(), "/ok")
	show(context.Background(), "/missing")
	show(context.Background(), "/hang")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	show(ctx, "/stall") // this call's own budget, shorter than the backstop
}
