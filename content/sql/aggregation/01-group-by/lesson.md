---
id: sql.aggregation.group-by
title: "GROUP BY, HAVING, FILTER (WHERE ...)"
track: sql
module: aggregation
order: 1
level: basic
minutes: 13
since: pg9.4
prerequisites: [sql.basics.set-based, sql.querying.null-logic, sql.joins.inner-outer]
compare_with: [sql.windows.over, ts.functions.higher-order, go.composite.slices-and-maps]
tags: [aggregation, group-by, having, filter, count, collation]
summary: >
  GROUP BY collapses rows into one row per group, and aggregates summarize each
  group; WHERE filters rows before grouping, HAVING filters groups after.
  FILTER (WHERE ...) restricts the input of one aggregate, so several
  conditional counts come out of a single pass.
examples:
  - id: group-having
    file: examples/01_group_having.sql
    title: "One row per group, what count(*) counts, and WHERE vs HAVING"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-21"
        note: >
          count(*) counts rows, count(duration_ms) only non-NULL values. avg
          skips NULLs too and divides by what it kept: export averages 900 over
          its one timed run, not 450 over two. avg of an integer column returns
          numeric, so search's 77.5 rounds to 78 instead of truncating.
      - lines: "26"
        note: >
          WHERE removes rows before any group exists. export has no failed run,
          so it never becomes a group. An aggregate here is an error (SQLSTATE
          42803): there is nothing to aggregate yet.
      - lines: "28"
        note: >
          HAVING runs on the finished groups, so it can test aggregates: auth
          has one failure and is dropped. It can't use the output name
          failures, because the SELECT list is computed after it; repeat the
          expression.
      - lines: "31-34"
        note: >
          Aggregates without GROUP BY treat all input as one group, and one
          group gives one row even when it is empty: count is 0, sum is NULL.
      - lines: "36-40"
        note: >
          Add GROUP BY and the same empty input makes no groups at all: SELECT 0.
          A report that needs a row for every suite has to start from a list of
          suites and LEFT JOIN the runs (example 3).
      - lines: "43"
        note: >
          A suite has several statuses, and PostgreSQL refuses to pick one.
          Group by status too, count it with FILTER (next example), or, if any
          value really will do, say so with any_value(status) (PG 16).
  - id: filter
    file: examples/02_filter.sql
    title: "FILTER for side-by-side counts, its CASE equivalent, and ratio traps"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-21"
        note: >
          Each FILTER restricts the input of its own aggregate only. Unlike
          WHERE, it doesn't remove rows from the other columns or groups: export
          stays, with failed = 0. One scan produces the whole pivot.
      - lines: "24-26"
        note: >
          The pre-9.4 spelling, still needed on databases without FILTER. It
          depends on an easy-to-miss detail: count(CASE ... END) works because a
          CASE without ELSE yields NULL, which count skips. Add ELSE 0 and it
          counts every row.
      - lines: "33"
        note: >
          count returns bigint, and bigint / bigint is integer division: 2 / 3
          is 0. No error, no warning, a column of zeros.
      - lines: "34"
        note: >
          100.0 is numeric, so the whole expression is computed in numeric.
          round(value, digits) exists for numeric only; with a double precision
          value it fails to resolve.
      - lines: "43-45"
        note: >
          export has no failed run, so the filtered sum sees no input: NULL,
          while count says 0. Every aggregate except count returns NULL on empty
          input; coalesce turns it into the 0 a chart or an API client expects.
  - id: counting-traps
    file: examples/03_counting_traps.sql
    title: "count(*) after a LEFT JOIN, grouping by the key, and Turkish İ/ı in lower()"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-15"
        note: >
          REQ-EXPORT has no tests, so the LEFT JOIN gives it one row padded with
          NULLs. count(*) counts that row (1); count(t.id) counts matched tests
          (0). Count a column of the optional side that is never NULL in real
          rows, its primary key.
      - lines: "18"
        note: >
          Grouping by the primary key makes every other column of requirements
          functionally dependent on it, so r.code may appear in SELECT without
          being grouped (PG 9.1). Group by the identity, not by a label that
          might repeat.
      - lines: "28-31"
        note: >
          This database uses the builtin C.UTF-8 locale, where lower() applies
          Unicode simple case mapping: İ becomes i and I also becomes i. So
          'DİYARBAKIR' turns into 'diyarbakir', which doesn't equal
          'diyarbakır'. Iğdır splits the same way.
      - lines: "34-37"
        note: >
          lower() follows the collation of its input. With tr-x-icu it applies
          Turkish rules (I to ı, İ to i), and each city becomes one group. The
          result carries the collation too, so ORDER BY sorts the keys in
          Turkish order.
use_cases:
  - "Reports and dashboards: runs per suite, failures per requirement, orders per customer per month"
  - "Pivot-style status columns (passed / failed / skipped) with FILTER, in one query instead of three or a client-side pivot"
  - "Data-quality checks with HAVING: duplicate e-mail addresses (HAVING count(*) > 1), requirements with fewer than two tests"
  - "Case-insensitive duplicate reports on names and cities, with the Turkish collation when the text is Turkish"
strengths:
  - "The summary is computed next to the data: one row per group crosses the network, instead of every row being fetched and folded in Go or TS"
  - "FILTER puts several conditional aggregates into one scan and reads like the question; Java's Collectors need a filtering(...) downstream collector per column"
  - "Ambiguous selects are an error (SQLSTATE 42803) rather than an arbitrary pick, and any_value() (PG 16) makes 'any value will do' explicit"
  - "Grouping by a primary key lets you select every column of that table without listing them all in GROUP BY"
weaknesses:
  - "NULL rules differ per aggregate: count gives 0 on empty input, sum, avg, min and max give NULL, and avg divides by the non-NULL count"
  - "Integer arithmetic on counts truncates silently: bigint / bigint is bigint"
  - "WHERE can't see aggregates and HAVING can't see SELECT aliases, so filtered totals repeat their expression or move to an outer query"
  - "Grouping compares text exactly; case-insensitive grouping needs an explicit normalization, and the correct one depends on the language"
pitfalls:
  - title: "count(*) after a LEFT JOIN"
    detail: "The NULL-padded row of an unmatched parent is still a row, so count(*) reports 1 where the answer is 0. Count a NOT NULL column of the optional side."
    bad: "SELECT r.code, count(*) FROM requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id GROUP BY r.id"
    good: "SELECT r.code, count(t.id) FROM requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id GROUP BY r.id"
  - title: "Integer division in ratios"
    detail: "count(*) is bigint, so passed / total is 0 for anything under 100%. Bring a numeric in first. A FILTERed denominator can be 0 (a suite with only skipped runs), so guard it with nullif or the query fails with division by zero."
    bad: "count(*) FILTER (WHERE status = 'passed') / count(*)"
    good: "round(100.0 * count(*) FILTER (WHERE status = 'passed') / nullif(count(*) FILTER (WHERE status <> 'skipped'), 0), 1)"
  - title: "An output name that shadows a column in GROUP BY"
    detail: "GROUP BY accepts output column names, but when an input column has the same name, the input column wins (ORDER BY makes the opposite choice). This query groups by the raw city and returns 'diyarbakır' twice, with no error."
    bad: "SELECT lower(city) AS city, count(*) FROM customers GROUP BY city"
    good: "SELECT lower(city) AS city_key, count(*) FROM customers GROUP BY lower(city)"
  - title: "sum() of nothing is NULL"
    detail: "A group whose FILTER matches no row, or a query with no input, gets NULL from sum, avg, min, max and array_agg; only count returns 0. Use coalesce where the consumer expects a number."
  - title: "Case-insensitive grouping of Turkish text"
    detail: "In the default C.UTF-8 collation lower() maps I to i, so 'DİYARBAKIR' and 'Diyarbakır' land in different groups. PG 18's casefold() doesn't fix it: in this database it leaves İ unchanged ('dİyarbakir'). Name the Turkish collation."
    bad: "GROUP BY lower(city)"
    good: "GROUP BY lower(city COLLATE \"tr-x-icu\")"
quiz:
  - q: "REQ-EXPORT has no test cases. What do count(*) and count(t.id) return for it in `requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id GROUP BY r.id`?"
    options: ["0 and 0", "1 and 0", "1 and NULL"]
    answer: 1
    why: "The LEFT JOIN keeps REQ-EXPORT as one row with NULLs on the test side. count(*) counts that row; count(t.id) skips the NULL."
  - q: "What does `SELECT count(*), sum(duration_ms) FROM test_runs WHERE false` return?"
    options: ["No rows", "One row: 0 and 0", "One row: 0 and NULL"]
    answer: 2
    why: "Without GROUP BY the input forms one group even when it is empty. count of nothing is 0; sum of nothing is NULL."
  - q: "You want one row per suite with its number of failed runs, including suites with no failures. Which do you use?"
    options: ["count(*) FILTER (WHERE status = 'failed') ... GROUP BY suite", "WHERE status = 'failed' ... GROUP BY suite", "Either; they return the same rows"]
    answer: 0
    why: "WHERE removes the rows before grouping, so a suite without failures has no rows left and disappears. FILTER only limits what that one aggregate sees."
  - q: "A suite has 2 passed runs out of 3. What is `count(*) FILTER (WHERE status = 'passed') / count(*)`?"
    options: ["0.67", "0", "1"]
    answer: 1
    why: "Both counts are bigint, and integer division truncates toward zero. Multiply by 100.0 (numeric) first."
further_reading:
  - title: "PostgreSQL docs: Aggregate Functions"
    url: "https://www.postgresql.org/docs/current/functions-aggregate.html"
  - title: "PostgreSQL docs: Aggregate Expressions (FILTER, ORDER BY, DISTINCT)"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-AGGREGATES"
  - title: "PostgreSQL docs: The GROUP BY and HAVING Clauses"
    url: "https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-GROUP"
  - title: "PostgreSQL docs: Collation Support (lower and upper follow the collation)"
    url: "https://www.postgresql.org/docs/current/collation.html"
---

## Concept

`GROUP BY` collapses all rows that share the grouping values into one output row, and
aggregate functions (`count`, `sum`, `avg`, `min`, `max`, ...) summarize the rows of each
group. Once a query groups, every expression in the SELECT list must be one of three
things: a grouping expression, an aggregate, or a column functionally dependent on a
grouped primary key. Anything else would need PostgreSQL to pick one value out of many,
and it refuses.

Two filters sit on either side of the grouping. `WHERE` decides which **rows** go in;
`HAVING` decides which **groups** come out, and can therefore test aggregates.
`FILTER (WHERE ...)` (PG 9.4) is a third, narrower one: it decides which rows a single
aggregate sees, so `count(*) FILTER (WHERE status = 'failed')` next to a plain `count(*)`
gives failures and totals from the same pass.

Most aggregates ignore NULL inputs. `count(*)` counts rows, `count(col)` counts non-NULL
values, and on empty input every aggregate except `count` returns NULL.

## Coming from C/C++/Java/Python/JS

- **Python:** pandas `df.groupby('suite').agg(...)` is the same idea.
  `itertools.groupby` only merges *adjacent* equal keys, so it needs sorted input;
  `GROUP BY` doesn't.
- **Java:** `runs.stream().collect(groupingBy(Run::suite, counting()))`. Each FILTER
  column would be its own `filtering(r -> ..., counting())` downstream collector.
- **JS:** `Object.groupBy(runs, r => r.suite)` (ES2024) builds the arrays and you reduce
  each one. SQL fuses grouping and reducing, and never materializes the groups' rows
  for you to look at.
- **Go:** a `map[string]*stats` filled in a loop, then sorted keys for stable output.
  The database does the same with a hash table or a sort, next to the data.
- In all of these, reading a field from "the group" without reducing it is something
  your code decides (the first element?). SQL makes you choose explicitly: group by it,
  aggregate it, or `any_value()` it.

## How it works

The clauses run in this logical order, which explains most of the rules above:

1. `FROM` and joins build the input rows.
2. `WHERE` keeps the rows for which its condition is true. No aggregates exist yet.
3. `GROUP BY` forms the groups; aggregates (with their `FILTER`s) are computed per group.
4. `HAVING` keeps the groups for which its condition is true.
5. The `SELECT` list is evaluated, output names come into existence.
6. `ORDER BY` sorts, and it may use those output names.

`HAVING` runs before step 5, so it can't refer to `failures`; it repeats `count(*)`.
PostgreSQL lets `GROUP BY` use output names as an extension, but an input column with
the same name takes precedence (see the pitfalls).

## Design notes

Prefer `FILTER` to `CASE` inside aggregates. It states the condition once, can't be
broken by a stray `ELSE 0`, and produces the same single-pass plan. Keep `CASE` for code
that must run on databases without `FILTER`.

Group by identities, not labels: `GROUP BY r.id` and select `r.code`, rather than
`GROUP BY r.code`, which silently merges two requirements if codes ever repeat.

Ratios are numeric work: bring in `100.0` or a cast before dividing, `nullif` any
denominator that can be zero, and round last.

Case-insensitive grouping is a normalization decision, and for Turkish text the default
one is wrong. `lower()` and `upper()` follow the collation of their input, so name it:
`lower(city COLLATE "tr-x-icu")`. Even that assumes Turkish typing: `IZMIR` typed on an
English keyboard becomes `ızmır` under Turkish rules. When the input is mixed, normalize
once on the way in instead of in every query.
