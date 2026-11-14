package main

import "fmt"

type Counter struct {
	Name string
	Hits int
}

func incByValue(c Counter) { c.Hits++ } // gets a copy; the caller never sees it

func incByPointer(c *Counter) { c.Hits++ } // c.Hits is shorthand for (*c).Hits

// newCounter returns the address of a local variable. That's safe in Go:
// escape analysis sees it and allocates c on the heap.
func newCounter(name string) *Counter {
	c := Counter{Name: name}
	return &c
}

func main() {
	c := Counter{Name: "api"}
	incByValue(c)
	fmt.Println("after incByValue:", c.Hits)
	incByPointer(&c)
	fmt.Println("after incByPointer:", c.Hits)

	p := newCounter("db")
	p.Hits += 5
	fmt.Println(p.Name, p.Hits)

	alias := p // copies the pointer, not the struct
	alias.Hits = 0
	fmt.Println("read through p:", p.Hits)

	var missing *Counter
	fmt.Println("nil pointer:", missing == nil)
	// missing.Hits would panic with a nil pointer dereference, and p+1
	// doesn't compile: Go has no pointer arithmetic.
}
