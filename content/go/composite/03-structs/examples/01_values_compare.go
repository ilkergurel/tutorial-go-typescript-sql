package main

import "fmt"

type Point struct {
	X, Y int
}

type Route struct {
	Name  string
	Stops []Point // a slice field: copying a Route copies only its header
}

func main() {
	var zero Point         // zero value: every field zeroed, ready to use
	p := Point{X: 3, Y: 4} // keyed literal: robust to new fields
	q := p                 // assignment copies all fields
	q.X = 100
	fmt.Println(zero, p, q)

	// Structs whose fields are all comparable support == and work as map keys.
	fmt.Println(p == Point{3, 4})
	visits := map[Point]int{{3, 4}: 2}
	visits[p]++
	fmt.Println(visits)

	// A copy is shallow: both Routes share one Stops array.
	r1 := Route{Name: "İzmir-Ankara", Stops: []Point{{0, 0}, {5, 5}}}
	r2 := r1
	r2.Name = "copy"
	r2.Stops[0] = Point{9, 9}
	fmt.Println(r1.Name, r1.Stops)

	// Anonymous structs: one-off shapes, common in table-driven tests.
	cases := []struct {
		in    string
		bytes int
	}{{"i", 1}, {"İ", 2}}
	fmt.Printf("%+v\n", cases)

	// struct{} has size zero: a map[string]struct{} is a set.
	seen := map[string]struct{}{"go": {}}
	_, ok := seen["go"]
	fmt.Println(ok)
}
