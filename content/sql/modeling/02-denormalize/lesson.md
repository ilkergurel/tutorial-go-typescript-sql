---
id: sql.modeling.denormalize
title: "When to denormalize; naming conventions, keys and audit columns"
track: sql
module: modeling
order: 2
level: intermediate
minutes: 14
since: pg12
prerequisites: [sql.modeling.normalization, sql.subqueries-ctes.modifying]
compare_with: [sql.constraints.generated, sql.server-side.triggers, sql.server-side.views, sql.modeling.temporal]
tags: [modeling, denormalization, snapshot, naming, audit-columns, conventions]
summary: >
  A stored copy of derivable data is a cache: decide what keeps it right
  before you add it. A value copied at the time of an event is not a copy at
  all. Conventions for names, keys and audit columns make every table
  predictable.
examples:
  - id: derived-copy
    file: examples/01_derived_copy.sql
    title: "A stored 'last result': one write path, how it drifts, and a reconciliation query"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-7"
        note: >
          The copy answers "what happened last" without reading test_runs,
          which matters on a list of thousands of test cases. It is also a
          second place where that fact lives; the rest of the example is
          what that costs.
      - lines: "18-26"
        note: >
          One statement inserts the run and updates the copy, so both commit
          or neither does. The last condition lets the copy only move
          forward: an older run arriving late (a retry, an import) can't
          overwrite a newer result. If two writers race, the second UPDATE
          waits for the first one's row lock and, under READ COMMITTED,
          re-checks this WHERE against the updated row.
      - lines: "29"
        note: >
          Any writer that skips the write path causes drift: bulk imports,
          psql sessions, a second service. Nothing fails. The copy is simply
          wrong from now on.
      - lines: "32-35"
        note: >
          The view defines the copy once, in terms of its source. id DESC
          breaks ties between two runs on the same day, so "latest" is
          deterministic.
      - lines: "37-41"
        note: >
          A reconciliation query: recompute, compare, list the differences.
          IS DISTINCT FROM is NULL-safe; with <> TC-2's NULL against 'passed'
          would be unknown and the row would be missed. Zero rows is the
          invariant: run it on a schedule or in integration tests.
      - lines: "44-48"
        note: >
          The repair reuses the view, so there's one definition of the truth.
          A trigger on test_runs or a materialized view would need no write
          path discipline at all, at their own costs (triggers and views
          lessons).
  - id: snapshot
    file: examples/02_snapshot.sql
    title: "A snapshot is a different fact: the price and KDV rate on the order"
    runtime: sql
    check: exact
    annotations:
      - lines: "11"
        note: >
          The foreign key stays. The copied columns freeze the terms; the
          reference keeps the identity (which product this was). It also
          means a product with orders can't be deleted, only retired with a
          flag.
      - lines: "13-14"
        note: >
          These look like copies of products.price and products.vat_pct, but
          they record the terms of this order. An invoice that changes when
          the price list changes is a bug, and in accounting a legal problem.
      - lines: "16"
        note: >
          A value derived from the same row is the cheap kind of
          denormalization: a generated column is recomputed by the database
          on every write and can't be written by hand, so it can't drift.
      - lines: "22-23"
        note: >
          The INSERT takes the copy itself. The client sends a SKU and a
          quantity, never a price, so a stale screen or a tampered request
          can't set what the customer pays.
      - lines: "26"
        note: >
          products says what is true today. Turkey's general KDV rate rose
          from 18% to 20% in July 2023, and every order from before kept 18.
      - lines: "29-34"
        note: >
          net_if_joined is what a "fully normalized" design reports for order
          1: 298.00 instead of the 258.00 that was charged. Here the join is
          the bug, and the copy is the correct model.
  - id: naming-audit
    file: examples/03_naming_audit.sql
    title: "Naming conventions, generated constraint names, audit columns and identifier folding"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-2"
        note: >
          No single convention matters; consistency does. With id everywhere
          and <singular>_id for references, every join condition is
          predictable: test_cases.project_id = projects.id. Plural table
          names also sidestep reserved words: CREATE TABLE user and CREATE
          TABLE order are syntax errors, users and orders are fine.
      - lines: "13-15"
        note: >
          Audit columns are timestamptz (the time zones lesson explains why).
          now() is the start time of the transaction, so every row written in
          one transaction shares one created_at.
      - lines: "14"
        note: >
          current_user is the database role. Behind a connection pool it is
          the application's role for every request, here playground. Store
          the end user explicitly, as a column the application fills.
      - lines: "19-22"
        note: >
          Generated names follow <table>_<columns>_<suffix>: pkey, key for
          UNIQUE, fkey, check. Errors carry the constraint name, so the API's
          error mapping can key on it; write CONSTRAINT name ... wherever you
          want to choose it. PG 18 also lists NOT NULL constraints here
          (contype 'n'), which the filter hides.
      - lines: "29-30"
        note: >
          updated_at = created_at even after the UPDATE, because a DEFAULT
          applies to INSERT only. Either every UPDATE sets updated_at = now()
          itself, or a BEFORE UPDATE trigger does (triggers lesson).
      - lines: "33-40"
        note: >
          Quoted "TestRuns" keeps its capitals, so every query must quote it
          forever. Unquoted Ürünler shows the folding rule as this UTF-8
          database applies it: Fiyat became fiyat, but the non-ASCII Ü
          stayed upper case.
      - lines: "42"
        note: >
          ürünler with a lower-case ü is a different name: SQLSTATE 42P01.
          Lower-case ASCII snake_case avoids both traps; Turkish belongs in
          the data and in comments, not in identifiers.
use_cases:
  - "Lists that show an aggregate on every row (last result, open defect count, member count), read far more often than written"
  - "Snapshots of the terms at the time of an event: prices and tax rates on order lines, a test case's title in a signed-off test report"
  - "Carrying a parent key down (project_id on test_runs) to filter or index without a join, tied to its source by a composite foreign key"
  - "Audit columns (created_at, created_by, updated_at) on every table, so 'when did this row change' doesn't need a log search"
strengths:
  - "A stored aggregate turns a per-row subquery into a column read, which an ORM-driven list page would otherwise repeat for every row"
  - "A snapshot keeps documents reproducible: an invoice printed again in five years says what it said, with no history tables"
  - "Kept by the same statement (a data-modifying CTE) or by the database (generated column, composite FK with ON UPDATE CASCADE), a copy is as atomic as the write it depends on"
  - "Predictable names and audit columns let tooling, error mapping and a reviewer find their way around any table"
weaknesses:
  - "Every copy needs a write path to keep it right, and imports, psql sessions or a second service that skip it cause silent drift"
  - "Reads get cheaper and writes get dearer: each recorded run now also updates, and row-locks, its test case"
  - "Drift isn't an error. You only find it with a reconciliation query that somebody has to remember to run"
  - "PostgreSQL accepts any name, quoted or not; conventions hold only as far as review and linting enforce them"
pitfalls:
  - title: "Denormalizing before measuring"
    detail: "A join on indexed keys is cheap. Copy a value when EXPLAIN ANALYZE on realistic data shows the join or the aggregate is the problem, not in advance."
  - title: "updated_at with only a DEFAULT"
    detail: "The default fills it on INSERT and never again, so every row looks untouched. Set it in every UPDATE, or let a trigger do it."
    bad: "updated_at timestamptz NOT NULL DEFAULT now()  -- and nothing else"
    good: "UPDATE test_cases SET title = $1, updated_at = now() WHERE id = $2;  -- or a BEFORE UPDATE trigger"
  - title: "current_user as the audit user"
    detail: "Behind a connection pool, every row records the application's database role. Record the person explicitly."
    bad: "created_by text NOT NULL DEFAULT current_user"
    good: "created_by bigint NOT NULL REFERENCES users  -- filled by the application"
  - title: "Quoted and non-ASCII identifiers"
    detail: "ORMs and GUI designers that quote mixed-case names make every hand-written query quote them too, and unquoted Turkish letters don't fold the way the ASCII ones do."
    bad: "CREATE TABLE \"TestRun\" (...);  CREATE TABLE Ürünler (...);"
    good: "CREATE TABLE test_runs (...);  CREATE TABLE products (...);"
  - title: "Identifiers longer than 63 bytes"
    detail: "Longer names are truncated with only a NOTICE, and two names that share their first 63 bytes collide. The limit is in bytes: ç, ğ, ı, ö, ş and ü take two bytes each in UTF-8."
quiz:
  - q: "test_cases.last_result is kept up to date by the endpoint that records runs. A CSV import inserts into test_runs directly. What happens?"
    options: ["The import fails", "last_result silently goes stale", "PostgreSQL recomputes last_result"]
    answer: 1
    why: "Nothing connects the copy to its source except the write path, and the import didn't use it. Only a reconciliation query, a trigger or a materialized view would notice."
  - q: "order_lines.unit_price is copied from products.price when the order is placed. Is that a normalization violation?"
    options: ["Yes: the price depends on the SKU", "No: it is the price agreed for that order, a different fact from today's list price"]
    answer: 1
    why: "unit_price depends on the order line, not just the SKU. Replacing it with a join would reprice old orders."
  - q: "A column is `updated_at timestamptz NOT NULL DEFAULT now()`. A row is updated. What does updated_at hold?"
    options: ["The time of the UPDATE", "Its old value: defaults apply only to INSERT", "NULL"]
    answer: 1
    why: "A DEFAULT is used when an INSERT omits the column (or says DEFAULT). Updates need an explicit SET or a trigger."
  - q: "In a UTF-8 database you run `CREATE TABLE Ürünler (id int)` and then `SELECT * FROM ürünler`. What happens?"
    options: ["It works: unquoted identifiers are case-insensitive", "relation \"ürünler\" does not exist", "A syntax error at Ü"]
    answer: 1
    why: "Folding lower-cased only the ASCII letters, so the table is named Ürünler with a capital Ü, and ürünler is a different name."
further_reading:
  - title: "PostgreSQL docs: Identifiers and Key Words"
    url: "https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS"
  - title: "PostgreSQL docs: Data-Modifying Statements in WITH"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-MODIFYING"
  - title: "PostgreSQL docs: Generated Columns"
    url: "https://www.postgresql.org/docs/current/ddl-generated-columns.html"
  - title: "PostgreSQL docs: Materialized Views"
    url: "https://www.postgresql.org/docs/current/rules-materializedviews.html"
  - title: "PostgreSQL docs: Current Date/Time (now() and transaction time)"
    url: "https://www.postgresql.org/docs/current/functions-datetime.html#FUNCTIONS-DATETIME-CURRENT"
---

## Concept

Denormalizing means storing a fact in more than one place on purpose, usually to make
reads cheaper. Three different things go by that name, and they deserve different
treatment:

- **A derived copy**: a value you could compute from other rows (the last result of a
  test case, a count of open defects). It is a cache. It needs a source of truth, a
  mechanism that keeps it right, and a way to detect when it isn't.
- **A derived value within one row** (a line total from quantity and price). A
  generated column covers this, and the database keeps it right for you.
- **A snapshot**: a value copied at the time of an event (the price on an order line).
  It looks like a copy but records a different fact, and normalizing it away would be
  the bug.

The rest of the lesson is the boring part that makes schemas readable: naming
conventions, key naming, and the audit columns every table should carry.

## Coming from C/C++/Java/Python/JS

- **Caching:** a derived column is a cache inside the database. Everything you know
  from caches applies: invalidation is the hard part, and a stale entry fails silently.
  The difference is that it can be updated in the same transaction as its source, which
  an external cache never can.
- **Java/JPA:** Hibernate's `@Formula` and a `@Transient` getter are the "compute on
  read" choice; a counter field updated in a service method is the "store it" choice,
  and every bulk JPQL `UPDATE` or native query bypasses it.
- **Go and TS:** with pgx or a query builder, the write path is a function you call. Put
  the copy's update in the same SQL statement (a data-modifying CTE), not in a second
  call that a later refactor can drop.
- **Naming:** Go wants `ProjectID`, TS wants `projectId`, SQL wants `project_id`. Keep
  each language's convention and map at the boundary (struct tags, a mapping layer);
  don't quote `"projectId"` into the database.

## Design notes

Before adding a derived copy, measure, then answer four questions in the migration's
comment: what is the source of truth; what keeps the copy right; what detects drift;
what repairs it. The mechanisms, from most to least help from the database:

1. **Generated column**, when the value comes from the same row.
2. **Composite foreign key with `ON UPDATE CASCADE`**, when the copy is a key from a
   parent row (the BCNF example in the previous lesson).
3. **Materialized view**, when a periodically refreshed result is acceptable.
4. **Trigger**, when the copy must change in the same transaction as its source and
   every writer must be covered.
5. **Application write path**, the cheapest to build and the easiest to bypass.

Conventions this codebase follows, as a starting point:

- Lower-case ASCII `snake_case` for every identifier; never quoted names.
- Plural table names (`test_runs`), which also avoids the reserved `user` and `order`.
- `id` as the primary key; `<singular>_id` for foreign keys (`project_id`).
- `_at` for `timestamptz` columns, `_on` for `date` columns, `is_`/`has_` for booleans.
- Explicit `CONSTRAINT` names wherever error mapping depends on them; otherwise the
  generated `_pkey`, `_key`, `_fkey`, `_check` names.
- `created_at`, `updated_at` and, where it matters, `created_by`/`updated_by` holding
  the application's user id. A full change history is a different tool: history tables
  (next lesson) or an audit trigger (triggers lesson).
