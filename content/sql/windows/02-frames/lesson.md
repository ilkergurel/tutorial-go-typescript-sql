---
id: sql.windows.frames
title: "Running totals, frames, moving averages, top-N per group"
track: sql
module: windows
order: 2
level: advanced
minutes: 15
since: pg11
prerequisites: [sql.windows.over]
compare_with: [sql.joins.lateral, sql.aggregation.grouping-sets, sql.tables-types.time]
tags: [window-functions, frames, rows, range, running-total, moving-average, top-n, last_value]
summary: >
  The frame is the part of the partition an aggregate or last_value reads. With
  ORDER BY it defaults to "start through the current row's last peer", which
  gives running totals; ROWS counts rows, RANGE measures distance on the value.
  Top-N per group is row_number or rank in a subquery.
examples:
  - id: running-totals
    file: examples/01_running_totals.sql
    title: "Running totals: the default RANGE frame, ROWS, and the last_value trap"
    runtime: sql
    check: exact
    annotations:
      - lines: "19"
        note: >
          With an ORDER BY the default frame is RANGE BETWEEN UNBOUNDED PRECEDING
          AND CURRENT ROW, and in RANGE mode CURRENT ROW means the current row's
          last peer. Runs 2 and 3 share 11-10, so each frame contains both: 8
          and 8, never the 4 in between.
      - lines: "20-21"
        note: >
          ROWS ends the frame at the row itself, so the total climbs one run at a
          time: 3, 4, 8, 10. ROWS UNBOUNDED PRECEDING is short for BETWEEN
          UNBOUNDED PRECEDING AND CURRENT ROW. id makes the order unique; the
          docs warn that ROWS gives unpredictable results when it isn't.
      - lines: "22"
        note: >
          No ORDER BY: every row is a peer of every other, the frame is the whole
          partition, and sum is the project total.
      - lines: "29-30"
        note: >
          first_value, last_value and nth_value read the frame, not the
          partition. The default frame stops at the current row (its last peer,
          and the order here is unique), so last_value returns the row's own
          value. The docs call this result unhelpful.
      - lines: "31-34"
        note: >
          A frame over the whole partition makes last_value the latest run's
          count: 2 for TraceLite on every row. first_value(failed) OVER
          (PARTITION BY project ORDER BY ran_on DESC, id DESC) gives the same
          answer with the default frame.
  - id: moving-average
    file: examples/02_moving_average.sql
    title: "Moving averages: last three builds (ROWS) vs last three days (RANGE)"
    runtime: sql
    check: exact
    annotations:
      - lines: "19"
        note: >
          ROWS BETWEEN 2 PRECEDING AND CURRENT ROW: this build and the two rows
          before it, whatever their dates. At the start of the data the frame is
          shorter, so the first two averages cover one and two builds; n_builds
          makes that visible.
      - lines: "20-21"
        note: >
          RANGE with an offset (PG 11) measures distance on the ORDER BY value:
          rows dated at most 2 days before this one. For a date or timestamp
          column the offset is an interval.
      - lines: "14-17"
        note: >
          They agree until the weekend gap. On 11-09 the three-build window
          reaches back to Thursday's 30-minute outlier and reports 19.0; the
          three-day window finds only Monday itself and reports 14.0.
      - lines: "29-30"
        note: >
          Both ends of a frame may precede the current row: the three builds
          before this one. Leaving the row out keeps a spike from raising its own
          baseline, so 30 minutes is compared with 11.0 and flagged. The first
          build has an empty frame, avg over no rows is NULL, and so is spike.
      - lines: "38-41"
        note: >
          SQLSTATE 42P20. A RANGE offset is a distance between the current
          row's value and other rows' values, which only makes sense for a
          single sort key. ROWS and GROUPS offsets count rows and peer groups and
          have no such restriction.
  - id: top-n
    file: examples/03_top_n.sql
    title: "Top-N per group with row_number and rank, and the run condition in the plan"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-25"
        note: >
          The tutorial's own pattern: number the rows of each partition in a
          subquery, keep rn <= 2 outside. With a unique order every suite gets
          at most two rows; search has only one test and returns one.
      - lines: "31"
        note: >
          rank answers "top 2 including ties": Giriş and Çıkış share second
          place, so auth returns three rows. dense_rank() <= 2 would mean "every
          test with one of the two slowest durations", which can be many more.
      - lines: "35"
        note: >
          The tied rows need an order for display; COLLATE "tr-x-icu" puts Çıkış
          before Giriş as the Turkish alphabet does.
      - lines: "38-46"
        note: >
          Run Condition (PG 15): rn <= 2 moved into the WindowAgg, and the
          Subquery Scan has no Filter left. The Window line says ROWS UNBOUNDED
          PRECEDING although the query never did: row_number ignores the frame,
          so the planner uses the cheaper ROWS mode (PG 16). It lists the order
          columns without direction; the DESC is in the Sort Key.
      - lines: "49-52"
        note: >
          SQLSTATE 42703. An output column's name can be used in ORDER BY and
          GROUP BY but not in WHERE or HAVING, and here the value doesn't even
          exist when WHERE runs. Hence the subquery.
use_cases:
  - "Running totals and cumulative counts: failures to date, burn-down charts, account balances"
  - "Moving averages that smooth noisy series: build times, response times, daily sign-ups over the last 7 days"
  - "Baselines for anomaly detection: compare each value with the average of the N values before it"
  - "Top-N per group: the 3 slowest tests per suite, the latest 5 runs per test case, best sellers per category"
  - "A partition's first or last value next to every row: the latest status shown on each historic run"
strengths:
  - "One sorted pass computes running and moving aggregates; the correlated-subquery version re-reads earlier rows for every row"
  - "sum, count and avg have an inverse transition function, so a sliding frame subtracts the row that leaves instead of recomputing (the docs' moving-aggregate mode)"
  - "RANGE with an interval follows the calendar, so missing days shrink the frame instead of silently stretching it as a fixed row count does"
  - "row_number top-N is standard SQL and works for any N, which DISTINCT ON (N = 1 only) can't; rank and dense_rank make tie handling explicit"
weaknesses:
  - "The default frame surprises: adding ORDER BY turns a total into a running total and last_value into 'this row', with no error"
  - "ROWS frames over a non-unique ORDER BY give unpredictable results, while RANGE treats peers alike but measures values, not rows"
  - "Top-N with row_number still sorts every row of every partition; with few groups and an index on (group, sort key), LATERAL ... LIMIT N reads only N index entries per group"
  - "RANGE offsets need exactly one ORDER BY column, of a type with distance arithmetic such as numbers, dates or timestamps"
pitfalls:
  - title: "Running total over a non-unique order"
    detail: "With the default RANGE frame, all rows of one day show the day's final total. If each row should add its own amount, make the order unique and use ROWS."
    bad: "sum(failed) OVER (PARTITION BY project ORDER BY ran_on)"
    good: "sum(failed) OVER (PARTITION BY project ORDER BY ran_on, id ROWS UNBOUNDED PRECEDING)"
  - title: "last_value with the default frame"
    detail: "The frame ends at the current row, so last_value returns the current row's value. Widen the frame, or flip the order and use first_value."
    bad: "last_value(status) OVER (PARTITION BY test_case ORDER BY ran_at)"
    good: "first_value(status) OVER (PARTITION BY test_case ORDER BY ran_at DESC, id DESC)"
  - title: "'Last 7 days' written as ROWS"
    detail: "ROWS counts rows. When days are missing, six preceding rows reach further back than a week. Use RANGE with an interval, or fill the gaps with generate_series first if a missing day should count as zero."
    bad: "avg(minutes) OVER (ORDER BY built_on ROWS BETWEEN 6 PRECEDING AND CURRENT ROW)"
    good: "avg(minutes) OVER (ORDER BY built_on RANGE BETWEEN '6 days' PRECEDING AND CURRENT ROW)"
  - title: "Partial frames at the start of the data"
    detail: "The first rows of a moving average average fewer values than the window size, without any marker. Return count(*) over the same frame, and hide or flag rows below the full size."
  - title: "Top-N with row_number when ties matter"
    detail: "row_number <= 3 cuts a tie for third place arbitrarily. If tied rows should all appear, use rank() <= 3 and accept more than three rows."
quiz:
  - q: "Two runs share ran_on = 11-10. What does `sum(failed) OVER (ORDER BY ran_on)` show on them?"
    options: ["Each its own running total", "The same total, which includes both runs", "NULL on the second one"]
    answer: 1
    why: "The default frame is RANGE ... CURRENT ROW, which ends at the last peer, so both rows' frames contain both runs."
  - q: "Builds on Thursday, Friday and Monday, none at the weekend. On Monday, which rows does `ROWS BETWEEN 2 PRECEDING AND CURRENT ROW` include?"
    options: ["Monday only", "Monday and the two builds before it, Thursday and Friday", "Saturday, Sunday and Monday"]
    answer: 1
    why: "ROWS counts rows regardless of their dates. RANGE BETWEEN '2 days' PRECEDING AND CURRENT ROW would include only Monday."
  - q: "You need the 3 slowest tests per suite, and every test tied for third place must be included. Which filter?"
    options: ["row_number() <= 3", "rank() <= 3", "dense_rank() <= 3"]
    answer: 1
    why: "rank gives tied rows the same position (1, 2, 3, 3), so all of them pass <= 3. row_number cuts the tie arbitrarily; dense_rank <= 3 returns every test with one of the three slowest durations."
  - q: "Why does `last_value(x) OVER (PARTITION BY p ORDER BY t)` return the current row's x?"
    options: ["last_value ignores ORDER BY", "The default frame ends at the current row's last peer", "last_value only works with ROWS mode"]
    answer: 1
    why: "last_value reads the last row of the frame, and the default frame with ORDER BY stops at the current row's peer group."
further_reading:
  - title: "PostgreSQL docs: Window Function Calls (frame clause)"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-WINDOW-FUNCTIONS"
  - title: "PostgreSQL docs: Window Functions (first_value, last_value and frames)"
    url: "https://www.postgresql.org/docs/current/functions-window.html"
  - title: "PostgreSQL docs: SELECT, WINDOW clause"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-WINDOW"
  - title: "PostgreSQL docs: Moving-Aggregate Mode"
    url: "https://www.postgresql.org/docs/current/xaggr.html#XAGGR-MOVING-AGGREGATES"
---

## Concept

Inside its partition, every window call has a **frame**: the rows that an aggregate, or
`first_value`, `last_value` and `nth_value`, read for the current row. Ranking functions
and `lag`/`lead` ignore the frame and work on the whole partition.

A frame clause has a mode and two ends:
`{ ROWS | RANGE | GROUPS } BETWEEN start AND end`, where each end is
`UNBOUNDED PRECEDING`, `n PRECEDING`, `CURRENT ROW`, `n FOLLOWING` or
`UNBOUNDED FOLLOWING`.

- **ROWS:** `n` counts rows, and `CURRENT ROW` is the row itself.
- **RANGE:** `n` is a distance on the single `ORDER BY` value: a number, or an
  `interval` for dates and timestamps (offsets since PG 11). `CURRENT ROW` includes all
  peers of the row.
- **GROUPS** (PG 11): `n` counts peer groups, sets of rows equal in the `ORDER BY`.

Without a frame clause: no `ORDER BY` means the whole partition; with `ORDER BY` it is
`RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW`, from the start through the current
row's last peer. That default is why `sum(...) OVER (ORDER BY ...)` is a running total.

**Top-N per group** doesn't need frames: number each partition with `row_number()` (or
`rank()`, to keep ties) in a subquery and keep the rows numbered up to N.

## Coming from C/C++/Java/Python/JS

- **pandas:** `groupby(...).cumsum()` is a `ROWS` running total, where ties don't share
  a value. `rolling(3).mean()` is `ROWS BETWEEN 2 PRECEDING AND CURRENT ROW`, except
  that pandas returns NaN until three values exist, while SQL averages whatever the
  frame holds. `rolling('3D')` on a datetime index is the `RANGE` version.
  `sort_values(...).groupby(...).head(2)` is top-N.
- **C/C++, Java, Go:** a running total is an accumulator in a loop. A `ROWS` moving
  average is a ring buffer of the last N values; a `RANGE` frame is the two-pointer
  sliding window, dropping old elements while they are more than the offset behind the
  current one.
- **Everyone:** the frame is per row. Each row gets its own slice of the partition, so
  one `SELECT` can report a running total, a moving average and a whole-partition value
  side by side.

## How it works

The `WindowAgg` node walks the sorted partition and keeps each frame's aggregate state.
When the frame's start moves forward (`2 PRECEDING`), aggregates with an inverse
transition function, such as `sum` and `count`, subtract the row that left instead of
recomputing; the docs call it moving-aggregate mode, and it makes the cost proportional
to the number of rows rather than rows times frame length.

Top-N has a planner optimization of its own. Since PostgreSQL 15 a filter such as
`rn <= 2` over `row_number()`, `rank()`, `dense_rank()` or `count()` becomes a **run
condition** of the `WindowAgg` (example 3's plan); PostgreSQL 16 added `ntile()`,
`cume_dist()` and `percent_rank()` (both versions' release notes). These values only grow
within a partition (`count()` when its frame starts at the partition's first row, as by
default), so once the condition fails no later row of that partition can pass. Per the
executor's source (src/backend/executor/nodeWindowAgg.c, function ExecWindowAgg), the
node then stops computing them: without `PARTITION BY` (and no other window above it) it
stops reading altogether; with one it still reads the rest of each partition, skipping
the window functions, and any Sort below has sorted every row anyway. On PostgreSQL 18.7 over 1,000,000 rows,
`EXPLAIN ANALYZE` showed the Seq Scan under `row_number() OVER ()` with `rn <= 2`
returning 3 rows; with `PARTITION BY` it returned all 1,000,000.

## Design notes

Write the frame clause explicitly whenever an aggregate's window has an `ORDER BY`. The
default is right for a running total over a unique order and surprising everywhere
else; an explicit `ROWS` or `RANGE` tells the next reader which one you meant.

For time series, decide what a missing period means. If no build means "no data", use
`RANGE` with an interval, as example 2 does. If it means zero (no sales that day),
generate the calendar with `generate_series`, `LEFT JOIN` the data to it, and then a
`ROWS` frame over complete days is correct.

For top-N per group there are three tools. `row_number()` in a subquery is portable
and works for any N; `rank()` when ties must stay. `DISTINCT ON` (the grouping-sets
lesson) is the shortest for N = 1. When groups are few and each is large, `LATERAL`
with `ORDER BY ... LIMIT N` over an index on `(group, sort key)` reads N index entries
per group instead of sorting everything (the LATERAL lesson).
