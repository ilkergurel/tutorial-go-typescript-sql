package main

import (
	"fmt"
	"time"
)

const interval = 20 * time.Millisecond // 50 requests per second

func main() {
	requests := []string{"r1", "r2", "r3", "r4", "r5"}

	// Steady rate: each request waits for the next tick.
	start := time.Now()
	tick := time.NewTicker(interval)
	for _, r := range requests {
		<-tick.C
		fmt.Println("steady", r)
	}
	tick.Stop()
	fmt.Println("steady: took >= 5 intervals:", time.Since(start) >= 5*interval)

	// Token bucket: capacity 3 allows a burst, then one token per tick.
	bucket := make(chan struct{}, 3)
	for range cap(bucket) {
		bucket <- struct{}{} // start full
	}
	done := make(chan struct{})
	defer close(done) // stops the refill goroutine: no leak
	start = time.Now()
	go func() {
		refill := time.NewTicker(interval)
		defer refill.Stop()
		for {
			select {
			case <-refill.C:
				select {
				case bucket <- struct{}{}:
				default: // bucket full: the token is dropped
				}
			case <-done:
				return
			}
		}
	}()

	for _, r := range requests {
		<-bucket
		fmt.Println("bursty", r)
	}
	fmt.Println("bursty: took >= 2 intervals:", time.Since(start) >= 2*interval)
}
