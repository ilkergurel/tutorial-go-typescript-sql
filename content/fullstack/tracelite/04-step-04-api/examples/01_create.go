package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"uuid"
)

type RequirementCreate struct { // what a client may send, and nothing else
	Code     string     `json:"code"`
	Title    string     `json:"title"`
	ParentID *uuid.UUID `json:"parentId"`
}
type Requirement struct { // abridged; step 1 checks the full type against the spec
	ID       uuid.UUID  `json:"id"`
	ParentID *uuid.UUID `json:"parentId"`
	Code     string     `json:"code"`
	Title    string     `json:"title"`
}
type Problem struct { // abridged: example 3 has the errors member
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
}

var ErrNotFound = errors.New("not found")
var errBadBody = errors.New("the body is not a RequirementCreate object")

func handleCreate(w http.ResponseWriter, r *http.Request) {
	if _, err := uuid.Parse(r.PathValue("projectId")); err != nil {
		writeError(w, ErrNotFound) // a malformed id names no resource
		return
	}
	var in RequirementCreate
	err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, 64<<10), &in, json.RejectUnknownMembers(true))
	if err != nil {
		writeError(w, errBadBody)
		return
	}
	created := Requirement{uuid.UUID{15: 1}, in.ParentID, in.Code, in.Title} // the service and the store would insert it
	w.Header().Set("Location", "/api/requirements/"+created.ID.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.MarshalWrite(w, created)
}

// writeError, abridged to the two errors this handler produces (example 3 has all of it).
func writeError(w http.ResponseWriter, err error) {
	p := Problem{Type: "about:blank", Status: http.StatusInternalServerError}
	switch {
	case errors.Is(err, errBadBody):
		p.Status = http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		p.Status = http.StatusNotFound
	}
	p.Title = http.StatusText(p.Status)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.MarshalWrite(w, p)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/projects/{projectId}/requirements", handleCreate)
	for _, req := range []struct{ project, body string }{
		{"01937b6e-0000-7000-8000-00000000000a", `{"code": "AUTH", "title": "Kimlik doğrulama"}`},
		{"01937b6e-0000-7000-8000-00000000000a", `{"code": "AUTH", "titel": "Kimlik doğrulama"}`},
		{"01937b6e-0000-7000-8000-00000000000a", `{"code": "AUTH", "Title": "Kimlik doğrulama"}`},
		{"42", `{"code": "AUTH", "title": "Kimlik doğrulama"}`},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/projects/"+req.project+"/requirements", strings.NewReader(req.body)))
		fmt.Println(rec.Code, rec.Header().Get("Location"), rec.Body)
	}
}
