package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// query stands in for any I/O call. It returns as soon as the caller stops
// caring, with ctx.Err() wrapped so errors.Is still works upstream.
func query(ctx context.Context, takes time.Duration) (string, error) {
	select {
	case <-time.After(takes):
		return "42 rows", nil
	case <-ctx.Done():
		return "", fmt.Errorf("query: %w", ctx.Err())
	}
}

func main() {
	// 1. A deadline. The query would take an hour; the budget is 20ms.
	ctx, cancel := context.WithTimeoutCause(context.Background(),
		20*time.Millisecond, errors.New("query budget of 20ms used up"))
	defer cancel() // always: it releases the timer if the work ends first
	_, err := query(ctx, time.Hour)
	fmt.Println(err)
	fmt.Println("is DeadlineExceeded:", errors.Is(err, context.DeadlineExceeded))
	fmt.Println("cause:", context.Cause(ctx))

	// 2. Explicit cancellation, e.g. the client went away mid-request.
	ctx2, cancel2 := context.WithCancel(context.Background())
	go cancel2() // any goroutine may cancel; every call is safe
	_, err = query(ctx2, time.Hour)
	fmt.Println(err)
	fmt.Println("is Canceled:", errors.Is(err, context.Canceled))

	// 3. Work that finishes in time never sees the context fire.
	ctx3, cancel3 := context.WithTimeout(context.Background(), time.Second)
	defer cancel3()
	fmt.Println(query(ctx3, 0))
}
