package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
)

// LEARN(go.files-io.reading-files): the logic accepts io.Reader; main owns the file.
func countLevels(r io.Reader) (map[string]int, error) {
	counts := map[string]int{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		counts[fields[1]]++
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return counts, nil
}

func main() {
	f, err := os.Open("app.log")
	if err != nil {
		log.Fatalf("open app.log: %v", err)
	}
	defer f.Close()

	counts, err := countLevels(f)
	if err != nil {
		log.Fatalf("count levels: %v", err)
	}
	for _, level := range slices.Sorted(maps.Keys(counts)) {
		fmt.Printf("%-5s %d\n", level, counts[level])
	}
}
