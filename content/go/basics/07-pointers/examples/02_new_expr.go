package main

import "fmt"

// Patch is a partial update: nil means "leave unchanged", which a plain
// string or int can't express ("" and 0 are real values).
type Patch struct {
	Name    *string
	Retries *int
}

func describe(p Patch) string {
	name, retries := "unchanged", "unchanged"
	if p.Name != nil {
		name = fmt.Sprintf("%q", *p.Name)
	}
	if p.Retries != nil {
		retries = fmt.Sprint(*p.Retries)
	}
	return "name=" + name + " retries=" + retries
}

// ptr is the helper codebases wrote before Go 1.26: &"api" and &0 don't
// compile, because only addressable operands have an address.
func ptr[T any](v T) *T { return &v }

func main() {
	before := Patch{Name: ptr("api")}
	after := Patch{Name: new("api"), Retries: new(0)} // Go 1.26: new(expr)
	fmt.Println(describe(before))
	fmt.Println(describe(after))

	zero, seven := new(int), new(7) // new(T) zeroes; new(expr) copies the value
	*zero++
	fmt.Println(*zero, *seven)

	empty := &Patch{} // &T{...}: the usual way to get a pointer to a struct
	fmt.Println(describe(*empty))
}
