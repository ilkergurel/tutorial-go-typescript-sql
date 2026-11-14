// LEARN(inside.codebase.testing-pyramid): the base of the pyramid: pure functions, a table of cases, no I/O, milliseconds per run.
package main

import (
	"slices"
	"strings"
	"testing"
)

// splitStatements is a miniature of sqlrun's splitter: split on ';' outside
// single-quoted strings and -- comments. The real one also knows dollar
// quotes, escape strings (E'...'), block comments and quoted identifiers.
func splitStatements(script string) []string {
	var out []string
	var cur strings.Builder
	inString, inComment := false, false
	for _, r := range script { // runes, not bytes: ş and İ are two bytes each
		switch {
		case inComment:
			inComment = r != '\n'
		case r == '\'':
			inString = !inString // '' inside a string toggles twice: still inside
		case !inString && r == '-' && strings.HasSuffix(cur.String(), "-"):
			inComment = true
		case !inString && r == ';':
			if s := strings.TrimSpace(cur.String()); s != "" {
				out = append(out, s)
			}
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name, script string
		want         []string
	}{
		{"two statements", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"semicolon in a string", "SELECT 'a;b'; SELECT 2", []string{"SELECT 'a;b'", "SELECT 2"}},
		{"doubled quote", "SELECT 'it''s; fine';", []string{"SELECT 'it''s; fine'"}},
		{"semicolon in a comment", "SELECT 1 -- a; b\n;", []string{"SELECT 1 -- a; b"}},
		{"Turkish identifier", "SELECT 1 AS şehir; SELECT 'İ'", []string{"SELECT 1 AS şehir", "SELECT 'İ'"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitStatements(tc.script); !slices.Equal(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}
