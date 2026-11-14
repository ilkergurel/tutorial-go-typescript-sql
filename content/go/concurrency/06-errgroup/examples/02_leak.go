package main

import (
	"fmt"
	"io"
	"runtime/pprof"
	"time"
)

// first asks every replica and returns the fastest answer. With an
// unbuffered channel the slower replicas block on their send forever:
// nobody will ever receive again. That is a goroutine leak.
func first(replicas []string, buffered bool) string {
	size := 0
	if buffered {
		size = len(replicas) // every replica can send and exit
	}
	answers := make(chan string, size)
	for _, r := range replicas {
		go func() { answers <- "answer from " + r }()
	}
	return <-answers
}

// leaked runs a leak-detecting GC cycle and returns how many goroutines can
// never wake up again (goroutineleak profile, Go 1.27).
func leaked() int {
	p := pprof.Lookup("goroutineleak")
	p.WriteTo(io.Discard, 0)
	return p.Count()
}

// settle polls until want leaks are found or a second has passed: the losing
// replicas need a moment to reach their blocked send.
func settle(want int) int {
	n := leaked()
	for start := time.Now(); n < want && time.Since(start) < time.Second; n = leaked() {
		time.Sleep(time.Millisecond)
	}
	return n
}

func main() {
	replicas := []string{"db-1", "db-2", "db-3"}

	first(replicas, true)
	fmt.Println("buffered channel, leaked goroutines:", settle(0))

	first(replicas, false)
	fmt.Println("unbuffered channel, leaked goroutines:", settle(2))
}
