package main

import "fmt"

// Mixed: Inc needs a pointer, String was written with a value receiver.
type Hits struct{ n int }

func (h *Hits) Inc()          { h.n++ }
func (h Hits) String() string { return fmt.Sprintf("hits=%d", h.n) }

// Consistent: every method of Visits has a pointer receiver.
type Visits struct{ n int }

func (v *Visits) Inc()           { v.n++ }
func (v *Visits) String() string { return fmt.Sprintf("visits=%d", v.n) }

func main() {
	var h Hits
	var s fmt.Stringer = h // compiles, and copies h into the interface
	show := h.String       // method value: evaluates and copies h now
	h.Inc()
	fmt.Println(s, show(), h.String())

	var v Visits
	// var s2 fmt.Stringer = v // does not compile: String has a pointer receiver
	var s2 fmt.Stringer = &v
	show2 := v.String // v is addressable: this binds &v
	v.Inc()
	fmt.Println(s2, show2(), v.String())

	fmt.Println(v)  // a Visits value has no String method
	fmt.Println(&v) // a *Visits has one
}
