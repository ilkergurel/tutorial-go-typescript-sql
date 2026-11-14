package main

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrDivByZero = errors.New("division by zero")

// divide returns a result and an error: the caller has to look at both.
func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrDivByZero
	}
	return a / b, nil
}

// minMax names its results: they document the return values and start at zero.
func minMax(xs []int) (lo, hi int) {
	if len(xs) == 0 {
		return // naked return: returns lo and hi as they are (0, 0)
	}
	lo, hi = xs[0], xs[0]
	for _, x := range xs[1:] {
		lo, hi = min(lo, x), max(hi, x) // min and max builtins: Go 1.21
	}
	return lo, hi
}

func main() {
	if q, err := divide(7, 2); err == nil {
		fmt.Println("7/2 =", q)
	}
	if _, err := divide(1, 0); errors.Is(err, ErrDivByZero) {
		fmt.Println("error:", err)
	}

	n, err := strconv.Atoi("42") // the same (value, error) shape everywhere
	fmt.Println(n, err)

	fmt.Println(minMax([]int{4, -2, 9, 0}))
	fmt.Println(minMax(nil))
}
