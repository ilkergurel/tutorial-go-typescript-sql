package main

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
)

type signup struct {
	Name string `json:"name"`
}

// strictV1 is the usual hardening of encoding/json: unknown fields fail.
func strictV1(body string) (signup, error) {
	var s signup
	dec := jsonv1.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	err := dec.Decode(&s)
	return s, err
}

// strictV2 needs one option; the rest is v2's default behavior.
func strictV2(body string) (signup, error) {
	var s signup
	err := json.UnmarshalRead(strings.NewReader(body), &s, json.RejectUnknownMembers(true))
	var sem *json.SemanticError
	if errors.As(err, &sem) { // its Error() text varies on purpose: use the fields
		err = fmt.Errorf("%v at %s", sem.Err, sem.JSONPointer)
	}
	return s, err
}

func main() {
	for _, c := range []struct{ label, body string }{
		{"duplicate name", `{"name":"Ayşe","name":"Mallory"}`},
		{"other case", `{"NAME":"Mallory"}`},
		{"Windows-1254 ş", "{\"name\":\"Ay\xfee\"}"}, // 0xFE is ş in Windows-1254
		{"trailing data", `{"name":"Ayşe"} {"name":"Eve"}`},
		{"unknown member", `{"name":"Ayşe","admin":true}`},
	} {
		fmt.Println(c.label)
		for _, try := range []struct {
			version string
			decode  func(string) (signup, error)
		}{{"v1", strictV1}, {"v2", strictV2}} {
			if s, err := try.decode(c.body); err != nil {
				fmt.Printf("  %s rejected: %v\n", try.version, err)
			} else {
				fmt.Printf("  %s accepted: name=%q\n", try.version, s.Name)
			}
		}
	}
}
