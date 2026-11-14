---
id: sql.subqueries-ctes.scalar
title: "Scalar and correlated subqueries; CTEs for readability; MATERIALIZED choice"
track: sql
module: subqueries-ctes
order: 1
level: intermediate
minutes: 14
since: pg12
prerequisites: [sql.aggregation.group-by, sql.joins.many-to-many]
compare_with: [sql.joins.lateral, sql.server-side.views, sql.indexes.plans]
tags: [subqueries, scalar-subquery, correlated-subquery, cte, with, materialized, explain]
summary: >
  A scalar subquery is an expression: it must return at most one row (none
  means NULL), and a correlated one is evaluated per outer row. CTEs name the
  steps of a query; since PG 12 a side-effect-free CTE referenced once is
  inlined, and MATERIALIZED / NOT MATERIALIZED override the planner's choice.
examples:
  - id: scalar-correlated
    file: examples/01_scalar_correlated.sql
    title: "Uncorrelated vs correlated scalar subqueries, and the more-than-one-row error"
    runtime: sql
    check: exact
    annotations:
      - lines: "17-20"
        note: >
          The subquery doesn't refer to the outer query, so it runs once (EXPLAIN
          shows it as an InitPlan) and its result, 357.5, is compared with
          every row. Writing WHERE duration_ms > avg(duration_ms) instead is an
          error: aggregates aren't allowed in WHERE.
      - lines: "24"
        note: >
          t.id comes from the outer row and acts as a constant during each
          evaluation. count without GROUP BY always returns exactly one row, so
          Export CSV gets 0, not NULL.
      - lines: "25-27"
        note: >
          A subquery that finds no row yields NULL: Export CSV has never run.
          ORDER BY ... LIMIT 1 makes "latest" explicit and guarantees at most
          one row, whatever the data looks like.
      - lines: "32-35"
        note: >
          SubPlan 1 hangs under the scan of test_cases: PostgreSQL executes it
          once per outer row, and here each execution scans test_runs. An index
          on test_runs (test_case_id) turns each call into a lookup; for a
          report over every test case, a join with GROUP BY lets the planner
          choose a hash join instead.
      - lines: "39-42"
        note: >
          SQLSTATE 21000, raised at run time by the first outer row whose
          subquery returns two rows. With one run per test in development data
          this query works; it breaks on the data that matters.
  - id: cte-steps
    file: examples/02_cte_steps.sql
    title: "The same query as nested derived tables and as named CTE steps"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-27"
        note: >
          Reading starts at the innermost SELECT (line 18) and works outward,
          and each name (per_case, f) appears after the code it names. One
          more step and the indentation is deeper than the logic.
      - lines: "30-41"
        note: >
          WITH names each step before it is used, so the query reads top to
          bottom, and any step can be checked on its own by selecting from it.
      - lines: "42-45"
        note: >
          Both CTEs are referenced once and have no side effects, so PG 12+
          inlines them: EXPLAIN shows the identical plan for both versions,
          with flaky's WHERE turned into a filter on the aggregate. The names
          cost nothing.
  - id: materialized
    file: examples/03_materialized.sql
    title: "Inlined, MATERIALIZED, referenced twice, NOT MATERIALIZED: four plans"
    runtime: sql
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          Inlined: a single Index Scan on test_runs. The CTE's condition (NOT
          passed) and the outer one (id = 4) were merged, and the primary key
          answers id = 4 directly.
      - lines: "15-17"
        note: >
          MATERIALIZED: the CTE runs on its own (Seq Scan, every failed run),
          its result is stored, and CTE Scan applies id = 4 to that. Before
          PG 12 every CTE behaved like this, which is where the advice "CTEs
          are an optimization fence" comes from.
      - lines: "20-24"
        note: >
          Two references: by default the CTE is materialized, computed once and
          read by two CTE Scans. That pays off when the CTE is expensive and
          each reference needs most of its rows.
      - lines: "27-31"
        note: >
          NOT MATERIALIZED copies the CTE into both places: two Seq Scans of
          test_runs, each with the filter. That pays off when each reference
          needs only a few rows that an index can find, and costs duplicate
          work when it doesn't.
use_cases:
  - "Comparing each row with a table-wide value: runs slower than the average, prices above the category median"
  - "Small per-row lookups in a SELECT list: the latest status, a count, a label from another table"
  - "Reports and migrations written as named steps (WITH) that a reviewer can follow and test step by step"
  - "Forcing or preventing materialization when EXPLAIN ANALYZE shows the planner's default choice is the slow one"
strengths:
  - "A scalar subquery can go anywhere a value can: SELECT list, WHERE, ORDER BY, even inside another expression"
  - "CTEs make multi-step queries read in order, and since PG 12 a single-use CTE costs nothing: same plan as the nested version"
  - "MATERIALIZED and NOT MATERIALIZED give an explicit, documented override instead of rewrite tricks to fool the planner"
  - "Unlike variables in Go or TS, the steps are not executed one by one: the planner optimizes the whole query"
weaknesses:
  - "The at-most-one-row rule is checked at run time, per execution, so a missing LIMIT or aggregate only fails on some data"
  - "A correlated subquery in the SELECT list runs once per outer row; over large outer sets a join or LATERAL can be much cheaper"
  - "A CTE referenced twice is materialized by default, so outer filters can't use the base table's indexes unless you say NOT MATERIALIZED"
  - "MATERIALIZED and NOT MATERIALIZED are PostgreSQL extensions, so portable SQL can't state the choice"
pitfalls:
  - title: "A scalar subquery that can return two rows"
    detail: "It works while every outer row matches at most one row, then fails with SQLSTATE 21000. Make 'at most one' structural: ORDER BY with LIMIT 1, an aggregate, or a lookup by unique key."
    bad: "(SELECT r.passed FROM test_runs r WHERE r.test_case_id = t.id)"
    good: "(SELECT r.passed FROM test_runs r WHERE r.test_case_id = t.id ORDER BY r.run_no DESC LIMIT 1)"
  - title: "Several correlated subqueries for one report"
    detail: "Each scalar subquery in the SELECT list is its own SubPlan, run per outer row: three columns mean three probes per test case. Aggregate once with a join and GROUP BY (or one LATERAL subquery) and select the columns from there."
  - title: "Pre-PG 12 folklore about CTEs"
    detail: "'CTEs are an optimization fence' was true until PG 11. Today a single-use, side-effect-free CTE is inlined. If old code relied on the fence to force an evaluation order, it now needs MATERIALIZED to keep that behaviour."
    bad: "WITH x AS (SELECT ...)  -- relies on the pre-12 fence"
    good: "WITH x AS MATERIALIZED (SELECT ...)"
  - title: "Volatile functions keep a CTE materialized"
    detail: "A CTE that calls random(), nextval() or clock_timestamp() is not side-effect-free, so it is never inlined, even when referenced once. That is correct (inlining could change how often the function runs), but it also means outer filters don't reach the base table."
quiz:
  - q: "A scalar subquery in the SELECT list finds no row for some outer row. What happens?"
    options: ["The outer row is dropped", "The value is NULL", "The query fails with 'more than one row returned'"]
    answer: 1
    why: "A scalar subquery that returns no rows yields NULL; only more than one row is an error."
  - q: "When does `(SELECT r.passed FROM test_runs r WHERE r.test_case_id = t.id)` fail?"
    options: ["Always", "When some test case has more than one run", "When some test case has no runs"]
    answer: 1
    why: "The error is raised at run time by an execution that returns two or more rows. No rows gives NULL."
  - q: "`WITH x AS (SELECT * FROM t WHERE NOT passed) SELECT * FROM x WHERE id = 4`, x referenced once, no volatile functions, PG 18. What does the planner do?"
    options: ["Inlines x, so id = 4 can use the index on t", "Computes x in full, then filters it", "Fails: CTEs can't be filtered"]
    answer: 0
    why: "Since PG 12, a non-recursive, side-effect-free CTE referenced once is folded into the outer query and optimized with it."
  - q: "A CTE is referenced twice and each reference needs one row by primary key. What does NOT MATERIALIZED change?"
    options: ["Nothing; two references always materialize", "The CTE is stored in a temporary table", "Each reference is inlined, so each can use the index instead of scanning a stored result"]
    answer: 2
    why: "NOT MATERIALIZED forces inlining even for multiple references, at the risk of computing the CTE twice."
further_reading:
  - title: "PostgreSQL docs: Scalar Subqueries"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SQL-SYNTAX-SCALAR-SUBQUERIES"
  - title: "PostgreSQL docs: Subquery Expressions (EXISTS, IN, ANY)"
    url: "https://www.postgresql.org/docs/current/functions-subquery.html"
  - title: "PostgreSQL docs: WITH Queries, SELECT in WITH"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-SELECT"
  - title: "PostgreSQL docs: CTE Materialization"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-CTE-MATERIALIZATION"
---

## Concept

A **subquery** is a `SELECT` inside another statement. Used where a single value is
expected, it is a **scalar subquery**: one column, at most one row. No row means NULL;
two rows are a run-time error. A subquery that refers to columns of the outer query is
**correlated**: those columns act as constants during each evaluation, so logically it
runs once per outer row. (`EXISTS` and `IN` subqueries, from the joins module, are
usually turned into semi- and anti-joins by the planner; a scalar subquery in the
SELECT list runs as a per-row SubPlan, as example 1's plan shows.)

A **common table expression** (`WITH name AS (...)`) gives a subquery a name, so a query
with several steps reads top to bottom instead of inside out. Whether a CTE is actually
computed as a separate step is the planner's decision. Since PG 12, a non-recursive CTE
without side effects that is referenced once is **inlined**: folded into the main query
and optimized together with it. Referenced more than once, it is **materialized**:
computed once, stored, and scanned. `MATERIALIZED` and `NOT MATERIALIZED` override that
default.

The goal is to structure queries without fear. Through PG 11 every CTE was materialized,
an "optimization fence", and much of the advice you will find online still dates from
then.

## Coming from C/C++/Java/Python/JS

- **Python/JS:** CTEs look like naming intermediate results
  (`const perCase = ...; const flaky = perCase.filter(...)`). The difference is that
  nothing runs step by step: the planner sees the whole query. Inlined is a lazy,
  fused pipeline (generators, Java streams); materialized is a list built in the middle
  (`list(...)`, `.collect(toList())`).
- **Java Streams:** `MATERIALIZED` is the explicit `.collect()` in the middle of a
  pipeline; `NOT MATERIALIZED` asks the planner to keep fusing even though two consumers
  read the same intermediate result.
- **C/C++:** inlining a function versus calling it once and caching the result; same
  trade-off, decided by the planner unless you say otherwise.
- **Correlated subqueries** are the nested loop you would write in code:
  `for t in cases: n = sum(1 for r in runs if r.case_id == t.id)`. An index on the
  inner column makes each probe cheap; a join plus `GROUP BY` lets the database pick a
  hash join instead of looping.

## How it works

A CTE stays materialized (computed once, then read through `CTE Scan` nodes) when any
of these holds:

- it is referenced more than once (unless `NOT MATERIALIZED`),
- it is written `MATERIALIZED`,
- it is recursive (next lesson) or contains a data-modifying statement (the lesson after),
- it calls a volatile function or locks rows (`FOR UPDATE`).

Otherwise it is inlined like a subquery in `FROM`, and conditions from the outer query
can be pushed down into it and onto its tables' indexes. `EXPLAIN` tells you which
happened: a `CTE Scan` node means materialized.

## Design notes

Write CTEs for any query with more than one logical step, and name each step after
what it contains (`per_case`, `flaky`). The single-use case costs nothing, so the only
question left is the multi-use case: `EXPLAIN` it, and add `NOT MATERIALIZED` when every
reference only needs a few rows an index can find.

Reach for `MATERIALIZED` rarely and deliberately: when `EXPLAIN ANALYZE` shows that
the inlined plan is worse (typically a misestimate the fence happens to fix), or when
old code depended on the pre-12 fence. Leave a comment saying why.

Use scalar subqueries for small, per-row lookups, and make "at most one row" structural
(an aggregate, `ORDER BY ... LIMIT 1`, or a unique key). When a report needs several
values from the same child table, or runs over every row of a large table, aggregate
once with a join and `GROUP BY`, or use one `LATERAL` subquery, instead of one scalar
subquery per column.
