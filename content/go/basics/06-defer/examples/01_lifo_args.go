package main

import "fmt"

func main() {
	for i := range 3 {
		defer fmt.Println("deferred in loop:", i) // last in, first out
	}

	x := 1
	defer fmt.Println("argument evaluated at defer time:", x)
	defer func() {
		fmt.Println("closure reads x when it runs:", x)
	}()
	x = 2

	fmt.Println("end of main body")
}
