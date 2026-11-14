package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Executor admits at most cap(slots) runs, like internal/runexec.
type Executor struct{ slots chan struct{} }

func New() *Executor { return &Executor{slots: make(chan struct{}, 2)} } // maxConcurrent

func (e *Executor) Run(ctx context.Context, name string, queueWait time.Duration) string {
	select {
	case e.slots <- struct{}{}:
		defer func() { <-e.slots }()
	case <-ctx.Done():
		return name + ": 503 server busy (the client left)"
	case <-time.After(queueWait):
		return name + ": 503 server busy"
	}
	time.Sleep(400 * time.Millisecond) // compile + run
	return name + ": ran"
}

func main() {
	e := New()
	results := make([]string, 4)
	var wg sync.WaitGroup
	for i, name := range []string{"run 1", "run 2"} {
		wg.Go(func() { results[i] = e.Run(context.Background(), name, time.Second) })
	}
	for len(e.slots) < 2 { // wait until both runs hold a slot
		time.Sleep(time.Millisecond)
	}

	results[2] = e.Run(context.Background(), "run 3", 100*time.Millisecond) // real queueWait: 10s
	results[3] = e.Run(context.Background(), "run 4", time.Second)          // gets a freed slot
	wg.Wait()
	for _, r := range results {
		fmt.Println(r)
	}
}
