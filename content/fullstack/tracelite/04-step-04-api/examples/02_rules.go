package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type FieldError struct {
	Pointer string `json:"pointer"` // a JSON Pointer into the request body
	Detail  string `json:"detail"`
}

// ValidationError lists every field that breaks a rule; writeError makes it a 422.
type ValidationError struct{ Fields []FieldError }

func (e *ValidationError) Error() string { return fmt.Sprint(e.Fields) }

var ErrParentNotFound = errors.New("parent not found in this project") // the store's (step 3)

var codeFormat = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$`)

// CreateRequirement normalizes, checks every rule, then lets the store decide what only
// the database can: uniqueness and the same-project parent.
func CreateRequirement(code, title string, insert func(code, title string) error) error {
	title = strings.TrimSpace(title)
	var fields []FieldError // all of them: a form shows every problem at once
	if !codeFormat.MatchString(code) || len(code) > 40 {
		fields = append(fields, FieldError{"#/code", "use A-Z, 0-9 and single hyphens, like AUTH-LOGIN"})
	}
	if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
		fields = append(fields, FieldError{"#/title", fmt.Sprintf("must be 1-200 characters, got %d", n)})
	}
	if fields != nil {
		return &ValidationError{fields}
	}
	err := insert(code, title)
	if errors.Is(err, ErrParentNotFound) {
		return &ValidationError{[]FieldError{{"#/parentId", "no such requirement in this project"}}}
	}
	return err
}

func main() {
	// The store, faked: it reports what reaches it, and AUTH-2FA's parent is in another project.
	insert := func(code, title string) error {
		if code == "AUTH-2FA" {
			return ErrParentNotFound
		}
		n := utf8.RuneCountInString(title)
		fmt.Printf("  stored %s, title %.20q: %d characters, %d bytes\n", code, title, n, len(title))
		return nil
	}
	for _, in := range [][2]string{
		{"AUTH-LOGIN", "  Parola ile giriş "},
		{"AUTH-LOGİN", "   "},
		{"LONG", strings.Repeat("ş", 200)},
		{"AUTH-2FA", "İki adım"},
	} {
		fmt.Printf("%s: %v\n", in[0], CreateRequirement(in[0], in[1], insert))
	}
}
