package main

import (
	"cmp"
	"fmt"
	"slices"
)

// Set needs comparable: its elements become map keys, and keys need ==.
type Set[T comparable] map[T]struct{}

func SetOf[T comparable](vs ...T) Set[T] {
	s := make(Set[T], len(vs))
	for _, v := range vs {
		s[v] = struct{}{}
	}
	return s
}

func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

// A function, not a method: a method can't demand cmp.Ordered of Set's T.
func Sorted[T cmp.Ordered](s Set[T]) []T {
	out := make([]T, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	slices.Sort(out)
	return out
}

// Stack never compares elements, so any type will do.
type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func main() {
	cities := SetOf("İzmir", "Ankara", "İzmir", "Iğdır") // T = string, inferred
	fmt.Println(len(cities), cities.Has("Ankara"), cities.Has("izmir"))
	fmt.Println(Sorted(cities))

	var st Stack[rune] // no argument to infer from: name T yourself
	for _, r := range "kış" {
		st.Push(r)
	}
	for r, ok := st.Pop(); ok; r, ok = st.Pop() {
		fmt.Print(string(r))
	}
	r, ok := st.Pop()
	fmt.Printf("\n%q %v\n", r, ok) // empty stack: the zero rune and false
}
