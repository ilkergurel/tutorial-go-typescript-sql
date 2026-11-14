// LEARN(fullstack.tracelite.step-10-external): every call to another service has a deadline, a reason to retry or not, and a cached answer to fall back on.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	ErrUnknownClause = errors.New("no such clause")
	errTransient     = errors.New("may go away") // a network error, a timeout, 429 or 5xx
)

// try is one attempt. For 429 and 5xx it also returns the server's Retry-After.
func try(ctx context.Context, client *http.Client, clauseURL string) (string, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond) // 2 s in TraceLite
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", clauseURL, nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", errTransient, err)
	}
	defer resp.Body.Close()
	switch code := resp.StatusCode; {
	case code == http.StatusOK:
		title, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return string(title), 0, err
	case code == http.StatusNotFound:
		return "", 0, ErrUnknownClause // an answer, which example 3 caches
	case code == http.StatusTooManyRequests || code >= 500:
		secs, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		return "", time.Duration(secs) * time.Second, fmt.Errorf("%w: status %d", errTransient, code)
	default:
		return "", 0, fmt.Errorf("status %d", code) // TraceLite's mistake: no retry fixes it
	}
}

func main() {
	answers := []string{"200", "slow", "503", "429", "404", "400"}
	var calls atomic.Int64
	catalogue := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := answers[calls.Add(1)-1]
		if answer == "slow" {
			<-r.Context().Done() // answers only after the client has given up
			return
		}
		if answer == "429" {
			w.Header().Set("Retry-After", "120")
		}
		code, _ := strconv.Atoi(answer)
		w.WriteHeader(code)
		fmt.Fprint(w, "Authentication information")
	}))
	defer catalogue.Close()
	client := &http.Client{Timeout: 10 * time.Second}                       // a backstop; http.DefaultClient has none
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // the API request's budget
	defer cancel()
	for _, answer := range answers {
		title, retryAfter, err := try(ctx, client, catalogue.URL+"/clauses/ISO27001:A.5.17")
		switch {
		case err == nil:
			fmt.Printf("%-4s -> %q\n", answer, title)
		case errors.Is(err, errTransient):
			fmt.Printf("%-4s -> retry, Retry-After %v\n", answer, retryAfter)
		default:
			fmt.Printf("%-4s -> final: %v\n", answer, err)
		}
	}
}
