package main

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

func worker(id int, jobs <-chan int, wg *sync.WaitGroup, seen map[int]string, mu *sync.Mutex) {
	defer wg.Done()
	for j := range jobs { // ends when jobs is closed and drained
		mu.Lock()
		seen[j] = fmt.Sprintf("job %d taken", j)
		mu.Unlock()
	}
}

func main() {
	jobs := make(chan int) // unbuffered: sends block until a worker receives
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[int]string{} // job -> record, collected from many goroutines

	for id := 1; id <= 3; id++ {
		wg.Add(1)
		go worker(id, jobs, &wg, seen, &mu)
	}

	for j := 1; j <= 6; j++ {
		jobs <- j // hands off to whichever worker is ready first
	}
	close(jobs) // no more jobs: workers' range loops end

	wg.Wait() // all workers done

	// Which worker takes which job is the scheduler's call, so raw output varies
	// between runs. Sorting gives stable output without hiding the concurrency:
	// every job appears exactly once.
	for _, job := range slices.Sorted(maps.Keys(seen)) {
		fmt.Println(seen[job])
	}
}
