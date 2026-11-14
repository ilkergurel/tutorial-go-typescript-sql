package main

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
)

type Product struct {
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	Stock int      `json:"stock,omitempty"` // in v2, omitzero is what drops a 0
}

// verdict names what v2 rejected, using the sentinel errors and types rather
// than the message text, which the docs say may change.
func verdict(err error) string {
	var syntax *jsontext.SyntacticError
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, jsontext.ErrDuplicateName):
		return "error: duplicate member name"
	case errors.Is(err, json.ErrUnknownName):
		return "error: unknown member"
	case errors.As(err, &syntax):
		return "syntax error at " + string(syntax.JSONPointer)
	}
	return "error"
}

func main() {
	v1, _ := jsonv1.Marshal(Product{Name: "çay"})
	v2, _ := json.Marshal(Product{Name: "çay"})
	fmt.Println("marshal v1:", string(v1))
	fmt.Println("marshal v2:", string(v2))

	for _, in := range []string{
		`{"name":"çay","name":"kahve"}`, // the same member twice
		"{\"name\":\"\xddzmir\"}",       // a windows-1254 byte, not UTF-8
		`{"name":"su","price":5}`,       // a member Product doesn't have
	} {
		var a, b Product
		err := jsonv1.Unmarshal([]byte(in), &a)
		fmt.Printf("%q\n  v1: %q %v\n", in, a.Name, err)
		err = json.Unmarshal([]byte(in), &b, json.RejectUnknownMembers(true))
		fmt.Printf("  v2 + RejectUnknownMembers: %s\n", verdict(err))
	}

	counts := map[string]int{"üç": 3, "bir": 1, "iki": 2}
	out, err := json.Marshal(counts, json.Deterministic(true)) // v2 doesn't sort maps by default
	fmt.Println(string(out), err)
}
