---
id: go.postgres.transactions
title: "Transactions: a WithTx helper, deferred rollback, injection-safe parameters"
track: go
module: postgres
order: 3
level: intermediate
minutes: 15
since: go1.26
prerequisites: [go.postgres.querying, go.errors.is-as-join]
compare_with: [sql.transactions.acid-mvcc, sql.transactions.serializable, ts.errors.try-catch]
tags: [postgres, pgx, transactions, savepoints, sql-injection]
summary: >
  Wrap writes that belong together in one helper that begins, defers the rollback
  and commits only on success, and run every statement through the tx it hands you.
  Values always travel as $1 parameters; identifiers come from an allow-list.
examples:
  - id: with-tx
    file: examples/01_with_tx.go
    title: A WithTx helper and a transfer that must not half-happen
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-17"
        note: >
          The consumer declares the one method it needs, so WithTx accepts a
          connection, a pool, or a transaction (which then nests as a savepoint).
          pgx.BeginFunc takes the same shape.
      - lines: "21-24"
        note: >
          pgx uses Begin's context for the BEGIN statement only; cancelling it
          later does not roll anything back. database/sql's BeginTx, by
          contrast, watches the context for the life of the transaction.
      - lines: "25"
        note: >
          The deferred Rollback covers every early return and a panic in fn.
          After Commit it just returns ErrTxClosed. If ctx is already cancelled
          the ROLLBACK can't be sent; pgx then closes the connection and the
          server rolls back on its own.
      - lines: "26-29"
        note: >
          Commit only when fn succeeded, and return Commit's error: a commit can
          fail (a serialization failure, a deferred constraint, a dropped
          connection), and then nothing was saved.
      - lines: "36-39"
        note: >
          The credit runs first on purpose: when the debit fails, Çağrı already
          has the money inside the transaction, and only the rollback takes it
          back. Note every statement goes through tx, never the pool.
      - lines: "40-44"
        note: >
          The CHECK (balance >= 0) violation, SQLSTATE 23514, becomes a domain
          error at the boundary, as in the errors module; callers test
          errors.Is(err, ErrInsufficientFunds), not SQLSTATE codes.
      - lines: "62-68"
        note: >
          The second transfer fails and both balances are exactly as the first
          one left them: the credit and the failed debit were one unit.
  - id: injection
    file: examples/02_injection.go
    title: The same input as a parameter and concatenated into SQL
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "28-30"
        note: >
          With an argument, pgx sends the statement and the value separately
          (the extended protocol): the server has parsed the SQL before it sees
          the value, so no value can change the statement's shape. Nobody is
          called "nobody' OR 'a' = 'a", so nothing changes.
      - lines: "32-34"
        note: >
          Concatenated, the quote closes the literal and OR 'a' = 'a' makes the
          WHERE true for every row. Exec without arguments even uses the simple
          protocol, which runs several statements: '; DROP TABLE accounts; --
          would have worked as well.
      - lines: "36-41"
        note: >
          Identifiers can't be parameters. pgx.Identifier.Sanitize quotes them
          and doubles embedded quotes, so the input becomes one strange table
          name that doesn't exist (42P01). It stops injection, not misuse: an
          allow-list of names is still the better design.
  - id: aborted-tx
    file: examples/03_aborted_tx.go
    title: An error aborts the whole transaction; a savepoint contains it
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "31-39"
        note: >
          After any error PostgreSQL rejects every further statement in the
          transaction with SQLSTATE 25P02 until it ends. Logging the error and
          carrying on achieves nothing; the second email is never inserted.
      - lines: "40"
        note: >
          COMMIT on an aborted transaction is accepted and performs a ROLLBACK.
          pgx reports that as ErrTxCommitRollback, which is the only reason you
          learn the "successful" work was thrown away.
      - lines: "45-48"
        note: >
          BeginFunc on a Tx starts a SAVEPOINT (pgx calls it a pseudo nested
          transaction); on error it rolls back to the savepoint only, and the
          outer transaction stays usable.
      - lines: "49-53"
        note: >
          Expected errors are handled, everything else still aborts the outer
          transaction. For this exact case INSERT ... ON CONFLICT DO NOTHING is
          simpler; savepoints are for failures SQL can't express.
use_cases:
  - "Any write that touches more than one row or table and must not half-happen: transfers, orders with order lines, a user plus their default settings"
  - "Read-modify-write sequences, together with SELECT ... FOR UPDATE or SERIALIZABLE isolation (sql.transactions module)"
  - "Savepoints for imports that skip or record bad rows without abandoning the whole batch"
strengths:
  - "Unlike Spring's @Transactional, nothing is implicit: the tx is a value you pass, so the statements inside are visibly the ones that use it"
  - "One defer covers every exit path, including panics, which Java needs a finally block or try-with-resources for"
  - "PostgreSQL runs most DDL inside transactions too, so a schema change and its data fix can commit or roll back together, which MySQL cannot do"
  - "Parameters are a protocol feature, not string escaping: the value never becomes SQL text"
weaknesses:
  - "Passing tx explicitly means every function in the unit takes it as a parameter (or works on an interface both pool and tx satisfy)"
  - "pgx does not tie the transaction to Begin's context; code ported from database/sql that relied on cancel-means-rollback must roll back itself"
  - "After an error the transaction is unusable; recovering mid-transaction needs savepoints, which other databases don't require"
pitfalls:
  - title: "Using the pool inside the transaction"
    detail: "pool.Exec runs on a different connection in its own implicit transaction: it commits immediately, even if fn later fails, and it can't see the tx's uncommitted rows."
    bad: |
      err := WithTx(ctx, pool, func(tx pgx.Tx) error {
          if _, err := tx.Exec(ctx, insertOrder, id); err != nil {
              return err
          }
          _, err := pool.Exec(ctx, insertLine, id, sku) // outside the tx
          return err
      })
    good: |
      err := WithTx(ctx, pool, func(tx pgx.Tx) error {
          if _, err := tx.Exec(ctx, insertOrder, id); err != nil {
              return err
          }
          _, err := tx.Exec(ctx, insertLine, id, sku)
          return err
      })
  - title: "Ignoring Commit's error"
    detail: "defer tx.Commit(ctx) drops the error. A commit can fail (40001 serialization_failure under SERIALIZABLE, a deferred constraint, a lost connection), or report ErrTxCommitRollback after a swallowed error; either way nothing was saved."
  - title: "Formatting values into SQL"
    detail: "fmt.Sprintf with '%s', and hand-written quote escaping, both turn data into code. Pass every value as $n. For dynamic ORDER BY columns or table names, map user choices to constants."
    bad: |
      q := fmt.Sprintf("SELECT * FROM users ORDER BY %s", r.URL.Query().Get("sort"))
    good: |
      sortBy := map[string]string{"name": "name", "newest": "created_at DESC"}
      order, ok := sortBy[r.URL.Query().Get("sort")]
      if !ok {
          order = "name"
      }
      q := "SELECT id, name FROM users ORDER BY " + order
  - title: "LIKE patterns from user input"
    detail: "A parameter stops injection but not wildcards: % and _ in the input still match anything. Escape them (backslash is LIKE's default escape character) or use starts_with(name, $1)."
  - title: "Slow work inside a transaction"
    detail: "An open transaction holds a connection and its row locks, and keeps VACUUM from cleaning up. Make HTTP calls and other slow work before Begin or after Commit; idle_in_transaction_session_timeout is the server's backstop."
quiz:
  - q: "Inside WithTx's callback you call pool.Exec instead of tx.Exec, and later the callback returns an error. What happens to the pool.Exec write?"
    options: ["It is rolled back with the transaction", "It is committed: it ran on another connection, outside the transaction", "pgx returns an error because the pool is busy"]
    answer: 1
    why: "The pool hands out a different connection, where the statement runs and commits on its own."
  - q: "A statement inside a transaction fails with a unique violation. You log it and run another INSERT on the same tx. What does PostgreSQL do?"
    options: ["Runs the INSERT normally", "Rejects it with SQLSTATE 25P02 until the transaction ends", "Rolls back automatically and runs the INSERT in a new transaction"]
    answer: 1
    why: "An aborted transaction ignores every command until ROLLBACK (or ROLLBACK TO SAVEPOINT)."
  - q: "Which is safe against SQL injection?"
    options: ["fmt.Sprintf(\"... WHERE name = '%s'\", strings.ReplaceAll(name, \"'\", \"''\"))", "conn.Query(ctx, \"... WHERE name = $1\", name)", "\"... WHERE name = \" + strconv.Quote(name)"]
    answer: 1
    why: "Only the parameter keeps the value out of the SQL text. strconv.Quote makes a double-quoted Go string, which PostgreSQL reads as an identifier."
  - q: "defer tx.Rollback(ctx) runs after tx.Commit(ctx) succeeded. What does it do?"
    options: ["Undoes the commit", "Nothing: it returns ErrTxClosed, which the defer ignores", "Panics"]
    answer: 1
    why: "Rollback on a closed Tx is a documented no-op that returns ErrTxClosed, which is why the deferred call is safe."
further_reading:
  - title: "package pgx: Tx, BeginFunc, Identifier"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Tx"
  - title: "PostgreSQL tutorial: Transactions (and savepoints)"
    url: "https://www.postgresql.org/docs/current/tutorial-transactions.html"
  - title: "PostgreSQL: SAVEPOINT"
    url: "https://www.postgresql.org/docs/current/sql-savepoint.html"
  - title: "PostgreSQL: Pattern matching with LIKE (escape characters)"
    url: "https://www.postgresql.org/docs/current/functions-matching.html#FUNCTIONS-LIKE"
  - title: "PostgreSQL: Error codes (23514, 23505, 25P02)"
    url: "https://www.postgresql.org/docs/current/errcodes-appendix.html"
---

## Concept

A transaction makes several statements one unit: all of them commit or none do. In pgx,
`Begin` returns a `pgx.Tx` bound to one connection; every statement of the unit must go
through that `tx`, and the transaction ends with `Commit` or `Rollback`. Getting the
ending right on every path (errors, early returns, panics) is the part people get wrong,
so put it in one helper:

```go
tx, err := db.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx) // no-op after Commit
if err := fn(tx); err != nil {
    return err
}
return tx.Commit(ctx)
```

`pgx.BeginFunc(ctx, db, fn)` is exactly this, ready-made. Two PostgreSQL behaviours shape
the code inside `fn`. First, an error aborts the whole transaction: later statements fail
with SQLSTATE 25P02 until it ends, so you either return the error or isolate the risky
statement in a savepoint (`tx.Begin`). Second, values are never part of the SQL text: you
write `$1, $2` and pass the values as arguments. Identifiers (table and column names)
can't be parameters, so choose them from a fixed list in your code.

## Coming from C/C++/Java/Python/JS

- Java: JDBC's `setAutoCommit(false)` plus commit, and rollback in `finally`, is the same
  shape with `defer` as the `finally`. Spring's `@Transactional` hides the connection in a
  thread-local; Go has no thread-locals, so the transaction is an explicit parameter.
  `$1` is `PreparedStatement`'s `?`.
- Python: psycopg 3's `with conn.transaction():` is `WithTx`, with a closure instead of a
  context manager. `cursor.execute(sql, (name,))` is the safe form there too; Python's
  `%` operator on the SQL string is the unsafe one.
- JS/TS: `prisma.$transaction(async (tx) => ...)` and knex's `transaction(trx => ...)`
  use the same callback shape as `WithTx`.
- C: libpq has no helper; you send `BEGIN` and `COMMIT` yourself, and `PQexecParams` is
  how values stay separate from the SQL.

## Design notes

The service layer decides what one unit of work is ("transfer" is two updates), so the
transaction belongs there, not inside each repository method. Repositories take an
interface that both the pool and a `pgx.Tx` satisfy (`Exec`, `Query`, `QueryRow`), so the
same code runs inside or outside a transaction; sqlc generates exactly that interface,
`DBTX`, in lesson 5. Pass the transaction as a value; hiding it in a `context.Context`
makes it invisible in signatures and easy to bypass.

Keep transactions short: no HTTP calls or user think-time between `Begin` and `Commit`.
The default isolation level is READ COMMITTED; `BeginTx(ctx, pgx.TxOptions{IsoLevel:
pgx.Serializable})` buys stronger guarantees at the price of retrying on SQLSTATE 40001,
which the `sql.transactions` module demonstrates with two live sessions.
