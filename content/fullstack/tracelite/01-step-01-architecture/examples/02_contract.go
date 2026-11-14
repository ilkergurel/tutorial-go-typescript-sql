package main

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"
	"uuid"
)

// The Requirement schema of openapi.yaml, abridged to six members. All of them are
// required, so this list is also every member the spec documents.
var (
	required = []string{"id", "parentId", "code", "title", "description", "status"}
	enums    = map[string][]any{"status": {"draft", "approved", "obsolete"}}
)

// conform encodes v exactly as a handler would and lists where it breaks the schema.
func conform(v any) (problems []string) {
	raw, _ := json.Marshal(v)
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	for _, name := range required {
		if _, ok := obj[name]; !ok {
			problems = append(problems, "missing "+name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(obj)) {
		if !slices.Contains(required, name) {
			problems = append(problems, "undocumented "+name)
		} else if enum := enums[name]; enum != nil && !slices.Contains(enum, obj[name]) {
			problems = append(problems, fmt.Sprintf("%s %q is not one of %v", name, obj[name], enum))
		}
	}
	return problems
}

// Requirement is internal/httpapi's wire type, kept by hand in step with the spec.
type Requirement struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parentId"`
	Code        string     `json:"code"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
}

// The same type after two tag edits that each look harmless in a Go review.
type editedRequirement struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id"` // the column's name, from habit
	Code        string     `json:"code"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"` // "" now disappears
	Status      string     `json:"status"`
}

func main() {
	good := Requirement{uuid.NewV7(), nil, "AUTH", "Kimlik doğrulama", "", "draft"}
	edited := editedRequirement(good)
	edited.Status = "Approved" // a display label written into the field
	for _, v := range []any{good, edited} {
		problems := conform(v)
		fmt.Printf("%T: %d problem(s)\n", v, len(problems))
		for _, p := range problems {
			fmt.Println("  " + p)
		}
	}
}
