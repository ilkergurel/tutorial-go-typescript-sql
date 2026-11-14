package main

import (
	"fmt"
	"sync"
	"time"
)

// tracker records how many tasks run at once, and the peak.
type tracker struct {
	mu        sync.Mutex
	now, peak int
}

func (t *tracker) task(id int) int {
	t.mu.Lock()
	t.now++
	t.peak = max(t.peak, t.now)
	t.mu.Unlock()
	defer func() { t.mu.Lock(); t.now--; t.mu.Unlock() }()
	time.Sleep(5 * time.Millisecond) // simulated I/O, not for ordering
	return id * id
}

func main() {
	results := make([]int, 10) // one slot per job: no lock, input order kept

	// Worker pool: three long-lived goroutines pull from one queue.
	var pool tracker
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			for id := range jobs {
				results[id] = pool.task(id)
			}
		})
	}
	for id := range results {
		jobs <- id
	}
	close(jobs)
	wg.Wait()
	fmt.Println("pool:", results, "peak <= 3:", pool.peak <= 3)

	// Semaphore: one goroutine per job, a buffered channel admits three.
	var sem tracker
	slots := make(chan struct{}, 3)
	for id := range results {
		wg.Go(func() {
			slots <- struct{}{}        // acquire: blocks while three are inside
			defer func() { <-slots }() // release
			results[id] = sem.task(id)
		})
	}
	wg.Wait()
	fmt.Println("semaphore:", results, "peak <= 3:", sem.peak <= 3)
}
