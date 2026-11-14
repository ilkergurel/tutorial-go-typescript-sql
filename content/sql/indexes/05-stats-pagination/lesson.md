---
id: sql.indexes.stats-pagination
title: "Statistics and ANALYZE; keyset vs OFFSET pagination; the N+1 problem; pg_stat_statements"
track: sql
module: indexes
order: 5
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.indexes.plans, sql.indexes.composite]
compare_with: [go.services.pagination-streaming, fullstack.tracelite.step-04-api, ts.services.advanced, go.postgres.querying, sql.security-ops.monitoring]
tags: [statistics, analyze, extended-statistics, pagination, keyset, n-plus-one, pg_stat_statements]
summary: >
  Keep the planner's statistics current and teach it about correlated columns;
  page with a keyset cursor instead of OFFSET; fetch a page's children in one
  statement instead of one per row; and use pg_stat_statements to find which
  queries actually cost the most.
examples:
  - id: statistics
    file: examples/01_statistics.sql
    title: "Estimates vs reality: correlated columns, then a stale table"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-12"
        note: >
          20,000 addresses spread evenly over ten districts, and each district
          lies in exactly one city. Real data is full of such dependencies:
          postcode and city, product and category, tenant and plan.
      - lines: "13"
        note: >
          ANALYZE stores per-column statistics in pg_statistic (readable through
          the pg_stats view): most common values with their frequencies, a
          histogram, the number of distinct values, correlation. Autovacuum
          normally runs it; this sandbox table is invisible to autovacuum.
      - lines: "17-24"
        note: >
          EXPLAIN (FORMAT JSON) returns the plan as data; "Plan Rows" is the
          estimate that text format prints as rows=... when costs are on. The
          helper then counts the real rows. Machine-readable plans are also how
          monitoring tools consume EXPLAIN.
      - lines: "26-30"
        note: >
          Alone, each condition is estimated exactly from the most-common-values
          list: 20% and 10% of the rows. Together the planner multiplies them,
          assuming independence: 0.2 × 0.1 × 20,000 = 400, where the truth is
          2,000. A fivefold underestimate is how a nested loop gets planned for
          an input that turns out large.
      - lines: "32-35"
        note: >
          CREATE STATISTICS asks the next ANALYZE to measure how strongly one
          column determines the other; now the estimate is exact. Functional
          dependencies only help equality and IN conditions with constants;
          the mcv and ndistinct kinds cover combinations and GROUP BY estimates.
      - lines: "37-40"
        note: >
          ANALYZE last saw five cities covering every row, so a sixth is
          estimated at 1 row while 5,000 exist. The planner does scale its row
          counts to the table's current size, but the distribution it knows is
          stale.
      - lines: "41-42"
        note: >
          Fixed. Autovacuum would analyze the table eventually, once enough rows
          have changed, but a job that loads data and queries it right away
          should run ANALYZE itself.
  - id: keyset
    file: examples/02_keyset.sql
    title: "Keyset vs OFFSET: the same page, the cost of getting there, and a moving feed"
    runtime: sql
    check: exact
    annotations:
      - lines: "1"
        note: >
          Pins the session time zone, which decides how timestamptz values and
          constants are printed, so the output is the same on any server.
      - lines: "7-11"
        note: >
          created_at alone isn't unique, so it can't mark a position: id is
          added as a tiebreaker, and the index on (created_at, id), read
          backwards, delivers exactly ORDER BY created_at DESC, id DESC.
      - lines: "14-21"
        note: >
          Identical rows. OFFSET counts positions from the top; the keyset query
          asks for rows that sort after the last one the reader saw, whose sort
          key the client kept as its cursor. The row comparison is
          lexicographic: created_at earlier, or the same created_at and a
          smaller id.
      - lines: "24-26"
        note: >
          actual rows=8005: the index scan produced 8,005 entries so that Limit
          could throw 8,000 of them away. The docs put it plainly: rows skipped
          by OFFSET still have to be computed. The cost grows with the page
          number. (Heap Fetches is the visibility-map effect from lesson 2.)
      - lines: "27-30"
        note: >
          The row comparison became an Index Cond, so the scan starts at the
          cursor and stops after 5 entries: 3 buffers, on page 2 or page 2
          million.
      - lines: "32-35"
        note: >
          OFFSET is positional. One post arriving at the top shifts everything
          down, so page 2 begins with 19998, which the reader already saw on
          page 1. A deleted post would make one silently disappear instead.
      - lines: "36-38"
        note: >
          The keyset query continues right after 19998, whatever happened above
          the cursor in the meantime.
  - id: n-plus-one
    file: examples/03_n_plus_one.sql
    title: "N+1 queries, and the one-statement alternatives"
    runtime: sql
    check: exact
    annotations:
      - lines: "12-17"
        note: >
          1 + N statements: what lazy loading in an ORM, or a Go loop calling
          QueryRow per post, sends. Each one is a cheap index lookup; the cost
          is the round trip and the per-statement overhead, paid N times. A
          50-post page means 51 statements per request.
      - lines: "19-24"
        note: >
          One statement per page. The ids travel as a single array parameter;
          from Go, pgx sends a []int32 as int[]. The SQL text stays the same for
          any page size, so it's one prepared statement and one
          pg_stat_statements entry.
      - lines: "26-31"
        note: >
          Or let the database join: the page and its counts in one query.
          LEFT JOIN keeps posts without comments, and count(c.id) gives 0 for
          them where count(*) would give 1.
      - lines: "33-36"
        note: >
          The per-row shape can hide inside a single statement too: SubPlan 1
          runs once per outer row (loops=3, Index Searches: 3). No round trips,
          so it's far cheaper than N+1 from the application, but it is still N
          lookups. The join form lets the planner choose a hash join when the
          page is large.
use_cases:
  - "Run ANALYZE right after bulk loads, imports and big deletes, before the next query depends on them"
  - "CREATE STATISTICS for columns filtered together that depend on each other: city and district, tenant and plan, country and currency"
  - "Keyset pagination for feeds, infinite scroll, public API cursors and batch jobs that walk a whole table"
  - "Batching child lookups with = ANY($1) or a join in every list endpoint"
  - "pg_stat_statements as the first stop when the database is slow: which normalized queries take the most total time"
strengths:
  - "Statistics make the planner adapt to the data without hints; extended statistics fix the independence assumption for the columns that need it"
  - "Keyset pagination costs the same on every page and stays correct while rows are inserted, unlike OFFSET, which gets slower and shifts"
  - "A batched query costs one round trip for any page size, where N+1 grows linearly with it"
  - "pg_stat_statements aggregates every execution per normalized query, so a 0.05 ms query run a million times shows up next to the one slow report"
weaknesses:
  - "Statistics are approximate and sampled on large tables; they go stale between autovacuum runs, and extended statistics must be declared by hand"
  - "Keyset pagination can't jump to page 37, needs an index matching the ORDER BY exactly, and gets complicated for mixed sort directions"
  - "Batching moves the fan-out into the application: results arrive keyed by parent id and must be stitched back together"
  - "pg_stat_statements needs shared_preload_libraries and a server restart, and isn't available in this sandbox"
pitfalls:
  - title: "Querying right after a bulk load"
    detail: "Until ANALYZE runs, the planner estimates new values from old statistics: 1 row for 5,000 in the example. Analyze in the same job."
    good: "INSERT INTO addresses SELECT ... FROM staging; ANALYZE addresses;"
  - title: "A keyset cursor without a unique tiebreaker"
    detail: "With ORDER BY created_at alone, rows sharing the boundary timestamp are skipped or repeated across pages. Add the primary key to both the ORDER BY and the cursor."
    bad: "WHERE created_at < $1 ORDER BY created_at DESC LIMIT 20"
    good: "WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC LIMIT 20"
  - title: "Comparing the cursor columns separately"
    detail: "created_at < $1 AND id < $2 is not the same as the row comparison: it drops rows with the same created_at and a smaller id, and rows with an earlier created_at but a larger id."
  - title: "N+1 in a Go loop"
    detail: "One QueryRow per item is a round trip per item. Collect the ids, send them as one array, and map the results back by id."
    bad: "for i := range posts { conn.QueryRow(ctx, \"SELECT count(*) FROM comments WHERE post_id = $1\", posts[i].ID).Scan(&posts[i].Comments) }"
    good: "rows, err := conn.Query(ctx, \"SELECT post_id, count(*) FROM comments WHERE post_id = ANY($1) GROUP BY post_id\", ids)"
  - title: "count(*) for 'page X of Y' on every request"
    detail: "An exact total reads the whole matching set each time. Fetch LIMIT n + 1 rows to know whether a next page exists, and show totals only where they are cheap or can be approximate."
quiz:
  - q: "city = 'İstanbul' matches 20% of 20,000 rows and district = 'Kadıköy' 10%. Without extended statistics, how many rows does the planner estimate for both conditions together?"
    options: ["2,000", "400", "4,000"]
    answer: 1
    why: "It assumes independence and multiplies: 0.2 × 0.1 × 20,000. The real answer is 2,000, because Kadıköy is always in İstanbul."
  - q: "`ORDER BY created_at DESC, id DESC LIMIT 5 OFFSET 8000`: how many rows must the server produce?"
    options: ["5", "8,005", "All rows in the table"]
    answer: 1
    why: "Skipped rows still have to be computed; with a matching index the scan can stop after offset + limit rows, but not before."
  - q: "Which condition fetches the next page after cursor ($1, $2) for `ORDER BY created_at DESC, id DESC`?"
    options: ["(created_at, id) < ($1, $2)", "created_at < $1 AND id < $2", "(created_at, id) > ($1, $2)"]
    answer: 0
    why: "Row comparison is lexicographic, matching the sort; the separate comparisons drop valid rows, and > goes the wrong direction for a descending order."
  - q: "In pg_stat_statements, what does an N+1 pattern usually look like?"
    options: ["One query with a huge mean_exec_time", "A simple lookup with an enormous calls count and a tiny mean_exec_time, high in total_exec_time", "A query with many shared_blks_read"]
    answer: 1
    why: "Each execution is fast; the damage is in the count. Sorting by total_exec_time surfaces it."
further_reading:
  - title: "PostgreSQL docs: Statistics Used by the Planner (including extended statistics)"
    url: "https://www.postgresql.org/docs/current/planner-stats.html"
  - title: "PostgreSQL docs: Row Estimation Examples"
    url: "https://www.postgresql.org/docs/current/row-estimation-examples.html"
  - title: "PostgreSQL docs: CREATE STATISTICS"
    url: "https://www.postgresql.org/docs/current/sql-createstatistics.html"
  - title: "PostgreSQL docs: ANALYZE"
    url: "https://www.postgresql.org/docs/current/sql-analyze.html"
  - title: "PostgreSQL docs: LIMIT and OFFSET"
    url: "https://www.postgresql.org/docs/current/queries-limit.html"
  - title: "PostgreSQL docs: Row Constructor Comparison"
    url: "https://www.postgresql.org/docs/current/functions-comparisons.html"
  - title: "PostgreSQL docs: pg_stat_statements"
    url: "https://www.postgresql.org/docs/current/pgstatstatements.html"
---

## Concept

Four habits that keep a working schema fast once real traffic arrives:

- **Statistics.** Every plan rests on row estimates, and every estimate comes from
  `ANALYZE`'s summary of the data. Stale statistics (after a bulk load) and the
  independence assumption (for correlated columns) are the two common ways estimates
  go wrong. Fix the first with `ANALYZE`, the second with `CREATE STATISTICS`.
- **Keyset pagination.** `OFFSET n` makes the server produce and discard `n` rows, and the
  pages shift when rows are inserted or deleted. A keyset cursor, `WHERE (created_at, id)
  < ($1, $2)` with a matching index, starts exactly where the last page ended.
- **No N+1.** Loading a list and then querying once per item turns one request into N+1
  round trips. Fetch children for the whole page at once, with `= ANY($1)` or a join.
- **Measure the workload, not one query.** `pg_stat_statements` records every normalized
  statement's call count, total and mean time, rows and buffers, so you optimize what
  actually costs the most.

## Coming from C/C++/Java/Python/JS

- **ORMs:** N+1 is the classic lazy-loading trap. Hibernate's answer is `JOIN FETCH` or
  batch fetching; Django's is `select_related` (a join) and `prefetch_related` (a second
  query with `IN (...)`); GraphQL servers in JS use DataLoader to batch per request. In Go
  there is usually no ORM, so N+1 is visible: a `QueryRow` inside a `for` loop.
- **REST APIs:** `?page=37&size=20` maps naturally to `OFFSET`; the `next_cursor` tokens of
  many public APIs are keyset pagination with the sort key encoded inside. Clients of
  such an API just pass the token back (the Go and TypeScript services lessons build such
  clients).
- **Profilers:** `pg_stat_statements` is a sampling-free aggregate profiler for the whole
  database, the equivalent of looking at cumulative time per function instead of one
  stack trace.

## How it works

How the planner turns statistics into a row count (the docs' row estimation examples):

- `col = value` with `value` in the most-common-values list: its stored frequency.
- A value not in that list: the frequency left over after the common values, spread
  evenly over the remaining distinct values. If the common values covered every row, the
  leftover is nothing, hence the estimate of 1 row for a city ANALYZE never saw.
- Ranges: the position of the constant within the histogram's equal-frequency buckets.
- `AND`: the selectivities multiplied, unless extended statistics say otherwise.

On large tables `ANALYZE` reads a random sample whose size grows with the statistics
target (`default_statistics_target`, 100 by default; per column with `ALTER TABLE ...
ALTER COLUMN ... SET STATISTICS`). Raise it for skewed columns whose rare values matter.

`pg_stat_statements` must be listed in `shared_preload_libraries` (a server restart),
then created per database with `CREATE EXTENSION pg_stat_statements`. The sandbox
doesn't load it, so here is the query you'd run first, as a sketch:

```sql
SELECT calls,
       round(total_exec_time) AS total_ms,
       round(mean_exec_time::numeric, 3) AS mean_ms,
       rows,
       shared_blks_hit + shared_blks_read AS buffers,
       query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;
```

Constants are replaced by `$1`, `$2`, ... so every execution of the per-post comment
lookup lands in one row: an N+1 shows up as a simple query with a huge `calls` and a tiny
`mean_ms`, near the top of `total_ms`. Since PostgreSQL 18, `IN` lists that differ only
in length are also merged into one entry, shown as `IN ($1 /*, ... */)`. Seeing other
users' query texts needs the `pg_read_all_stats` role; `pg_stat_statements_reset()`
starts a fresh measurement, for example after a deploy.

## Design notes

- Make `ANALYZE` part of every job that bulk-loads or bulk-deletes. Autovacuum catches up
  on its own schedule, which is too late for the job's own next query.
- Declare extended statistics when `EXPLAIN ANALYZE` shows a large gap between estimated
  and actual rows on a multi-column condition, not preemptively.
- Default to keyset pagination for anything a user scrolls or a program iterates; the
  same cursor technique drives batched backfills (`WHERE id > $last ORDER BY id LIMIT
  1000`). Keep `OFFSET` for small, bounded lists where "page 7 of 12" matters. Encode the
  cursor as an opaque token so clients can't depend on its contents.
- Design list endpoints as one query per level: the page, then the children of all its
  rows. TraceLite's API (step 4) builds its list endpoints on keyset pagination.
- Turn on `pg_stat_statements` (and `track_io_timing`) in every environment you care
  about. Rank by `total_exec_time` before tuning anything; the slowest single query is
  often not the most expensive one.
