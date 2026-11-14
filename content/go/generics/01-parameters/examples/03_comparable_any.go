package main

import "fmt"

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T) { s[v] = struct{}{} }

func main() {
	// Set[[]int] would not compile: slices have no ==.
	// Set[any] does: since Go 1.20 interface types satisfy comparable,
	// and the check moves to run time, per dynamic value.
	s := Set[any]{}
	s.Add(1)
	s.Add("bir")
	fmt.Println(len(s))

	defer func() { fmt.Println("recovered:", recover()) }()
	s.Add([]int{1, 2}) // the dynamic type []int has no ==
	fmt.Println("not reached")
}
