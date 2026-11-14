package sqlrun

import (
	"slices"
	"testing"

	"stacklab/server/internal/runapi"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   []string
	}{
		{"comment and string semicolons", "SELECT 1; -- a ; comment\nSELECT 'a;b'; SELECT 2",
			[]string{"SELECT 1", "-- a ; comment\nSELECT 'a;b'", "SELECT 2"}},
		{"doubled quote", "SELECT 'it''s; fine'; SELECT 2;", []string{"SELECT 'it''s; fine'", "SELECT 2"}},
		{"E string backslash quote", `SELECT E'it\'s; fine'; SELECT 2;`, []string{`SELECT E'it\'s; fine'`, "SELECT 2"}},
		{"backslash is literal in a standard string", `SELECT 'C:\'; SELECT 2;`, []string{`SELECT 'C:\'`, "SELECT 2"}},
		{"quoted identifier", `SELECT 1 AS "a;b"; SELECT 2`, []string{`SELECT 1 AS "a;b"`, "SELECT 2"}},
		{"nested block comment", "SELECT /* a /* b; */ c; */ 1; SELECT 2", []string{"SELECT /* a /* b; */ c; */ 1", "SELECT 2"}},
		{"dollar-quoted body", "CREATE FUNCTION f() RETURNS int LANGUAGE plpgsql AS $$ BEGIN RETURN 1; END $$; SELECT f();",
			[]string{"CREATE FUNCTION f() RETURNS int LANGUAGE plpgsql AS $$ BEGIN RETURN 1; END $$", "SELECT f()"}},
		{"tagged dollar quote holding $$", "DO $fn$ BEGIN RAISE NOTICE '$$;'; END $fn$; SELECT 2;",
			[]string{"DO $fn$ BEGIN RAISE NOTICE '$$;'; END $fn$", "SELECT 2"}},
		{"positional parameter is not a tag", "PREPARE p AS SELECT $1::int; EXECUTE p(1);", []string{"PREPARE p AS SELECT $1::int", "EXECUTE p(1)"}},
		{"comment-only pieces dropped", "SELECT 1; -- trailing\n/* only a comment */;\n\n", []string{"SELECT 1"}},
		{"İ in an identifier before $", "SELECT 1 AS şehir$; SELECT 2", []string{"SELECT 1 AS şehir$", "SELECT 2"}},
		// SQL-standard routine bodies, by psql's rule (psqlscan.l).
		{"BEGIN ATOMIC body", "CREATE FUNCTION f() RETURNS int LANGUAGE sql BEGIN ATOMIC SELECT 1; SELECT 2; END; SELECT f();",
			[]string{"CREATE FUNCTION f() RETURNS int LANGUAGE sql BEGIN ATOMIC SELECT 1; SELECT 2; END", "SELECT f()"}},
		{"create or replace procedure", "create or replace procedure p() language sql\nbegin atomic\n  insert into t values (1);\n  insert into t values (2);\nend;\ncall p();",
			[]string{"create or replace procedure p() language sql\nbegin atomic\n  insert into t values (1);\n  insert into t values (2);\nend", "call p()"}},
		{"CASE ... END inside the body", "CREATE FUNCTION g(x int) RETURNS text LANGUAGE sql BEGIN ATOMIC SELECT CASE WHEN x > 0 THEN 'artı' ELSE 'eksi' END; END; SELECT g(1);",
			[]string{"CREATE FUNCTION g(x int) RETURNS text LANGUAGE sql BEGIN ATOMIC SELECT CASE WHEN x > 0 THEN 'artı' ELSE 'eksi' END; END", "SELECT g(1)"}},
		{"unterminated body runs to the end", "CREATE PROCEDURE p() BEGIN ATOMIC SELECT 1; SELECT 2;",
			[]string{"CREATE PROCEDURE p() BEGIN ATOMIC SELECT 1; SELECT 2;"}},
		{"BEGIN starts a transaction", "BEGIN; SELECT 1; COMMIT;", []string{"BEGIN", "SELECT 1", "COMMIT"}},
		{"BEGIN TRANSACTION and START TRANSACTION", "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; END; START TRANSACTION; ROLLBACK;",
			[]string{"BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE", "END", "START TRANSACTION", "ROLLBACK"}},
		{"BEGIN only counts in CREATE FUNCTION or PROCEDURE", "CREATE VIEW v AS SELECT 1 AS begin; SELECT 2;", []string{"CREATE VIEW v AS SELECT 1 AS begin", "SELECT 2"}},
		{"words in parentheses don't count", "CREATE FUNCTION h(begin int) RETURNS int LANGUAGE sql RETURN 1; SELECT 2;",
			[]string{"CREATE FUNCTION h(begin int) RETURNS int LANGUAGE sql RETURN 1", "SELECT 2"}},
		{"quoted BEGIN doesn't count", `CREATE FUNCTION k() RETURNS text LANGUAGE sql RETURN 'begin'; SELECT "begin";`,
			[]string{`CREATE FUNCTION k() RETURNS text LANGUAGE sql RETURN 'begin'`, `SELECT "begin"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitStatements(tc.script); !slices.Equal(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestParseSessionScript(t *testing.T) {
	script := "-- header comment\n-- A: not a tag, a comment\n\nA: BEGIN;\nA: SELECT balance\n   FROM accounts;   -- 100\n\n-- B moves money\nB: UPDATE accounts SET balance = 0;\nB:COMMIT;\n"
	got := parseSessionScript(script)
	want := []MultiStep{
		{'A', "BEGIN;"},
		{'A', "SELECT balance\n   FROM accounts;   -- 100"},
		{'B', "UPDATE accounts SET balance = 0;"},
	}
	// "B:COMMIT;" has no space after the colon, so it is not a tag: it stays
	// part of B's step, which then holds two statements.
	want[2].SQL += "\nB:COMMIT;"
	if !slices.Equal(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func ptr(s string) *string { return &s }

func TestTextAlignsByCharactersAndSpellsNull(t *testing.T) {
	got := Text([]runapi.StatementResult{
		{Command: "CREATE TABLE"},
		{Command: "SELECT 2", Columns: []runapi.SQLColumn{{Name: "şehir"}, {Name: "n"}},
			Rows: [][]*string{{ptr("İstanbul"), ptr("1")}, {ptr("Ağrı"), nil}}},
		{Notices: []string{"NOTICE: hi"}, Command: "DO"},
		{Error: &runapi.SQLStmtError{Message: `relation "x" does not exist`, Code: "42P01", Detail: "pid 1234"}},
	})
	want := "CREATE TABLE\n" +
		"SELECT 2\n" +
		"    şehir    | n\n" +
		"    ---------+-----\n" +
		"    İstanbul | 1\n" +
		"    Ağrı     | NULL\n" +
		"DO\n" +
		"    NOTICE: hi\n" +
		"ERROR: relation \"x\" does not exist (SQLSTATE 42P01)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
