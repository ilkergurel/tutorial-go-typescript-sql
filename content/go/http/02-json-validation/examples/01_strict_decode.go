package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// decodeJSON reads exactly one JSON value of at most 1 KiB into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if !errors.Is(dec.Decode(&struct{}{}), io.EOF) {
		return errors.New("body must hold a single JSON value")
	}
	return nil
}

// decodeStatus maps a decode error to a status and a client-safe message.
func decodeStatus(err error) (int, string) {
	var tooBig *http.MaxBytesError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &tooBig):
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("body exceeds %d bytes", tooBig.Limit)
	case errors.As(err, &typeErr):
		return http.StatusBadRequest, fmt.Sprintf("%s: got %s, want %s", typeErr.Field, typeErr.Value, typeErr.Type)
	}
	return http.StatusBadRequest, err.Error() // syntax, unknown field, trailing data
}

func greet(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		status, msg := decodeStatus(err)
		http.Error(w, msg, status) // plain text here; example 3 answers problem+json
		return
	}
	fmt.Fprintf(w, "hello %s (%d)\n", in.Name, in.Age)
}

func main() {
	for _, c := range []struct{ label, body string }{
		{"valid", `{"name":"Ayşe","age":30}`},
		{"typo", `{"name":"Ayşe","agee":30}`},
		{"type", `{"name":"Ayşe","age":"30"}`},
		{"two", `{"name":"Ayşe"} {"name":"Eve"}`},
		{"null", `null`},
		{"too big", `{"name":"` + strings.Repeat("a", 2000) + `"}`},
	} {
		rec := httptest.NewRecorder() // calls the handler directly, no server needed
		greet(rec, httptest.NewRequest("POST", "/greet", strings.NewReader(c.body)))
		fmt.Printf("%-7s %d %s", c.label, rec.Code, rec.Body)
	}
}
