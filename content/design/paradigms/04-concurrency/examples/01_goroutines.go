package main

import (
	"fmt"
	"sync"
)

// Stock is shared by every buyer goroutine, so its state is guarded.
type Stock struct {
	mu   sync.Mutex // guards left
	left int
}

// Reserve checks and writes under one lock: no other goroutine, on any
// core, can run between the check and the decrement.
func (s *Stock) Reserve() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.left == 0 {
		return false
	}
	s.left--
	return true
}

func main() {
	s := &Stock{left: 10}
	results := make(chan bool, 20)

	var wg sync.WaitGroup
	for range 20 { // 20 buyers at once, in parallel where there are cores
		wg.Go(func() { results <- s.Reserve() })
	}
	wg.Wait()
	close(results)

	sold, refused := 0, 0
	for ok := range results {
		if ok {
			sold++
		} else {
			refused++
		}
	}
	fmt.Printf("sold %d, refused %d, left %d\n", sold, refused, s.left)
}
