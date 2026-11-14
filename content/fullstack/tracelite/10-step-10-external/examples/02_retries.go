package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

var errTransient = errors.New("may go away") // example 1's: a network error, a timeout, 429 or 5xx

// fetchTitle repeats example 1's try until it succeeds, fails for good, or the budget runs out.
func fetchTitle(ctx context.Context, try func() (string, time.Duration, error)) (string, error) {
	for attempt := 1; ; attempt++ {
		title, retryAfter, err := try()
		if !errors.Is(err, errTransient) {
			return title, err // done, a 404, or a mistake that retrying won't fix
		}
		if attempt == 3 {
			return "", fmt.Errorf("gave up after %d attempts: %w", attempt, err)
		}
		wait := max(retryAfter, rand.N(100*time.Millisecond<<attempt)) // full jitter, or the server's wish
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < wait {
			return "", fmt.Errorf("asked to wait %v, past our deadline: %w", wait, err)
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// catalogue plays example 1's try: each call returns the next scripted answer.
func catalogue(answers ...string) func() (string, time.Duration, error) {
	return func() (string, time.Duration, error) {
		answer := answers[0]
		answers = answers[1:]
		fmt.Println("  attempt:", answer)
		switch answer {
		case "200":
			return "Authentication information", 0, nil
		case "429":
			return "", 120 * time.Second, fmt.Errorf("%w: 429", errTransient)
		default: // 5xx, or no answer within the attempt's timeout
			return "", 0, fmt.Errorf("%w: %s", errTransient, answer)
		}
	}
}

func main() {
	for _, answers := range [][]string{{"503", "timeout", "200"}, {"429"}, {"502", "502", "502"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // the API request's budget
		title, err := fetchTitle(ctx, catalogue(answers...))
		cancel()
		fmt.Printf("lookup: %q, err: %v\n", title, err)
	}
}
