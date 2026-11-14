package main

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

type List[E any] []E

func (l List[E]) Filter(keep func(E) bool) List[E] {
	var out List[E]
	for _, x := range l {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

// Map declares its own type parameter F next to the receiver's E (Go 1.27).
func (l List[E]) Map[F any](f func(E) F) List[F] {
	out := make(List[F], 0, len(l))
	for _, x := range l {
		out = append(out, f(x))
	}
	return out
}

// The pre-1.27 shape of the same operation, kept for comparison.
func MapList[E, F any](l List[E], f func(E) F) List[F] { return l.Map(f) }

func even(n int) bool       { return n%2 == 0 }
func square(n int) int      { return n * n }
func label(s string) string { return "#" + s }

func main() {
	nums := List[int]{1, 2, 3, 4, 5, 6}

	fmt.Println(MapList(MapList(MapList(nums.Filter(even), square), strconv.Itoa), label))
	fmt.Println(nums.Filter(even).Map(square).Map(strconv.Itoa).Map(label))

	// As a value, a generic method must be instantiated first.
	toText := List[int].Map[string] // func(List[int], func(int) string) List[string]
	fmt.Println(strings.Join(toText(nums, strconv.Itoa), "+"))

	// The one generic method in the Go 1.27 standard library: (*rand.Rand).N.
	r := rand.New(rand.NewPCG(1, 2))
	fmt.Printf("%T %T\n", r.N(10), r.N(5*time.Second))
}
