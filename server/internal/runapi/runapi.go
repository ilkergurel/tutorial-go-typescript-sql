// Package runapi defines the wire format shared by the runner service and the
// backend's /api/run proxy. One package keeps both sides' JSON in lockstep.
// The frontend mirrors these shapes by hand in src/features/runner/types.ts
// and trusts the response with a cast (ServerRunner in CodeRunner.ts):
// nothing checks the mirror against this file, at build time or at run time.
// That is why Result.MarshalJSON makes the Go side match the mirror's plain
// arrays: an empty list goes out as [], never null.
package runapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

// Runtime selects the execution engine. ts-browser never reaches the runner —
// the browser executes it locally (ADR-0003) — but it shares this enum so the
// RunResult shape is uniform end to end.
type Runtime string

const (
	RuntimeGo           Runtime = "go"
	RuntimeSQL          Runtime = "sql"
	RuntimeTSNode       Runtime = "ts-node"
	RuntimeTSBrowser    Runtime = "ts-browser"
	RuntimeGoTest       Runtime = "go-test"
	RuntimeGoServer     Runtime = "go-server"
	RuntimeSQLNonTx     Runtime = "sql-nontx"
	RuntimeSQLMultiSess Runtime = "sql-multisession"
	// RuntimeTSTest runs Vitest or Playwright tests (local mode only).
	RuntimeTSTest Runtime = "ts-test"
)

// Request is POST /run and POST /api/run body.
type Request struct {
	Runtime  Runtime           `json:"runtime"`
	Example  string            `json:"example"`
	Files    map[string]string `json:"files"`
	Fixtures map[string]string `json:"fixtures,omitempty"`
	Stdin    string            `json:"stdin,omitempty"`
	TimeoutS int               `json:"timeoutS,omitempty"`
	Vet      bool              `json:"vet,omitempty"`
	// SQL-only fields.
	Setup string `json:"setup,omitempty"`

	// Lesson/example identity for run history; ignored by the runner itself.
	LessonID string `json:"lessonId,omitempty"`
}

// Result is the single typed shape every runtime returns.
type Result struct {
	Stdout      string        `json:"stdout"`
	Stderr      string        `json:"stderr"`
	ExitCode    int           `json:"exitCode"`
	DurationMs  int64         `json:"durationMs"`
	TimedOut    bool          `json:"timedOut"`
	Truncated   bool          `json:"truncated"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Files       []WrittenFile `json:"files"`
	// SQL runtimes: statements in script order (for sql-multisession, the
	// setup statements). Stdout then holds the same results as text, which is
	// what expected/*.txt files compare against.
	SQL *SQLResult `json:"sql,omitempty"`
	// sql-multisession: every step in script order, each tagged with its
	// session. One list (not one per session) keeps the interleaving.
	Steps []StepResult `json:"steps,omitempty"`
}

// MarshalJSON writes every empty list as [], never null. encoding/json writes
// a nil slice as null, producers build these slices by appending (so a clean
// run's diagnostics start out nil), and the TypeScript mirror
// (src/features/runner/types.ts) declares plain arrays that the browser
// trusts with a cast. This method is the one place that keeps the two sides
// in step: the backend's reply, cmd/runner's reply and the stored run
// history all marshal a Result. Steps keeps omitempty, so an empty Steps is
// still absent, which is what the mirror's steps?: says.
//
// It copies before filling in, so marshaling never modifies the caller's
// Result (the handler and runstore marshal the same value).
func (r Result) MarshalJSON() ([]byte, error) {
	type plain Result // same fields, no MarshalJSON: stops the recursion
	p := plain(r)
	p.Diagnostics = orEmpty(r.Diagnostics)
	p.Files = orEmpty(r.Files)
	if r.SQL != nil {
		sql := *r.SQL
		sql.Statements = make([]StatementResult, len(r.SQL.Statements))
		for i, st := range r.SQL.Statements {
			sql.Statements[i] = st.withLists()
		}
		p.SQL = &sql
	}
	if len(r.Steps) > 0 {
		p.Steps = make([]StepResult, len(r.Steps))
		for i, st := range r.Steps {
			st.StatementResult = st.StatementResult.withLists()
			p.Steps[i] = st
		}
	}
	return json.Marshal(p)
}

// withLists is s with its nil lists replaced by empty ones. Not a
// MarshalJSON on StatementResult: StepResult embeds it, and an embedded
// type's MarshalJSON is promoted, so a step would marshal as its statement
// alone and lose step, session, sql, waited and finishedOrder.
func (s StatementResult) withLists() StatementResult {
	s.Columns = orEmpty(s.Columns)
	s.Rows = orEmpty(s.Rows)
	s.Notices = orEmpty(s.Notices)
	return s
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// StepResult is one tagged statement from a multisession script.
type StepResult struct {
	Step    int    `json:"step"`    // 1-based position in the script
	Session string `json:"session"` // "A" or "B"
	SQL     string `json:"sql"`
	// Waited: still running when the next step was sent (blocked on a lock,
	// or just slow). FinishedOrder says when its result arrived.
	Waited        bool `json:"waited"`
	FinishedOrder int  `json:"finishedOrder"`
	StatementResult
}

type Diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
}

type WrittenFile struct {
	Path    string `json:"path"`
	Size    int    `json:"size"`
	Preview string `json:"preview"`
}

// SQLResult carries per-statement outcomes for the sql runtime.
type SQLResult struct {
	Schema     string            `json:"schema"`
	Statements []StatementResult `json:"statements"`
}

// StatementResult is one statement's outcome. Cells are Postgres' text form;
// a nil cell is SQL NULL, which must never look like an empty string.
type StatementResult struct {
	Command      string        `json:"command"` // command tag, e.g. INSERT 0 3
	Columns      []SQLColumn   `json:"columns"`
	Rows         [][]*string   `json:"rows"`
	RowsAffected int64         `json:"rowsAffected"`
	Notices      []string      `json:"notices"` // e.g. "NOTICE: ..." from RAISE NOTICE
	Error        *SQLStmtError `json:"error,omitempty"`
	DurationMs   int64         `json:"durationMs"`
}

type SQLColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SQLStmtError carries the server's error fields; Pos is the 1-based
// character position the server blamed (0 when it didn't say).
type SQLStmtError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Detail  string `json:"detail,omitempty"`
	Hint    string `json:"hint,omitempty"`
	Pos     int32  `json:"pos"`
}

// Error is the runner's non-200 body (RFC 9457 subset the backend re-maps).
type Error struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// Validate enforces the boundaries the runner trusts the backend to check.
func (r Request) Validate() error {
	if r.Runtime == "" {
		return errors.New("runtime is required")
	}
	if len(r.Files) == 0 && r.Runtime != RuntimeSQL {
		return errors.New("files are required")
	}
	if len(r.Files) > 32 {
		return errors.New("too many files")
	}
	var total int
	for name, body := range r.Files {
		if name == "" || len(name) > 128 {
			return errors.New("invalid file name")
		}
		total += len(name) + len(body)
	}
	for name, body := range r.Fixtures {
		if name == "" || len(name) > 128 {
			return errors.New("invalid fixture name")
		}
		total += len(name) + len(body)
	}
	total += len(r.Stdin) + len(r.Setup)
	if total > 1<<20 { // 1 MiB of combined sources
		return errors.New("request too large")
	}
	if r.TimeoutS < 0 || r.TimeoutS > 15 {
		return errors.New("timeoutS out of range")
	}
	return nil
}

// Encode marshals compactly; the single call site justifies no streaming.
func Encode(v any) ([]byte, error) { return json.Marshal(v) }

// Decode is strict about unknown fields so wire drift fails loudly in dev.
func Decode(data []byte, v any) error {
	dec := json.NewDecoder(bytesReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
