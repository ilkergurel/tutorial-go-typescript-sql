package main

import (
	"fmt"
	"strings"
)

func sum(nums ...int) int { // nums is an ordinary []int inside
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func double(nums ...int) {
	for i := range nums {
		nums[i] *= 2
	}
}

// counter returns a closure; each closure keeps its own count alive.
func counter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

func apply(items []string, f func(string) string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, f(it))
	}
	return out
}

func main() {
	fmt.Println(sum(), sum(1, 2, 3))

	xs := []int{1, 2, 3}
	fmt.Println(sum(xs...)) // spread a slice into a variadic call
	double(xs...)           // no copy: double writes into xs's backing array
	fmt.Println(xs)

	next, other := counter(), counter()
	fmt.Println(next(), next(), next(), other())

	shout := func(s string) string { return strings.ToUpper(s) + "!" }
	fmt.Println(apply([]string{"go", "ts"}, shout)) // functions are values
}
