package main

import "fmt"

type Config struct {
	Name    string
	Retries int
	Debug   bool
	Tags    []string
	Limits  map[string]int
}

func main() {
	var c Config // no constructor: every field starts at its zero value
	fmt.Printf("%q %d %t %v %v\n", c.Name, c.Retries, c.Debug, c.Tags, c.Limits)
	fmt.Println("nil slice:", c.Tags == nil, "nil map:", c.Limits == nil)

	// Zero values are usable, not just "empty":
	c.Tags = append(c.Tags, "beta")           // append allocates on demand
	fmt.Println(len(c.Limits), c.Limits["x"]) // reading a nil map is fine
	// c.Limits["x"] = 1 would panic: writing needs make(map[string]int)

	n := 3               // short declaration: infers int, function bodies only
	n, extra := 4, "new" // legal: extra is new, so n is only assigned
	a, b := 1, 2
	a, b = b, a // tuple assignment, no temp variable
	fmt.Println(c.Tags, n, extra, a, b)
}
