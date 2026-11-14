---
id: sql.aggregation.grouping-sets
title: "GROUPING SETS/ROLLUP/CUBE, DISTINCT ON"
track: sql
module: aggregation
order: 3
level: advanced
minutes: 14
since: pg9.5
prerequisites: [sql.aggregation.group-by, sql.tables-types.time]
compare_with: [sql.joins.lateral, sql.windows.over, sql.windows.frames]
tags: [aggregation, grouping-sets, rollup, cube, grouping, distinct-on, top-n]
summary: >
  GROUPING SETS computes several GROUP BYs in one pass: ROLLUP adds subtotals
  along a hierarchy, CUBE every combination, and GROUPING() tells a subtotal's
  NULL from a NULL in the data. DISTINCT ON, a PostgreSQL extension, keeps the
  first row of each group in ORDER BY order.
examples:
  - id: rollup
    file: examples/01_rollup.sql
    title: "ROLLUP subtotals, and GROUPING() to label them"
    runtime: sql
    check: exact
    annotations:
      - lines: "17"
        note: >
          ROLLUP (suite, browser) means GROUPING SETS ((suite, browser),
          (suite), ()): every prefix of the list, which fits a hierarchy
          (browsers within a suite). One query returns 5 detail rows, 3 suite
          subtotals and the grand total.
      - lines: "15-18"
        note: >
          A column that isn't part of a row's grouping set comes back NULL. auth
          now has two rows with browser NULL: the API tests (2 runs, 70 ms) and
          the suite subtotal (4 runs, 530 ms). Without GROUPING(browser) in the
          ORDER BY, even their relative order would be unspecified.
      - lines: "22-25"
        note: >
          GROUPING(browser) = 1 marks the rolled-up rows, so coalesce only ever
          sees the NULLs that came from the data. A plain coalesce(browser,
          'all browsers') would have labelled the API tests as a subtotal.
      - lines: "26"
        note: >
          With several arguments GROUPING returns a bitmask, rightmost argument
          in the lowest bit, 1 meaning "not grouped here": 0 is a detail row, 1
          a suite subtotal, 3 the grand total. ROLLUP never produces 2 (browser
          without suite); CUBE does.
      - lines: "30"
        note: >
          Sorting on GROUPING() places each subtotal after its details and the
          total last, whatever NULL ordering is in effect. suite and browser
          here are the input columns, not the labels, which would sort
          'all suites' among the real names.
  - id: sets-cube
    file: examples/02_sets_cube.sql
    title: "GROUPING SETS instead of UNION ALL, CUBE, and the one-scan plan"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-11"
        note: >
          Each set in the list is an independent GROUP BY: per suite, per
          status, and () for the total. The docs define it as the UNION ALL of
          those queries, with NULL in the columns a set doesn't use.
      - lines: "14-17"
        note: >
          CUBE (suite, status) expands to all four subsets, so it adds the
          per-status totals (level 2) that ROLLUP leaves out. With n columns
          that is 2^n groupings; CUBE of five columns asks for 32.
      - lines: "20-23"
        note: >
          MixedAggregate: one Seq Scan feeds a hash table per column (Hash Key:
          suite, Hash Key: status) plus a running total for (). The UNION ALL
          spelling would scan test_runs three times.
  - id: distinct-on
    file: examples/03_distinct_on.sql
    title: "DISTINCT ON: the latest run per test case, ties, and the portable alternative"
    runtime: sql
    check: exact
    annotations:
      - lines: "14"
        note: >
          Two runs of Export in the same second: a tie on ran_at. Real data has
          these (retries, batch imports), and they decide whether the query is
          deterministic.
      - lines: "18-20"
        note: >
          DISTINCT ON (test_case) keeps the first row of each test_case group in
          ORDER BY order. The leading ORDER BY items must be the DISTINCT ON
          expressions; the ones after them pick the winner: newest ran_at, and
          for the tie the higher id. Drop id DESC and the docs call the chosen
          row unpredictable.
      - lines: "23-29"
        note: >
          The inner ORDER BY is tied to the DISTINCT ON groups. To present the
          winners in a different order, wrap the query and sort outside.
      - lines: "32-39"
        note: >
          row_number() (window functions get their own module) numbers the rows
          of each partition in the given order, and rn = 1 keeps the first. It
          is standard SQL, and it turns into "latest three per test" with
          rn <= 3, which DISTINCT ON can't do.
      - lines: "42"
        note: >
          SQLSTATE 42P10. The sort would order rows by ran_at alone, so rows
          of one test case aren't adjacent and "first of the group" has no
          meaning.
use_cases:
  - "Report exports with subtotals and a grand total: per suite and browser, per suite, overall (ROLLUP)"
  - "Dashboard tiles that each need a different breakdown (per status, per suite, total) from one query (GROUPING SETS)"
  - "Cross-tab margins: row totals, column totals and the corner total (CUBE over two columns)"
  - "Latest row per key with DISTINCT ON: last run per test, current price per product, newest address per customer"
strengths:
  - "Several groupings come out of one scan of the data; the UNION ALL equivalent reads the table once per grouping"
  - "GROUPING() distinguishes subtotal rows from NULLs in the data, which no amount of coalesce can"
  - "DISTINCT ON returns whole rows, not just the max: the latest run's status, duration and id together, without joining max(ran_at) back to the table"
  - "Grouping sets, ROLLUP, CUBE and multi-argument GROUPING are standard SQL (features T431-T434 in PostgreSQL's conformance list)"
weaknesses:
  - "Result rows have different meanings in one column set, so every consumer must check GROUPING() or a level column"
  - "CUBE grows as 2^n; a few extra columns produce far more groupings than anyone reads"
  - "DISTINCT ON is a PostgreSQL extension and only keeps one row per group; top-N needs row_number() or LATERAL"
  - "DISTINCT ON sorts its whole input (or reads an index in that order); with few keys and many rows each, LATERAL ... LIMIT 1 over an index reads far less"
pitfalls:
  - title: "Labelling subtotals with coalesce"
    detail: "coalesce(browser, 'all') can't tell a subtotal's NULL from a NULL in the data, so detail rows get mislabelled as totals. Test GROUPING()."
    bad: "coalesce(browser, 'all browsers')"
    good: "CASE WHEN GROUPING(browser) = 1 THEN 'all browsers' ELSE coalesce(browser, 'api') END"
  - title: "DISTINCT ON without a complete tie-breaker"
    detail: "When two rows tie on every ORDER BY item after the DISTINCT ON expressions, either may be kept, and the answer can change between runs. End the ORDER BY with a unique column."
    bad: "SELECT DISTINCT ON (test_case) * FROM test_runs ORDER BY test_case, ran_at DESC"
    good: "SELECT DISTINCT ON (test_case) * FROM test_runs ORDER BY test_case, ran_at DESC, id DESC"
  - title: "Combining ROLLUPs repeats grouping sets"
    detail: "Several grouping items multiply into their Cartesian product, so GROUP BY ROLLUP (a, b), ROLLUP (a, c) contains (a, b) and (a, c) twice and (a) three times, and their rows repeat accordingly. GROUP BY DISTINCT (PG 14) removes the duplicate sets."
    bad: "GROUP BY ROLLUP (suite, browser), ROLLUP (suite, status)"
    good: "GROUP BY DISTINCT ROLLUP (suite, browser), ROLLUP (suite, status)"
  - title: "Expecting the subtotal rows in the right place"
    detail: "Without ORDER BY, grouping-set rows come out in whatever order the plan produces them. Sort on GROUPING() and the grouping columns explicitly."
quiz:
  - q: "Which grouping sets does `GROUP BY ROLLUP (suite, browser)` compute?"
    options: ["(suite, browser), (suite), ()", "(suite, browser), (suite), (browser), ()", "(suite), (browser)"]
    answer: 0
    why: "ROLLUP takes every prefix of its list, down to the empty set. All subsets, including (browser) alone, is CUBE."
  - q: "A ROLLUP result row has browser NULL. How do you tell whether it is a subtotal?"
    options: ["browser IS NULL", "GROUPING(browser) = 1", "count(*) > 1"]
    answer: 1
    why: "GROUPING(browser) is 1 only when browser is not part of the row's grouping set; a NULL in the data gives 0."
  - q: "What does GROUPING(suite, status) return on the grand-total row?"
    options: ["0", "1", "3"]
    answer: 2
    why: "Neither column is grouped, so both bits are set: 2 for suite plus 1 for status."
  - q: "What happens with `SELECT DISTINCT ON (test_case) test_case, passed FROM test_runs ORDER BY ran_at DESC`?"
    options: ["It returns the latest run per test case", "It fails: the DISTINCT ON expressions must match the leftmost ORDER BY expressions", "It returns one arbitrary run per test case"]
    answer: 1
    why: "DISTINCT ON relies on the sort placing each group's rows together, so the ORDER BY must start with test_case (SQLSTATE 42P10)."
further_reading:
  - title: "PostgreSQL docs: GROUPING SETS, CUBE, and ROLLUP"
    url: "https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-GROUPING-SETS"
  - title: "PostgreSQL docs: Grouping Operations (GROUPING)"
    url: "https://www.postgresql.org/docs/current/functions-aggregate.html#FUNCTIONS-GROUPING-TABLE"
  - title: "PostgreSQL docs: SELECT, DISTINCT clause (DISTINCT ON)"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-DISTINCT"
  - title: "PostgreSQL docs: SELECT, GROUP BY clause (GROUP BY DISTINCT)"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-GROUPBY"
---

## Concept

A report rarely wants just one level of summary. It wants runs per suite and browser,
per suite, and overall. Plain `GROUP BY` gives one level per query. **Grouping sets**
(PG 9.5) give several in one result:

- `GROUPING SETS ((a), (b), ())` lists the groupings explicitly. Each set behaves like
  its own `GROUP BY`, and its rows have NULL in the columns the set doesn't use.
- `ROLLUP (a, b)` is shorthand for every prefix, `(a, b), (a), ()`: subtotals down a
  hierarchy.
- `CUBE (a, b)` is shorthand for every subset, `(a, b), (a), (b), ()`: the margins of a
  cross-tab.
- `GROUPING(a, b)` returns a bitmask saying which columns are rolled up in the current
  row, so a subtotal's NULL can be told from a NULL in the data.

**DISTINCT ON** answers a different summary question: not "how many per group" but
"which row per group". `SELECT DISTINCT ON (key) ... ORDER BY key, preference` keeps the
first row of each key in that order: the latest run per test case, with all its
columns. It is a PostgreSQL extension; `row_number()` is the standard equivalent.

## Coming from C/C++/Java/Python/JS

- **Spreadsheets and BI tools:** a pivot table with subtotals and a grand total is
  `ROLLUP`; "totals for rows and columns" is `CUBE` over the two pivot axes.
- **Python:** pandas `pivot_table(..., margins=True)` adds the `All` row and column, as
  `CUBE` over two columns does. "First row per group" is
  `df.sort_values(...).drop_duplicates('test_case')`, which keeps the first occurrence:
  the same sort-then-keep-first algorithm as `DISTINCT ON`.
- **Java, JS, Go:** you would run one grouping pass per level, or add subtotals in a
  second loop. Grouping sets compute every level from one scan.
- **JS:** latest per key is usually a `reduce` into a `Map` that replaces an entry when a
  newer one arrives. That compares two rows at a time; `DISTINCT ON` sorts once and keeps
  the first of each run of equal keys.

## How it works

`GROUP BY GROUPING SETS ((suite), (status), ())` is defined as the `UNION ALL` of three
queries: `GROUP BY suite` with `status` as NULL, `GROUP BY status` with `suite` as NULL,
and the ungrouped total. The executor computes them together: example 2's plan is one
scan feeding a hash table per set (`MixedAggregate`).

Grouping items combine as a Cartesian product: `GROUP BY suite, ROLLUP (browser,
status)` means `GROUPING SETS ((suite, browser, status), (suite, browser), (suite))`.
That is how you get subtotals *within* each suite without a grand total.

`DISTINCT ON` runs after the sort: rows arrive ordered by the `DISTINCT ON` expressions
first, and a `Unique` step keeps the first row of each run of equal keys. That is why
the `ORDER BY` must start with those expressions.

## Design notes

Use grouping sets when the database is the reporting engine: CSV exports, admin
summaries, dashboard tiles. Always return `GROUPING()` (or a level column derived from
it) so the consumer never has to guess what a NULL means, and always sort explicitly.
Prefer `GROUPING SETS` that list exactly the levels you need over `CUBE`, which grows as
2^n.

For "one row per group", pick by shape. `DISTINCT ON` is the shortest when you need
one row per key and run on PostgreSQL only; always end its `ORDER BY` with a unique
column. `row_number()` is portable and extends to top-N. When there are few keys and
many rows per key, `LEFT JOIN LATERAL (... ORDER BY ran_at DESC LIMIT 1)` over an index
on `(test_case, ran_at DESC)` reads one index entry per key instead of every row (see the
LATERAL lesson).
