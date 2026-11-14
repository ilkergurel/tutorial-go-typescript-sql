package sqlrun

// Statement guards: statements the runner refuses instead of sending, because
// running them would break a promise the runtime makes. Refusing up front gives
// the learner the reason and what to do instead, rather than a hang or a
// change that quietly persists.
//
// Matching is on the statement's leading keyword (after comments), which is
// enough here: transaction control can't hide inside another statement at the
// top level, and a procedure or DO block that tries COMMIT inside the sql
// runtime's transaction already fails with "invalid transaction termination".
// The runner still checks the transaction status after every statement as a
// safety net (runPhase).

import (
	"regexp"

	"stacklab/server/internal/runapi"
)

var (
	// COPY ... FROM STDIN makes the server wait for the client to stream rows;
	// this runner only sends SQL text, so the run would hang until it times out.
	copyFromStdinRe = regexp.MustCompile(`(?is)^copy\b.*\bfrom\s+stdin\b`)

	// Statements that end the current transaction. ROLLBACK TO [SAVEPOINT]
	// only rewinds to a savepoint inside it and stays allowed.
	endsTransactionRe   = regexp.MustCompile(`(?i)^(commit|end|abort|rollback|prepare\s+transaction)\b`)
	rollbackToSavepoint = regexp.MustCompile(`(?i)^rollback(\s+(work|transaction))?\s+to\b`)
)

// refuseCopyFromStdin returns the error for a COPY ... FROM STDIN, in any mode.
func refuseCopyFromStdin(stmt string) *runapi.SQLStmtError {
	if !copyFromStdinRe.MatchString(codeStart(stmt)) {
		return nil
	}
	return &runapi.SQLStmtError{
		Message: "COPY ... FROM STDIN is refused here: the server would wait for the client to stream rows, and this runner only sends SQL text",
		Hint:    "Use INSERT ... VALUES. Clients that stream rows are psql's \\copy and pgx's CopyFrom (lesson sql.files-movement.copy).",
	}
}

// refuseTxControl returns the error for a statement that would end the sql
// runtime's sandbox transaction.
func refuseTxControl(stmt string) *runapi.SQLStmtError {
	code := codeStart(stmt)
	if !endsTransactionRe.MatchString(code) || rollbackToSavepoint.MatchString(code) {
		return nil
	}
	return &runapi.SQLStmtError{
		Message: "COMMIT, ROLLBACK and END are refused here: the sql runtime runs the whole script in one transaction and rolls it back at the end",
		Detail:  "Ending that transaction early would let the rest of the script write outside the sandbox, and the changes would persist.",
		Hint:    "Use SAVEPOINT and ROLLBACK TO SAVEPOINT to undo part of a script. Lessons about transactions use the sql-nontx or sql-multisession runtime.",
	}
}

// escapedSandbox is the safety net behind refuseTxControl: the statement ended
// the transaction some way the keyword check missed.
var escapedSandbox = &runapi.SQLStmtError{
	Message: "this statement ended the sql runtime's transaction, so the run stopped here",
	Detail:  "Statements after it would have run outside the sandbox. The run's schema was dropped.",
	Hint:    "Lessons about transactions use the sql-nontx or sql-multisession runtime.",
}

// refused is the result recorded for a statement the runner didn't send.
func refused(err *runapi.SQLStmtError) runapi.StatementResult {
	return runapi.StatementResult{
		Columns: []runapi.SQLColumn{},
		Rows:    [][]*string{},
		Notices: []string{},
		Error:   err,
	}
}
