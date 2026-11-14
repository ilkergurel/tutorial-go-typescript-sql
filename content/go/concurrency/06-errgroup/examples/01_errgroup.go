package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// fetch stands in for an HTTP call. Page 2 fails at once; every other page
// takes an hour unless the context is canceled first.
func fetch(ctx context.Context, page int) error {
	if page == 2 {
		return errors.New("upstream returned 503")
	}
	select {
	case <-time.After(time.Hour):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(2) // at most two fetches in flight; Go blocks for a free slot

	status := make([]string, 5) // one slot per task: no two goroutines share one
	for i := range status {
		page := i + 1
		g.Go(func() error {
			err := fetch(ctx, page)
			switch {
			case err == nil:
				status[i] = "ok"
			case errors.Is(err, context.Canceled):
				status[i] = "canceled"
			default:
				status[i] = "failed"
			}
			if err != nil {
				return fmt.Errorf("page %d: %w", page, err)
			}
			return nil
		})
	}

	err := g.Wait() // waits for all five, returns the first error
	for i, s := range status {
		fmt.Printf("page %d: %s\n", i+1, s)
	}
	fmt.Println("Wait:", err)
	fmt.Println("cause:", context.Cause(ctx))
}
