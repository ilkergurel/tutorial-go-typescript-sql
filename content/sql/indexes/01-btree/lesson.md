---
id: sql.indexes.btree
title: "How B-trees work; EXPLAIN vs EXPLAIN ANALYZE with buffers"
track: sql
module: indexes
order: 1
level: intermediate
minutes: 14
since: pg18
prerequisites: [sql.basics.set-based, sql.constraints.keys]
compare_with: [go.performance.pprof, go.testing.benchmarks-fuzz, sql.security-ops.monitoring]
tags: [indexes, btree, explain, explain-analyze, buffers, planner]
summary: >
  A B-tree finds a row in a few page reads at any table size, but only EXPLAIN
  tells you whether the planner uses it, and only EXPLAIN ANALYZE, which really
  runs the statement, tells you what it cost: rows, loops and pages touched.
examples:
  - id: explain-vs-analyze
    file: examples/01_explain_vs_analyze.sql
    title: "EXPLAIN predicts, EXPLAIN ANALYZE executes and measures"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-9"
        note: >
          20,000 orders, 20 per customer, and customer 42's rows are 1,000 ids
          apart, so each sits on a different page. ANALYZE the command collects
          the statistics the planner estimates from; it has nothing to do with
          EXPLAIN's ANALYZE option below. In production autovacuum runs it for
          you; this sandbox table is uncommitted and invisible to autovacuum.
      - lines: "12-13"
        note: >
          Seq Scan with a Filter: read every row, test the condition on each.
          COSTS OFF hides the estimates (cost, rows, width) so the output is
          stable; "How it works" reads a full cost line.
      - lines: "15"
        note: >
          A B-tree on customer_id: the keys in sorted order, each pointing at
          the row's physical location (its TID, block and offset) in the table.
      - lines: "18-19"
        note: >
          A bitmap scan, in two steps: the Bitmap Index Scan collects the
          locations of all matching rows, then the Bitmap Heap Scan reads those
          table pages in physical order, each once. Recheck Cond is the
          condition it re-tests when it has to, for example when the bitmap
          grew too large to remember individual rows.
      - lines: "22-23"
        note: >
          Now with actuals. rows=20.00 loops=1: PG 18 prints row counts with
          two decimals because they are averages per loop. Heap Blocks:
          exact=20 is the 20 rows on 20 pages. Buffers counts 8 kB pages: hit
          means found in PostgreSQL's shared buffer cache, read means fetched
          from the operating system (its file cache or the disk). Nothing had
          read the new index's two pages through the cache yet; in the DELETE
          below they are hits. Since PG 18 ANALYZE implies BUFFERS; on older
          servers write EXPLAIN (ANALYZE, BUFFERS). Index Searches (also new in
          18) counts descents of the tree.
      - lines: "25-31"
        note: >
          Plain EXPLAIN only plans the DELETE. With ANALYZE it executes, and
          the 20 rows are really gone; the Delete node reports 0 rows because
          there is no RETURNING. Outside a sandbox, wrap it in BEGIN ...
          ROLLBACK. Running the plain EXPLAIN first has a side benefit: the
          measured run finds the planner's lookups cached, so PG 18 has no
          "Planning: Buffers" section to print.
  - id: btree-depth
    file: examples/02_btree_depth.sql
    title: "Tree depth from buffer counts: 100, 10,000 and 200,000 rows"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-7"
        note: >
          Same shape, three sizes. A PRIMARY KEY is backed by a unique B-tree
          index, and the inserts went through it, so its pages are in the
          cache and every buffer below is a hit.
      - lines: "9-13"
        note: >
          The first plan in a session reads catalog data and each index's
          metapage (which records where the root is). Planning these lookups
          once here keeps that out of the measurements below.
      - lines: "16-21"
        note: >
          2, 3 and 4 pages: one per tree level plus one heap page for the row.
          100 keys fit on one page that is root and leaf at once. 10,000 need
          about 30 leaf pages under a root. 200,000 need about 550 leaves,
          more than one root page can point to, so a level of internal pages
          appears. With 4-byte keys a page holds roughly 400 entries, so three
          levels cover tens of millions of rows.
      - lines: "24-27"
        note: >
          A range descends once (Index Searches: 1) and then follows the leaf
          level's sibling links: 6 index pages are 2 to descend plus the 4
          leaves holding keys 1000 to 1999. The 1,000 rows themselves sit on 6
          heap pages. The planner picked a bitmap scan; the index walk is the
          same either way.
  - id: index-ignored
    file: examples/03_index_ignored.sql
    title: "An index the planner is right to ignore"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          7919 is prime, so (g * 7919) % 1000 hits every amount from 0 to 999
          exactly 20 times, scattered over the table. Physical order and amount
          are unrelated, as for most columns that aren't timestamps or ids.
      - lines: "12-18"
        note: >
          Same index, same query shape, two plans: 1% of the rows (200) goes
          through the index, 90% (18,000) doesn't. The planner compares
          estimated costs per query, using the statistics from line 10.
      - lines: "20-22"
        note: >
          147 pages: every page of the table once, in physical order, which the
          operating system can read ahead. Rows Removed by Filter: 2000 is the
          price of not using the index: cheap here.
      - lines: "24-27"
        note: >
          SET LOCAL lasts until the transaction ends. enable_seqscan = off
          doesn't forbid sequential scans; it makes the planner avoid them
          whenever another plan exists (PG 18 marks a node it had to use anyway
          with "Disabled: true"). The forced bitmap scan reads the same 147
          table pages plus 18 index pages: more work for the same answer.
      - lines: "29-31"
        note: >
          A plain Index Scan walks the index in amount order and jumps to the
          table for each row: 18,018 page accesses, each table page visited
          about 120 times. With a cold cache each jump can be a random disk
          read, which is why random_page_cost (4.0 by default) is priced above
          seq_page_cost (1.0).
use_cases:
  - "Before adding an index: EXPLAIN the slow query and look for a Seq Scan whose Filter throws most rows away (Rows Removed by Filter)"
  - "After adding one: EXPLAIN ANALYZE inside BEGIN ... ROLLBACK, on production-sized data, to confirm the new plan and that it touches fewer buffers"
  - "A plain B-tree serves =, <, <=, >, >=, BETWEEN, IN, IS NULL, ORDER BY and min/max on its column, and anchored LIKE 'abc%' under the C collation or with text_pattern_ops"
  - "Reviewing an ORM's generated query: the plan shows whether its WHERE clause can use any index at all"
strengths:
  - "Lookup cost grows with tree depth, not table size: 4 pages for 200,000 rows here and only a few more for billions, while a Seq Scan grows linearly"
  - "Keys stay sorted, so one B-tree answers equality, ranges, ORDER BY and min/max; a hash structure (Java HashMap, Python dict, PostgreSQL's hash index) answers only equality"
  - "EXPLAIN ANALYZE gives exact per-node row and page counts for one real execution, more precise than a sampling profiler, and hit + read is the same total whether the cache was warm or cold"
  - "No hints needed: add or drop an index and the planner re-decides per query from statistics, without touching the SQL"
weaknesses:
  - "Every index slows every INSERT and every UPDATE of an indexed column, and takes disk and cache space"
  - "EXPLAIN ANALYZE runs the statement in full: a 10-minute query takes 10 minutes, and writes happen unless you roll back"
  - "A plan reflects this data volume and these statistics; the plan on a 100-row development table says nothing about production"
  - "Timing every node costs overhead (significant on some systems, the docs warn); TIMING OFF keeps row and buffer counts without it"
pitfalls:
  - title: "EXPLAIN ANALYZE on a write, in autocommit"
    detail: "ANALYZE executes the statement, so the rows really change. Wrap it in a transaction and roll back."
    bad: "EXPLAIN ANALYZE DELETE FROM orders WHERE placed_at < '2025-01-01';"
    good: "BEGIN; EXPLAIN ANALYZE DELETE FROM orders WHERE placed_at < '2025-01-01'; ROLLBACK;"
  - title: "Judging an index on a toy table"
    detail: "The docs: on a table that occupies one disk page you'll nearly always get a sequential scan, index or not. Generate realistic volumes with generate_series, ANALYZE, then read the plan."
  - title: "Comparing timings instead of buffers"
    detail: "The second run is faster because the pages are now cached. Buffers (hit + read) counts the same work either way, so compare those, and use timings only on the same cache state."
  - title: "enable_* settings as a fix"
    detail: "enable_seqscan = off and friends are diagnostic switches for seeing what an alternative plan costs. Use SET LOCAL inside a transaction; turned off server-wide they distort every query's plan."
quiz:
  - q: "In autocommit, you run `EXPLAIN ANALYZE DELETE FROM orders WHERE customer_id = 42`. What happens?"
    options: ["Nothing changes; EXPLAIN never modifies data", "The rows are deleted and the plan is printed", "The rows are deleted only if BUFFERS is also given"]
    answer: 1
    why: "ANALYZE executes the statement and discards only its output. Use BEGIN ... ROLLBACK around it."
  - q: "A primary-key lookup on a 200,000-row table shows `Buffers: shared hit=4`. What are the 4?"
    options: ["Four matching rows", "Three index levels plus one table page", "Four separate index searches"]
    answer: 1
    why: "A lookup reads one page per tree level, then the heap page holding the row. Index Searches: 1 shows it was a single descent."
  - q: "`amount < 900` matches 90% of the rows and there is an index on amount. Why does the planner choose a Seq Scan?"
    options: ["The index hasn't been analyzed", "Fetching 90% of the rows through the index visits table pages in random order, many times each; reading every page once is less work", "B-tree indexes can't evaluate range conditions"]
    answer: 1
    why: "The forced index scan touched 18,018 pages against 147 for the Seq Scan. B-trees handle ranges fine; the range was just too wide."
  - q: "What does `Buffers: shared read=2` tell you?"
    options: ["Two pages weren't in PostgreSQL's buffer cache and were requested from the operating system", "Two pages were certainly read from disk", "Two rows were read"]
    answer: 0
    why: "A read may still be served by the OS file cache; PostgreSQL only knows its own cache missed. Turn on track_io_timing to see how long those reads took."
further_reading:
  - title: "PostgreSQL docs: Using EXPLAIN"
    url: "https://www.postgresql.org/docs/current/using-explain.html"
  - title: "PostgreSQL docs: EXPLAIN (options: ANALYZE, BUFFERS, TIMING, SUMMARY)"
    url: "https://www.postgresql.org/docs/current/sql-explain.html"
  - title: "PostgreSQL docs: Index Types"
    url: "https://www.postgresql.org/docs/current/indexes-types.html"
  - title: "PostgreSQL docs: B-Tree Indexes (implementation: levels, page splits, deduplication)"
    url: "https://www.postgresql.org/docs/current/btree.html"
  - title: "PostgreSQL docs: Planner Cost Constants (seq_page_cost, random_page_cost)"
    url: "https://www.postgresql.org/docs/current/runtime-config-query.html"
---

## Concept

A table in PostgreSQL is a **heap**: rows sit in 8 kB pages in no particular order. An
index is a separate structure that maps values to row locations (a **TID**: block number
and offset). The default kind, and the one behind every `PRIMARY KEY` and `UNIQUE`, is a
**B-tree**: keys kept sorted in pages, with internal levels above the leaves. A lookup
starts at the root and reads one page per level. A page holds hundreds of keys, so the
tree stays shallow: three levels reach tens of millions of rows.

Whether a query uses the index is the **planner's** decision, made per query from
statistics. Two commands show it:

- `EXPLAIN` prints the chosen plan with estimates. Nothing runs; it's cheap and safe.
- `EXPLAIN ANALYZE` runs the statement and adds what really happened: rows per node,
  loops, and (by default since PostgreSQL 18) **buffers**, the pages each node touched.

Read the plan before you trust an index: the planner may ignore it, and sometimes it is
right to.

## Coming from C/C++/Java/Python/JS

- **Java's `TreeMap`, C++'s `std::map`:** sorted, O(log n), ranges and ordered
  iteration. A B-tree is the on-disk version: nodes are pages with hundreds of keys, so
  the cost is a handful of page reads rather than dozens of pointer hops.
- **`HashMap`, Python `dict`, JS `Map`:** equality only, no order. PostgreSQL has hash
  indexes too, and they're rarely the right choice for the same reason.
- **Profilers:** `EXPLAIN` is reading what the compiler intends to emit; `EXPLAIN
  ANALYZE` is a precise profiler for one execution, with exact counts instead of samples.
- **If you've read MySQL advice:** InnoDB stores the table inside its primary-key tree.
  PostgreSQL doesn't; every index, the primary key included, points into the heap, so an
  index scan always pays for the trip to the table (lesson 2's index-only scans are the
  exception).

## How it works

```text
                 [ root: 1 .. 200000 ]                  level 2
                 /                    \
      [ internal: 1 .. ]         [ internal: .. 200000 ]  level 1
       /      |       \               ...
  [leaf] <-> [leaf] <-> [leaf] <-> ... <-> [leaf]        level 0: keys + TIDs
     |         |  \
   heap      heap  heap     table pages, in insertion order, not key order
```

Leaves are linked to their neighbours, so a range scan descends once and then walks
sideways. Page splits add leaves as the table grows; when the root itself splits, the
tree gains a level. That's the whole reason lookup cost barely moves with table size.

A plan is a tree of nodes, printed with indentation; each node pulls rows from the nodes
indented below it, so start reading at the innermost line. With costs on, a node reads:

```text
Seq Scan on orders  (cost=0.00..378.00 rows=20 width=14)
```

`378.00` is the estimated total cost in abstract units: the table's 128 pages at
`seq_page_cost` (1.0), plus 20,000 rows at `cpu_tuple_cost` (0.01), plus 20,000
evaluations of the filter at `cpu_operator_cost` (0.0025). `0.00` is the start-up cost
(work before the first row, such as a sort). `rows=20` is the estimate you compare with
`actual rows` from `EXPLAIN ANALYZE`; a large gap means bad statistics, the subject of
lesson 5. The planner picks the cheapest total among the plans it considers.

## Design notes

A workflow that holds up:

1. `EXPLAIN` the query first. Look for the scan type on the big tables and for filters
   that discard most rows.
2. `EXPLAIN (ANALYZE)` it inside `BEGIN ... ROLLBACK`, on realistic data volumes. Compare
   estimated and actual rows, then look at Buffers. `TIMING OFF` and `SUMMARY OFF` are
   what keep this sandbox's output stable; on a real investigation, keep the timings.
3. Change one thing (an index, the query, `ANALYZE`) and compare Buffers again.

Indexes are not free: each one is updated by every `INSERT`, and by every `UPDATE` that
changes an indexed column. Add them for queries you have measured, not for columns that
look important. The planner rejecting an index for a wide range is usually correct;
the fix for a slow wide query is a different index or a different query, which the rest
of this module covers.
