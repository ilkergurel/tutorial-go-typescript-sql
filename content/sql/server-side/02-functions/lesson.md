---
id: sql.server-side.functions
title: "SQL functions vs PL/pgSQL; procedures"
track: sql
module: server-side
order: 2
level: advanced
minutes: 15
since: pg14
prerequisites: [sql.server-side.views, sql.indexes.plans]
compare_with: [sql.server-side.triggers, sql.subqueries-ctes.modifying, go.postgres.transactions, design.architecture.business-rules]
tags: [functions, plpgsql, procedures, volatility, inlining, transactions]
summary: >
  A simple LANGUAGE sql function is a named query the planner can inline;
  PL/pgSQL adds variables, loops and exceptions but is a black box to the
  planner; a procedure can COMMIT between batches. Push logic into the database
  when it saves round trips or must sit next to the data, not by default.
examples:
  - id: inlining
    file: examples/01_inlining.sql
    title: "One table function three ways: inlined SQL, VOLATILE SQL, PL/pgSQL"
    runtime: sql
    check: exact
    annotations:
      - lines: "15-17"
        note: >
          LANGUAGE sql, a single SELECT, STABLE, not STRICT, no SECURITY
          DEFINER and no SET clause: the planner can replace the call with the
          query itself, as it does for a view. This is a parameterized view.
      - lines: "19-21"
        note: >
          The same body without a volatility label, so it defaults to
          VOLATILE. A volatile function may have side effects, so the planner
          must call it as written instead of merging it into the outer query.
      - lines: "23-25"
        note: >
          PL/pgSQL functions are not inlined. The planner sees a function that
          returns a set of rows and, lacking a ROWS clause, assumes 1000 of
          them.
      - lines: "27"
        note: >
          6 failed runs for test case 42. The other two functions return the
          same rows; only their plans differ.
      - lines: "29"
        note: >
          Inlined: the function is gone from the plan. Its WHERE and the
          caller's status filter end up in one scan, and the index on
          test_case_id is used.
      - lines: "30-31"
        note: >
          Function Scan: the function runs its own query, and the caller's
          filter is applied to whatever it returns (the Filter line sits on the
          Function Scan). Harmless for 40 rows; when the caller's condition is
          the selective one, it's the difference between using an index and
          not.
  - id: body-checks
    file: examples/02_body_checks.sql
    title: "String body vs SQL-standard body: what happens when the schema changes"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-7"
        note: >
          A string body is validated at CREATE (check_function_bodies is on by
          default), then stored as text and parsed again when it runs. Nothing
          records that it uses test_runs.status.
      - lines: "11-13"
        note: >
          The SQL-standard body (PG 14+, LANGUAGE sql only) is parsed once and
          stored as a parse tree, with dependencies on test_runs and the
          columns it reads. Dropping the table or the column, or changing the
          column's type, is now refused while the function exists.
      - lines: "18-19"
        note: >
          The rename doesn't break the standard body: it refers to the column
          itself, not its name, exactly like a view.
      - lines: "20"
        note: >
          pg_get_functiondef decompiles the stored tree, so the body already
          reads outcome. (substring keeps just the RETURN line; the full output
          starts with the schema-qualified function name.)
      - lines: "21"
        note: >
          SQLSTATE 42703. The string body still says status and fails only when
          called, possibly long after the migration that broke it ran green.
  - id: procedure
    file: examples/03_procedure.sql
    title: "A procedure that commits between batches, and why CALL inside BEGIN can't"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          A procedure returns nothing (apart from OUT parameters) and is run
          with CALL, not SELECT. Unlike a function it may end the transaction
          it runs in, when CALL is a transaction of its own.
      - lines: "16-21"
        note: >
          One batch is one data-modifying CTE (sql.subqueries-ctes.modifying):
          a run can't be deleted without being archived.
      - lines: "22-23"
        note: >
          GET DIAGNOSTICS reads the row count of the last command; an empty
          batch ends the loop.
      - lines: "24"
        note: >
          COMMIT makes the batch durable, releases its row locks and starts a
          new transaction automatically. A crash halfway leaves the earlier
          batches done, so the procedure must be safe to run again. This one
          is: it only moves runs that are still old.
      - lines: "30"
        note: >
          Three NOTICEs, three committed transactions, one CALL: 299 runs were
          older than 2026-01-04, moved as 120 + 120 + 59.
      - lines: "36-37"
        note: >
          SQLSTATE 2D000. Inside BEGIN the client owns the transaction, so the
          procedure may not commit it. That's also why this example uses
          sql-nontx: the sql runtime wraps the whole script in one
          transaction.
use_cases:
  - "Parameterized views: a STABLE SQL table function such as runs_of(case_id) that every client calls and the planner inlines"
  - "IMMUTABLE helpers for expression indexes, generated columns and CHECK constraints, e.g. normalizing a requirement code"
  - "An operation several applications must perform identically and atomically next to the data, such as closing a requirement only when all its test cases passed"
  - "Procedures for batched maintenance with COMMIT between batches: archiving, backfills, purges"
  - "Trigger functions, the most common job PL/pgSQL has (next lesson)"
strengths:
  - "Unlike a Go or TS loop over query results, a function runs next to the data: 10,000 statements cost one round trip, not 10,000"
  - "STABLE SQL functions are inlined, so a reusable, named query costs nothing at runtime"
  - "SQL-standard bodies (PG 14) record their dependencies, so the database refuses schema changes that would break them; a query string in Go code gets no such check"
  - "A procedure can commit between batches, so one CALL can do a long cleanup without one long transaction"
weaknesses:
  - "PL/pgSQL is a black box to the planner: no inlining, a fixed 1000-row estimate for set-returning functions, statements planned on first use and cached per session"
  - "Function bodies live outside the app's code review, debugger, tests and deploys; they change only through migrations, and app/schema version skew shows up at runtime"
  - "PL/pgSQL checks only syntax at CREATE: a misspelled table name fails on first call, not at deploy"
  - "Work done in functions runs on the database server, the component in the stack that is hardest to scale out"
pitfalls:
  - title: "Forgetting the volatility label"
    detail: "Functions default to VOLATILE. A volatile SQL table function isn't inlined, can't be used in an index condition, and is re-evaluated for every row. Label read-only functions STABLE and pure ones IMMUTABLE."
    bad: "CREATE FUNCTION runs_of(case_id int) RETURNS SETOF test_runs\nLANGUAGE sql\nAS $$ SELECT * FROM test_runs WHERE test_case_id = case_id $$;"
    good: "CREATE FUNCTION runs_of(case_id int) RETURNS SETOF test_runs\nLANGUAGE sql STABLE\nAS $$ SELECT * FROM test_runs WHERE test_case_id = case_id $$;"
  - title: "Labeling a function IMMUTABLE when it isn't"
    detail: "IMMUTABLE allows the planner to fold the call to a constant and lets it into expression indexes. A function that reads a table, calls now() or depends on a setting is at most STABLE. Casting timestamptz to date depends on the TimeZone setting (the cast is STABLE); naming the zone explicitly makes it immutable. Otherwise cached plans go stale and an expression index can silently disagree with the data."
    bad: "CREATE FUNCTION run_day(ts timestamptz) RETURNS date\nLANGUAGE sql IMMUTABLE\nRETURN ts::date;  -- depends on the session's TimeZone"
    good: "CREATE FUNCTION run_day(ts timestamptz) RETURNS date\nLANGUAGE sql IMMUTABLE\nRETURN (ts AT TIME ZONE 'Europe/Istanbul')::date;"
  - title: "Row-by-row loops in PL/pgSQL"
    detail: "A FOR loop that runs one UPDATE per row is the N+1 problem moved into the server. It beats doing the same from Go, but one set-based statement beats both."
    bad: "FOR r IN SELECT id FROM test_cases WHERE requirement_id = 7 LOOP\n  UPDATE test_cases SET status = 'retest' WHERE id = r.id;\nEND LOOP;"
    good: "UPDATE test_cases SET status = 'retest' WHERE requirement_id = 7;"
  - title: "EXCEPTION blocks as a habit"
    detail: "Every block with an EXCEPTION clause runs as a subtransaction, which the docs call significantly more expensive to enter and exit. Catch where you handle the error; prefer ON CONFLICT over catching unique_violation."
  - title: "SECURITY DEFINER without a pinned search_path"
    detail: "A SECURITY DEFINER function runs with its owner's privileges. If its search_path includes a schema the caller can write to, the caller can shadow a table or function the body uses and run code as the owner. Set search_path on the function, with pg_temp last, and revoke EXECUTE from PUBLIC."
    good: "CREATE FUNCTION close_requirement(code text) RETURNS void\nLANGUAGE plpgsql SECURITY DEFINER\nSET search_path = app, pg_temp\nAS $$ ... $$;\nREVOKE EXECUTE ON FUNCTION close_requirement(text) FROM PUBLIC;"
quiz:
  - q: "Which of these can the planner inline into the calling query?"
    options: ["A LANGUAGE sql set-returning function with one SELECT, marked STABLE", "The same function without a volatility label", "A PL/pgSQL function using RETURN QUERY", "All three"]
    answer: 0
    why: "Example 1: only the STABLE SQL function disappears from the plan. VOLATILE (the default) and PL/pgSQL show up as Function Scan."
  - q: "A procedure COMMITs inside a loop. Your Go code opens a transaction with BEGIN and runs CALL inside it. What happens?"
    options: ["It commits your transaction too", "It fails with invalid transaction termination", "The COMMITs are silently ignored"]
    answer: 1
    why: "Transaction control is only allowed when CALL runs in its own transaction; inside a transaction block, the caller owns it."
  - q: "You rename a column used by a function written with the SQL-standard body (`RETURN ...`). What happens to the function?"
    options: ["It fails the next time it's called", "The rename is refused", "It keeps working, and its stored body shows the new name"]
    answer: 2
    why: "The standard body is a stored parse tree that refers to the column, not its name. A string body would fail at its next call."
  - q: "A function looks up a value in a reference table and returns it. Which volatility is correct?"
    options: ["IMMUTABLE", "STABLE", "VOLATILE"]
    answer: 1
    why: "It reads the database, so the result can change between statements: not IMMUTABLE. It doesn't modify anything and is consistent within a statement: STABLE."
further_reading:
  - title: "PostgreSQL docs: Query Language (SQL) Functions"
    url: "https://www.postgresql.org/docs/current/xfunc-sql.html"
  - title: "PostgreSQL docs: Function Volatility Categories"
    url: "https://www.postgresql.org/docs/current/xfunc-volatility.html"
  - title: "PostgreSQL docs: CREATE FUNCTION (SQL-standard body, SECURITY DEFINER, SET)"
    url: "https://www.postgresql.org/docs/current/sql-createfunction.html"
  - title: "PostgreSQL docs: PL/pgSQL under the hood (plan caching)"
    url: "https://www.postgresql.org/docs/current/plpgsql-implementation.html"
  - title: "PostgreSQL docs: Transaction Management in PL/pgSQL"
    url: "https://www.postgresql.org/docs/current/plpgsql-transactions.html"
  - title: "PostgreSQL docs: CALL"
    url: "https://www.postgresql.org/docs/current/sql-call.html"
---

## Concept

Server-side code in PostgreSQL comes in three shapes:

- **SQL functions** (`LANGUAGE sql`) are one or more SQL statements; the last one's
  result is returned (all rows for `RETURNS SETOF` / `RETURNS TABLE`). A simple one is a
  named, parameterized query, and the planner can inline it like a view.
- **PL/pgSQL functions** add variables, `IF`, loops, `RAISE` and exception handling.
  Each SQL statement inside is prepared on first use and its plan cached for the session.
  The planner treats the function as a black box.
- **Procedures** (`CREATE PROCEDURE`, PG 11) are called with `CALL`, return nothing but
  `OUT` parameters, and may `COMMIT` or `ROLLBACK` when the `CALL` is a transaction of
  its own.

Every function has a **volatility** that the planner trusts: `VOLATILE` (the default:
anything goes, re-evaluated for every row), `STABLE` (no writes, same result for the same
arguments within one statement), `IMMUTABLE` (same result forever, so it may be folded at
plan time and used in index expressions). Getting it wrong costs either performance
(too loose) or correctness (too strict).

A SQL function body can be a string (`AS $$ ... $$`) or, since PG 14, SQL-standard
(`RETURN expr` or `BEGIN ATOMIC ... END`). The standard form is parsed once, stored as a
parse tree and tracked in the dependency graph, like a view.

## Coming from C/C++/Java/Python/JS

- **Java stored procedures via JDBC `CallableStatement`:** same idea, different call
  syntax. In PostgreSQL functions are called inside queries (`SELECT f(x)`,
  `FROM f(x)`), and only procedures use `CALL`.
- **C++ `inline` and `constexpr`:** a STABLE SQL function is inlined like an `inline`
  function; IMMUTABLE is close to `constexpr`: pure, so it may be evaluated once at plan
  time when its arguments are constants.
- **Go/TS query strings:** a string function body is like SQL in a Go string, checked
  only when it runs. The standard body is closer to sqlc-generated code: the schema is
  checked against it, and a breaking migration fails instead of the next request.
- **Python:** PL/pgSQL reads like Pascal or Ada. For heavier logic there are PL/Python
  (`plpython3u`) and others, but the `u` means untrusted: superuser only.

## How it works

Inlining is the planner replacing a function call with the function's query before
optimization. It applies only to `LANGUAGE sql` functions, and for set-returning ones
only when the body is a single `SELECT`, the function is `STABLE` or `IMMUTABLE`, not
`STRICT`, not `SECURITY DEFINER` and has no `SET` clause. The call must also sit in
`FROM`, without `WITH ORDINALITY`, and its arguments may not contain volatile functions
or subqueries. `EXPLAIN` shows the outcome: the function's name disappears from the plan.
The manual doesn't list these conditions; the planner's source does
(src/backend/optimizer/util/clauses.c, function inline_set_returning_function), and it
reads a `BEGIN ATOMIC` body as well as a string one. On PostgreSQL 18.7, example 1's
`runs_of` stopped being inlined (Function Scan again) when declared `STRICT`,
`SECURITY DEFINER` or with `SET work_mem`, or when called with `random()` in its
argument; a `BEGIN ATOMIC` copy of it was inlined.

PL/pgSQL is compiled to an instruction tree on its first call in each session, and each
SQL statement inside becomes a prepared statement the first time execution reaches it.
After a few executions the plan cache may switch to a generic plan that ignores the
current variable values. Two consequences: a mistake in a statement surfaces only when
that statement first runs, and table or column names can't be variables; for those you
build a string and run it with `EXECUTE ... USING`.

The examples use the single-statement `RETURN` form; a `BEGIN ATOMIC ... END` body holds
several statements and has the same dependency tracking. Its semicolons don't end the
`CREATE`: like psql, this sandbox's splitter counts `BEGIN` and `END` (and `CASE ... END`
inside them) only in a `CREATE FUNCTION` or `CREATE PROCEDURE` statement, so a plain
`BEGIN;` still starts a transaction.

## Design notes

Push logic into the database when it pays:

- **The work is data-heavy and set-based:** aggregations, bulk moves, batched
  maintenance. Shipping rows to the application and back is the expensive part.
- **Round trips dominate:** a multi-step operation per request that would otherwise
  be several statements with the application in between.
- **The rule must hold for every client** and you can enforce that, for example by
  granting the application role `EXECUTE` on a function instead of `UPDATE` on the table.

Keep it in the application when the logic branches on business rules that change often,
calls anything outside the database (HTTP, e-mail, queues: never from inside a
transaction), or needs logging, feature flags and unit tests with fakes. In TraceLite,
"coverage of a requirement" is a good SQL function or view; "notify the owner when
coverage drops" belongs in Go.

When you do write server-side code: prefer plain SQL over PL/pgSQL when one statement
can do it, use the SQL-standard body for new SQL functions, label volatility on every
function, and keep each function in a migration file next to the tables it reads, so
review and version control see it.
