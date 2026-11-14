package main

import "fmt"

func main() {
	arr := [5]int{1, 2, 3, 4, 5} // an array: fixed length, copied on assignment
	s := arr[1:3]                // a slice: pointer into arr, len 2, cap 4
	fmt.Println(s, len(s), cap(s))

	s[0] = 20 // writes through to arr
	fmt.Println("arr:", arr)

	s = append(s, 30) // room left in cap: append overwrites arr[3]
	fmt.Println("arr:", arr)

	s = append(s, 40, 50) // no room: new backing array, arr is left alone
	s[0] = 99
	fmt.Println("arr:", arr, "s:", s)

	// The classic bug: two appends to the same slice share its spare capacity.
	base := make([]int, 3, 10)
	a := append(base, 1)
	b := append(base, 2)
	fmt.Println("a:", a, "b:", b)

	// A full slice expression base[low:high:max] caps the capacity, so the
	// next append has to copy.
	a = append(base[:3:3], 1)
	b = append(base[:3:3], 2)
	fmt.Println("a:", a, "b:", b)
}
