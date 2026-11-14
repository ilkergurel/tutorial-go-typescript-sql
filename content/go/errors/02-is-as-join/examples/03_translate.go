package main

import (
	"errors"
	"fmt"
)

// PgError stands in for a database driver's error type: pgx returns a
// *pgconn.PgError whose Code field holds the SQLSTATE.
type PgError struct{ Code, Message string }

func (e *PgError) Error() string { return e.Message + " (SQLSTATE " + e.Code + ")" }

// ErrEmailTaken is part of the repository's API: callers check this, never
// SQLSTATE codes.
var ErrEmailTaken = errors.New("email already registered")

func execInsert(email string) error { // pretend the database said this
	return &PgError{Code: "23505", Message: `duplicate key value violates unique constraint "users_email_key"`}
}

// Leaky: %w makes *PgError part of this function's API.
func createUserLeaky(email string) error {
	if err := execInsert(email); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// Translated: %w on the domain sentinel, %v on the driver error. The message
// keeps every detail for the log; only the sentinel can be inspected.
func createUser(email string) error {
	err := execInsert(email)
	if pgErr, ok := errors.AsType[*PgError](err); ok && pgErr.Code == "23505" {
		return fmt.Errorf("create user %s: %w: %v", email, ErrEmailTaken, err)
	}
	if err != nil {
		return fmt.Errorf("create user: %v", err) // other driver errors stay opaque
	}
	return nil
}

func main() {
	for _, err := range []error{createUserLeaky("ayse@example.com"), createUser("ayse@example.com")} {
		_, driverVisible := errors.AsType[*PgError](err)
		fmt.Println(err)
		fmt.Println("  email taken:", errors.Is(err, ErrEmailTaken), "| driver type visible:", driverVisible)
	}
}
