package main

import (
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
)

var sink []byte

// churn allocates 64 MiB in 4 KiB pieces and keeps only the last one.
func churn() {
	for range 16 << 10 {
		sink = make([]byte, 4<<10)
	}
}

// reuse does the same number of writes into one buffer.
func reuse() {
	buf := sink
	for range 16 << 10 {
		clear(buf)
	}
}

// cycles counts the GC cycles that ran while f did.
func cycles(f func()) uint32 {
	var before, after runtime.MemStats
	runtime.GC() // start from a collected heap
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.NumGC - before.NumGC
}

func main() {
	fmt.Println("GOGC=100, allocating:   cycles > 0:", cycles(churn) > 0)
	fmt.Println("GOGC=100, reusing:      cycles =", cycles(reuse))

	old := debug.SetGCPercent(-1) // GOGC=off: no proportional trigger at all
	fmt.Println("GOGC=off, allocating:   cycles =", cycles(churn))

	debug.SetMemoryLimit(16 << 20) // GOMEMLIMIT=16MiB still triggers collections
	fmt.Println("GOGC=off, 16 MiB limit: cycles > 0:", cycles(churn) > 0)

	debug.SetMemoryLimit(math.MaxInt64)
	debug.SetGCPercent(old)
	fmt.Println("default GOGC was", old)
}
