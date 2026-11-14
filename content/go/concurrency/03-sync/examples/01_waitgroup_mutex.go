package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
)

// wordCount is shared by every goroutine. The mutex turns each
// read-modify-write of the map into one indivisible step.
type wordCount struct {
	mu     sync.Mutex
	counts map[string]int
}

func (w *wordCount) add(word string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.counts[word]++
}

func main() {
	lines := []string{
		"go channels go",
		"mutex guards state",
		"go wait group",
	}
	wc := &wordCount{counts: map[string]int{}}

	var wg sync.WaitGroup
	for _, line := range lines {
		wg.Go(func() { // Go 1.25: Add(1), go, defer Done() in one call
			for _, w := range strings.Fields(line) {
				wc.add(w)
			}
		})
	}
	wg.Wait()

	// No lock needed here: every goroutine has returned, and Wait makes
	// their writes visible to this goroutine.
	for _, w := range slices.Sorted(maps.Keys(wc.counts)) {
		fmt.Printf("%-8s %d\n", w, wc.counts[w])
	}
}
