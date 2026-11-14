package main

import (
	"bufio"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
)

func main() {
	f, err := os.Open("app.log")
	if err != nil {
		log.Fatalf("open app.log: %v", err)
	}
	defer f.Close() // runs when main returns

	counts := map[string]int{}
	sc := bufio.NewScanner(f)
	for sc.Scan() { // one line per iteration, newline stripped
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue // skip malformed lines
		}
		counts[fields[1]]++ // fields[1] is the level: INFO, WARN, ERROR
	}
	if err := sc.Err(); err != nil { // Scan returns false on EOF *and* on error
		log.Fatalf("scan app.log: %v", err)
	}

	// Map iteration order is random in Go: sort the keys for stable output.
	for _, level := range slices.Sorted(maps.Keys(counts)) {
		fmt.Printf("%-5s %d\n", level, counts[level])
	}
}
