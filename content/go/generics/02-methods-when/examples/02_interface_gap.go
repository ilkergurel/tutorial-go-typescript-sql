package main

import (
	"fmt"
	"reflect"
	"strconv"
)

type Box[T any] struct{ v T }

func (b Box[T]) String() string { return fmt.Sprintf("Box(%v)", b.v) }

// Convert adds its own type parameter U to the receiver's T.
func (b Box[T]) Convert[U any](f func(T) U) Box[U] { return Box[U]{f(b.v)} }

// What a caller might hope Box[int] satisfies. Interface methods can't
// declare type parameters, so the interface has to pick one instance.
type ToText interface {
	Convert(func(int) string) Box[string]
}

func main() {
	b := Box[int]{42}
	fmt.Printf("%q\n", b.Convert(strconv.Itoa).v) // direct call: U = string

	var s fmt.Stringer = b // String is an ordinary method
	fmt.Println(s)

	// var _ ToText = b    // compile error: wrong type for method Convert
	_, ok := any(b).(ToText)
	fmt.Println("implements ToText:", ok)

	// reflect cannot instantiate a method, so it lists only ordinary ones.
	t := reflect.TypeOf(b)
	for i := range t.NumMethod() {
		fmt.Println("reflect sees:", t.Method(i).Name)
	}
}
