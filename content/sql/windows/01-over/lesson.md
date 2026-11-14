---
id: sql.windows.over
title: "OVER and PARTITION BY: ROW_NUMBER, RANK, DENSE_RANK, LAG/LEAD"
track: sql
module: windows
order: 1
level: intermediate
minutes: 14
since: pg8.4
prerequisites: [sql.aggregation.group-by, sql.subqueries-ctes.scalar]
compare_with: [sql.aggregation.grouping-sets, sql.joins.lateral, sql.windows.frames]
tags: [window-functions, over, partition-by, row_number, rank, dense_rank, lag, lead]
summary: >
  A window function computes a value for each row from a set of related rows,
  its partition, and keeps every row: aggregates with OVER, rankings with
  row_number/rank/dense_rank, neighbours with lag/lead. Windows run after WHERE
  and GROUP BY, so filtering on one takes a subquery.
examples:
  - id: partition
    file: examples/01_partition.sql
    title: "Aggregates OVER (PARTITION BY ...): group figures without losing the rows"
    runtime: sql
    check: exact
    annotations:
      - lines: "16-19"
        note: >
          The baseline: GROUP BY answers "per suite" with one row per suite, and
          the individual tests are no longer in the result.
      - lines: "22-27"
        note: >
          Each OVER (PARTITION BY suite) aggregate sees the rows of its own suite
          and writes the result onto every one of them: five rows in, five rows
          out. OVER () makes the whole result one partition, so all_ms is the
          grand total on every row.
      - lines: "25-26"
        note: >
          A row's share of its group in one pass. Without windows this needs a
          join back to a grouped subquery. 100.0 keeps the division numeric;
          with integers, 100 * 80 / 600 would print 13 instead of 13.3.
      - lines: "32-39"
        note: >
          The inner query computes suite_avg for every row; the outer query sees
          it as an ordinary column and can filter on it. A CTE works the same
          way. PostgreSQL has no clause for filtering window results in place.
      - lines: "42-43"
        note: >
          SQLSTATE 42P20. Logically FROM and WHERE run first, then GROUP BY and
          HAVING, then the window functions, then DISTINCT, ORDER BY and LIMIT.
          When WHERE runs, there is no average yet. The same rule forbids
          window functions in GROUP BY and HAVING.
  - id: ranking
    file: examples/02_ranking.sql
    title: "row_number, rank and dense_rank on ties, and named windows"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-11"
        note: >
          Three auth tests take 300 ms. To a window ordered by duration they are
          peers: rows its ORDER BY can't tell apart. Ties are where the three
          ranking functions differ.
      - lines: "19-20"
        note: >
          rank gives peers the same number and then skips: 1, 2, 2, 2, 5.
          dense_rank doesn't skip, 1, 2, 2, 2, 3: it counts distinct durations.
          Both are fully determined by the data, so they need no tie-breaker.
      - lines: "21-22"
        note: >
          row_number always counts 1, 2, 3, so the peers must be put in some
          order. Without the title the docs say tied rows are numbered in an
          unspecified order, and the numbers can move between runs. COLLATE
          "tr-x-icu" sorts the tie Turkish-alphabetically, Çıkış before Giriş;
          the database default C.UTF-8 compares code points and would put Ç
          after O.
      - lines: "24-25"
        note: >
          The WINDOW clause names a definition once. by_time copies per_suite's
          PARTITION BY and adds an ORDER BY; the row_number call does the same
          with a longer ORDER BY. A copy may add an ORDER BY only if the
          original has none, and never its own PARTITION BY.
      - lines: "29-33"
        note: >
          Windows run after GROUP BY, so they see one row per suite, and their
          ORDER BY may use an aggregate: the suites ranked by total time. An
          aggregate inside a window is fine; a window inside an aggregate is
          not.
  - id: lag-lead
    file: examples/03_lag_lead.sql
    title: "lag and lead: compare each run with the previous and next run"
    runtime: sql
    check: exact
    annotations:
      - lines: "1"
        note: >
          ran_at::date and the printed timestamps depend on the session time
          zone; pinning it keeps the output the same on every server.
      - lines: "25"
        note: >
          PARTITION BY test_case keeps the two histories apart: lag never
          reaches into another test's runs. ORDER BY ran_at defines "previous";
          id breaks a tie between two runs started in the same instant.
      - lines: "20-21"
        note: >
          lag looks one row back. On a test's first run there is no such row,
          so it returns NULL, or the third argument when you give one.
          lag(status, 2) would look two rows back.
      - lines: "22"
        note: >
          Arithmetic on a neighbour's value: the 940 ms failure shows up as +810
          against the day before, and the recovery as -815.
      - lines: "23"
        note: >
          lead looks forward. timestamptz minus timestamptz is an interval
          ('1 day', '1 day 14:35:00'), and each test's last run has no next
          run, hence NULL.
      - lines: "29-36"
        note: >
          Like any window result, prev can only be filtered one level up. The
          passed-then-failed transitions are regressions: Giriş on 11-12. Çıkış
          has never passed, so it never regressed; finding how long it has been
          red is the gaps-and-islands lesson.
use_cases:
  - "A row's share of its group: each test's share of its suite's time, each order's share of the customer's revenue"
  - "Rankings inside groups: slowest tests per suite, leaderboard positions with ties (rank, dense_rank)"
  - "Comparing a row with its neighbour: duration change since the previous run, passed-to-failed transitions, time between events (lag, lead)"
  - "Numbering rows to keep one per key: deduplication and latest-row-per-key with row_number() = 1"
strengths:
  - "Detail rows and group figures in one result: nothing is collapsed as with GROUP BY, and unlike a join back to a grouped subquery the table is read once"
  - "lag and lead compare a row with its neighbour in the query itself, where application code would sort, group and track a `prev` variable by hand"
  - "rank and dense_rank define what happens to ties, so a leaderboard doesn't depend on the order rows happen to arrive in"
  - "Window functions are standard SQL, unlike DISTINCT ON, so the row_number() form of 'first row per group' is portable"
weaknesses:
  - "Window results can't be filtered at the same query level: every 'keep rank <= 3' needs a subquery or CTE"
  - "row_number over a non-unique ORDER BY returns arbitrary numbers for ties, and nothing warns you"
  - "Windows with different PARTITION BY or ORDER BY typically need a sort each; several unrelated windows in one query mean several sorts"
  - "PostgreSQL doesn't implement the standard's IGNORE NULLS option, so 'the previous non-NULL value' takes extra work"
pitfalls:
  - title: "A window function in WHERE"
    detail: "WHERE is evaluated before window functions exist (SQLSTATE 42P20). Compute the value in a subquery or CTE and filter outside."
    bad: "SELECT * FROM test_runs WHERE row_number() OVER (PARTITION BY suite ORDER BY id) = 1"
    good: "SELECT * FROM (SELECT *, row_number() OVER (PARTITION BY suite ORDER BY id) AS rn FROM test_runs) AS t WHERE rn = 1"
  - title: "row_number without a unique ORDER BY"
    detail: "Peers are numbered in an unspecified order, so 'the first row' can differ between runs. End the window's ORDER BY with a unique column."
    bad: "row_number() OVER (PARTITION BY suite ORDER BY duration_ms DESC)"
    good: "row_number() OVER (PARTITION BY suite ORDER BY duration_ms DESC, id)"
  - title: "lag without PARTITION BY"
    detail: "Without a partition, the 'previous run' of one test's first run is the last run of some other test, and the transition looks real."
    bad: "lag(status) OVER (ORDER BY ran_at)"
    good: "lag(status) OVER (PARTITION BY test_case ORDER BY ran_at, id)"
  - title: "Expecting the window's ORDER BY to sort the result"
    detail: "It orders the rows inside the computation. The output happens to follow some window's sort today, and the docs say not to rely on it. Write the query's own ORDER BY."
  - title: "Adding ORDER BY to an aggregate's window"
    detail: "sum(x) OVER (PARTITION BY s) is the group total; sum(x) OVER (PARTITION BY s ORDER BY t) is a running total, because ORDER BY also changes the default frame. The next lesson covers frames."
quiz:
  - q: "Durations, slowest first: 500, 300, 300, 200. What do rank() and dense_rank() give the 200 ms test?"
    options: ["3 and 3", "4 and 3", "4 and 4"]
    answer: 1
    why: "rank skips after a tie (1, 2, 2, 4); dense_rank doesn't (1, 2, 2, 3)."
  - q: "Why does `SELECT * FROM t WHERE rank() OVER (ORDER BY x) <= 3` fail?"
    options: ["rank() needs a PARTITION BY", "Window functions are computed after WHERE, so WHERE can't use them", "rank() is only allowed together with GROUP BY"]
    answer: 1
    why: "Windows are evaluated after WHERE, GROUP BY and HAVING; they're allowed only in the SELECT list and ORDER BY. Filter in an outer query."
  - q: "What does `lag(status) OVER (PARTITION BY test_case ORDER BY ran_at)` return on a test case's first run?"
    options: ["The last status of the previous test case", "NULL", "An error"]
    answer: 1
    why: "lag never crosses a partition boundary; with no row before, it returns its default, which is NULL unless you pass a third argument."
  - q: "What does `sum(duration_ms) OVER ()` put on each row?"
    options: ["The row's own duration", "The total over all rows the query kept after WHERE", "NULL, because there is no PARTITION BY"]
    answer: 1
    why: "An empty OVER () makes all rows one partition, and with no ORDER BY the frame is the whole partition."
further_reading:
  - title: "PostgreSQL docs: Tutorial, Window Functions"
    url: "https://www.postgresql.org/docs/current/tutorial-window.html"
  - title: "PostgreSQL docs: Window Functions (function list)"
    url: "https://www.postgresql.org/docs/current/functions-window.html"
  - title: "PostgreSQL docs: Window Function Calls (syntax)"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-WINDOW-FUNCTIONS"
  - title: "PostgreSQL docs: Window Function Processing"
    url: "https://www.postgresql.org/docs/current/queries-table-expressions.html#QUERIES-WINDOW"
  - title: "PostgreSQL docs: SELECT, WINDOW clause"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-WINDOW"
---

## Concept

A **window function** computes a value for each row from a set of rows related to it,
and unlike `GROUP BY` it returns every row. Writing `OVER (...)` after a call makes it
a window call:

- `PARTITION BY` splits the rows into groups. A window function never looks outside
  the current row's partition; without `PARTITION BY` the whole result is one partition.
- `ORDER BY` inside `OVER` orders the rows within each partition. It defines what
  "previous" and "first" mean and how ranks are assigned. It does not sort the output.

Three families of functions use it:

- **Any aggregate** (`sum`, `count`, `avg`, ...) followed by `OVER`: the group's
  figure, written onto each of its rows.
- **Ranking:** `row_number()` counts 1, 2, 3, ... no matter what; `rank()` gives ties
  the same number and then skips (1, 2, 2, 4); `dense_rank()` gives ties the same
  number without skipping (1, 2, 2, 3).
- **Offsets:** `lag(x, n, default)` and `lead(x, n, default)` read `x` from the row `n`
  positions before or after the current one, within the partition.

Window functions are evaluated after `WHERE`, `GROUP BY` and `HAVING`, and before
`DISTINCT`, `ORDER BY` and `LIMIT`. They are allowed only in the `SELECT` list and the
query's `ORDER BY`; to filter on one, compute it in a subquery or CTE.

## Coming from C/C++/Java/Python/JS

- **Python (pandas):** `df.groupby('suite')['ms'].transform('sum')` is
  `sum(ms) OVER (PARTITION BY suite)`: a group value broadcast back onto each row.
  `groupby(...).rank(method='min')`, `method='dense'` and `method='first'` are `rank`,
  `dense_rank` and `row_number`; `groupby(...).shift(1)` is `lag`.
- **Java, JS, Go:** there is no built-in equivalent. You would group rows into a map
  of lists, sort each list, then loop with an index and a `prev` variable. The window
  does the grouping, the sort and the reset at each group boundary for you.
- **C/C++:** `lag` and `lead` are the `a[i-1]` and `a[i+1]` of a loop over a sorted
  array, with a bounds check that returns a default, restarted for every group.
- **Everyone:** the window's `ORDER BY` is not the result's `ORDER BY`. It decides the
  computation, like the comparator you pass to a sort inside a group-by. The output
  order is set only by the query's own `ORDER BY`.

## How it works

The executor sorts the rows by the partition and order columns and walks them once in
a `WindowAgg` step, resetting its state at each partition boundary. Window functions
with equivalent `PARTITION BY` and `ORDER BY` share that pass, and the docs guarantee
they see the same row order even when the `ORDER BY` has ties. A window with a
different definition typically needs another sort, and that sort is not guaranteed to
keep tied rows in the same order: one more reason to make `row_number`'s order unique.

The order of evaluation explains the restrictions. Rows are filtered (`WHERE`), grouped
(`GROUP BY`, `HAVING`), and only then windowed. So a window may rank groups by an
aggregate (example 2), but `WHERE` and `HAVING` can never see a window's result.

## Design notes

Use `GROUP BY` when you want one row per group, and a window when you need the detail
rows and the group figures together, or a row compared with its neighbour. `lag` and
`lead` replace the old self-join on "the previous row", which needed a correlated
`max(ran_at) ... < r.ran_at` lookup per row.

Pick the ranking function by what a tie should mean. `rank` or `dense_rank` when ties
must stay visible (a leaderboard, "the slowest tests" where two are equally slow);
`row_number` with a unique tie-breaker when you need exactly one row per position
(deduplication, "the latest run per test").

Filters belong on one side of the window or the other, on purpose. A condition in
`WHERE` changes what the window sees: `WHERE status = 'failed'` before `lag` makes
"previous" mean "previous failure". A condition on the window's result goes in the
outer query. Name shared definitions in a `WINDOW` clause, so several functions
provably use the same partition and order.
