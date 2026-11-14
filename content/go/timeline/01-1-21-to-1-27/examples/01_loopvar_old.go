//go:build go1.21

// This file is compiled with Go 1.21 semantics, because its build
// constraint names go1.21: the last version before per-iteration loop
// variables. On a go 1.22+ module without the line, it prints differently.
package main

import (
	"fmt"
	"strings"
)

func main() {
	cities := []string{"Adana", "Bursa", "Çorum"}

	var shared []func() string
	for _, c := range cities {
		shared = append(shared, func() string { return c })
	}

	var copied []func() string
	for _, c := range cities {
		c := c // the pre-1.22 idiom: declare a fresh variable per iteration
		copied = append(copied, func() string { return c })
	}

	var ptrs []*int
	for i := 0; i < 3; i++ {
		ptrs = append(ptrs, &i)
	}

	fmt.Println("closures:        ", call(shared))
	fmt.Println("closures, c := c:", call(copied))
	fmt.Println("pointers to i:   ", *ptrs[0], *ptrs[1], *ptrs[2])
}

func call(fs []func() string) string {
	var out []string
	for _, f := range fs {
		out = append(out, f())
	}
	return strings.Join(out, " ")
}
