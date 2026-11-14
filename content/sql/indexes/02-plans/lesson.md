---
id: sql.indexes.plans
title: "Reading plans: seq/index/index-only/bitmap scans, nested loop/hash/merge joins"
track: sql
module: indexes
order: 2
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.indexes.btree, sql.joins.inner-outer]
compare_with: [go.composite.slices-and-maps, sql.joins.lateral, go.postgres.querying]
tags: [explain, query-plan, index-only-scan, bitmap-scan, nested-loop, hash-join, merge-join]
summary: >
  The planner picks one of four ways to read each table and one of three ways
  to join each pair, per query, from estimated row counts. Each choice tells
  you what it believed; EXPLAIN ANALYZE tells you whether it was right.
examples:
  - id: scans
    file: examples/01_scans.sql
    title: "One table, six plans: seq, index, index-only, bitmap, BitmapOr, ordered"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          value is a scrambled permutation (the 7919 trick from the previous
          lesson) and sensor_id repeats every 200 rows: neither follows the
          physical order of the table.
      - lines: "13-14"
        note: >
          90% of the rows match, so the planner reads every page once and tests
          the Filter on each row. A Seq Scan is always on the menu; every other
          plan needs an index that matches the condition or the ORDER BY.
      - lines: "16-17"
        note: >
          Index Scan: descend the tree, then fetch each match from the table.
          Unbeatable for a handful of rows, expensive per row for many.
      - lines: "19-20"
        note: >
          Index Only Scan: the query needs only id, and the index holds it, so
          the table can be skipped for pages the visibility map marks
          all-visible. B-trees always support this; GiST and SP-GiST only for
          some operator classes; GIN never, because its entries hold only parts
          of the values.
      - lines: "22-23"
        note: >
          Bitmap scan: 200 matches scattered over the table. The bitmap orders
          their locations by page, so each page is read once, in physical
          order. That also means the index order is lost: an ORDER BY on top of
          a bitmap scan needs its own Sort.
      - lines: "25-26"
        note: >
          A single index scan can only use conditions joined by AND on its own
          columns. Here each index builds a bitmap and BitmapOr merges them.
          AND conditions on two indexes combine the same way (BitmapAnd),
          though one composite index usually beats that (next lesson).
      - lines: "28-29"
        note: >
          No Sort node: the index hands rows over in value order, and Limit
          stops pulling after five. The cost is the same on 20,000 rows or 20
          million, which is what keyset pagination (lesson 5) builds on.
      - lines: "33-34"
        note: >
          Heap Fetches: 100. The visibility map of a brand-new table has no
          bits set (VACUUM sets them, and these uncommitted rows aren't visible
          to anyone else anyway), so every row is checked in the table. Only 3
          buffers, because the 100 rows share one heap page next to 2 index
          pages. On a vacuumed table that changes slowly, Heap Fetches drops
          toward 0, and only then does index-only pay off.
  - id: joins
    file: examples/02_joins.sql
    title: "The three join methods, each where it fits"
    runtime: sql
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          Ten orders per customer, scattered over the table. The index on
          orders.customer_id is explicit: a foreign key doesn't create one.
      - lines: "15-19"
        note: >
          Nested Loop: the first child (outer) yields 3 customers; the second
          child (inner) runs once per outer row with that row's id plugged in,
          which is why its Index Cond mentions c.id. The IN list became
          = ANY(array), a single index scan for all three keys.
      - lines: "21-25"
        note: >
          Hash Join: the Hash node reads customers once into an in-memory hash
          table keyed on id, then all 20,000 orders stream past it. No index
          needed, each table read once. HashAggregate then groups by city with
          a second hash table.
      - lines: "27-31"
        note: >
          Merge Join: both inputs arrive sorted on the join key (customers from
          its primary key, orders from orders_customer), so one merge pass
          joins them, and the result is already in c.id order: no Sort node for
          the ORDER BY.
  - id: join-costs
    file: examples/03_join_costs.sql
    title: "One join, three methods: what the rejected plans would have cost"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          The same tables as before, but now 500 customers qualify: too many
          for one index lookup each, so the planner hashes them instead.
      - lines: "19-22"
        note: >
          120 buffers: customers' 11 pages and orders' 109, each read once.
          Batches: 1 means the hash table fit in memory; the limit is work_mem
          times hash_mem_multiplier (4MB × 2.0 by default), and a hash table
          past it spills to temporary files in more batches. Planning: Buffers
          counts pages the planner itself read while choosing.
      - lines: "24-28"
        note: >
          Nested Loop, forced: the inner Index Scan ran 500 times (loops=500,
          Index Searches: 500). actual rows=10.00 is per loop, so 5,000 rows in
          total. Each loop reads 2 index pages and 10 table pages: 6,011 page
          accesses, 50 times the hash join. With 3 outer rows (previous
          example) the same shape was the cheapest plan; the method depends on
          the row counts, not on the query text.
      - lines: "30-33"
        note: >
          Merge Join, forced: customers gets an explicit Sort (500 rows in
          memory), orders comes sorted from its index at one page jump per row:
          20,005 accesses. It reads 19,981 orders and stops: past the last
          İzmir customer (id 1998) there's nothing left to merge.
      - lines: "35-39"
        note: >
          All three methods off, and a join is still required. enable_* only
          discourages, so the planner takes the cheapest disabled plan and
          PG 18 marks it Disabled: true; older versions printed no marker.
use_cases:
  - "Diagnosing a slow endpoint: find the node where actual rows, loops or buffers explode, and compare its actual rows with the estimate"
  - "Deciding between a composite index and two single-column indexes combined with BitmapAnd"
  - "Checking that ORDER BY ... LIMIT stops early on an index instead of sorting the whole table"
  - "Explaining why one customer's page is fast and a whole city's report is slow: the join method changes with the row counts"
strengths:
  - "The algorithm is chosen per query from statistics: the same join is a nested loop for 3 customers and a hash join for 500, which a hand-written loop in Go or an ORM can't do"
  - "Bitmap scans combine any number of single-column indexes for AND/OR conditions and still read each table page only once"
  - "Index order replaces sorting: index scans and merge joins deliver ordered rows, so ORDER BY ... LIMIT can stop after a few rows"
  - "Rejected alternatives are measurable: switch a method off with SET LOCAL and compare buffers in the same transaction"
weaknesses:
  - "Every choice is only as good as the row estimates: a nested loop planned for 3 outer rows that meets 50,000 is the classic slow query"
  - "Core PostgreSQL has no per-query hints; the enable_* switches are coarse and apply to everything in the session or transaction"
  - "Index-only scans depend on the visibility map, so on a busy table they degrade into index scans until VACUUM catches up"
  - "Hashes and sorts are bounded per node by work_mem (hashes: times hash_mem_multiplier) and spill to temporary files beyond it"
pitfalls:
  - title: "Reading actual rows without loops"
    detail: "On the inner side of a nested loop, actual rows is the average per execution. rows=10.00 loops=500 is 5,000 rows. Multiply before comparing with the estimate or blaming the node."
  - title: "Reading Buffers as per-node work"
    detail: "A node's Buffers include all of its children. The Aggregate's 120 in the hash join plan is the two Seq Scans' 11 + 109, not 120 more."
  - title: "Expecting an index-only scan right after a bulk load"
    detail: "Until VACUUM has set the visibility map, every row is checked in the table (Heap Fetches). After a large load, vacuum and analyze explicitly instead of waiting for autovacuum."
    good: "VACUUM (ANALYZE) readings;"
  - title: "Raising work_mem server-wide for one report"
    detail: "work_mem applies per sort or hash node, per session, so a global increase multiplies across every concurrent query. Raise it for the one transaction that needs it."
    bad: "ALTER SYSTEM SET work_mem = '256MB';"
    good: "BEGIN; SET LOCAL work_mem = '256MB'; SELECT ...; COMMIT;"
quiz:
  - q: "An inner Index Scan shows `actual rows=10.00 loops=500`. How many rows did it produce in total?"
    options: ["10", "500", "5,000"]
    answer: 2
    why: "Actual rows is the per-loop average; multiply by loops."
  - q: "Why does a Bitmap Heap Scan need a separate Sort for ORDER BY, even on an indexed column?"
    options: ["Bitmaps store values, not locations", "It visits rows in physical page order, so the index order is lost", "Bitmap scans only support equality"]
    answer: 1
    why: "The bitmap is laid out by page so each page is read once; the docs note that any ordering of the original indexes is lost."
  - q: "An Index Only Scan reports `Heap Fetches` equal to the number of rows. What usually fixes it in a real database?"
    options: ["A bigger work_mem", "VACUUM, so the visibility map marks pages all-visible", "Adding INCLUDE columns"]
    answer: 1
    why: "The table can be skipped only for all-visible pages, and VACUUM is what records them."
  - q: "You join every row of two large tables on a key with no index, and the output order doesn't matter. Which method do you expect?"
    options: ["Nested Loop", "Hash Join", "Merge Join"]
    answer: 1
    why: "One pass over each table, no index or sort needed. A nested loop would rescan the inner side per row; a merge join would have to sort both sides first."
further_reading:
  - title: "PostgreSQL docs: Using EXPLAIN"
    url: "https://www.postgresql.org/docs/current/using-explain.html"
  - title: "PostgreSQL docs: Planner/Optimizer (the join strategies)"
    url: "https://www.postgresql.org/docs/current/planner-optimizer.html"
  - title: "PostgreSQL docs: Combining Multiple Indexes (bitmap scans)"
    url: "https://www.postgresql.org/docs/current/indexes-bitmap-scans.html"
  - title: "PostgreSQL docs: Index-Only Scans and Covering Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-index-only-scans.html"
  - title: "PostgreSQL docs: Query Planning settings (enable_*, cost constants)"
    url: "https://www.postgresql.org/docs/current/runtime-config-query.html"
  - title: "PostgreSQL docs: Resource Consumption (work_mem, hash_mem_multiplier)"
    url: "https://www.postgresql.org/docs/current/runtime-config-resource.html"
---

## Concept

For each table in a query the planner considers a **Seq Scan** (always possible) and an
index-based scan for every index whose columns match a condition or the `ORDER BY`. For
each pair of inputs it considers three join methods. It estimates the cost of the
combinations and runs the cheapest. The menu:

- **Seq Scan**: every page, once, in order. Right for a large fraction of the table.
- **Index Scan**: descend, then one table fetch per match. Right for a few rows, or when
  the index order saves a sort.
- **Index Only Scan**: answer from the index alone, for pages the visibility map marks
  all-visible.
- **Bitmap Index Scan + Bitmap Heap Scan**: collect matching locations first, read each
  page once in physical order. The middle ground, and the way to combine indexes.
- **Nested Loop**: for each outer row, run the inner side. Right when the outer side is
  small and the inner side has an index.
- **Hash Join**: build a hash table on one side, stream the other. Right for large,
  unsorted inputs.
- **Merge Join**: walk two inputs sorted on the key. Right when both arrive sorted, or the
  output must be sorted anyway.

None of them is good or bad by itself. A plan is a statement about row counts: a nested
loop says "the outer side is small". Your job when reading it is to check that belief.

## Coming from C/C++/Java/Python/JS

You've written all three joins by hand:

- **Nested loop:** `for c in customers: for o in index[c.id]` in Python, or a lookup in a
  `TreeMap` per outer element in Java. Without the index it is JS's
  `customers.map(c => orders.filter(o => o.customerId === c.id))`, O(n·m).
- **Hash join:** build a `HashMap<Integer, Customer>` (Go: `map[int]Customer`) from the
  smaller list, then loop over the larger one and probe it.
- **Merge join:** the merge step of merge sort, over two lists sorted by the same key.

The difference is that you choose once, when you write the code, while the planner
chooses per execution from estimated sizes. An ORM that loads customers and then queries
each one's orders is running a nested loop from the application, with a network round
trip per inner lookup: the N+1 problem of lesson 5.

## How it works

Reading `EXPLAIN ANALYZE` output, node by node:

- **`actual rows=10.00 loops=500`**: rows are averages per loop. Since PG 18 they are
  printed with two decimals, because an average needn't be whole.
- **`Index Searches: 500`** (new in PG 18): descents of the tree across all loops. On the
  inner side of a nested loop that's one per outer row. A range is one search; an
  `= ANY(...)` list can take up to one per element, fewer when neighbouring keys share a
  leaf page.
- **`Buffers`**: cumulative, a node's count includes its children's.
- **Hash**: `Buckets`, `Batches`, `Memory Usage`. One batch means it fit in memory.
- **Sort**: `Sort Method: quicksort  Memory: 25kB` fit in `work_mem`; `external merge
  Disk: ...` did not.
- **`Heap Fetches`**: table visits an index-only scan couldn't skip.
- **`Rows Removed by Filter`**: rows read and thrown away; a large number on a selective
  query is the classic sign of a missing or unusable index.

Two more nodes you'll meet: **Memoize** (PG 14+) sits between a nested loop and its inner
scan and caches inner results per key when the same keys repeat; **Gather** with
`Parallel Seq Scan` below it means the work was split across worker processes, which
starts once a table passes `min_parallel_table_scan_size` (8 MB by default).

## Design notes

Read a slow plan in this order:

1. Find the node with the largest actual rows × loops, or the largest Buffers.
2. Compare its actual rows with the estimate (`EXPLAIN` without `COSTS OFF`). A tenfold
   gap is the usual root cause: the planner picked a nested loop for "3 rows" and got
   thousands. Fix the estimate (`ANALYZE`, extended statistics, lesson 5) before
   anything else.
3. If the estimate is right and the plan is still slow, the fix is an index that matches
   the query (lesson 3) or a different query shape.

`enable_*` switches belong between steps 2 and 3: confirming what the alternative would
cost, as in the third example. The docs call them a crude method and a temporary
solution, and core PostgreSQL has no per-query hints. A plan you had to force today
breaks silently when the data changes tomorrow; a correct estimate keeps adapting.
