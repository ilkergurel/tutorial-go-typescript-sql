package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
)

// Problem is RFC 9457 problem details; errors is TraceLite's extension member.
type Problem struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}
type FieldError struct {
	Pointer string `json:"pointer"`
	Detail  string `json:"detail"`
}
type ValidationError struct{ Fields []FieldError } // the service's (example 2)

func (e *ValidationError) Error() string { return fmt.Sprint(e.Fields) }

var (
	ErrNotFound      = errors.New("not found") // the store's (step 3)
	ErrDuplicateCode = errors.New("duplicate code")
	errBadBody       = errors.New("the body is not a RequirementCreate object") // the handler's (example 1)
)

// writeError is the one place where errors become statuses.
func writeError(w http.ResponseWriter, err error) {
	var invalid *ValidationError
	p := Problem{Type: "about:blank"}
	switch {
	case errors.As(err, &invalid):
		p = Problem{Type: "https://tracelite.example/problems/validation", Title: "Validation failed",
			Status: http.StatusUnprocessableEntity, Errors: invalid.Fields}
	case errors.Is(err, ErrDuplicateCode):
		p = Problem{Type: "https://tracelite.example/problems/conflict", Title: "Conflict", Status: http.StatusConflict,
			Detail: "the code is already used in this project", Errors: []FieldError{{"#/code", "already used"}}}
	case errors.Is(err, errBadBody):
		p.Status, p.Detail = http.StatusBadRequest, err.Error()
	case errors.Is(err, ErrNotFound):
		p.Status = http.StatusNotFound
	default:
		log.Printf("internal error: %v", err) // the cause goes to the log, not to the client
		p.Status = http.StatusInternalServerError
	}
	if p.Type == "about:blank" {
		p.Title = http.StatusText(p.Status)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.MarshalWrite(w, p)
}

func main() {
	for _, err := range []error{
		&ValidationError{[]FieldError{{"#/code", "use A-Z, 0-9 and single hyphens"}, {"#/title", "must be 1-200 characters, got 0"}}},
		fmt.Errorf("create AUTH: %w", ErrDuplicateCode),
		errBadBody,
		ErrNotFound,
		errors.New(`insert requirement: relation "requirements" does not exist`),
	} {
		rec := httptest.NewRecorder()
		writeError(rec, err)
		fmt.Println(rec.Code, rec.Body)
	}
}
