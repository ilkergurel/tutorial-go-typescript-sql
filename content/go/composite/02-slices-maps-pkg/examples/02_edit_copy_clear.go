package main

import (
	"fmt"
	"slices"
)

func main() {
	tags := []string{"go", "sql", "sql", "ts", "go"}

	// De-duplicate: sort, then Compact drops adjacent repeats.
	uniq := slices.Compact(slices.Sorted(slices.Values(tags)))
	fmt.Printf("%q\n", uniq)

	uniq = slices.Insert(uniq, 1, "react")
	fmt.Printf("%q\n", uniq)

	// Delete works in place and zeroes the slots it vacated (Go 1.22), so an
	// old header that still covers them sees "" instead of stale data.
	before := uniq
	uniq = slices.Delete(uniq, 0, 2)
	fmt.Printf("%q %q\n", uniq, before)

	// Filter in place instead of a loop that builds a second slice.
	nums := []int{1, 2, 3, 4, 5, 6}
	nums = slices.DeleteFunc(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println(nums)

	// The copy built-in copies min(len(dst), len(src)) elements; it never grows dst.
	dst := make([]int, 2)
	n := copy(dst, []int{7, 8, 9})
	fmt.Println(n, dst)

	// Clone gives an independent copy; Equal compares element by element.
	c := slices.Clone(nums)
	c[0] = 100
	fmt.Println(slices.Equal(c, nums), nums, c)

	// clear (Go 1.21) zeroes a slice's elements but keeps its length.
	clear(c)
	fmt.Println(c, len(c))
}
