// LEARN(inside.codebase.at-10x): runexec admits two runs at a time and turns a 10 s wait into 503; fake time shows where that breaks.
package main

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// The constants and the select are runexec.go's; only the run is a sleep.
const (
	maxConcurrent = 2
	queueWait     = 10 * time.Second
)

// press simulates `learners` people pressing Run at the same instant.
func press(learners int, run time.Duration) (served, busy int, longest time.Duration) {
	slots := make(chan struct{}, maxConcurrent)
	var mu sync.Mutex
	var wg sync.WaitGroup
	start := time.Now()
	for range learners {
		wg.Go(func() {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-time.After(queueWait):
				mu.Lock()
				busy++ // the learner sees "server busy" (503)
				mu.Unlock()
				return
			}
			mu.Lock()
			served++
			longest = max(longest, time.Since(start))
			mu.Unlock()
			time.Sleep(run) // compile + run
		})
	}
	wg.Wait()
	return served, busy, longest
}

func TestRunSlotsUnderLoad(t *testing.T) {
	for _, tc := range []struct {
		learners int
		run      time.Duration
	}{{1, 800 * time.Millisecond}, {10, 800 * time.Millisecond}, {10, 3 * time.Second}, {30, 3 * time.Second}} {
		synctest.Test(t, func(t *testing.T) {
			served, busy, longest := press(tc.learners, tc.run)
			t.Logf("%2d learners, %v per run: %2d served, %2d busy, longest wait %v",
				tc.learners, tc.run, served, busy, longest)
		})
	}
}
