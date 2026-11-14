package sqlrun

import "strings"

// splitStatements cuts a script at top-level semicolons, as psql does. A ';'
// inside a string (doubled-quote and E-string backslash escapes included), a
// quoted identifier, a comment (block comments nest in Postgres) or a
// dollar-quoted body ($$ ... $$, $fn$ ... $fn$: every PL/pgSQL function) does
// not end a statement, and neither does one inside the BEGIN ATOMIC ... END
// body of a SQL-standard function or procedure (routineBody). Pieces holding
// only whitespace and comments are dropped: the server would answer them with
// an empty result that means nothing to a learner.
//
// Rejected alternative: sending the whole script as one simple-protocol
// message. The server would run it, but an error would hide which statement
// failed, and results could not be attributed per statement.
func splitStatements(script string) []string {
	var stmts []string
	var body routineBody
	n := len(script)
	start, i := 0, 0
	for i < n {
		c := script[i]
		switch {
		case isIdentStart(c) && (i == 0 || !isIdentByte(script[i-1])):
			j := i + 1
			for j < n && isIdentByte(script[j]) {
				j++
			}
			body.word(script[i:j])
			i = j
		case c == '(':
			body.parens++
			i++
		case c == ')':
			body.parens = max(body.parens-1, 0)
			i++
		case c == ';' && body.depth > 0:
			i++ // inside BEGIN ATOMIC ... END: part of the routine's body
		case c == '-' && i+1 < n && script[i+1] == '-':
			if j := strings.IndexByte(script[i:], '\n'); j >= 0 {
				i += j + 1
			} else {
				i = n
			}
		case c == '/' && i+1 < n && script[i+1] == '*':
			i = skipBlockComment(script, i)
		case c == '\'':
			// E'...' strings use backslash escapes; standard strings don't.
			escapes := i > 0 && (script[i-1] == 'E' || script[i-1] == 'e') && (i < 2 || !isIdentByte(script[i-2]))
			i = skipQuoted(script, i, '\'', escapes)
		case c == '"':
			i = skipQuoted(script, i, '"', false)
		case c == '$' && (i == 0 || !isIdentByte(script[i-1])):
			tag := dollarTag(script[i:])
			if tag == "" {
				i++ // a positional parameter like $1, or a lone $
				continue
			}
			if end := strings.Index(script[i+len(tag):], tag); end >= 0 {
				i += len(tag) + end + len(tag)
			} else {
				i = n // unterminated: the server reports it
			}
		case c == ';':
			stmts = appendStatement(stmts, script[start:i])
			body = routineBody{}
			i++
			start = i
		default:
			i++
		}
	}
	return appendStatement(stmts, script[start:])
}

// routineBody follows psql's rule for SQL-standard routine bodies
// (psqlscan_track_identifier in src/fe_utils/psqlscan.l, PostgreSQL 14+):
// only in a statement that starts CREATE [OR REPLACE] FUNCTION or PROCEDURE,
// count BEGIN and END outside parentheses, and CASE once inside a BEGIN
// (CASE also ends with END). While the count is above zero, ';' is part of
// the body. Any other statement starting with BEGIN (a transaction, as in
// "BEGIN;" or "BEGIN TRANSACTION;") is not counted, so it still ends at its
// ';'. Like psql, it reads unquoted words only; strings, quoted identifiers,
// comments and dollar-quoted bodies are skipped before it sees them.
//
// One difference: psql doesn't end a statement at a ';' inside parentheses
// either; this splitter does, as it did before, and resets the count there.
type routineBody struct {
	first  []string // the statement's first unquoted words outside parentheses, lower case, at most four
	parens int
	depth  int // open BEGIN (and CASE inside one) without their END
}

// word takes one unquoted identifier or keyword of the current statement.
func (b *routineBody) word(w string) {
	if b.parens > 0 {
		return // psql ignores words inside parentheses
	}
	if len(b.first) < 4 {
		b.first = append(b.first, strings.ToLower(w))
	}
	if !b.inRoutine() {
		return
	}
	switch {
	case strings.EqualFold(w, "begin"):
		b.depth++
	case strings.EqualFold(w, "case") && b.depth > 0:
		b.depth++
	case strings.EqualFold(w, "end") && b.depth > 0:
		b.depth--
	}
}

// inRoutine reports whether the statement starts CREATE [OR REPLACE]
// FUNCTION or PROCEDURE (psqlscan_is_create_routine).
func (b *routineBody) inRoutine() bool {
	f := b.first
	routine := func(w string) bool { return w == "function" || w == "procedure" }
	return len(f) >= 2 && f[0] == "create" &&
		(routine(f[1]) || len(f) == 4 && f[1] == "or" && f[2] == "replace" && routine(f[3]))
}

// skipBlockComment returns the index just past the comment opening at i.
func skipBlockComment(s string, i int) int {
	depth := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "/*"):
			depth++
			i += 2
		case strings.HasPrefix(s[i:], "*/"):
			depth--
			i += 2
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return i
}

// skipQuoted returns the index just past the quoted token opening at i. A
// doubled quote is an escaped quote in both strings and identifiers.
func skipQuoted(s string, i int, quote byte, backslashEscapes bool) int {
	for i++; i < len(s); i++ {
		switch {
		case backslashEscapes && s[i] == '\\':
			i++
		case s[i] == quote && i+1 < len(s) && s[i+1] == quote:
			i++
		case s[i] == quote:
			return i + 1
		}
	}
	return i
}

// dollarTag returns the opening "$tag$" at the start of s, or "" when s does
// not start one. Tags follow identifier rules, so "$1" is not a tag.
func dollarTag(s string) string {
	j := 1
	for j < len(s) && s[j] != '$' && isIdentByte(s[j]) && (j > 1 || s[j] < '0' || s[j] > '9') {
		j++
	}
	if j < len(s) && s[j] == '$' {
		return s[:j+1]
	}
	return ""
}

// isIdentByte reports bytes that can continue an identifier. Bytes >= 0x80
// count, so a dollar sign after a Turkish letter is still part of a name.
func isIdentByte(b byte) bool {
	return b == '_' || b == '$' || b >= 0x80 ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// isIdentStart reports bytes that can start an identifier: no digit or $.
func isIdentStart(b byte) bool {
	return b == '_' || b >= 0x80 || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func appendStatement(stmts []string, piece string) []string {
	if piece = strings.TrimSpace(piece); hasCode(piece) {
		stmts = append(stmts, piece)
	}
	return stmts
}

// hasCode reports whether piece holds anything besides comments.
func hasCode(piece string) bool {
	return codeStart(piece) != ""
}

// codeStart returns piece from its first token that isn't a comment, so a
// statement's leading keyword can be read even after "-- why" lines.
func codeStart(piece string) string {
	for {
		piece = strings.TrimSpace(piece)
		switch {
		case strings.HasPrefix(piece, "--"):
			j := strings.IndexByte(piece, '\n')
			if j < 0 {
				return ""
			}
			piece = piece[j+1:]
		case strings.HasPrefix(piece, "/*"):
			piece = piece[skipBlockComment(piece, 0):]
		default:
			return piece
		}
	}
}
