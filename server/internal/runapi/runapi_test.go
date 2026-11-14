package runapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// The handler encodes a *Result with json.Encoder; marshal it the same way.
func encode(t *testing.T, r *Result) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(r); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return b.String()
}

// A clean Go run has no diagnostics and wrote no files. Its slices are nil
// (runexec.fromGorun appends onto nil), and types.ts declares both as arrays.
func TestCleanRunSendsEmptyArrays(t *testing.T) {
	got := encode(t, &Result{Stdout: "hi\n", DurationMs: 12})
	for _, want := range []string{`"diagnostics":[]`, `"files":[]`} {
		if !strings.Contains(got, want) {
			t.Errorf("clean run lacks %s: %s", want, got)
		}
	}
	for _, absent := range []string{"null", `"sql"`, `"steps"`} {
		if strings.Contains(got, absent) {
			t.Errorf("clean run contains %s: %s", absent, got)
		}
	}
}

// Contract guard for every list in the response, SQL included: a Result
// whose every slice is nil (one zero statement and one zero step, so the
// nested lists exist) must not contain null anywhere. A new slice field
// added without normalization fails here.
func TestNoListMarshalsAsNull(t *testing.T) {
	got := encode(t, &Result{
		SQL:   &SQLResult{Schema: "run_1", Statements: []StatementResult{{Command: "CREATE TABLE"}}},
		Steps: []StepResult{{Step: 1, Session: "A", SQL: "BEGIN", FinishedOrder: 1}},
	})
	if strings.Contains(got, "null") {
		t.Fatalf("null on the wire: %s", got)
	}
	for _, want := range []string{
		`"columns":[],"rows":[],"rowsAffected":0,"notices":[]`,
		// The step keeps its own fields next to the statement's; a promoted
		// MarshalJSON from the embedded StatementResult would drop these.
		`"step":1,"session":"A","sql":"BEGIN","waited":false,"finishedOrder":1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %s: %s", want, got)
		}
	}

	if got := encode(t, &Result{SQL: &SQLResult{Schema: "run_2"}}); !strings.Contains(got, `"statements":[]`) {
		t.Errorf("empty SQL script: %s", got)
	}
}

// SQL NULL is the one null the wire must keep: a nil cell is NULL, and a
// pointer to "" is an empty text value. Normalizing lists must not touch cells.
func TestSQLNullCellsStayNull(t *testing.T) {
	empty := ""
	got := encode(t, &Result{SQL: &SQLResult{Statements: []StatementResult{{
		Columns: []SQLColumn{{Name: "a", Type: "text"}, {Name: "b", Type: "text"}},
		Rows:    [][]*string{{nil, &empty}},
	}}}})
	if !strings.Contains(got, `"rows":[[null,""]]`) {
		t.Fatalf("cells changed: %s", got)
	}
}

// Marshaling fills in copies: the handler and runstore marshal the same
// value, and neither should see it change underneath.
func TestMarshalLeavesTheResultAlone(t *testing.T) {
	r := &Result{
		SQL:   &SQLResult{Statements: []StatementResult{{}}},
		Steps: []StepResult{{}},
	}
	_ = encode(t, r)
	if r.Diagnostics != nil || r.Files != nil || r.SQL.Statements[0].Columns != nil || r.Steps[0].Notices != nil {
		t.Fatalf("marshal modified its input: %+v", r)
	}
}
