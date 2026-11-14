---
id: sql.indexes.composite
title: "Composite index column order, partial and expression indexes, covering (INCLUDE)"
track: sql
module: indexes
order: 3
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.indexes.plans, sql.constraints.keys]
compare_with: [sql.constraints.generated, sql.modifying.soft-delete, sql.modeling.multi-tenant, go.postgres.querying]
tags: [indexes, composite-index, partial-index, expression-index, covering-index, include, collation]
summary: >
  Start from the query, not the table: equality columns first, then one range or
  sort column; a WHERE clause to index only the rows you search; an expression
  to index what you compare; INCLUDE to answer from the index alone.
examples:
  - id: column-order
    file: examples/01_column_order.sql
    title: "(shop_id, placed_on) vs (placed_on, shop_id) for the same query"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-11"
        note: >
          g / 100 is integer division: 100 consecutive orders per day, and
          g % 100 + 1 spreads them over 100 shops. Dates rather than
          timestamptz keep the plan text short and independent of the
          server's time zone setting.
      - lines: "19-26"
        note: >
          Entries are sorted by day first, so the only contiguous stretch for
          this query is "March and April, every shop": about 6,100 entries,
          checked one by one for shop_id = 7. shop_id still appears in the
          Index Cond, but it filters inside the scanned range instead of
          narrowing it. 60 buffers: the 39 table pages holding the 61 rows,
          and about 20 index pages mostly full of other shops' entries. The
          Planning section counts pages the planner itself read while
          choosing the plan.
      - lines: "29-35"
        note: >
          Sorted by shop first, shop 7's two months are one contiguous run of
          61 entries: the Bitmap Index Scan reads 2 index pages instead of
          about 20. The table side (39 pages for 61 rows) is identical, since
          both indexes find the same rows. Each measured run follows a plain
          EXPLAIN of the same query, so planner lookups are already cached.
      - lines: "38-39"
        note: >
          A prefix of the key is an index too: shop_id alone works, and
          within one shop the entries are already in placed_on order, so
          ORDER BY ... DESC LIMIT reads the run backwards and stops after 10.
          An index on (placed_on, shop_id) could do neither of these well.
      - lines: "42-43"
        note: >
          total isn't in the index, so its order is unknown: all of shop 7's
          rows are fetched and sorted (a top-N sort that keeps only 10). An
          index on (shop_id, total) would serve this query instead.
  - id: partial-expression
    file: examples/02_partial_expression.sql
    title: "Partial index for the hot rows; expression index for Turkish lower-casing"
    runtime: sql
    check: exact
    annotations:
      - lines: "17-20"
        note: >
          The full index has an entry for all 20,000 rows; the partial one
          only for the 400 open tickets, and it shrinks further as tickets
          close. Fewer pages to cache, and nothing to maintain when closed
          tickets change. pg_relation_size takes the index name, resolved via
          search_path, so other schemas' objects can't interfere.
      - lines: "23-24"
        note: >
          The plan shows no condition at all: every entry in the partial index
          is open by construction. The planner used it because the query's
          WHERE implies the index's WHERE. It proves that at planning time, so
          the condition must be in the SQL text, not a $1 parameter.
      - lines: "27-28"
        note: >
          The databases here use the builtin C.UTF-8 locale, whose lower()
          maps I to i. Turkish rules map I to ı (and İ to i), which is what a
          Turkish user typing "ısparta" expects to match.
      - lines: "29-30"
        note: >
          The index stores lower(city COLLATE "tr-x-icu") for every row,
          computed once per insert or update. ANALYZE also gathers statistics
          on the expression itself, which helps the estimate for queries on
          it.
      - lines: "33-35"
        note: >
          The query repeats the indexed expression, collation included. EXPLAIN
          shows the collation change as (city)::text rather than spelling out
          COLLATE, so read the index definition, not the plan, to see it.
      - lines: "38-40"
        note: >
          lower(city) is a different expression: Seq Scan, and with the default
          rules 'ISPARTA-8' becomes 'isparta-8', so nothing matches. Choosing
          the collation is a correctness decision first and an indexing one
          second.
  - id: covering
    file: examples/03_covering.sql
    title: "INCLUDE: an index-only scan, and what payload columns can't do"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "1-2"
        note: >
          In the transactional sql runtime nothing is ever committed, so the
          visibility map stays empty and an index-only scan still visits every
          row (previous lesson). Here VACUUM can mark the pages all-visible. If
          another transaction in the database was open while it ran, some pages
          may stay unmarked and Heap Fetches would be above 0.
      - lines: "14-18"
        note: >
          20 table pages for 20 rows, because the index only knows where
          customer 42's rows are, not their totals.
      - lines: "20-25"
        note: >
          Index Only Scan with Heap Fetches: 0: 2 index pages plus 1 hit on the
          visibility map, against 22 pages before. INCLUDE columns sit in the
          leaf entries only; they are not part of the key.
      - lines: "27-29"
        note: >
          Payload never guides a scan: a condition on total can't use either
          index. If you need to search or sort by total, make it a key column.
      - lines: "31-33"
        note: >
          79 pages against 21. The key-only index stores each customer_id once
          with a list of 20 row pointers (deduplication); the docs state that
          INCLUDE indexes never deduplicate, and every entry also carries its
          numeric total.
      - lines: "35-39"
        note: >
          Uniqueness applies to the key columns only. A plain UNIQUE (email,
          name) would have accepted this row, since the pair differs; with
          INCLUDE, name is stored but not compared. SQLSTATE 23505.
use_cases:
  - "(tenant_id, created_at) for 'this tenant's latest items' in multi-tenant apps; (status, priority) for a work queue"
  - "Partial: open tickets, pending jobs, rows WHERE deleted_at IS NULL; partial UNIQUE for 'one active subscription per user'"
  - "Expression: case-insensitive lookups (lower(email)), Turkish-aware search keys, jsonb fields (data->>'sku')"
  - "INCLUDE: a hot lookup that needs one or two extra small columns, served by an index-only scan"
strengths:
  - "One composite index serves every query on a prefix of its columns, and returns rows in key order, unlike a hash map keyed by the whole tuple"
  - "A partial index costs nothing for rows outside its WHERE: no space, and no maintenance when those rows change"
  - "An expression index makes a computed comparison as cheap as a column comparison, with no extra column in the table"
  - "UNIQUE ... INCLUDE enforces uniqueness on the key and serves index-only scans for the payload, with one index instead of two"
weaknesses:
  - "Column order serves some queries and not others: (a, b) helps a alone but not b alone (short of PG 18's skip scan, next lesson)"
  - "Partial and expression indexes only match queries written to fit them: the same predicate, the same expression and collation, with no parameters in the predicate"
  - "Every index adds write cost, and an UPDATE touching any indexed column, INCLUDE columns too, gives up HOT (in-place) updates"
  - "INCLUDE indexes never deduplicate and repeat payload in every entry, so they can be several times larger"
pitfalls:
  - title: "Range or sort column first"
    detail: "The range column decides where the scan starts and stops; any column after it only filters entries inside that stretch. Put equality columns first."
    bad: "CREATE INDEX ON orders (placed_on, shop_id);"
    good: "CREATE INDEX ON orders (shop_id, placed_on);"
  - title: "A partial index's condition sent as a parameter"
    detail: "Matching happens at planning time. pgx prepares and caches statements by default, and after five executions PostgreSQL may switch to a generic plan that knows only $1, which can't imply the predicate. Keep the predicate's value literal in the SQL."
    bad: "rows, err := conn.Query(ctx, \"SELECT id FROM tickets WHERE status = $1 ORDER BY opened_on LIMIT 10\", \"open\")"
    good: "rows, err := conn.Query(ctx, \"SELECT id FROM tickets WHERE status = 'open' ORDER BY opened_on LIMIT 10\")"
  - title: "Expression or collation mismatch"
    detail: "An index on lower(city COLLATE \"tr-x-icu\") does nothing for lower(city), upper(city) or city ILIKE ..., and those also compare differently. Pick one normalization, put it in a STORED generated column or a SQL function, and use it everywhere."
  - title: "One partial index per category"
    detail: "The docs warn against a set of indexes like ON t (data) WHERE category = 1, = 2, ...: the planner has to test each one. Use one index on (category, data), or partitioning for truly large tables."
  - title: "Wide INCLUDE columns"
    detail: "Payload is copied into every leaf entry. A long text column bloats the index, and a value too large for an index entry makes the INSERT fail."
quiz:
  - q: "Which index serves `WHERE shop_id = $1 AND placed_on >= $2 ORDER BY placed_on LIMIT 20` best?"
    options: ["(placed_on, shop_id)", "(shop_id, placed_on)", "Separate indexes on shop_id and placed_on"]
    answer: 1
    why: "Equality first makes the shop's rows one contiguous run, already in placed_on order, so the scan starts at $2 and stops after 20. Two separate indexes would need a bitmap, which loses the order."
  - q: "There's a partial index `ON tickets (opened_on) WHERE status = 'open'`. A prepared statement running `WHERE status = $1 ...` has switched to a generic plan. Can it use the index?"
    options: ["Yes, when $1 is 'open'", "No, the plan is made without knowing $1, so it can't prove the predicate", "Only with an index-only scan"]
    answer: 1
    why: "The docs: matching takes place at planning time, so parameterized clauses don't work with a partial index."
  - q: "The index is on `lower(city COLLATE \"tr-x-icu\")`. What does `WHERE lower(city) = 'ısparta-8'` do?"
    options: ["Uses the index and finds the row", "Uses the index but finds nothing", "Seq Scan, and under C.UTF-8 rules finds nothing"]
    answer: 2
    why: "It's a different expression, so the index doesn't apply; and default lower() maps I to i, giving 'isparta-8'."
  - q: "With `CREATE UNIQUE INDEX ON users (email) INCLUDE (name)`, can two rows share an email if their names differ?"
    options: ["Yes, uniqueness covers (email, name)", "No, uniqueness covers email only"]
    answer: 1
    why: "INCLUDE columns are payload; the uniqueness condition applies only to the key columns."
further_reading:
  - title: "PostgreSQL docs: Multicolumn Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-multicolumn.html"
  - title: "PostgreSQL docs: Indexes and ORDER BY"
    url: "https://www.postgresql.org/docs/current/indexes-ordering.html"
  - title: "PostgreSQL docs: Partial Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-partial.html"
  - title: "PostgreSQL docs: Indexes on Expressions"
    url: "https://www.postgresql.org/docs/current/indexes-expressional.html"
  - title: "PostgreSQL docs: Index-Only Scans and Covering Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-index-only-scans.html"
  - title: "PostgreSQL docs: PREPARE (generic vs custom plans)"
    url: "https://www.postgresql.org/docs/current/sql-prepare.html"
---

## Concept

An index is a sorted copy of some expression of each row, plus a pointer back. Four knobs
decide what it is good for:

- **Column order** (composite indexes). Entries are sorted by the first column, then the
  second within it, and so on. The docs' rule: equality conditions on leading columns,
  plus a range on the first column without an equality, limit the part of the index
  that's scanned. Conditions on later columns only filter inside that part. So:
  **equality columns first, then one range or `ORDER BY` column**.
- **`WHERE`** (partial indexes). Only rows matching the predicate get entries. The query
  must imply the predicate, provably, at planning time.
- **Expressions**. Index `lower(email)`, `(data->>'sku')`, `date_trunc('day', at)`. The
  query must use the same expression.
- **`INCLUDE`** (covering indexes, PG 11+). Extra columns stored in the leaf entries so an
  index-only scan can answer without the table. Not searchable, not part of uniqueness.

The design question is never "which columns are important?" but "which queries run
often, and what exactly do they filter, sort and return?"

## Coming from C/C++/Java/Python/JS

- **Java's `Comparator.comparing(Order::shopId).thenComparing(Order::placedOn)`**, or Go's
  `slices.SortFunc` with `cmp.Or(cmp.Compare(a.ShopID, b.ShopID), a.PlacedOn.Compare(b.PlacedOn))`,
  is exactly the order of a `(shop_id, placed_on)` index. A `TreeSet.subSet(from, to)` over
  it is the range scan: contiguous only when the leading part of the key is fixed.
- **Python:** `bisect` on a list of `(shop_id, placed_on)` tuples finds shop 7's March in
  one step; on `(placed_on, shop_id)` tuples, shop 7's rows are interleaved with 99 other
  shops.
- **JS:** a `Map` keyed by `name.toLocaleLowerCase('tr-TR')` is an expression index. A
  lookup with `name.toLowerCase()` uses a different key function and misses, which is
  the collation mismatch from the second example.
- **Covering index:** a denormalized cache inside the index. Like any cache, it costs space
  and must be kept in sync on every write; PostgreSQL does the syncing for you.

## How it works

The two indexes of the first example, as their leaf entries are ordered:

```text
(shop_id, placed_on):  ... (6, 2026-07-19) | (7, 2026-01-01) ... (7, 2026-03-01) ... (7, 2026-04-30) ... | (8, ...)
                                                                 \___ one run of 61 entries ___/

(placed_on, shop_id):  (2026-03-01, 1) (2026-03-01, 2) ... (2026-03-01, 7) ... (2026-04-30, 100)
                       \___ 6,100 entries scanned, each checked for shop_id = 7 ___/
```

`INCLUDE` vs adding the column to the key: `(customer_id, total)` would also cover the
query, and could additionally search and sort by `total`. INCLUDE columns don't need a
B-tree operator class, are kept out of the upper levels of the tree, and stay out of
uniqueness checks, which is why `UNIQUE (email) INCLUDE (name)` exists.

Since PG 18, a B-tree can sometimes use `(a, b)` for a condition on `b` alone by skipping
through the distinct values of `a`. That **skip scan** is the next lesson's topic; it
softens the column-order rule but doesn't replace it.

## Design notes

Work from the query list (lesson 5 shows where to get it: `pg_stat_statements`), one
query shape at a time:

1. Equality columns of the `WHERE` first. Among them, prefer the order that lets other
   queries share the index as a prefix (`tenant_id` first in a multi-tenant schema).
2. Then the one column that is a range or the `ORDER BY`. A second range column after it
   only filters.
3. If the query touches a small, well-defined subset (open, pending, not deleted), make it
   partial, and keep that condition literal in the SQL.
4. If it compares a computed value, index the expression, or store it in a `STORED`
   generated column when several queries need it (the generated-columns lesson). For
   Turkish text, put the collation inside the expression.
5. Only for hot, read-mostly lookups: `INCLUDE` the one or two small columns the query
   returns, then confirm `Heap Fetches` stays near zero in production.

Then merge: an index on `(a, b)` makes a separate index on `(a)` redundant. Each index you
keep slows every write, so drop the ones no query uses (`pg_stat_user_indexes.idx_scan`
tells you which, see the monitoring lesson).
