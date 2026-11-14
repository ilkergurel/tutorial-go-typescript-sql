package main

import (
	"cmp"
	"fmt"
)

// Number is a constraint: an interface used only to describe a set of types.
// ~int64 means "every type whose underlying type is int64".
type Number interface {
	~int | ~int64 | ~float64
}

// Kurus is a money amount in kuruş (1/100 lira). Underlying type: int64.
type Kurus int64

func (k Kurus) String() string { return fmt.Sprintf("%d,%02d TL", k/100, k%100) }

func Sum[T Number](xs ...T) T {
	var total T // the zero value of whatever T is
	for _, x := range xs {
		total += x // allowed: every type in Number supports +
	}
	return total
}

// Clamp works for every ordered type: integers, floats, strings.
func Clamp[T cmp.Ordered](v, lo, hi T) T {
	return min(max(v, lo), hi)
}

// Convert lists To first: callers name it, and From is inferred.
func Convert[To, From Number](xs []From) []To {
	out := make([]To, len(xs))
	for i, x := range xs {
		out[i] = To(x)
	}
	return out
}

func main() {
	fmt.Println(Sum(1, 2, 3), Sum(0.5, 0.25)) // T inferred: int, then float64
	prices := []Kurus{1999, 4550}
	fmt.Println(Sum(prices...)) // T = Kurus: the result keeps its String method
	// Sum("a", "b")            // compile error: string does not satisfy Number

	fmt.Println(Clamp(15, 0, 10), Clamp(Kurus(-50), 0, 10000))
	fmt.Println(Clamp("zeytin", "armut", "kiraz"))
	fmt.Println(cmp.Compare("çilek", "zeytin")) // byte order: ç sorts after z

	ratios := Convert[float64]([]int{1, 2, 3}) // To named, From = int inferred
	fmt.Printf("%T %v\n", ratios, ratios)
}
