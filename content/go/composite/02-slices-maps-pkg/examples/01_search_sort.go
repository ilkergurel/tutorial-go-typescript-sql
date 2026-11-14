package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type City struct {
	Name   string
	Region string
	Pop    int // population in thousands, approximate
}

func main() {
	nums := []int{42, 7, 19, 7, 3}
	fmt.Println(slices.Contains(nums, 19), slices.Index(nums, 7), slices.Max(nums))

	slices.Sort(nums) // in place, ascending
	pos, found := slices.BinarySearch(nums, 19)
	fmt.Println(nums, pos, found)

	cities := []City{
		{"İzmir", "Ege", 4479},
		{"Manisa", "Ege", 1475},
		{"Ankara", "İç Anadolu", 5864},
		{"Konya", "İç Anadolu", 2330},
		{"Aydın", "Ege", 1166},
	}
	// SortFunc takes a three-way comparison. cmp.Or returns the first
	// non-zero result: region ascending, then population descending.
	slices.SortFunc(cities, func(a, b City) int {
		return cmp.Or(
			strings.Compare(a.Region, b.Region),
			cmp.Compare(b.Pop, a.Pop),
		)
	})
	for _, c := range cities {
		fmt.Printf("%-11s %-7s %5d\n", c.Region, c.Name, c.Pop)
	}

	over5M := func(c City) bool { return c.Pop > 5000 }
	if i := slices.IndexFunc(cities, over5M); i >= 0 {
		fmt.Println("first city over 5M:", cities[i].Name)
	}

	// Sorting compares bytes, not the Turkish alphabet: Ç and İ land after Z.
	names := []string{"Zonguldak", "Çorum", "İzmir", "Ankara", "Iğdır"}
	slices.Sort(names)
	fmt.Println(names)
}
