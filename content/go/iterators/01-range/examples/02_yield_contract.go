package main

import "fmt"

// careless ignores what yield returns.
func careless(yield func(int) bool) {
	for i := range 3 {
		yield(i)
	}
}

// careful stops as soon as the loop body is done.
func careful(yield func(int) bool) {
	for i := range 3 {
		if !yield(i) {
			return
		}
	}
}

func firstPositive(seq func(func(int) bool)) int {
	for v := range seq {
		if v > 0 {
			return v // returns from firstPositive, not just from yield
		}
	}
	return -1
}

func deferInBody() {
	for v := range careful {
		defer fmt.Println("deferred", v) // runs when deferInBody returns
	}
	fmt.Println("loop done")
}

func main() {
	fmt.Println(firstPositive(careful))
	deferInBody()

	defer func() { fmt.Println("recovered:", recover()) }()
	fmt.Println(firstPositive(careless))
}
