package main

import "fmt"

type ValidationError struct{ Field string }

func (e *ValidationError) Error() string { return "invalid " + e.Field }

// Wrong: the success path returns a nil *ValidationError as an error.
func validateBad(name string) error {
	var verr *ValidationError
	if name == "" {
		verr = &ValidationError{Field: "name"}
	}
	return verr // when name is fine: (type *ValidationError, value nil)
}

// Right: return the literal nil on success.
func validate(name string) error {
	if name == "" {
		return &ValidationError{Field: "name"}
	}
	return nil
}

func main() {
	err := validateBad("Işıl")
	fmt.Println("bad:  err == nil?", err == nil)
	fmt.Printf("      %%v=%v %%T=%T\n", err, err)

	err = validate("Işıl")
	fmt.Println("good: err == nil?", err == nil)
	fmt.Printf("      %%v=%v %%T=%T\n", err, err)

	// Every interface type behaves this way, not only error.
	var p *int
	var x any = p
	fmt.Println("any holding a nil *int == nil?", x == nil)
	fmt.Println("the pointer inside == nil?   ", x.(*int) == nil)
}
