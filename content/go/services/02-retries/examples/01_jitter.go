package main

import (
	"fmt"
	"math/rand/v2"
	"time"
)

const (
	base    = 100 * time.Millisecond
	maxWait = 2 * time.Second
)

// ceiling doubles per attempt (100ms, 200ms, 400ms, ...) up to maxWait.
// Clamping attempt first keeps the shift from overflowing into a negative.
func ceiling(attempt int) time.Duration {
	return min(base<<min(attempt, 20), maxWait)
}

func row(label string, wait func(attempt int) time.Duration) {
	fmt.Printf("  %-9s", label)
	for attempt := range 6 {
		fmt.Printf("%8v", wait(attempt).Round(time.Millisecond))
	}
	fmt.Println()
}

func main() {
	// Seeded so this output repeats; production code calls rand.N, whose
	// source is seeded randomly for every process.
	r := rand.New(rand.NewPCG(1, 2))

	fmt.Println("no jitter: every client retries at the same instants")
	for _, c := range []string{"client A", "client B", "client C"} {
		row(c, ceiling)
	}
	fmt.Println("full jitter: a uniformly random wait in [0, ceiling)")
	for _, c := range []string{"client A", "client B", "client C"} {
		row(c, func(attempt int) time.Duration { return r.N(ceiling(attempt)) })
	}
	attempt := 40 // a long outage, or a loop without an attempt limit
	fmt.Println("attempt 40, unclamped:", base<<attempt)
	fmt.Println("attempt 40, clamped:  ", ceiling(attempt))
}
