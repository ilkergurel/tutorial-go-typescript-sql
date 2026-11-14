package main

import "fmt"

func main() {
	var seen []int
	for i := range 3 { // Go 1.22: range over an int counts 0..n-1
		seen = append(seen, i)
	}
	fmt.Println(seen)

	n := 1
	for n < 100 { // Go's "while" is a for with only a condition
		n *= 3
	}
	fmt.Println("n =", n)

	grid := [][]int{{1, 2}, {3, -1}, {5, 6}}
outer: // a label names the loop that break/continue should target
	for r, row := range grid {
		for _, v := range row {
			if v < 0 {
				fmt.Println("negative value in row", r)
				break outer // a plain break would only leave the inner loop
			}
		}
	}

	// Go 1.22: each iteration gets its own i, so the closures see 0, 1, 2.
	// Before 1.22 they all shared one i and returned 3, 3, 3.
	var funcs []func() int
	for i := 0; i < 3; i++ {
		funcs = append(funcs, func() int { return i })
	}
	var got []int
	for _, f := range funcs {
		got = append(got, f())
	}
	fmt.Println(got)
}
