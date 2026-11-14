---
id: sql.basics.set-based
title: "SQL as a declarative, set-based language (vs loops in Go/TS)"
track: sql
module: basics
order: 3
level: basic
minutes: 13
since: pg9.0
prerequisites: [sql.basics.mental-model]
compare_with: [go.composite.slices-maps-pkg, ts.functions.higher-order, design.paradigms.side-by-side]
tags: [declarative, set-based, explain, set-operations]
summary: >
  A query describes the rows you want; the planner picks the loops, hash tables
  and sorts. Statements act on every qualifying row at once, against the table
  as it was when they started, so stop writing the loop and write the result.
examples:
  - id: describe-result
    file: examples/01_describe_result.sql
    title: "The question as one query, and the plan the server chose"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-23"
        note: >
          Each clause states a property of the answer: which rows (WHERE),
          grouped how (GROUP BY), which groups survive (HAVING), in what
          order. None says how to compute it. GROUP BY and HAVING get their
          own lesson in the aggregation module.
      - lines: "20"
        note: >
          A half-open range, from the first of the month up to (not including)
          the first of the next, stays correct when the column later becomes
          a timestamp; BETWEEN ... '2026-11-30' would then miss that day's
          afternoon.
      - lines: "26-32"
        note: >
          EXPLAIN prints the plan instead of running the query. Bottom-up:
          scan and filter the table, sort by customer, sum each run of equal
          names (GroupAggregate), keep groups of 500 or more, sort by the sum.
          On other data the planner may pick a HashAggregate, which is your
          Go map, or an index. The query text would not change. COSTS OFF
          hides the estimates, which shift with table statistics.
  - id: all-at-once
    file: examples/02_all_at_once.sql
    title: "One statement, every row, one snapshot"
    runtime: sql
    check: exact
    annotations:
      - lines: "7"
        note: >
          Every right-hand side is computed from the row as it was before
          the UPDATE, so the two assignments don't see each other. Go's
          a, b = b, a works the same way; Java needs a temporary variable.
      - lines: "14"
        note: >
          The SELECT reads a snapshot taken when the statement started: 3
          rows in, 3 rows added. A JavaScript for...of that pushes onto the
          array it iterates never ends. Go's range evaluates the slice once
          and stops after 3, the semantics every SQL statement has.
      - lines: "19-21"
        note: >
          The crack: PostgreSQL checks a non-deferrable UNIQUE as each row is
          written, not at the end of the statement as the SQL standard asks.
          pos 1 becomes 2 while the old 2 still exists, so the statement
          fails, and fails whole: no row moves. A DEFERRABLE constraint
          (constraints module) gets the end-of-statement check.
  - id: set-operations
    file: examples/03_set_operations.sql
    title: "Diff two lists with EXCEPT, INTERSECT and UNION"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          supplier_file has no key and lists B-7 twice: tables are bags
          (multisets), not sets. UNION, INTERSECT and EXCEPT remove
          duplicates unless you add ALL.
      - lines: "8-11"
        note: >
          EXCEPT is set difference. In Go you'd build a map from one side and
          loop over the other; here the planner picks hashing or sorting, and
          there is no map to get wrong.
      - lines: "20-23"
        note: >
          INTERSECT keeps rows present on both sides. B-7 comes out once even
          though the file has it twice.
      - lines: "27-30"
        note: >
          UNION has to sort or hash every row to find duplicates. When
          duplicates are impossible, or are real data (two identical
          payments), UNION ALL is both correct and cheaper.
use_cases:
  - "Reports and dashboards: filter, group, sort in one query instead of fetching rows and folding them in Go or TS"
  - "Bulk changes: one UPDATE or DELETE for every qualifying row, not a loop issuing one per row"
  - "Reconciling lists (import vs catalog, expected vs actual) with EXCEPT and INTERSECT"
strengths:
  - "The planner chooses algorithms from statistics and indexes; adding an index can speed a query up without touching its text, which a hand-written loop can't offer"
  - "One statement is atomic: it applies to all qualifying rows or to none, so there is no half-finished loop to clean up after an error"
  - "Set operators express list diffs in a line, where Go needs a map and two loops and Java a HashSet and removeAll"
weaknesses:
  - "You don't pick the plan directly (core PostgreSQL has no per-query hints, only crude enable_* settings); when the planner misestimates, you fix statistics, indexes or the query's shape"
  - "Work with per-row side effects outside the database (call an API, send an e-mail) doesn't fit in one statement"
  - "PostgreSQL's row-by-row UNIQUE check breaks the all-at-once model for updates that permute key values"
pitfalls:
  - title: "One statement per row from the application"
    detail: "A Go or TS loop that sends an UPDATE per item makes N round trips and N statements, and leaves half the rows changed if it fails midway outside a transaction. Send the whole set in one statement: UPDATE ... FROM over arrays passed through unnest, with one entry per sku (the modifying module shows why that matters)."
    bad: "for _, it := range items { db.Exec(ctx, \"UPDATE stock SET qty = qty - $1 WHERE sku = $2\", it.Qty, it.SKU) }"
    good: "UPDATE stock s SET qty = s.qty - i.qty FROM unnest($1::text[], $2::int[]) AS i(sku, qty) WHERE s.sku = i.sku"
  - title: "Expecting rows in insertion order"
    detail: "A table has no order. Without ORDER BY, rows come back in whatever order the chosen plan produces, and that changes with updates and plans."
  - title: "UNION where UNION ALL was meant"
    detail: "UNION silently merges identical rows. Two identical 100 TL payments become one, and the total is wrong."
quiz:
  - q: "nums holds 1, 2, 3. What does INSERT INTO nums SELECT n + 3 FROM nums leave behind?"
    options:
      - "6 rows: 1 to 6"
      - "An endless loop, stopped by statement_timeout"
      - "An error: a table can't be read and written in one statement"
    answer: 0
    why: "The SELECT sees the table as of the statement's start, so it reads 3 rows and inserts 3."
  - q: "What does UPDATE t SET a = b, b = a do?"
    options:
      - "Sets both columns to the old b"
      - "Swaps a and b in every row"
      - "Fails: a column is used after being assigned"
    answer: 1
    why: "Right-hand sides use the old values of the row, like Go's a, b = b, a."
  - q: "What does the text of a SELECT decide?"
    options:
      - "The rows and order of the result"
      - "Whether a hash table or a sort is used"
      - "The order in which the table is read from disk"
    answer: 0
    why: "The query is a description of the result. The algorithms are the planner's choice, visible with EXPLAIN."
  - q: "Which query lists skus in the supplier's file that we don't stock?"
    options:
      - "SELECT sku FROM supplier_file UNION SELECT sku FROM stock"
      - "SELECT sku FROM supplier_file INTERSECT SELECT sku FROM stock"
      - "SELECT sku FROM supplier_file EXCEPT SELECT sku FROM stock"
    answer: 2
    why: "EXCEPT returns rows of the first query that the second doesn't have: set difference."
further_reading:
  - title: "PostgreSQL docs: Queries (the SELECT pipeline)"
    url: "https://www.postgresql.org/docs/current/queries.html"
  - title: "PostgreSQL docs: Combining Queries (UNION, INTERSECT, EXCEPT)"
    url: "https://www.postgresql.org/docs/current/queries-union.html"
  - title: "PostgreSQL docs: Using EXPLAIN"
    url: "https://www.postgresql.org/docs/current/using-explain.html"
  - title: "PostgreSQL docs: CREATE TABLE, non-deferred uniqueness constraints"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
---

## Concept

SQL is declarative: a query describes the rows you want, and the planner decides how to
produce them. Which index, which join algorithm, hash table or sort, in which order to
read the tables: all of that is chosen per query from statistics, and `EXPLAIN` shows
the choice.

The unit of work is the set of rows (strictly a bag, since duplicates are allowed), not
the row. An `UPDATE` changes every row its `WHERE` matches, in one atomic step, and
every expression in it reads the table as it was when the statement started. The habit
to build: when you're about to write a loop around the database, ask what the result
looks like and write that.

## Coming from C/C++/Java/Python/JS

The November report as imperative Go:

```go
spent := map[string]int64{} // in kuruş: never float for money
for _, o := range orders {  // every row, fetched over the network first
	if !o.Placed.Before(nov1) && o.Placed.Before(dec1) {
		spent[o.Customer] += o.TotalKurus
	}
}
var out []row
for c, s := range spent {
	if s >= 500_00 {
		out = append(out, row{customer: c, spent: s})
	}
}
slices.SortFunc(out, func(a, b row) int { return cmp.Compare(b.spent, a.spent) })
```

Here you chose the algorithm (a hash map), the memory (all rows in the process) and the
order of operations. The SQL version states the result and lets the server pick, next to
the data and its indexes.

- **TS's `filter`/`reduce`, Java Streams, Python comprehensions** look declarative, but
  they still run as written, in your process, after every row has been fetched. The
  planner can do what they can't: reorder steps, use an index, or skip reading most of
  the table.
- **N+1 queries**: a loop that runs one query per item is a loop the database cannot see
  or optimize. It's the most common performance bug in application code that talks to
  SQL.
- **Halloween problem**: C++ iterator invalidation and JS's endless `for...of` over a
  growing array don't exist inside one SQL statement; the snapshot rule prevents them.

## Design notes

"Set-based" doesn't mean "one giant statement for everything". Loops still have a
place, one level up: batch a ten-million-row backfill into chunks of a few thousand rows,
each chunk a single set-based `UPDATE`, so locks stay short and progress is visible. Per-row
work outside the database (an HTTP call per customer) belongs in application code.
What disappears is the loop *inside* the data work: filtering, joining, grouping and
diffing are the database's job, and it does them better because it knows the data's
size, its indexes and its statistics.
