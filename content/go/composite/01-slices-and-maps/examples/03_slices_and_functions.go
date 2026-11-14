package main

import (
	"encoding/json"
	"fmt"
)

// double changes elements in place: callee and caller share the array.
func double(xs []int) {
	for i := range xs {
		xs[i] *= 2
	}
}

// addBroken appends to its own copy of the slice header: the caller's
// length never changes.
func addBroken(xs []int, x int) { xs = append(xs, x) }

// add returns the new slice, the way append itself does.
func add(xs []int, x int) []int { return append(xs, x) }

func main() {
	xs := []int{1, 2, 3}
	double(xs)
	addBroken(xs, 4)
	fmt.Println(xs)
	xs = add(xs, 4)
	fmt.Println(xs)

	for _, x := range xs {
		x *= 10 // x is a copy of the element: xs doesn't change
	}
	fmt.Println(xs)

	var none []int   // nil slice: no array at all
	empty := []int{} // empty but not nil
	fmt.Println(len(none), len(empty), none == nil, empty == nil)
	out, err := json.Marshal(map[string][]int{"none": none, "empty": empty})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	none = append(none, 7) // append allocates for a nil slice
	fmt.Println(none)
}
