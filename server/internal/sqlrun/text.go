package sqlrun

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"stacklab/server/internal/runapi"
)

// Text renders SQL results as the plain text that expected/*.txt files hold,
// so the UI's "matches expected" check works for SQL like it does for Go. One
// block per statement: the command tag (or the error), then any notices and
// the result table, indented under it:
//
//	CREATE TABLE
//	SELECT 2
//	    name | email
//	    -----+----------------
//	    ada  | ada@example.com
//	    brin | NULL
//
// NULL prints as NULL so it never looks like an empty string. Columns are
// padded by characters, not bytes, so Turkish text lines up. Error DETAIL is
// left out: for deadlocks it names process IDs, which change on every run.
// The UI shows it from the structured result.
func Text(stmts []runapi.StatementResult) string {
	var b strings.Builder
	for _, sr := range stmts {
		writeStatement(&b, "", sr)
	}
	return b.String()
}

// Text renders a multisession run in the order things happened: each step's
// result when it arrived, and a "waiting" line for a step still running when
// the next one was sent (typically blocked on the other session's lock):
//
//	[3] B: UPDATE 1
//	[4] A: waiting
//	[5] B: COMMIT
//	[4] A: UPDATE 1
func (m MultiResult) Text() string {
	var b strings.Builder
	for _, sr := range m.Setup {
		if sr.Error != nil {
			writeStatement(&b, "setup: ", sr)
		}
	}
	for _, ev := range m.events {
		st := m.Steps[ev.step]
		prefix := fmt.Sprintf("[%d] %s: ", st.Step, st.Session)
		if ev.waiting {
			b.WriteString(prefix + "waiting\n")
			continue
		}
		writeStatement(&b, prefix, st.StatementResult)
	}
	return b.String()
}

const blockIndent = "    "

func writeStatement(b *strings.Builder, prefix string, sr runapi.StatementResult) {
	switch {
	case sr.Error != nil && sr.Error.Code != "":
		fmt.Fprintf(b, "%sERROR: %s (SQLSTATE %s)\n", prefix, sr.Error.Message, sr.Error.Code)
	case sr.Error != nil:
		fmt.Fprintf(b, "%sERROR: %s\n", prefix, sr.Error.Message)
	default:
		b.WriteString(prefix + sr.Command + "\n")
	}
	for _, n := range sr.Notices {
		b.WriteString(blockIndent + n + "\n")
	}
	if sr.Error == nil && len(sr.Columns) > 0 {
		writeTable(b, sr.Columns, sr.Rows)
	}
}

func writeTable(b *strings.Builder, cols []runapi.SQLColumn, rows [][]*string) {
	cells := make([][]string, 0, len(rows)+1)
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.Name
	}
	cells = append(cells, header)
	for _, row := range rows {
		line := make([]string, len(row))
		for i, v := range row {
			if v == nil {
				line[i] = "NULL"
			} else {
				// One result row stays one text line.
				line[i] = strings.ReplaceAll(*v, "\n", `\n`)
			}
		}
		cells = append(cells, line)
	}

	widths := make([]int, len(cols))
	for _, line := range cells {
		for i, cell := range line {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}
	writeRow := func(line []string) {
		var row strings.Builder
		for i, cell := range line {
			if i > 0 {
				row.WriteString(" | ")
			}
			row.WriteString(cell + strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell)))
		}
		b.WriteString(strings.TrimRight(blockIndent+row.String(), " ") + "\n")
	}

	writeRow(cells[0])
	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	b.WriteString(blockIndent + strings.Join(sep, "-+-") + "\n")
	for _, line := range cells[1:] {
		writeRow(line)
	}
}
