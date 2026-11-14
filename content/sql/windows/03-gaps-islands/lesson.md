---
id: sql.windows.gaps-islands
title: "Gaps and islands"
track: sql
module: windows
order: 3
level: advanced
minutes: 13
since: pg8.4
prerequisites: [sql.windows.over, sql.windows.frames]
compare_with: [sql.subqueries-ctes.recursive, sql.files-movement.dump-generate, sql.joins.lateral]
tags: [window-functions, gaps-and-islands, lag, lead, row_number, running-sum, streaks, sessionization]
summary: >
  Ordered rows form islands (runs of consecutive rows) separated by gaps. lead
  finds the gaps; for islands, subtract row_number from a fixed-step sequence,
  or flag each row that starts an island and number the islands with a running
  sum. Then GROUP BY the island number.
examples:
  - id: days
    file: examples/01_days.sql
    title: "Missing days with lead, streaks with day - row_number"
    runtime: sql
    check: exact
    annotations:
      - lines: "13"
        note: >
          lead(day) puts the next existing day on each row. The last day has no
          next one, so next_day is NULL there, NULL > 1 is not true, and the
          WHERE drops the row.
      - lines: "11-12"
        note: >
          date + int, date - int and date - date are day arithmetic. From 11-03
          to 11-06 the hole is 11-04 to 11-05, two days.
      - lines: "21"
        note: >
          Inside a streak, day and rn both grow by 1, so day - rn stays the
          same; a gap of k days shifts it by k. The value (2026-10-31) means
          nothing, only its sameness does. row_number returns bigint and there
          is no date - bigint operator, hence ::int.
      - lines: "26-32"
        note: >
          Grouping on that constant turns each island into one row with its
          first day, last day and length. The same trick works on integer
          sequences such as invoice numbers (n - row_number).
  - id: status-streaks
    file: examples/02_status_streaks.sql
    title: "Status streaks per test: flag the changes, number them with a running sum"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-11"
        note: >
          Çıkış didn't run in builds 102 and 104. The islands are consecutive
          runs of a test, not consecutive build numbers, so subtracting
          row_number from build would split them. This needs the general
          technique.
      - lines: "20-22"
        note: >
          Step 1: a run starts an island when its status differs from the
          previous run's. On each test's first run lag is NULL, and IS DISTINCT
          FROM counts NULL versus 'passed' as different, so island 1 starts
          there.
      - lines: "17"
        note: >
          Step 2: a running sum of the flags. Each 1 opens a new island number
          and the 0s keep the current one. The default RANGE frame is safe
          because build is unique within a test; with ties, peers would share a
          sum.
      - lines: "15"
        note: >
          A view names the steps and lets two queries share them; the sandbox
          rolls it back with everything else. A CTE does the same inside one
          statement.
      - lines: "29-34"
        note: >
          Step 3 is a plain GROUP BY on (test_case, island). current compares
          each island with the test's highest island number, a window over the
          grouped rows. Çıkış has failed three runs in a row, builds 103 to 106:
          the alert you want.
      - lines: "37-40"
        note: >
          SQLSTATE 42P20: window function calls cannot be nested. The flag has to
          be computed one query level down before sum can run over it, which is
          the whole reason the technique has two levels.
  - id: sessions
    file: examples/03_sessions.sql
    title: "Sessions: islands separated by 30 minutes of inactivity"
    runtime: sql
    check: exact
    annotations:
      - lines: "1"
        note: >
          The ::time values below are local times; pinning the time zone keeps
          them the same on every server.
      - lines: "24-26"
        note: >
          The break condition is a time gap: more than 30 minutes since the
          user's previous event opens a new session. On the first event lag is
          NULL, the comparison is NULL, and CASE falls through to ELSE 1. A NULL
          that happens to do the right thing deserves a comment in real code.
      - lines: "21"
        note: >
          The same running sum as in example 2, ordered by (at, id) so that two
          events in the same instant can't become peers and share a number.
      - lines: "16-18"
        note: >
          ayşe's first session lasts 70 minutes although the timeout is 30:
          each event only has to be within 30 minutes of the previous one, so
          sessions chain. emre's events are 31 minutes apart: two sessions of
          length zero.
use_cases:
  - "Missing data: days without a backup or upload, holes in invoice or ticket numbering"
  - "Streaks: consecutive failures of a test (alert at three in a row), consecutive days with activity"
  - "State history as intervals: 'passed builds 101 to 102, failed 103 to 104' from a per-run status log"
  - "Sessionization: splitting clickstreams or API calls into sessions by an inactivity timeout"
  - "Availability reports: outage periods from per-minute health checks"
strengths:
  - "A few sorts and one pass each; the self-join and correlated-subquery versions compare every row with many others"
  - "Flag-and-sum works for any break condition: a changed value, a time gap, a threshold crossed, combined with PARTITION BY"
  - "The result is ordinary rows (from, to, length), ready for joins, filters or another GROUP BY, instead of a stream the application must fold"
  - "The work stays in the database, so an alert query returns two rows, not the full event history"
weaknesses:
  - "Two or three nested query levels are hard to read without named steps (CTEs or views) and comments"
  - "Window functions can't be nested or filtered in place, which is what forces the extra levels"
  - "The row_number subtraction only works on a fixed-step sequence (consecutive integers or days); anything irregular needs flag-and-sum"
  - "Every partition is sorted; on a large event table the sort dominates unless an index supplies the order"
pitfalls:
  - title: "<> instead of IS DISTINCT FROM in the flag"
    detail: "If the compared value can be NULL (a skipped run), <> yields NULL, the CASE reads it as 'no change', and the NULL rows silently join the previous island."
    bad: "CASE WHEN status <> lag(status) OVER w THEN 1 ELSE 0 END"
    good: "CASE WHEN status IS DISTINCT FROM lag(status) OVER w THEN 1 ELSE 0 END"
  - title: "lag inside sum() OVER"
    detail: "Window function calls cannot be nested (SQLSTATE 42P20). Compute the flag in a subquery or CTE, then sum it one level up."
    bad: "sum(CASE WHEN status <> lag(status) OVER w THEN 1 ELSE 0 END) OVER w"
    good: "SELECT ..., sum(starts) OVER w FROM (SELECT ..., CASE WHEN ... lag(status) OVER w ... END AS starts FROM test_runs) AS flagged"
  - title: "Subtracting row_number when values repeat"
    detail: "Two uploads on one day give that day two row numbers, so day - row_number changes in the middle of a streak and splits it. dense_rank numbers distinct values and keeps the key constant."
    bad: "day - row_number() OVER (ORDER BY day)::int"
    good: "day - dense_rank() OVER (ORDER BY day)::int"
  - title: "A running sum over a non-unique order"
    detail: "With the default RANGE frame, rows that tie on the ORDER BY get the same sum, so an island that starts on a tied row can merge with the previous one. Add a unique column to the order."
    bad: "sum(new_session) OVER (PARTITION BY user_id ORDER BY at)"
    good: "sum(new_session) OVER (PARTITION BY user_id ORDER BY at, id)"
quiz:
  - q: "Days 1, 2, 3, 6, 7. What does `day - row_number() OVER (ORDER BY day)` give?"
    options: ["0, 0, 0, 3, 3", "0, 0, 0, 2, 2", "1, 2, 3, 6, 7"]
    answer: 1
    why: "Row numbers are 1 to 5: 1-1, 2-2, 3-3, 6-4, 7-5. The two days missing between 3 and 6 shift the key by 2."
  - q: "Why does `sum(CASE WHEN status <> lag(status) OVER w THEN 1 ELSE 0 END) OVER w` fail?"
    options: ["sum can't be used as a window function", "Window function calls can't be nested; compute the flag in a subquery first", "CASE isn't allowed in a window function's argument"]
    answer: 1
    why: "PostgreSQL rejects a window call inside another window call's argument (SQLSTATE 42P20)."
  - q: "Events at 09:00, 09:20, 09:45 and 10:10, with a 30-minute inactivity timeout. How many sessions?"
    options: ["One, lasting 70 minutes", "Two: 09:00-09:20 and 09:45-10:10", "Three"]
    answer: 0
    why: "Each event is within 30 minutes of the previous one, so no break condition fires; sessions chain."
  - q: "status can be NULL for a skipped run. Which flag starts a new island when NULL follows 'failed'?"
    options: ["status <> lag(status)", "status IS DISTINCT FROM lag(status)", "NOT (status = lag(status))"]
    answer: 1
    why: "Comparisons with NULL yield NULL, which CASE treats as false; IS DISTINCT FROM treats NULL as a comparable value and returns true."
further_reading:
  - title: "PostgreSQL docs: Window Functions (lag, lead, row_number, dense_rank)"
    url: "https://www.postgresql.org/docs/current/functions-window.html"
  - title: "PostgreSQL docs: Comparison Functions and Operators (IS DISTINCT FROM)"
    url: "https://www.postgresql.org/docs/current/functions-comparison.html"
  - title: "PostgreSQL docs: Date/Time Functions and Operators (date arithmetic)"
    url: "https://www.postgresql.org/docs/current/functions-datetime.html"
  - title: "PostgreSQL docs: Set Returning Functions (generate_series)"
    url: "https://www.postgresql.org/docs/current/functions-srf.html"
---

## Concept

"Gaps and islands" names a family of questions about ordered data: where does a sequence
have holes (**gaps**), and which runs of consecutive rows belong together (**islands**)?
The days a nightly upload didn't arrive, how many builds in a row a test has failed,
which clicks make up one session.

A table has no order, so "consecutive" is built from window functions, in two steps:
**mark** each row (compare it with its neighbour, or compute a key that stays the same
along a run), then **group** on the mark to get one row per island. Three techniques
cover nearly every case:

- **Gaps with `lead`:** put the next value on each row; a jump bigger than the step is a
  gap, from `value + 1` to `next - 1`.
- **Subtract `row_number`:** for a fixed-step sequence (integers, days), `value -
  row_number()` is constant inside an island and changes at every gap.
- **Flag and running sum:** set a flag to 1 on each row where an island starts (the
  value changed, too much time passed), then `sum(flag) OVER (ORDER BY ...)` numbers
  the islands. It works for any break condition.

## Coming from C/C++/Java/Python/JS

- **Python:** `itertools.groupby` starts a new group every time the key changes, so it
  groups *consecutive* equal values: islands, in one line, because an iterator has an
  order. SQL needs a window `ORDER BY` to supply the order, and the running sum plays the
  part of groupby's "new group starts here".
- **pandas:** `(s != s.shift()).cumsum()` is flag-and-sum written out: `shift` is `lag`,
  `!=` is the flag, `cumsum` is the running sum. NaN compares unequal to everything, so
  the first row starts a group, the role `IS DISTINCT FROM` plays here.
- **C/C++, Java, Go:** the loop is `if (cur != prev) island++;`. SQL splits those two
  statements across two query levels because a window function can't take another
  window function as its argument.

## How it works

Why `value - row_number` works: inside an island both grow by exactly one per row, so
their difference doesn't move. A gap of `g` missing values makes the value jump by
`g + 1` while the row number still grows by 1, so the difference jumps by `g`. If a value
can appear twice, `row_number` would still grow on the duplicate; `dense_rank` doesn't.

Flag-and-sum, on Giriş from example 2:

```
build    101     102     103     104     105     106
status   passed  passed  failed  failed  passed  passed
starts   1       0       1       0       1       0
island   1       1       2       2       3       3
```

The running sum only grows, and only where an island starts, so every island gets its
own number. From there it is ordinary `GROUP BY test_case, island`.

## Design notes

Name the steps. A CTE or view per step (`flagged`, `numbered`, then the grouped result)
makes the pattern recognizable; three anonymous nested subqueries make it a puzzle.

Default to flag-and-sum. It handles partitions, NULLs (with `IS DISTINCT FROM`), time
thresholds and combined conditions. Keep the `row_number` subtraction for clean
fixed-step sequences, where it is shorter. To list every missing day rather than ranges,
generate the calendar with `generate_series` and anti-join the data to it.

If a dashboard asks for streaks on every page load over a long history, consider
recording the transitions when they happen: a row per status change turns every island
into a stored range, and the window query becomes a lookup. An index on the partition and
order columns (here the primary key, `(test_case, build)`) can supply the sort order.
