package main

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Filter[V any](seq iter.Seq[V], keep func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

func Map[V, W any](seq iter.Seq[V], f func(V) W) iter.Seq[W] {
	return func(yield func(W) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		if n <= 0 {
			return
		}
		taken := 0 // per call, so the Seq can be ranged again
		for v := range seq {
			if !yield(v) {
				return
			}
			if taken++; taken == n {
				return
			}
		}
	}
}

func main() {
	words := []string{"çay", "ılık", "incir", "simit", "ıhlamur", "kiraz"}
	long := func(s string) bool {
		fmt.Println("  check", s)
		return utf8.RuneCountInString(s) >= 5 // letters; len("ılık") is 6
	}
	upper := func(s string) string { return strings.ToUpperSpecial(unicode.TurkishCase, s) }

	pipeline := Take(Map(Filter(slices.Values(words), long), upper), 2)
	fmt.Println("built, nothing read yet")
	for w := range pipeline {
		fmt.Println(w)
	}
}
