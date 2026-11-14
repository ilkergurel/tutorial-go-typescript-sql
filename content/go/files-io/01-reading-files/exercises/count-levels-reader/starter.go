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

func countLevels(r io.Reader) (map[string]int, error) {
	// TODO: scan r line by line, count fields[1] per line, skip malformed lines,
	// return an error if the scanner itself failed.
	return nil, nil
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
