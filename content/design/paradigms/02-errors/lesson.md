---
id: design.paradigms.errors
title: "The same problem: error handling (Go errors vs TS exceptions/Result vs SQL constraints)"
track: design
module: paradigms
order: 2
level: intermediate
minutes: 14
since: go1.26
prerequisites: [design.paradigms.side-by-side, go.errors.is-as-join, ts.errors.result, sql.constraints.keys]
compare_with: [go.errors.errors-as-values, go.errors.is-as-join, go.errors.panic-strategy, ts.errors.try-catch, ts.errors.result, sql.constraints.keys, sql.transactions.acid-mvcc]
tags: [errors, exceptions, result, constraints, sqlstate, comparison]
summary: >
  One sign-up, three failure models: Go returns error values the caller inspects, TypeScript
  throws or returns a Result union, and PostgreSQL refuses the write with a SQLSTATE and
  aborts the transaction. Only the constraint still holds when two requests race.
examples:
  - id: go-error-values
    file: examples/01_error_values.go
    title: "Go: errors are values, inspected with errors.Is and errors.AsType"
    runtime: go
    check: exact
    annotations:
      - lines: "9-15"
        note: >
          Two kinds of expected failure: a sentinel answers "which failure?", an
          error type also carries details. Both are ordinary values; nothing is
          thrown and nothing unwinds.
      - lines: "20"
        note: >
          The error result is in the signature, so every caller sees that
          Register can fail. It doesn't list which errors: that is documentation,
          the same gap a TS signature has for exceptions.
      - lines: "24-26"
        note: >
          %w wraps the sentinel with context. The message gains "register
          ayse@example.com", and errors.Is still finds ErrEmailTaken underneath
          (lesson go.errors.errors-as-values).
      - lines: "32-43"
        note: >
          The mapping to the outside world lives in one function. errors.AsType
          (Go 1.26) is the generic form of errors.As. In a real service this is
          the HTTP boundary, and it also logs the 500 case exactly once.
      - lines: "48-49"
        note: >
          Nothing forces the check: write users.Register(email) as a bare
          statement and the program still compiles, and go vet stays quiet too.
          Third-party linters such as errcheck flag it; code review expects the
          check on the next line.
  - id: ts-throw-or-result
    file: examples/02_throw_or_result.ts
    title: "TypeScript: the same rules, thrown and returned"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-14"
        note: >
          The throwing version reads like Java or Python. The signature promises
          void, so tsc can't tell callers that this function fails, or how.
      - lines: "20-24"
        note: >
          The catch variable is unknown (strict mode), so you narrow with
          instanceof and rethrow what you didn't expect. The catch block plays
          the role of Go's status function, but nothing in registerOrThrow's
          signature told the caller it was needed.
      - lines: "28-36"
        note: >
          The Result version puts the failure modes in the return type. A caller
          can't treat the result as a success without checking ok first, which
          Go's (value, error) pair doesn't enforce.
      - lines: "39"
        note: >
          satisfies checks that every Problem has a status without widening the
          object's type. Add "rate-limited" to Problem and this line stops
          compiling; Go's status function would fall through to 500 instead.
  - id: sql-constraints
    file: examples/03_constraints.sql
    title: "SQL: the table refuses the write, even under a race"
    runtime: sql-multisession
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          The failure modes are declared, not coded: NOT NULL, UNIQUE and CHECK
          name the writes this table refuses, for every client.
      - lines: "11-12"
        note: >
          Each refusal is an error with a SQLSTATE: 23505 unique_violation, 23514
          check_violation. A Go repository maps the code (and the constraint name)
          to its own sentinel; pgx exposes both on *pgconn.PgError.
      - lines: "15-18"
        note: >
          Both sessions ask "is it free?" and both get 0. This is the check an
          application does before inserting, and it proves nothing: the other
          request hasn't written yet.
      - lines: "19-21"
        note: >
          The unique index decides. B's insert waits on A's uncommitted row,
          because A could still roll back. When A commits, B fails with 23505. Go's
          map check and TS's Set check would both have said yes.
      - lines: "24-25"
        note: >
          An error inside a transaction aborts all of it: every further statement
          is refused with 25P02 until ROLLBACK. That is SQL's unwinding, and unlike
          an exception it also undoes the work already done.
      - lines: "27"
        note: >
          The committed result, and a trace of the failures: ids 3, 4 and 6 were
          used by inserts that failed. Sequences are never rolled back, so treat
          identity values as unique, not gap-free.
use_cases:
  - "A sign-up or create endpoint: SQL UNIQUE and CHECK as the guarantee, Go mapping 23505 to 409 and InvalidError to 422, TS turning 409 and 422 into form messages"
  - "Choosing per failure: Go error or TS Result for outcomes callers must handle, panic or throw for bugs, a constraint for anything that must hold across all clients"
  - "Reviewing code that checks before writing (SELECT then INSERT) and replacing the check with a constraint plus error mapping"
strengths:
  - "Go: the error result is in every signature and every propagation is a visible return, so there is no line that can throw unexpectedly as in Java or Python"
  - "TS Result: the success value is unreachable until ok is checked, and an exhaustive table over the Problem union makes a new failure a compile error"
  - "SQL: a constraint holds under concurrency and for every client; two racing requests can't both insert, which no application-side check can promise"
  - "SQL: an error rolls back the whole transaction, so a failure never leaves half the writes behind, unlike an exception thrown between two in-memory updates"
weaknesses:
  - "Go: the compiler accepts an ignored error, and error is an interface, so the possible failures aren't listed anywhere a tool can check"
  - "TS exceptions: signatures can't declare what they throw, catch receives unknown, and libraries keep throwing even when your code returns Results"
  - "SQL: errors arrive as codes and constraint names that every layer above must translate; a 23505 means nothing to a user"
  - "SQL: an aborted transaction refuses everything until rollback, so an unhandled error poisons every later statement on that connection"
pitfalls:
  - title: "Checking before writing instead of relying on the constraint"
    detail: "A SELECT that finds no row doesn't reserve anything; the third example shows two sessions passing the same check. Insert, and map the unique violation; or use ON CONFLICT when a duplicate is not an error."
    bad: "if n, _ := countByEmail(ctx, email); n == 0 { insertUser(ctx, email) }"
    good: "INSERT INTO users (email) VALUES ($1) ON CONFLICT (email) DO NOTHING RETURNING id"
  - title: "Matching on message text"
    detail: "Go's err.Error(), a JS error's message and PostgreSQL's message text are for humans, and PostgreSQL translates its messages to the server's lc_messages language. Match on identity: errors.Is, instanceof, the SQLSTATE code and the constraint name."
    bad: "if strings.Contains(err.Error(), \"duplicate key\") { ... }"
    good: "if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == \"23505\" { ... }"
  - title: "Carrying on after a failed statement in a transaction"
    detail: "Every later statement fails with 25P02, and COMMIT on an aborted transaction rolls back and reports ROLLBACK instead of failing (pgx returns pgx.ErrTxCommitRollback). Return the error, or put the risky statement in a savepoint (lesson sql.transactions.acid-mvcc)."
  - title: "Letting one layer's vocabulary leak into the next"
    detail: "A TS client that parses SQLSTATE codes, or a Go handler that wraps *pgconn.PgError with %w, couples every layer to the database driver. Translate once per boundary: SQLSTATE to sentinel in the repository, sentinel to status in the handler, status to a Problem union in the client."
quiz:
  - q: "Two requests register zeynep@example.com at the same moment. Each first runs SELECT count(*) and gets 0. What prevents a duplicate?"
    options:
      - "Nothing: both inserts succeed"
      - "The UNIQUE constraint: one insert waits, then fails with 23505"
      - "READ COMMITTED isolation makes the second SELECT wait"
    answer: 1
    why: "Plain SELECTs never wait for writers under MVCC. The unique index makes the second insert wait for the first transaction and fail when it commits."
  - q: "Where does the TS example tell tsc that registerOrThrow can fail?"
    options:
      - "In its return type"
      - "Nowhere: the signature says void"
      - "In the catch clause of statusOf"
    answer: 1
    why: "TypeScript has no throws clause. The Result version moves the failures into the return type, where the checker can see them."
  - q: "B's INSERT failed inside BEGIN. What does B's next SELECT do?"
    options:
      - "Returns the rows as usual"
      - "Fails with 25P02 until B ends the transaction"
      - "Waits for A"
    answer: 1
    why: "PostgreSQL aborts the whole transaction on an error; only ROLLBACK (or ROLLBACK TO a savepoint) makes the session usable again."
  - q: "A new failure kind, \"rate-limited\", is added. Which version makes the compiler point at every place that needs a new case?"
    options:
      - "Go's status function"
      - "The TS Result version with satisfies Record<Problem, number>"
      - "The TS exception version"
    answer: 1
    why: "The record must have a key for every member of the Problem union. Go's if-chain and the TS catch block both fall through to a default silently."
further_reading:
  - title: "The Go Blog: Working with Errors in Go 1.13"
    url: "https://go.dev/blog/go1.13-errors"
  - title: "package errors (Is, As, AsType, Join)"
    url: "https://pkg.go.dev/errors"
  - title: "TypeScript Handbook: Narrowing (instanceof, discriminated unions)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "MDN: try...catch"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/try...catch"
  - title: "PostgreSQL docs: Constraints"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Appendix A. Error Codes"
    url: "https://www.postgresql.org/docs/current/errcodes-appendix.html"
---

## Concept

Every failure falls into one of three groups: expected and handled nearby (bad input,
a duplicate e-mail), expected but handled far away (a lost connection, a timeout), and
bugs. The three languages give honest, different answers.

- **Go returns errors as values.** The signature says a function can fail, the caller
  checks on the next line, and context is added with `%w` on the way up. Expected
  failures are sentinels or error types found with `errors.Is` and `errors.AsType`; bugs
  panic. Every propagation is a visible `return`, and nothing forces you to look.
- **TypeScript throws, or returns a union.** JavaScript can throw any value from any
  line, signatures don't say so, and `catch` receives `unknown`. For failures a caller
  must handle every time, a Result union moves them into the return type, where the
  checker makes callers look. Both channels coexist, because the platform and libraries
  throw.
- **SQL refuses the write.** Constraints declare which rows a table will never hold. A
  write that breaks one fails with a SQLSTATE code, and the failure aborts the whole
  transaction. Because the database sees every write, the constraint still holds when two
  requests race, which no check in Go or TypeScript can promise.

All three examples register the same three e-mails against a store that already holds
`ayse@example.com`: one succeeds, one is a duplicate, one is invalid. The SQL example then
runs the race that the in-memory versions can't see.

## Coming from C/C++/Java/Python/JS

- **Java**: checked exceptions put failures in signatures and force handling; Go puts
  them in signatures without forcing; TypeScript does neither. JDBC's
  `SQLException.getSQLState()` is the same five-character code that pgx exposes as
  `PgError.Code`.
- **Python**: TypeScript's exceptions behave like Python's, minus typed `except` clauses.
  psycopg raises `errors.UniqueViolation` for 23505, a class per SQLSTATE, where pgx
  gives one error type with a code field.
- **C**: Go's error values are return codes that carry a message and a chain of causes;
  SQLSTATE is the database's errno, the same on every client and every platform.
- **Rust and C++23**: `Result` and `std::expected` are TypeScript's Result union with
  language or library support; neither Go nor TypeScript has a `?` operator, so
  propagation is an explicit `return`.
- **JS**: nothing in TypeScript changes how `throw` works at run time; the Result union
  is a convention the type checker enforces.

## How it works

One failed sign-up crosses every layer, and each layer translates once:

```text
layer              receives                          produces
TS form            user input                        "must contain @" before sending (UX only)
Go handler         JSON body                         *InvalidError -> 422
Go repository      INSERT fails: SQLSTATE 23505,     ErrEmailTaken, wrapped with %w
                   constraint users_email_key
Go handler         errors.Is(err, ErrEmailTaken)     409 with a problem+json body
TS client          response.status 409               { ok: false, problem: "taken" }
TS form            Problem union                     "this e-mail is already registered"
```

No layer speaks another's vocabulary: the client never sees a SQLSTATE, and the handler
never imports the database driver.

## Design notes

Choose the mechanism per failure, not per codebase. Outcomes a caller must handle every
time (invalid input, a duplicate, "not found") are values: a Go error the caller inspects,
a TS Result the caller narrows. Bugs are panics or exceptions that travel to one boundary
which logs them once. Rules that must hold whoever writes are constraints, and the
application's own checks exist only to produce a better message earlier.

That last point is what the race in the SQL example proves. Validating in TypeScript and
again in Go is good UX and cheap defense, but "is this e-mail free?" can only be answered
by the database at the moment of the write. Let the INSERT fail, and give the constraint
a name the repository can map: the name says which rule broke, the SQLSTATE says what kind.

Transactions change what "handling" means in SQL. After an error the transaction is
dead; the only moves are ROLLBACK or ROLLBACK TO a savepoint. In Go, that is the
`defer tx.Rollback(ctx)` of a WithTx helper (lesson go.postgres.transactions); in
TypeScript, it means a failed request should be retried or reported as a whole, never
patched up halfway.
