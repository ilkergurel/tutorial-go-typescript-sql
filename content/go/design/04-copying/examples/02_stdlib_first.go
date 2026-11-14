package main

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Run struct {
	Lesson string
	Millis int
}

func main() {
	runs := []Run{
		{"go.basics.defer", 420}, {"sql.joins.lateral", 95},
		{"go.basics.defer", 380}, {"ts.react.query", 1210},
	}

	// Group by key: a map and a loop, no GroupBy helper needed.
	byLesson := map[string][]int{}
	for _, r := range runs {
		byLesson[r.Lesson] = append(byLesson[r.Lesson], r.Millis)
	}
	for _, id := range slices.Sorted(maps.Keys(byLesson)) { // Go 1.23
		fmt.Println(id, byLesson[id], "max:", slices.Max(byLesson[id]))
	}

	// Sort by one field, then another: cmp.Or picks the first non-zero result.
	slices.SortFunc(runs, func(a, b Run) int {
		return cmp.Or(cmp.Compare(b.Millis, a.Millis), strings.Compare(a.Lesson, b.Lesson))
	})
	fmt.Println("slowest:", runs[0].Lesson)

	// Unique values: sort, then drop adjacent duplicates.
	var ids []string
	for _, r := range runs {
		ids = append(ids, r.Lesson)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	fmt.Println("unique:", ids)

	// Batches of two (Go 1.23).
	for batch := range slices.Chunk(ids, 2) {
		fmt.Println("batch:", batch)
	}

	// Split once without a regexp; built-in min and max (Go 1.21).
	track, rest, _ := strings.Cut("go.basics.defer", ".")
	fmt.Println(track, "|", rest, "| min:", min(3, 1, 2), "| default:", cmp.Or("", "unnamed"))
}
