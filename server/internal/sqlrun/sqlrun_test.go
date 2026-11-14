// sqlrun_test.go: the guarantees of all three SQL modes, exercised against a
// real Postgres via TEST_DATABASE_URL (the local playground database, or
// playground-db in Docker mode).
//
//go:build integration

package sqlrun

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"stacklab/server/internal/runapi"
)

func testSession(t *testing.T) *Session {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	s, err := Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return s
}

// schemaExists asks from a fresh connection, so it sees only committed state.
func schemaExists(t *testing.T, schema string) bool {
	t.Helper()
	s := testSession(t)
	r := s.exec(context.Background(), "SELECT count(*) FROM pg_namespace WHERE nspname = '"+schema+"'")
	if r.Error != nil {
		t.Fatalf("query schema: %v", r.Error.Message)
	}
	return *r.Rows[0][0] != "0"
}

func TestTransactionalRunRollsBack(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{
		Setup:  "CREATE TABLE t (id int); INSERT INTO t VALUES (1),(2);",
		Script: "SELECT count(*) FROM t;",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Statements) != 3 {
		t.Fatalf("want 3 statement results, got %d", len(res.Statements))
	}
	last := res.Statements[len(res.Statements)-1]
	if last.Command != "SELECT 1" || len(last.Rows) != 1 {
		t.Fatalf("unexpected final statement: %+v", last)
	}
	if !strings.HasPrefix(res.Schema, "run_") {
		t.Fatalf("schema name shape: %q", res.Schema)
	}
	if schemaExists(t, res.Schema) {
		t.Fatalf("schema %s persisted after a transactional run", res.Schema)
	}
}

func TestRunStatementErrorStopsPhase(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{
		Script: "SELECT 1; SELECT * FROM missing_table; SELECT 2;",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Statements) != 2 {
		t.Fatalf("statement after the failing one must not run: %+v", res)
	}
	serr := res.Statements[1].Error
	if serr == nil || serr.Code != "42P01" {
		t.Fatalf("want SQLSTATE 42P01, got %+v", serr)
	}
}

// NULL and the empty string are different values; lessons on NULL logic depend on seeing that.
func TestNullIsNotEmptyString(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{Script: "SELECT NULL::text AS a, '' AS b;"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	row := res.Statements[0].Rows[0]
	if row[0] != nil || row[1] == nil || *row[1] != "" {
		t.Fatalf("want [nil, \"\"], got %v", row)
	}
	if got := Text(res.Statements); !strings.Contains(got, "NULL |") {
		t.Fatalf("text must spell NULL out, got:\n%s", got)
	}
}

func TestNoticesBelongToTheirStatement(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{
		Script: "DO $$ BEGIN RAISE NOTICE 'hello; from plpgsql'; END $$; SELECT 1;",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Statements) != 2 {
		t.Fatalf("the ';' inside $$ must not split the DO block: %d statements", len(res.Statements))
	}
	if n := res.Statements[0].Notices; len(n) != 1 || n[0] != "NOTICE: hello; from plpgsql" {
		t.Fatalf("notices = %q", n)
	}
	if len(res.Statements[1].Notices) != 0 {
		t.Fatalf("notice leaked into the next statement: %q", res.Statements[1].Notices)
	}
}

// SQL-standard routine bodies hold semicolons. The splitter used to cut them
// ("syntax error at end of input"); now each CREATE is one statement, and
// the routines can be called.
func TestBeginAtomicRoutinesAreCreatedAndCallable(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{
		Setup: "CREATE TABLE log (score int);",
		Script: `CREATE FUNCTION grade(score int) RETURNS text LANGUAGE sql
BEGIN ATOMIC
  INSERT INTO log VALUES (score);
  SELECT CASE WHEN score >= 50 THEN 'geçti' ELSE 'kaldı' END;
END;
CREATE OR REPLACE PROCEDURE keep_passes() LANGUAGE sql
BEGIN ATOMIC
  DELETE FROM log WHERE score < 50;
  INSERT INTO log VALUES (100);
END;
SELECT grade(72) AS a, grade(31) AS b;
CALL keep_passes();
SELECT string_agg(score::text, ',' ORDER BY score) FROM log;`,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, s := range res.Statements {
		if s.Error != nil {
			t.Fatalf("statement failed: %+v", s.Error)
		}
	}
	// setup, CREATE FUNCTION, CREATE PROCEDURE, SELECT, CALL, SELECT
	if n := len(res.Statements); n != 6 {
		t.Fatalf("want 6 statement results, got %d", n)
	}
	if row := res.Statements[3].Rows[0]; *row[0] != "geçti" || *row[1] != "kaldı" {
		t.Fatalf("grade() = %q, %q", *row[0], *row[1])
	}
	if got := *res.Statements[5].Rows[0][0]; got != "72,100" {
		t.Fatalf("log after the procedure = %s, want 72,100", got)
	}
}

// sql-nontx: statements that refuse a transaction block work, and the schema
// is gone afterwards even though everything committed.
func TestNonTxRunsVacuumAndDropsSchema(t *testing.T) {
	res, err := RunNonTx(context.Background(), testSession(t), Request{
		Setup:  "CREATE TABLE t (id int); INSERT INTO t SELECT generate_series(1, 100);",
		Script: "DELETE FROM t WHERE id > 10; VACUUM t; SELECT count(*) FROM t;",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, s := range res.Statements {
		if s.Error != nil {
			t.Fatalf("statement failed: %+v", s.Error)
		}
	}
	if got := *res.Statements[len(res.Statements)-1].Rows[0][0]; got != "10" {
		t.Fatalf("count = %s, want 10", got)
	}
	if schemaExists(t, res.Schema) {
		t.Fatalf("schema %s survived a non-transactional run", res.Schema)
	}
}

func TestStaleSchemasAreDropped(t *testing.T) {
	s := testSession(t)
	stale := "run_1000000000000000000_deadbeef" // 2001: far older than staleAfter
	if r := s.exec(context.Background(), "CREATE SCHEMA "+stale); r.Error != nil {
		t.Fatalf("create: %v", r.Error.Message)
	}
	t.Cleanup(func() { _ = s.quiet(context.Background(), "DROP SCHEMA IF EXISTS "+stale) })
	dropStaleSchemas(context.Background(), s)
	if schemaExists(t, stale) {
		t.Fatal("stale schema survived")
	}
}

// A script's COMMIT used to end the sandbox transaction: the run's schema was
// committed, SET LOCAL search_path was gone, and later statements wrote
// outside the sandbox and persisted.
func TestRunRefusesCommitAndNothingPersists(t *testing.T) {
	res, err := Run(context.Background(), testSession(t), Request{
		Setup:  "CREATE TABLE t (id int);",
		Script: "INSERT INTO t VALUES (1);\nCOMMIT;\nCREATE TABLE after_commit (id int);",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if n := len(res.Statements); n != 3 {
		t.Fatalf("want 3 statement results (setup, insert, refused COMMIT), got %d", n)
	}
	refusedCommit := res.Statements[2]
	if refusedCommit.Error == nil || !strings.Contains(refusedCommit.Error.Message, "refused") {
		t.Fatalf("COMMIT was not refused: %+v", refusedCommit)
	}
	if schemaExists(t, res.Schema) {
		t.Fatalf("schema %s persisted after a refused COMMIT", res.Schema)
	}
}

// The safety net behind the keyword check: a statement that leaves the
// session outside a transaction stops the phase.
func TestRunPhaseStopsWhenTheTransactionIsGone(t *testing.T) {
	s := testSession(t)
	res := &runapi.SQLResult{}
	if runPhase(context.Background(), s, "SELECT 1; SELECT 2;", res, true) {
		t.Fatal("phase succeeded outside a transaction")
	}
	if len(res.Statements) != 1 || res.Statements[0].Error != escapedSandbox {
		t.Fatalf("want one statement stopped by the safety net, got %+v", res.Statements)
	}
}

// COPY ... FROM STDIN used to hang until the run budget ran out.
func TestCopyFromStdinIsRefusedQuickly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := Run(ctx, testSession(t), Request{Setup: "CREATE TABLE t (a int);", Script: "COPY t FROM STDIN;"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	last := res.Statements[len(res.Statements)-1]
	if last.Error == nil || !strings.Contains(last.Error.Message, "COPY ... FROM STDIN is refused") {
		t.Fatalf("COPY FROM STDIN was not refused: %+v", last)
	}
	if ctx.Err() != nil {
		t.Fatal("the refusal took the whole timeout")
	}
}
