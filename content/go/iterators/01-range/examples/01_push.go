package main

import (
	"fmt"
	"iter"
	"maps"
	"slices"
)

// Countdown returns a push iterator: it calls yield once per value and
// stops as soon as yield returns false.
func Countdown(from int) iter.Seq[int] {
	return func(yield func(int) bool) {
		defer fmt.Println("  countdown: cleanup")
		for i := from; i > 0; i-- {
			fmt.Println("  countdown: yield", i)
			if !yield(i) {
				return
			}
		}
	}
}

func main() {
	for n := range Countdown(5) {
		fmt.Println("body:", n)
		if n == 4 {
			break // yield returns false inside Countdown
		}
	}

	// Seq2 yields pairs. maps.All, slices.All and slices.Backward are Seq2s.
	menu := []string{"çay", "simit", "ayran"}
	for i, item := range slices.Backward(menu) {
		fmt.Println(i, item)
	}

	stock := map[string]int{"çay": 12, "simit": 40, "ayran": 7}
	for _, k := range slices.Sorted(maps.Keys(stock)) { // Seq in, sorted slice out
		fmt.Println(k, stock[k])
	}
}
