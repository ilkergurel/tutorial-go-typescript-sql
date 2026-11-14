---
id: sql.constraints.fk-check
title: "Foreign keys and ON DELETE choices, CHECK, deferrable constraints"
track: sql
module: constraints
order: 2
level: basic
minutes: 14
since: pg9.0
prerequisites: [sql.constraints.keys, sql.querying.null-logic]
compare_with: [design.architecture.business-rules, go.postgres.transactions, sql.modifying.soft-delete]
tags: [constraints, foreign-key, on-delete, check, deferrable]
summary: >
  A foreign key makes the database refuse orphans and decide what deleting a
  parent does to its children. CHECK states rules about a single row, and
  DEFERRABLE moves a check to the end of the statement or the transaction.
examples:
  - id: on-delete
    file: examples/01_on_delete.sql
    title: "CASCADE, SET NULL and RESTRICT, each where it fits"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          CASCADE models composition: a requirement can't outlive its project.
          The cascaded deletes run inside the parent's DELETE, recursively,
          so one statement can remove a whole tree. Reserve it for children
          that really are part of the parent.
      - lines: "9-10"
        note: >
          SET NULL models a loose association, and the column must be nullable
          for it to work. PG 15+ accepts a column list, ON DELETE SET NULL
          (author_id), so a composite key like (tenant_id, author_id) can keep
          its tenant_id.
      - lines: "15-16"
        note: >
          RESTRICT and the default NO ACTION both refuse to delete a referenced
          row. The difference: NO ACTION is checked at the end of the statement,
          or later if the constraint is deferred, so a statement or transaction
          can still fix things; RESTRICT can never be deferred.
      - lines: "19-22"
        note: >
          The referenced side always has an index (a foreign key must point at
          a primary key or unique constraint). The referencing columns get
          none automatically, and without one every DELETE on projects scans
          all of requirements looking for children.
      - lines: "32-34"
        note: >
          Deleting user 11 sets owner_id to NULL on 101 and 200. Deleting
          project 2 then takes requirement 200 with it. One DELETE each, and
          no application code walked the relationships.
      - lines: "36"
        note: >
          Requirement 100 has a test run, so RESTRICT refuses. PG 18 reports
          this as SQLSTATE 23001 (restrict_violation); NO ACTION, and RESTRICT
          before PG 18, use 23503 (foreign_key_violation), the same code an
          INSERT with a missing parent gets.
  - id: check
    file: examples/02_check.sql
    title: "CHECK constraints, and why NULL slips through them"
    runtime: sql
    check: exact
    annotations:
      - lines: "3"
        note: >
          A column CHECK with an IN list is the lightweight alternative to an
          enum type or a lookup table (compared in the tables and types module).
      - lines: "4"
        note: >
          duration_ms >= 0 rejects negative numbers, but NULL >= 0 is NULL, and
          a CHECK only fails on false. Unknown passes.
      - lines: "7-8"
        note: >
          A table constraint can compare columns of the same row. It can't look
          at other rows or tables: PostgreSQL assumes a CHECK gives the same
          answer for the same row forever, and only re-checks on INSERT and
          UPDATE. Cross-row rules belong in UNIQUE, EXCLUDE or a foreign key.
      - lines: "9-11"
        note: >
          When NULL must not slip through, say so: duration is required unless
          the run was skipped. IS NOT NULL is never unknown, so this CHECK
          means exactly what it says.
      - lines: "14-16"
        note: >
          Row 2 has NULL duration and NULL finished_at: the first two checks
          evaluate to unknown and pass; duration_unless_skipped is true because
          the status is 'skipped'.
      - lines: "20-21"
        note: >
          finished_at is before started_at, so the named table constraint
          fails with SQLSTATE 23514 (check_violation). The name in the message
          is the reason to name table constraints.
  - id: deferrable
    file: examples/03_deferrable.sql
    title: "DEFERRABLE: check at the end of the statement, or at COMMIT"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          A plain UNIQUE in PostgreSQL is checked row by row, so the UPDATE on
          line 9 would fail with 23505 as soon as row 1 moved to pos 2 while
          row 2 still held it. DEFERRABLE INITIALLY IMMEDIATE checks at the end
          of the statement instead, as the SQL standard specifies, at some
          cost in speed.
      - lines: "8-10"
        note: >
          Every row moved one place and the end state is unique, so the
          statement succeeds. This is the shape of every "reorder the list"
          feature.
      - lines: "17-18"
        note: >
          INITIALLY DEFERRED postpones the foreign key check to COMMIT. Each
          transaction starts in that mode; SET CONSTRAINTS changes it for the
          current transaction only.
      - lines: "21-23"
        note: >
          The child row arrives before its parent, which is what imports and
          circular references need. Switching to IMMEDIATE runs the pending
          checks right there. StackLab rolls every script back and never
          reaches COMMIT, so this is how the lesson makes the check happen.
      - lines: "25-27"
        note: >
          The orphan is accepted, and the error surfaces at SET CONSTRAINTS,
          not at the INSERT that caused it. In an application the same error
          would come from COMMIT and abort the whole transaction: that
          distance between cause and error is the price of deferring.
use_cases:
  - "ON DELETE CASCADE for parts of an aggregate: a project's requirements, an order's lines"
  - "ON DELETE SET NULL for optional associations such as owner or assignee"
  - "RESTRICT (or NO ACTION) where history must survive: test runs, invoices, audit rows"
  - "DEFERRABLE INITIALLY DEFERRED for bulk imports in arbitrary order and for circular references"
  - "CHECK for value ranges, allowed states and simple cross-column rules"
strengths:
  - "Unlike JPA's cascade settings, which only run when the ORM performs the delete, ON DELETE actions fire for every client, including a psql session"
  - "The database decides atomically: no window where a concurrent request inserts a child under a parent that is being deleted"
  - "CHECK and FK violations have their own SQLSTATEs (23514, 23503, and 23001 for RESTRICT in PG 18), so the API can tell 'invalid value' from 'missing parent'"
weaknesses:
  - "Referencing columns are not indexed automatically; forgetting the index makes parent deletes scan the child table"
  - "CASCADE can delete far more than the statement names, and nothing in the DELETE says so"
  - "NOT NULL and CHECK can't be deferred, and deferrable constraints can't serve as ON CONFLICT arbiters"
  - "A CHECK sees one row only; rules across rows need UNIQUE, EXCLUDE, a foreign key or a trigger"
pitfalls:
  - title: "CHECK passes on NULL"
    detail: "A CHECK fails only when its expression is false. With a nullable column, NULL makes most expressions unknown, and unknown passes. Add NOT NULL or write the NULL case into the expression."
    bad: "duration_ms int CHECK (duration_ms > 0)  -- NULL accepted"
    good: "duration_ms int NOT NULL CHECK (duration_ms > 0)"
  - title: "No index on the foreign key column"
    detail: "Deleting or re-keying a parent looks for children in the referencing table. Without an index on the referencing columns, that is a sequential scan per deleted parent row."
    bad: "project_id int REFERENCES projects  -- and nothing else"
    good: "project_id int REFERENCES projects; CREATE INDEX ON requirements (project_id);"
  - title: "Reaching for CASCADE by default"
    detail: "CASCADE turns an accidental delete of a parent into silent data loss in every child table. Use it for true composition; prefer RESTRICT or NO ACTION for anything that is evidence or history."
  - title: "Expecting a deferred error at the INSERT"
    detail: "With INITIALLY DEFERRED, the violating INSERT succeeds and COMMIT fails. Code that only checks errors on each statement must also check the commit."
quiz:
  - q: "test_runs.requirement_id references requirements ON DELETE RESTRICT. What happens on DELETE of a requirement that has runs?"
    options: ["The runs are deleted too", "requirement_id is set to NULL", "The DELETE fails", "The DELETE waits until COMMIT"]
    answer: 2
    why: "RESTRICT refuses the delete outright and can't be deferred; PG 18 reports it as SQLSTATE 23001."
  - q: "A column has CHECK (qty > 0) and no NOT NULL. Is a row with qty = NULL accepted?"
    options: ["Yes", "No"]
    answer: 0
    why: "NULL > 0 is NULL, and a CHECK only rejects rows for which the expression is false."
  - q: "An FK is DEFERRABLE INITIALLY DEFERRED. An INSERT creates an orphan. When is the error raised?"
    options: ["At the INSERT", "At the end of the INSERT statement", "At COMMIT, or at SET CONSTRAINTS ... IMMEDIATE if that comes first"]
    answer: 2
    why: "Deferred constraints are checked at commit; switching them to IMMEDIATE checks all pending changes at that moment."
  - q: "Which constraints can be declared DEFERRABLE?"
    options: ["All of them", "UNIQUE, PRIMARY KEY, EXCLUDE and foreign keys", "Only foreign keys", "CHECK and NOT NULL"]
    answer: 1
    why: "NOT NULL and CHECK are always checked immediately, per row; the other four accept DEFERRABLE."
further_reading:
  - title: "PostgreSQL docs: Constraints (foreign keys, check constraints)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: CREATE TABLE (referential actions, DEFERRABLE)"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
  - title: "PostgreSQL docs: SET CONSTRAINTS"
    url: "https://www.postgresql.org/docs/current/sql-set-constraints.html"
---

## Concept

A **foreign key** says "this column's value must exist as a key in that table". The
database refuses rows that point nowhere, and when the parent is deleted, the
`ON DELETE` action decides what happens to the children:

| Action | Child rows when the parent is deleted |
|---|---|
| `NO ACTION` (default) | The delete fails if children remain when the check runs (end of statement, or later if deferred) |
| `RESTRICT` | The delete fails; never deferred |
| `CASCADE` | Deleted too |
| `SET NULL` / `SET DEFAULT` | The referencing columns are set to NULL or their default |

A **CHECK** constraint is a boolean expression over one row, and the row is rejected only
when it evaluates to false. **DEFERRABLE** constraints (UNIQUE, PRIMARY KEY, EXCLUDE and
foreign keys) can be checked later than usual: at the end of the statement
(`INITIALLY IMMEDIATE`) or at COMMIT (`INITIALLY DEFERRED`).

## Coming from C/C++/Java/Python/JS

- Java/JPA: `cascade = CascadeType.REMOVE` and `orphanRemoval` run only when the ORM
  performs the delete. `ON DELETE CASCADE` runs for every client, including a bulk
  `DELETE` statement and a psql session.
- Go/TS: the check-then-insert you would write in a service (`SELECT` the parent, then
  `INSERT` the child) races with a concurrent delete. The foreign key closes that window.
- C++ ownership is a good mental model: CASCADE is a member owned by value, SET NULL is a
  weak pointer that gets reset, RESTRICT refuses to destroy something still referenced.

## How it works

When each constraint is checked:

| Constraint | Checked |
|---|---|
| NOT NULL, CHECK | Per row, as it is written; never deferrable |
| UNIQUE, PRIMARY KEY, EXCLUDE (not deferrable) | Immediately as each row is written |
| Foreign key (not deferrable) | At the end of each statement |
| Any `DEFERRABLE INITIALLY IMMEDIATE` | At the end of each statement |
| Any `DEFERRABLE INITIALLY DEFERRED` | At COMMIT, or at `SET CONSTRAINTS ... IMMEDIATE` |

The first two rows explain the reorder example: a plain UNIQUE sees the intermediate
duplicate, while the deferrable one only sees the final state.

## Design notes

Choose the ON DELETE action from the domain, not from convenience: is the child part of
the parent (CASCADE), associated with it (SET NULL), or evidence that must outlive any
cleanup (RESTRICT)? The same question decides whether you need soft delete instead.

Deferral is a tool for the few places that need it, such as imports and circular
references. Deferred errors arrive at COMMIT, far from the statement that caused them,
so don't make constraints deferrable by default.

PostgreSQL 18 adds `NOT ENFORCED` for CHECK and foreign key constraints: the constraint
is recorded but never checked. That documents a rule the application enforces itself;
it is not a cheaper way to get the guarantee.
