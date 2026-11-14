package main

import (
	"fmt"
	"regexp"
	"runtime"
)

// A constant pattern is part of the program: if it doesn't compile, that is
// a bug, and MustCompile panics at startup, before any request is served.
var slugRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// classify runs f and names the kind of panic it raised, if any.
func classify(f func()) (kind string) {
	defer func() {
		switch r := recover().(type) {
		case nil:
			kind = "no panic"
		case runtime.Error:
			kind = "runtime.Error: " + r.Error()
		default:
			kind = fmt.Sprintf("%T: %v", r, r)
		}
	}()
	f()
	return
}

func main() {
	fmt.Println("slug ok:", slugRE.MatchString("errors-as-values"))

	// A pattern typed by a user is input, not a bug: Compile returns an error.
	_, err := regexp.Compile("[a-z")
	fmt.Println("error:", err)
	// The same pattern in a Must call panics: Must means "this cannot fail".
	fmt.Println(classify(func() { regexp.MustCompile("[a-z") }))

	// Bugs the runtime detects panic with a runtime.Error.
	var hits map[string]int
	i := 3
	fmt.Println(classify(func() { hits["x"]++ }))
	fmt.Println(classify(func() { _ = []int{1, 2, 3}[i] }))
	fmt.Println(classify(func() { panic(nil) })) // since Go 1.21, never a silent nil
	fmt.Println(classify(func() {}))
}
