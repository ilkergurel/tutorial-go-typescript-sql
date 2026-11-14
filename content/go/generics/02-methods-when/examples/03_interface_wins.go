package main

import (
	"fmt"
	"math"
)

type Shape interface{ Area() float64 }

type Circle struct{ R float64 }
type Rect struct{ W, H float64 }

func (c Circle) Area() float64 { return math.Pi * c.R * c.R }
func (r Rect) Area() float64   { return r.W * r.H }

// Over-genericized rewrite of Area: a closed type set plus a type switch is
// a method in disguise. A new shape means editing this function.
func AreaOf[T Circle | Rect](s T) float64 {
	switch v := any(s).(type) {
	case Circle:
		return math.Pi * v.R * v.R
	case Rect:
		return v.W * v.H
	}
	panic("unreachable")
}

// The interface only consumes behavior, so it needs no type parameter,
// and one slice can mix shapes.
func Total(shapes []Shape) float64 {
	sum := 0.0
	for _, s := range shapes {
		sum += s.Area()
	}
	return sum
}

// A type parameter earns its keep when the type flows back out.
func Largest[S Shape](xs []S) S {
	best := xs[0]
	for _, x := range xs[1:] {
		if x.Area() > best.Area() {
			best = x
		}
	}
	return best
}

func main() {
	fmt.Printf("%.2f\n", AreaOf(Rect{W: 2, H: 3}))
	// var mixed []Circle | Rect // not a type: unions exist only in constraints
	fmt.Printf("%.2f\n", Total([]Shape{Circle{R: 1}, Rect{W: 2, H: 3}}))

	c := Largest([]Circle{{R: 1}, {R: 3}, {R: 2}})
	fmt.Println("largest radius:", c.R) // c is a Circle: no type assertion
}
