package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// cache is read far more often than it is written. RWMutex lets readers
// share the lock; a writer waits for them and then holds it alone.
type cache struct {
	mu   sync.RWMutex
	data map[string]string
	// Counters on their own need no lock: each Add is one atomic step.
	hits, misses atomic.Int64
}

func (c *cache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	if ok {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}
	return v, ok
}

func (c *cache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func main() {
	c := &cache{data: map[string]string{"tr": "Türkiye", "de": "Deutschland"}}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { // 50 readers, each looking up three keys
			c.get("tr")
			c.get("de")
			c.get("fr")
		})
		wg.Go(func() { c.set(fmt.Sprintf("x%d", i), "?") }) // 50 writers
	}
	wg.Wait()
	fmt.Println("hits:", c.hits.Load(), "misses:", c.misses.Load())
	fmt.Println("entries:", len(c.data))
}
