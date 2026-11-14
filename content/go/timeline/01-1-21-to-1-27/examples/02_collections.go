package main

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"
	"strings"
)

var plates = map[string]int{"İzmir": 35, "Adana": 1, "Muğla": 48, "Bursa": 16}

// Old: collect the keys, sort them, return a slice.
func sortedKeysOld(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Evens is an iterator (Go 1.23): it calls yield once per value and
// stops as soon as yield returns false.
func Evens(limit int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range limit { // range over an int (Go 1.22)
			if i%2 == 0 && !yield(i) {
				return
			}
		}
	}
}

func main() {
	fmt.Println("sort.Strings:   ", sortedKeysOld(plates))
	fmt.Println("slices.Sorted:  ", slices.Sorted(maps.Keys(plates))) // Go 1.23
	fmt.Println("min, max:       ", min(35, 1, 48), max(35, 1, 48))   // Go 1.21
	fmt.Println("cmp.Or:         ", cmp.Or("", "", "tr-TR"))          // Go 1.22

	var firstFour []int
	for n := range Evens(1_000_000) {
		if len(firstFour) == 4 {
			break // yield returns false and Evens returns
		}
		firstFour = append(firstFour, n)
	}
	fmt.Println("range over func:", firstFour)

	fmt.Print("SplitSeq:        ")
	for part := range strings.SplitSeq("Çay,Simit,Ayran", ",") { // Go 1.24
		fmt.Printf("[%s]", part)
	}
	fmt.Println()
}
