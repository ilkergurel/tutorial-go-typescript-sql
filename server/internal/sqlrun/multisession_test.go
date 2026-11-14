//go:build integration

package sqlrun

import (
	"context"
	"strings"
	"testing"
)

const accounts = "CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL); INSERT INTO accounts VALUES ('a', 100), ('b', 0);"

func runMulti(t *testing.T, script string) *MultiResult {
	t.Helper()
	res, err := RunMulti(context.Background(), testSession(t), testSession(t), accounts, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, sr := range res.Setup {
		if sr.Error != nil {
			t.Fatalf("setup failed: %+v", sr.Error)
		}
	}
	return res
}

// The isolation lesson: setup is visible to both sessions, and under READ
// COMMITTED A's second read sees B's commit.
func TestMultiNonRepeatableRead(t *testing.T) {
	res := runMulti(t, `
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a';
B: UPDATE accounts SET balance = balance - 40 WHERE id = 'a';
A: SELECT balance FROM accounts WHERE id = 'a';
A: COMMIT;
`)
	want := "[1] A: BEGIN\n" +
		"[2] A: SELECT 1\n    balance\n    -------\n    100\n" +
		"[3] B: UPDATE 1\n" +
		"[4] A: SELECT 1\n    balance\n    -------\n    60\n" +
		"[5] A: COMMIT\n"
	if got := res.Text(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if schemaExists(t, res.Schema) {
		t.Fatalf("schema %s survived", res.Schema)
	}
}

// "B waits until A commits": the blocked step is reported as waiting, the
// script carries on, and B's result arrives after A's COMMIT.
func TestMultiLockWaitResolvesOnCommit(t *testing.T) {
	res := runMulti(t, `
A: BEGIN;
A: SELECT balance FROM accounts WHERE id = 'a' FOR UPDATE;
B: UPDATE accounts SET balance = 0 WHERE id = 'a';
A: COMMIT;
`)
	got := res.Text()
	want := "[1] A: BEGIN\n" +
		"[2] A: SELECT 1\n    balance\n    -------\n    100\n" +
		"[3] B: waiting\n" +
		"[4] A: COMMIT\n" +
		"[3] B: UPDATE 1\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if !res.Steps[2].Waited || res.Steps[2].FinishedOrder != 4 {
		t.Fatalf("step 3: waited=%v finishedOrder=%d", res.Steps[2].Waited, res.Steps[2].FinishedOrder)
	}
}

// A real deadlock forms, and the first session to wait is always the victim
// (its deadlock check runs first). Three runs to catch timing flakiness.
func TestMultiDeadlockIsDeterministic(t *testing.T) {
	script := `
A: BEGIN;
B: BEGIN;
A: UPDATE accounts SET balance = 1 WHERE id = 'a';
B: UPDATE accounts SET balance = 2 WHERE id = 'b';
A: UPDATE accounts SET balance = 3 WHERE id = 'b';
B: UPDATE accounts SET balance = 4 WHERE id = 'a';
A: ROLLBACK;
B: COMMIT;
`
	want := "[1] A: BEGIN\n[2] B: BEGIN\n[3] A: UPDATE 1\n[4] B: UPDATE 1\n" +
		"[5] A: waiting\n[6] B: waiting\n" +
		"[5] A: ERROR: deadlock detected (SQLSTATE 40P01)\n" +
		"[6] B: UPDATE 1\n[7] A: ROLLBACK\n[8] B: COMMIT\n"
	for i := range 3 {
		if got := runMulti(t, script).Text(); got != want {
			t.Fatalf("run %d got:\n%s\nwant:\n%s", i+1, got, want)
		}
	}
}

// A failed statement doesn't end the session: Postgres decides what still
// works, and ROLLBACK must get through so a retry can follow.
func TestMultiSessionRecoversAfterError(t *testing.T) {
	got := runMulti(t, `
A: BEGIN;
A: SELECT 1/0;
A: SELECT 1;
A: ROLLBACK;
A: SELECT balance FROM accounts WHERE id = 'b';
`).Text()
	for _, line := range []string{
		"[2] A: ERROR: division by zero (SQLSTATE 22012)",
		"[3] A: ERROR: current transaction is aborted, commands ignored until end of transaction block (SQLSTATE 25P02)",
		"[4] A: ROLLBACK",
		"[5] A: SELECT 1",
	} {
		if !strings.Contains(got, line+"\n") {
			t.Fatalf("missing %q in:\n%s", line, got)
		}
	}
}

func TestMultiSetupErrorIsShown(t *testing.T) {
	res, err := RunMulti(context.Background(), testSession(t), testSession(t), "CREATE TABLE broken (", "A: SELECT 1;")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := res.Text(); !strings.HasPrefix(got, "setup: ERROR: syntax error at end of input (SQLSTATE 42601)") {
		t.Fatalf("got:\n%s", got)
	}
}
