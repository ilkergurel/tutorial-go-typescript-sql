package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
)

// Tally needs no constructor: the zero Mutex is unlocked, and the map is
// created on first write. Reading a nil map is fine; writing to one panics.
type Tally struct {
	mu     sync.Mutex
	counts map[string]int
}

func (t *Tally) Add(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.counts == nil {
		t.counts = map[string]int{}
	}
	t.counts[key]++
}

func (t *Tally) Count(key string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.counts[key]
}

// Report holds a Tally by value, so the zero Report is ready too.
type Report struct {
	Title string
	Runs  Tally
}

func main() {
	var r Report
	r.Runs.Add("go")
	r.Runs.Add("go")
	fmt.Println("runs:", r.Runs.Count("go"), r.Runs.Count("sql"))

	var buf bytes.Buffer // the standard library follows the same rule
	buf.WriteString("merhaba")
	var wg sync.WaitGroup
	wg.Go(func() { buf.WriteString(", dünya") })
	wg.Wait()
	fmt.Println(buf.String())

	// Useful is not the same as safe: zero here means "no limit".
	var c http.Client
	var s http.Server
	fmt.Println("http.Client Timeout:", c.Timeout)
	fmt.Println("http.Server ReadHeaderTimeout:", s.ReadHeaderTimeout)
}
