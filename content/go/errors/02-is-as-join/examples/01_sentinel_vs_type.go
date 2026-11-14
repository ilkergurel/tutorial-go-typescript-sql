package main

import (
	"errors"
	"fmt"
)

// Sentinel: one fixed value. It tells the caller WHICH failure happened.
var ErrNotFound = errors.New("not found")

// Error type: carries data the caller can act on.
type ValidationError struct {
	Field, Reason string
}

func (e *ValidationError) Error() string { return e.Field + " " + e.Reason }

func findUser(id int) (string, error) {
	if id <= 0 {
		return "", &ValidationError{Field: "id", Reason: "must be positive"}
	}
	if id != 42 {
		return "", fmt.Errorf("find user %d: %w", id, ErrNotFound)
	}
	return "Gülşen", nil
}

func main() {
	_, err := findUser(7)
	fmt.Println(err)
	fmt.Println("  err == ErrNotFound:", err == ErrNotFound)
	fmt.Println("  errors.Is:         ", errors.Is(err, ErrNotFound))

	_, err = findUser(-1)
	err = fmt.Errorf("GET /users/-1: %w", err) // one more layer
	fmt.Println(err)
	_, ok := err.(*ValidationError)
	fmt.Println("  type assertion:", ok)

	if verr, ok := errors.AsType[*ValidationError](err); ok {
		fmt.Println("  errors.AsType: ", verr.Field, "/", verr.Reason)
	}
	var verr *ValidationError // errors.As: the spelling before Go 1.26
	if errors.As(err, &verr) {
		fmt.Println("  errors.As:     ", verr.Field, "/", verr.Reason)
	}
}
