package main

import "fmt"

func main() {
	var count int = 3
	var total int64 = 10
	// fmt.Println(count + total) // compile error: mismatched types int and int64
	fmt.Println(int64(count) + total)

	price := 9.99
	// fmt.Println(price * count) // compile error: mismatched types float64 and int
	fmt.Printf("%.2f\n", price*float64(count))

	f := -2.9
	fmt.Println(int(f), int(-f)) // truncates toward zero: not floor, not round

	n := 200
	fmt.Println(uint8(n), int8(n)) // same 8 bits, different meaning: 200 vs -56

	var u uint32
	u-- // unsigned arithmetic wraps around instead of failing
	fmt.Println(u)

	a, b := -7, 2
	fmt.Println(a/b, a%b) // integer division truncates toward zero too
}
