package main

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// retry calls op until it succeeds, waiting 1s, 2s, 4s... between attempts.
// It gives up early when ctx is done.
func retry(ctx context.Context, attempts int, op func() error) error {
	delay := time.Second
	for i := 1; ; i++ {
		err := op()
		if err == nil || i == attempts {
			return err
		}
		select {
		case <-time.After(delay):
			delay *= 2
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
}

func TestRetryBacksOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now() // the bubble's fake clock: 2000-01-01 00:00 UTC
		calls := 0
		err := retry(t.Context(), 4, func() error {
			calls++
			if calls < 4 {
				return errors.New("503")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := time.Since(start); got != 7*time.Second { // exact, not "about"
			t.Fatalf("waited %v, want 7s (1s + 2s + 4s)", got)
		}
		t.Logf("%d calls, %v of fake time", calls, time.Since(start))
	})
}

func TestRetryStopsAtDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		start := time.Now()
		err := retry(ctx, 10, func() error { return errors.New("503") })
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want DeadlineExceeded", err)
		}
		t.Logf("gave up after %v: %v", time.Since(start), err)
	})
}
