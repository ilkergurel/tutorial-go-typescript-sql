package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"strings"
	"unicode/utf8"
)

type signup struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type fieldError struct {
	Pointer string `json:"pointer"`
	Detail  string `json:"detail"`
}

// problem is an RFC 9457 problem details object; Errors is an extension.
type problem struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []fieldError `json:"errors,omitempty"`
}

// validate reports every broken rule at once, so a form can mark them all.
func (s signup) validate() (errs []fieldError) {
	if n := utf8.RuneCountInString(s.Name); n < 2 || n > 20 {
		errs = append(errs, fieldError{"#/name", fmt.Sprintf("must be 2-20 characters, got %d", n)})
	}
	if a, err := mail.ParseAddress(s.Email); err != nil || a.Address != s.Email {
		errs = append(errs, fieldError{"#/email", "must be a bare address like ad@example.com"})
	}
	return errs
}

func writeProblem(w http.ResponseWriter, p problem) {
	w.Header().Set("Content-Type", "application/problem+json") // before WriteHeader, or it is dropped
	w.WriteHeader(p.Status)                                    // status member and real status agree
	json.NewEncoder(w).Encode(p)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var s signup
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil { // example 1's decodeJSON in real code
		writeProblem(w, problem{Type: "about:blank", Title: "Bad Request",
			Status: http.StatusBadRequest, Detail: "malformed JSON"})
		return
	}
	if errs := s.validate(); len(errs) > 0 {
		writeProblem(w, problem{Type: "https://example.com/problems/validation",
			Title: "Your request is not valid.", Status: http.StatusUnprocessableEntity, Errors: errs})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func main() {
	for _, body := range []string{
		`{"name":"Gülşen Çağlayıcı","email":"gulsen@example.com"}`,
		`{"name":"Ö","email":"Gülşen <g@example.com>"}`,
		`{"name":`,
	} {
		rec := httptest.NewRecorder()
		createUser(rec, httptest.NewRequest("POST", "/users", strings.NewReader(body)))
		var pretty bytes.Buffer
		json.Indent(&pretty, rec.Body.Bytes(), "", "  ")
		fmt.Printf("%d %s\n%s", rec.Code, rec.Header().Get("Content-Type"), pretty.Bytes())
	}
}
