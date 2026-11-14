// LEARN(inside.codebase.contract): Go struct tags are the run API's contract; nil slices and lax decoding are where it leaks.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Abridged from server/internal/runapi/runapi.go: same names, same tags.
type Diagnostic struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type Result struct {
	Stdout      string       `json:"stdout"`
	ExitCode    int          `json:"exitCode"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	SQL         *SQLResult   `json:"sql,omitempty"`
}

type SQLResult struct {
	Schema string `json:"schema"`
}

type Request struct {
	Runtime  string `json:"runtime"`
	TimeoutS int    `json:"timeoutS,omitempty"`
}

// gorunResult stands in for gorun.Result, which starts with Diagnostics: []Diagnostic{}.
type gorunResult struct {
	Stdout      string
	ExitCode    int
	Diagnostics []Diagnostic
}

// fromGorun converts the way runexec.fromGorun does: append onto a nil slice.
func fromGorun(res gorunResult) Result {
	out := Result{Stdout: res.Stdout, ExitCode: res.ExitCode}
	for _, d := range res.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, d)
	}
	return out
}

func show(label string, v any) {
	b, _ := json.Marshal(v)
	fmt.Printf("%-8s %s\n", label, b)
}

func main() {
	show("go ok", fromGorun(gorunResult{Stdout: "hi\n", Diagnostics: []Diagnostic{}}))
	show("go err", fromGorun(gorunResult{ExitCode: 1, Diagnostics: []Diagnostic{{3, "undefined: x"}}}))
	show("sql", Result{SQL: &SQLResult{Schema: "run_1"}})
	show("fixed", Result{Stdout: "hi\n", Diagnostics: []Diagnostic{}})

	body := `{"runtime":"go","timeoutSecs":15}` // a typo: timeoutSecs
	var lax Request
	err := json.NewDecoder(strings.NewReader(body)).Decode(&lax)
	fmt.Printf("lax:     %+v, err=%v\n", lax, err)

	strict := json.NewDecoder(strings.NewReader(body))
	strict.DisallowUnknownFields()
	fmt.Println("strict:", strict.Decode(&Request{}))
}
