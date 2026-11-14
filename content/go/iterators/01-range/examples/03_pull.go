package main

import (
	"fmt"
	"iter"
	"slices"
)

// Merge interleaves two ascending sequences. A range loop can only walk one
// push iterator at a time, so Merge pulls from both.
func Merge(a, b iter.Seq[int]) iter.Seq[int] {
	return func(yield func(int) bool) {
		nextA, stopA := iter.Pull(a)
		defer stopA()
		nextB, stopB := iter.Pull(b)
		defer stopB()

		va, okA := nextA()
		vb, okB := nextB()
		for okA || okB {
			if okA && (!okB || va <= vb) {
				if !yield(va) {
					return
				}
				va, okA = nextA()
			} else {
				if !yield(vb) {
					return
				}
				vb, okB = nextB()
			}
		}
	}
}

func main() {
	odds := slices.Values([]int{1, 3, 5, 7})
	evens := slices.Values([]int{2, 4, 6})
	fmt.Println(slices.Collect(Merge(odds, evens)))

	// Pulling by hand: next reports false once the sequence is finished.
	next, stop := iter.Pull(slices.Values([]string{"bir", "iki"}))
	defer stop()
	for v, ok := next(); ok; v, ok = next() {
		fmt.Println(v)
	}
	v, ok := next()
	fmt.Printf("%q %v\n", v, ok)
}
