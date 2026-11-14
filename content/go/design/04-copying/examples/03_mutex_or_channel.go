package main

import (
	"fmt"
	"maps"
	"sync"
)

// "Share memory by communicating": one goroutine owns the counts, and
// everyone else sends it messages.
type ChanCounter struct {
	incs  chan string
	reads chan chan map[string]int
}

func NewChanCounter() *ChanCounter {
	c := &ChanCounter{incs: make(chan string), reads: make(chan chan map[string]int)}
	go func() {
		counts := map[string]int{}
		for {
			select {
			case key := <-c.incs:
				counts[key]++
			case reply := <-c.reads:
				reply <- maps.Clone(counts)
			}
		}
	}()
	return c
}

func (c *ChanCounter) Inc(key string) { c.incs <- key }

func (c *ChanCounter) Snapshot() map[string]int {
	reply := make(chan map[string]int)
	c.reads <- reply
	return <-reply
}

// "Mutexes serialize": the same counter, guarded state instead of an owner.
type MutexCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func (c *MutexCounter) Inc(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	c.counts[key]++
}

func (c *MutexCounter) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.counts) // a copy: callers must not share our map
}

func main() {
	type counter interface {
		Inc(string)
		Snapshot() map[string]int
	}
	for _, c := range []counter{NewChanCounter(), &MutexCounter{}} {
		var wg sync.WaitGroup
		for i := range 1000 {
			wg.Go(func() { c.Inc([]string{"go", "sql"}[i%2]) })
		}
		wg.Wait()
		fmt.Printf("%T: %v\n", c, c.Snapshot())
	}
}
