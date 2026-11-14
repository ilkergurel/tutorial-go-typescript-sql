---
id: sql.server-side.views
title: "Views and materialized views"
track: sql
module: server-side
order: 1
level: intermediate
minutes: 14
since: pg9.4
prerequisites: [sql.aggregation.group-by, sql.indexes.plans]
compare_with: [sql.subqueries-ctes.scalar, sql.modeling.denormalize, sql.security-ops.rls, sql.transactions.row-locks]
tags: [views, materialized-views, updatable-views, check-option, refresh, locking]
summary: >
  A view is a named query that the planner expands into every query using it:
  no stored rows, never stale. A materialized view stores the result and is
  exactly as fresh as its last REFRESH; CONCURRENTLY keeps readers unblocked
  during the refresh, at the price of a unique index and more work.
examples:
  - id: named-query
    file: examples/01_named_query.sql
    title: "A view is a stored query: expanded by the planner, frozen columns, pinned types"
    runtime: sql
    check: exact
    annotations:
      - lines: "19-25"
        note: >
          The view stores this SELECT, not its result. Every query on
          requirement_coverage runs it against the current tables, so the view
          can't be stale, and "coverage" is defined once instead of being
          copied into Go handlers, TS reports and ad-hoc psql sessions.
      - lines: "27"
        note: >
          Callers filter and sort the view like a table. REQ-AUDIT has no tests
          and still shows up with 0: the LEFT JOIN keeps it, and count(t.id)
          counts only matched test cases where count(*) would say 1.
      - lines: "30"
        note: >
          No "compute the whole view, then filter" step. The planner
          substitutes the definition as a subquery and pushes code =
          'REQ-LOGIN' through the GROUP BY (code is a grouping column) down to
          an index scan on requirements. A plain view costs what the query it
          expands to costs.
      - lines: "33-35"
        note: >
          The * became id, code, title when the view was created. The new
          priority column is invisible through the view until you recreate it.
      - lines: "36"
        note: >
          pg_get_viewdef decompiles the stored parse tree; it isn't your
          original text. The column list is spelled out, which is why it can't
          grow on its own.
      - lines: "39"
        note: >
          SQLSTATE 0A000. The view depends on test_cases.status, so its type
          can't change while the view exists. In a migration, drop the view,
          change the column and recreate the view in one transaction.
  - id: updatable
    file: examples/02_updatable.sql
    title: "Writing through a simple view, and WITH CHECK OPTION"
    runtime: sql
    check: exact
    annotations:
      - lines: "11-12"
        note: >
          One table in FROM, and no aggregates, DISTINCT, GROUP BY, LIMIT or
          set operations: the view is automatically updatable. PostgreSQL turns
          INSERT, UPDATE and DELETE on it into the same commands on test_cases,
          with no trigger or rule to write.
      - lines: "15-16"
        note: >
          The view's WHERE is added to the UPDATE, so project 2's failed case
          (id 3) is untouched. A view is a narrow interface to a table for
          writes as well as reads.
      - lines: "19-20"
        note: >
          The insert succeeds, but the row belongs to project 2 and the view
          can't see it. Without a check option a view filters reads, not
          writes.
      - lines: "23-25"
        note: >
          CREATE OR REPLACE VIEW may change the query and the check option as
          long as the existing columns keep their names, types and order. A
          bare WITH CHECK OPTION means CASCADED: the conditions of any views
          this one is built on are checked as well.
      - lines: "26"
        note: >
          Allowed: the updated row still satisfies project_id = 1.
      - lines: "27"
        note: >
          SQLSTATE 44000 (with_check_option_violation). Moving a row out of
          the view is now an error instead of a silent disappearance.
  - id: materialized
    file: examples/03_materialized.sql
    title: "Stale until refreshed; REFRESH blocks readers, CONCURRENTLY doesn't"
    runtime: sql-multisession
    setup: |
      CREATE TABLE test_runs (
        id          int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        requirement text    NOT NULL,
        passed      boolean NOT NULL
      );
      INSERT INTO test_runs (requirement, passed) VALUES
        ('REQ-LOGIN', true), ('REQ-LOGIN', false), ('REQ-EXPORT', true);
    check: exact
    annotations:
      - lines: "2-5"
        note: >
          The query runs now and its 2 result rows are stored like a table's
          (hence the command tag SELECT 2). From here on, reads come from the
          stored rows, whatever happens to test_runs.
      - lines: "6"
        note: >
          Indexes work as on a table. This unique index is also what REFRESH
          ... CONCURRENTLY requires: unique, on plain columns (no expressions),
          no WHERE clause.
      - lines: "9-10"
        note: >
          The third REQ-LOGIN run is committed, yet pass_rates still says 2
          runs. A materialized view is exactly as fresh as its last refresh;
          that is the whole trade.
      - lines: "13-16"
        note: >
          A plain REFRESH takes ACCESS EXCLUSIVE, so even B's SELECT waits
          until A commits, then sees the new numbers. On a big view, readers
          wait for the refresh's whole runtime.
      - lines: "21-24"
        note: >
          CONCURRENTLY takes EXCLUSIVE, which still admits SELECT: B reads the
          old rows without waiting. It builds the new result, matches it to
          the stored rows on the unique index's columns and applies only the
          differences. Two refreshes of one view still can't run at once.
      - lines: "25"
        note: >
          After A commits, B sees the refreshed rows. Readers got the old
          version or the new one, never a mix and never an empty view.
use_cases:
  - "One shared definition of a business metric (coverage, pass rate, open defects) that every report, endpoint and BI tool queries instead of re-implementing"
  - "A stable read interface while tables are refactored: split or rename columns behind the view, keep its shape for old clients"
  - "A writable slice of one table (one project's test cases) WITH CHECK OPTION, for a tool that must not touch other rows"
  - "Materialized: dashboards over expensive aggregations that may be minutes old, and local copies of slow foreign data (the docs' file_fdw example)"
strengths:
  - "Unlike a query-building helper in Go or TS, a view works for every client: psql, BI tools, other services, other languages"
  - "A plain view is free at runtime: the planner inlines it and optimizes the combined query, filters and indexes included"
  - "Simple views are writable without triggers, and WITH CHECK OPTION turns the view's WHERE into a write constraint"
  - "A materialized view turns an expensive aggregation into an indexed table read, and CONCURRENTLY refreshes it without blocking readers"
weaknesses:
  - "Views pin the schema: a column a view uses can't change type, and CREATE OR REPLACE VIEW can only append columns, so migrations drop and recreate views"
  - "Views built on views hide the real query: one short SELECT can expand into a dozen joins that only EXPLAIN reveals"
  - "A materialized view has no automatic refresh: it is stale until something you schedule runs REFRESH"
  - "Every refresh recomputes the whole query, even if one row changed; CONCURRENTLY applies only the differences but is slower than a plain refresh when much changes"
pitfalls:
  - title: "SELECT * in a view"
    detail: "The column list is fixed when the view is created; columns added to the table later don't appear. List the columns: the view is a contract, write it down."
    bad: "CREATE VIEW requirement_list AS SELECT * FROM requirements;"
    good: "CREATE VIEW requirement_list AS SELECT id, code, title FROM requirements;"
  - title: "Treating a view as a write boundary without CHECK OPTION"
    detail: "INSERT and UPDATE through a view can create or move rows outside it. If the view is the only thing a tool may write through, add WITH CHECK OPTION."
    bad: "CREATE VIEW project1_cases AS SELECT * FROM test_cases WHERE project_id = 1;"
    good: "CREATE VIEW project1_cases AS\n  SELECT id, project_id, title, status FROM test_cases WHERE project_id = 1\n  WITH CHECK OPTION;"
  - title: "Plain REFRESH on a view that is read all day"
    detail: "Readers queue behind the ACCESS EXCLUSIVE lock for the whole refresh. Add a unique index and refresh CONCURRENTLY. Keep the plain form for the first population (CONCURRENTLY needs a populated view) and for refreshes that replace most rows."
    good: "CREATE UNIQUE INDEX ON pass_rates (requirement);\nREFRESH MATERIALIZED VIEW CONCURRENTLY pass_rates;"
  - title: "Relying on ORDER BY inside a materialized view"
    detail: "The docs say REFRESH does not guarantee to preserve the defining query's order. Sort in the query that reads the view."
  - title: "Assuming a view checks the caller's privileges"
    detail: "By default a view reads its tables with the view owner's privileges, which makes it a way to grant access to a slice of a table. Set security_invoker = true (PG 15) when the caller's privileges and row-level security policies must apply, and use security_barrier for a view that filters rows for less trusted roles."
quiz:
  - q: "A view was created as `SELECT * FROM requirements`. You then add a column to `requirements`. What does the view return?"
    options: ["All columns, including the new one", "The original columns only", "An error until the view is recreated"]
    answer: 1
    why: "The * is expanded to a column list when the view is created; later columns are not part of the view."
  - q: "A dashboard reads a materialized view constantly. An hourly job runs a plain REFRESH MATERIALIZED VIEW. What do dashboard queries do during the refresh?"
    options: ["Read the old rows", "Wait until the refresh commits", "Read a half-refreshed view"]
    answer: 1
    why: "Plain REFRESH holds ACCESS EXCLUSIVE, which conflicts with SELECT. CONCURRENTLY takes EXCLUSIVE, which lets reads through."
  - q: "What does REFRESH MATERIALIZED VIEW CONCURRENTLY require?"
    options: ["Nothing beyond the view itself", "A unique index on plain columns without WHERE, and an already populated view", "Superuser privileges"]
    answer: 1
    why: "The unique index is how the refresh matches new rows to stored ones; an unpopulated view has nothing to match against."
  - q: "`CREATE VIEW v AS SELECT * FROM test_cases WHERE project_id = 1;` then `INSERT INTO v (id, project_id, title, status) VALUES (9, 2, 'x', 'failed')`. What happens?"
    options: ["It is rejected", "The row is inserted but isn't visible through v", "The row is inserted with project_id rewritten to 1"]
    answer: 1
    why: "Without WITH CHECK OPTION, writes through a view aren't checked against its WHERE."
further_reading:
  - title: "PostgreSQL docs: CREATE VIEW (updatable views, CHECK OPTION, security_invoker)"
    url: "https://www.postgresql.org/docs/current/sql-createview.html"
  - title: "PostgreSQL docs: Materialized Views"
    url: "https://www.postgresql.org/docs/current/rules-materializedviews.html"
  - title: "PostgreSQL docs: REFRESH MATERIALIZED VIEW"
    url: "https://www.postgresql.org/docs/current/sql-refreshmaterializedview.html"
  - title: "PostgreSQL docs: Explicit Locking (table-level lock modes)"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html"
  - title: "PostgreSQL docs: Rules and Privileges (security_barrier views)"
    url: "https://www.postgresql.org/docs/current/rules-privileges.html"
---

## Concept

A **view** is a named `SELECT`. It stores no rows. When a query mentions the view, the
planner substitutes the view's definition and optimizes the combined query, so a view
can never be out of date and costs what the expanded query costs. Its job is to give a
query one name and one definition, "coverage" or "active test cases", that psql, BI
tools and every service share.

A **materialized view** runs its query once and stores the result like a table, indexes
included. Reads are cheap table reads. The price is freshness: the data is as current as
the last `REFRESH MATERIALIZED VIEW`, which re-runs the whole query.

- Plain `REFRESH` replaces the contents under an ACCESS EXCLUSIVE lock: readers wait.
- `REFRESH ... CONCURRENTLY` (9.4+) builds the new result, compares it with the stored
  rows on a unique index's columns and applies the differences, while readers keep
  seeing the old rows.

Simple views (one table, no aggregates, `DISTINCT`, `GROUP BY`, `LIMIT` or `UNION`) are
**automatically updatable**: writes to the view become writes to the table, restricted
to the view's rows. `WITH CHECK OPTION` also requires every written row to satisfy the
view's `WHERE`.

## Coming from C/C++/Java/Python/JS

- **A query-building helper** (a Go function returning SQL, a Django manager method, a
  Prisma or Knex helper) is the app-side version of a view. The view serves every
  client, including psql and reporting tools, and the planner optimizes through it.
- **C macros:** a view is expanded where it's used, like a macro, rather than called
  like a function. Nesting views is nesting macros: easy to write, hard to read
  expanded.
- **Caches (Redis, `@Cacheable`, memoization):** a materialized view is a cache inside
  the database with manual invalidation. `REFRESH` is the rebuild; there's no TTL and no
  automatic invalidation on writes.
- **An interface in front of a class:** a view with an explicit column list is a stable
  contract while the tables behind it are refactored.

## How it works

A view is stored as a parse tree, not as text (`pg_get_viewdef` decompiles it). The
tree refers to columns themselves rather than to their names, which explains the two
surprises in example 1: `*` was resolved once at creation, and a column the view uses
can't change type underneath it. `ALTER TABLE ... RENAME COLUMN` is fine: the view
follows the rename and keeps its own output column name.

Writes through an updatable view are rewritten into writes on the base table with the
view's `WHERE` added. Anything more complex (a join, an aggregate) is read-only unless
you write `INSTEAD OF` triggers for it.

A materialized view is a table plus the stored query. Plain `REFRESH` replaces the
contents and holds ACCESS EXCLUSIVE until commit. `CONCURRENTLY` takes EXCLUSIVE, which
still admits plain `SELECT`, and brings the stored rows in line with the new result in
two set-based statements. The docs state only the unique-index requirement; the source
(src/backend/commands/matview.c, function refresh_by_match_merge) shows the method: a full
outer join of old and new rows, on the unique index's columns and the whole row, finds
rows only on one side; those only in the old result are deleted, then those only in the
new one inserted. A changed row is therefore deleted and re-inserted, never updated, and
unchanged rows aren't touched. On PostgreSQL 18.7, after one base row changed, only the
matching view row got a new `ctid`, and the refresh added one delete, one insert and no
update to the view's row counters (`pg_stat_xact_user_tables`).
Neither form is incremental: both run the full query every time, and the docs note that
`CONCURRENTLY` may be faster when few rows change and slower when many do. A view created
`WITH NO DATA` (or refreshed that way) can't be read at all until its first refresh:
SQLSTATE 55000, "has not been populated".

## Design notes

Create a view when a query's definition is a rule that several consumers must agree on,
not to save typing in one handler. List columns explicitly, keep stacks of views shallow
(one or two levels), and put view definitions in the same migrations as the tables they
read: a column type change becomes "drop view, alter, recreate view" in one transaction.

Use views as an access layer deliberately. By default a view reads its tables with its
owner's privileges, so granting `SELECT` on a view grants exactly that slice of data. When
the caller's own privileges and row-level security policies should apply instead, create
it `WITH (security_invoker = true)` (PG 15+); the security module covers both patterns.

Reach for a materialized view only after a plain view with the right indexes is too slow,
and decide the acceptable staleness up front. Refresh it from a scheduled job (TraceLite's
job table with `FOR UPDATE SKIP LOCKED` is a natural home), switch to `CONCURRENTLY` as
soon as readers notice the lock, and watch refresh duration as the data grows. If the
result must be current after every write, a materialized view is the wrong tool: keep a
summary table updated in the same transaction as the writes, by the application or by a
trigger (lesson 3).
