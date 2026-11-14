package main

import (
	"errors"
	"fmt"
	"strings"
)

// Expected failures are values a caller can inspect: a sentinel for "which
// failure", a type when the caller needs details.
var ErrEmailTaken = errors.New("email already registered")

type InvalidError struct{ Field, Reason string }

func (e *InvalidError) Error() string { return e.Field + ": " + e.Reason }

type Users struct{ emails map[string]bool }

// The signature says it can fail; it doesn't say how.
func (u *Users) Register(email string) error {
	if !strings.Contains(email, "@") {
		return &InvalidError{Field: "email", Reason: "must contain @"}
	}
	if u.emails[email] {
		return fmt.Errorf("register %s: %w", email, ErrEmailTaken)
	}
	u.emails[email] = true
	return nil
}

// One place decides what each failure means outside the program.
func status(err error) int {
	if err == nil {
		return 201
	}
	if errors.Is(err, ErrEmailTaken) {
		return 409
	}
	if _, ok := errors.AsType[*InvalidError](err); ok {
		return 422
	}
	return 500 // unexpected: log it here, once
}

func main() {
	users := &Users{emails: map[string]bool{"ayse@example.com": true}}
	for _, email := range []string{"mehmet@example.com", "ayse@example.com", "can.example.com"} {
		err := users.Register(email)
		fmt.Printf("%-19s %d  %v\n", email, status(err), err)
	}
}
