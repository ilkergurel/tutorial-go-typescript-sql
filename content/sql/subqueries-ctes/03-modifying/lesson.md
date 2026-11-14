---
id: sql.subqueries-ctes.modifying
title: "Data-modifying CTEs"
track: sql
module: subqueries-ctes
order: 3
level: advanced
minutes: 13
since: pg18
prerequisites: [sql.subqueries-ctes.scalar, sql.modifying.returning-upsert]
compare_with: [sql.transactions.acid-mvcc, go.postgres.transactions, sql.server-side.triggers]
tags: [cte, with, returning, insert, update, delete, upsert, audit, snapshot, atomicity]
summary: >
  INSERT, UPDATE, DELETE and MERGE can sit in a WITH clause, and their
  RETURNING rows feed the next step: moving rows, inserting a parent with its
  children, or updating with an audit trail becomes one atomic statement. All
  parts read one snapshot, so they see each other's changes only through
  RETURNING.
examples:
  - id: move-and-link
    file: examples/01_move_and_link.sql
    title: "Move rows to an archive, and insert a parent with its children"
    runtime: sql
    check: exact
    annotations:
      - lines: "8"
        note: >
          LIKE ... INCLUDING ALL copies columns, defaults, constraints and
          indexes, so the archive has the same column order and its own
          primary key: archiving the same run twice would fail loudly.
      - lines: "15-21"
        note: >
          DELETE ... RETURNING * hands the removed rows to the INSERT. It is one
          statement, so it commits or fails as a whole: no window where a run
          is deleted but not yet archived, and no second copy if a client
          retries a half-finished pair of statements.
      - lines: "18"
        note: >
          RETURNING * yields the columns in table order, which matches the
          archive because LIKE copied it. Once the two tables can drift, list
          the columns on both sides.
      - lines: "20-21"
        note: >
          The main statement can itself be an INSERT, UPDATE or DELETE. With no
          RETURNING of its own, the client sees only its command tag:
          INSERT 0 2.
      - lines: "39-47"
        note: >
          req's RETURNING carries the generated id into the second INSERT, with
          no round trip to the client in between. The foreign key on
          test_cases holds: a non-deferrable constraint is checked after the
          whole command, and by then the requirement row exists.
      - lines: "44"
        note: >
          req has one row and VALUES has two; the comma join pairs the new id
          with each child title.
  - id: one-snapshot
    file: examples/02_one_snapshot.sql
    title: "One snapshot: the table shows the old state, RETURNING the new; get-or-create"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-15"
        note: >
          The UPDATE runs, but every part of the statement reads the snapshot
          taken before any of it ran, so the outer scan of test_cases still
          sees failed. The docs are explicit: RETURNING is the only way for a
          WITH sub-statement's changes to reach the rest of the statement.
      - lines: "18"
        note: >
          A later statement always sees the changes of earlier statements in the
          same transaction, at any isolation level: the update is visible.
      - lines: "25-31"
        note: >
          kvkk is new: the INSERT returns its row ('created'), and the SELECT,
          reading the old snapshot, finds nothing, so there is no duplicate
          line.
      - lines: "33-39"
        note: >
          security already exists: DO NOTHING returns no row, and the SELECT
          finds the existing one ('found'). Either way the caller gets the id
          in one round trip.
      - lines: "42-43"
        note: >
          SQLSTATE 0A000. A data-modifying statement may only be in a WITH
          attached to the top-level statement, not in a subquery's own WITH.
  - id: audit
    file: examples/03_audit.sql
    title: "Update and audit in one statement with RETURNING old/new (PG 18)"
    runtime: sql
    check: exact
    annotations:
      - lines: "14"
        note: >
          status <> 'passed' leaves out rows the update wouldn't change, so the
          log records real changes only: Logout is neither updated nor logged.
      - lines: "15"
        note: >
          old.status and new.status (PG 18) are the row before and after the
          update. Before PG 18, an UPDATE's RETURNING could only show the new
          values.
      - lines: "16-19"
        note: >
          The second step reads the first one's RETURNING. The audit rows and
          the update commit or fail together, and the audit logic is visible
          in the query instead of hidden in a trigger.
      - lines: "21"
        note: >
          The main query reads logged's RETURNING, so the client receives
          exactly what was written to the log.
use_cases:
  - "Archiving: DELETE ... RETURNING into an archive or history table in one atomic statement"
  - "Creating an aggregate in one round trip: a requirement and its test cases, an order and its lines, with generated ids flowing through RETURNING"
  - "Audit trails without triggers: UPDATE ... RETURNING old.*, new.* (PG 18) feeding an INSERT into a log table"
  - "Get-or-create for tags, labels and lookup values: INSERT ... ON CONFLICT DO NOTHING RETURNING, falling back to a SELECT"
strengths:
  - "Several writes are one statement: atomic even in autocommit, one round trip from Go or TS, no transaction plumbing in the application"
  - "Generated ids travel through RETURNING instead of being read back by the client and sent again"
  - "Each data-modifying step runs exactly once and to completion, whether or not anything reads its output"
  - "With PG 18's old/new in RETURNING, before-and-after values come from the same statement that changed them"
weaknesses:
  - "All parts share one snapshot, so a step can't see another step's changes in the tables; only RETURNING carries them"
  - "Sub-statements run concurrently with each other and the main query, in no guaranteed order"
  - "Two steps modifying the same row is unsupported: only one change takes effect, and which one is hard to predict"
  - "Long chains get hard to debug: a step can't be run on its own without performing its write"
pitfalls:
  - title: "Reading the table instead of RETURNING"
    detail: "The main query and the other steps read the snapshot from before the statement. A SELECT on the modified table shows the old rows; read the CTE."
    bad: "WITH u AS (UPDATE test_cases SET status = 'retest' WHERE status = 'failed' RETURNING id) SELECT id, status FROM test_cases WHERE id IN (SELECT id FROM u)"
    good: "WITH u AS (UPDATE test_cases SET status = 'retest' WHERE status = 'failed' RETURNING id, status) SELECT id, status FROM u"
  - title: "Two steps touching the same row"
    detail: "The docs say updating the same row twice in one statement is not supported: only one modification happens, and which one is hard to predict. Deleting a row another step updated performs only the update. Merge the changes into one UPDATE."
    bad: "WITH a AS (UPDATE test_cases SET status = 'passed' WHERE id = 1), b AS (UPDATE test_cases SET title = 'Login (SSO)' WHERE id = 1) SELECT 1"
    good: "UPDATE test_cases SET status = 'passed', title = 'Login (SSO)' WHERE id = 1"
  - title: "Get-or-create under concurrency"
    detail: "In READ COMMITTED, if another transaction inserts the same tag and commits after your statement's snapshot, DO NOTHING skips the insert and the fallback SELECT can't see the other row: zero rows come back. Treat an empty result as 'retry', or use ON CONFLICT DO UPDATE, which always returns the row at the cost of writing it."
  - title: "A data-modifying CTE nobody reads still runs"
    detail: "Data-modifying steps run exactly once and to completion even if the main query ignores them. A WITH (DELETE ...) added 'just to see' what it would return deletes."
quiz:
  - q: "`WITH u AS (UPDATE test_cases SET status = 'retest' WHERE status = 'failed' RETURNING id) SELECT status FROM test_cases WHERE id IN (SELECT id FROM u)`: which status does the SELECT show for the updated rows?"
    options: ["retest", "failed", "An error"]
    answer: 1
    why: "The outer scan of test_cases uses the statement's snapshot, taken before the UPDATE. Only u's RETURNING rows carry the new values."
  - q: "`WITH d AS (DELETE FROM tags RETURNING id) SELECT 1`: the main query never reads d. What happens?"
    options: ["Every tag is deleted", "Nothing is deleted", "An error about an unused CTE"]
    answer: 0
    why: "Data-modifying statements in WITH are executed exactly once and to completion, regardless of whether their output is read."
  - q: "How do the changes made by one WITH sub-statement reach the others and the main query?"
    options: ["By reading the modified table", "Through a second snapshot taken after each step", "Only through its RETURNING rows"]
    answer: 2
    why: "All parts share one snapshot and run concurrently, so RETURNING is the only channel."
  - q: "Before PG 18, what could `UPDATE ... RETURNING` show?"
    options: ["Old and new values", "Only the new values", "Only the old values"]
    answer: 1
    why: "PG 18 added the old and new aliases; before that an UPDATE returned the new row only."
further_reading:
  - title: "PostgreSQL docs: Data-Modifying Statements in WITH"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-MODIFYING"
  - title: "PostgreSQL docs: Returning Data from Modified Rows (old and new)"
    url: "https://www.postgresql.org/docs/current/dml-returning.html"
  - title: "PostgreSQL docs: INSERT, ON CONFLICT clause"
    url: "https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT"
  - title: "PostgreSQL docs: Read Committed Isolation Level (ON CONFLICT and snapshots)"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html#XACT-READ-COMMITTED"
---

## Concept

A `WITH` clause may contain `INSERT`, `UPDATE`, `DELETE` and `MERGE`, not only
`SELECT`. A data-modifying step usually ends in `RETURNING`, and its returned rows act
as a temporary table that later steps and the main query can read. The main statement
can itself be a write. The result is several writes in one statement: one round trip,
and atomic even in autocommit.

Three rules make it predictable:

- **One snapshot.** Every step and the main query read the database as it was before
  the statement started. A step can't see another step's changes in the tables;
  `RETURNING` is the only channel.
- **Exactly once.** Each data-modifying step runs once and to completion, whether or
  not anything reads its output.
- **Top level only.** A data-modifying `WITH` must belong to the top-level statement.

That is the point of the feature: read what you wrote, in the same atomic statement.
Archive what you deleted, link children to the id you just generated, log the old and
new values of the rows you just changed.

## Coming from C/C++/Java/Python/JS

- **Go/TS with a driver:** the usual version is `BEGIN`, `INSERT ... RETURNING id`,
  read the id into a variable, `INSERT` the children with it, `COMMIT`: four round trips
  and a transaction to manage. The CTE is one statement; the id never leaves the server.
- **Java/JPA:** cascading a persist from a parent to its children sends one `INSERT`
  per entity, and the generated ids travel to the application in between. The ORM
  hides those round trips; the CTE removes them.
- **Triggers:** an audit trigger does the logging invisibly on every update. The CTE
  makes it explicit for the statements that need it, at the cost of being opt-in.
- **Any language with immutable snapshots:** the rule matches reading from a persistent
  data structure: every step got the same version, and the new version exists only in
  what the steps returned.

## Design notes

Use a data-modifying CTE when several writes form one logical operation and the later
ones need the earlier ones' output: archive moves, parent-with-children creation,
update-plus-audit. Make every step return exactly what the next one needs, and make the
main query read from `RETURNING` rather than from the tables.

Keep each row touched by one step. If two steps would modify the same row, merge them
into one statement; the docs leave the outcome unspecified.

Get-or-create with `ON CONFLICT DO NOTHING RETURNING` plus a fallback `SELECT` is the
cheapest form and correct without concurrent writers of the same key. With them, an
empty result means "someone else inserted it after my snapshot": retry, or use
`ON CONFLICT DO UPDATE ... RETURNING`, which always returns the row but writes it.

When a chain grows past three or four steps, or its steps need to branch on results,
move it into a transaction in Go or TS (or a function) where each step can be tested on
its own.
