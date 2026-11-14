package main

import (
	"cmp"
	"fmt"
	"slices"
)

// MinOf returns the smallest element, or the zero value and false. A type
// parameter has no "nothing" of its own, so absence travels as the bool.
func MinOf[T cmp.Ordered](xs []T) (T, bool) {
	if len(xs) == 0 {
		var zero T
		return zero, false
	}
	return slices.Min(xs), true // slices.Min panics on an empty slice
}

func main() {
	offers := map[string][]int64{ // prices in kuruş
		"kahve": {},
		"simit": {1500, 1250},
		"su":    {500, 0}, // a free promotion: 0 is a real price
	}
	for _, p := range []string{"ayran", "kahve", "simit", "su"} {
		prices, listed := offers[p] // missing key: a nil slice and false
		cheapest, ok := MinOf(prices)
		switch {
		case !listed:
			fmt.Printf("%-6s not listed\n", p)
		case !ok:
			fmt.Printf("%-6s no offers\n", p)
		default:
			fmt.Printf("%-6s cheapest %d\n", p, cheapest)
		}
	}

	// Drop the bools and three different situations print the same 0.
	missing, _ := MinOf(offers["ayran"])
	none, _ := MinOf(offers["kahve"])
	free, _ := MinOf(offers["su"])
	fmt.Println("without the bools:", missing, none, free)
}
