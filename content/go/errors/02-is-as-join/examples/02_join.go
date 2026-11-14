package main

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

var ErrRequired = errors.New("required")

type TooLongError struct {
	Field    string
	Len, Max int
}

func (e *TooLongError) Error() string {
	return fmt.Sprintf("%s: %d characters, max %d", e.Field, e.Len, e.Max)
}

// validate reports every problem at once instead of stopping at the first.
func validate(name, city string) error {
	var errs []error
	if name == "" {
		errs = append(errs, fmt.Errorf("name: %w", ErrRequired))
	}
	// Characters, not bytes: len("Şanlıurfa") is 11, but it has 9 runes.
	if n := utf8.RuneCountInString(city); n > 10 {
		errs = append(errs, &TooLongError{Field: "city", Len: n, Max: 10})
	}
	return errors.Join(errs...) // nil when errs is empty
}

func main() {
	err := validate("", "Kahramanmaraş")
	fmt.Println(err) // one line per joined error
	fmt.Println("required?", errors.Is(err, ErrRequired))
	if tl, ok := errors.AsType[*TooLongError](err); ok {
		fmt.Println("too long:", tl.Field, tl.Len)
	}
	fmt.Println("valid:", validate("Ayşe", "Şanlıurfa") == nil)

	// fmt.Errorf with several %w builds the same kind of tree.
	errConflict := errors.New("conflict")
	errClosed := errors.New("connection closed")
	err = fmt.Errorf("save: %w; rollback: %w", errConflict, errClosed)
	fmt.Println(err)
	fmt.Println(errors.Is(err, errConflict), errors.Is(err, errClosed), errors.Unwrap(err))
}
