---
id: sql.querying.select
title: "SELECT/WHERE/ORDER BY/LIMIT"
track: sql
module: querying
order: 1
level: basic
minutes: 14
since: pg13
prerequisites: [sql.basics.set-based]
compare_with: [ts.functions.higher-order, go.composite.slices-maps-pkg, go.basics.strings-runes]
tags: [select, where, order-by, limit, collation]
summary: >
  The everyday query: name the columns, filter rows with WHERE, sort with a
  deterministic ORDER BY, cut with LIMIT. Clauses run in a fixed logical order,
  NULLs sort first under DESC, and the default sort order is not Turkish
  alphabetical.
examples:
  - id: where
    file: examples/01_where.sql
    title: "Columns, filters, and when each clause runs"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-21"
        note: >
          Name the columns you need: the result's shape stays under your
          control when someone adds a column. ORDER BY may use the alias
          stock_value because sorting happens after the SELECT list is
          computed.
      - lines: "26"
        note: >
          AND binds tighter than OR, so this is outdoor OR (kitchen AND in
          stock), and the out-of-stock Hamak comes back. The same precedence
          as && over || in Go, Java and JS; SQL just makes it easier to miss.
      - lines: "31"
        note: >
          IN (a, b) means = a OR = b, grouped for you. With the parentheses
          implied, the AND applies to both categories.
      - lines: "36"
        note: >
          BETWEEN includes both ends. With timestamps that is a trap:
          BETWEEN '2026-11-01' AND '2026-11-30' stops at midnight on the 30th.
          Use a half-open range there (>= start AND < next start).
      - lines: "40-42"
        note: >
          Logically, FROM and WHERE run before the SELECT list exists, so
          WHERE can't see its aliases (SQLSTATE 42703). Repeat the
          expression, or wrap the query in a subquery and filter outside.
  - id: order-limit
    file: examples/02_order_limit.sql
    title: "ORDER BY, NULL placement, ties, LIMIT and pages"
    runtime: sql
    check: exact
    annotations:
      - lines: "12-13"
        note: >
          Without ORDER BY, rows come back in whatever order the plan produces.
          Here that is storage order, and the UPDATE wrote deniz's new row
          version after the others. It is repeatable on this tiny table, which
          is exactly why code that relies on it passes tests and breaks later.
      - lines: "16"
        note: >
          NULL sorts as if larger than any value: last under ASC, first under
          DESC. Emre, who hasn't played, tops the leaderboard.
      - lines: "20-22"
        note: >
          umut and zeynep tie at 85. Without the id tie-breaker either could
          be cut at LIMIT 2, and the choice can change with the plan. A unique
          last sort key makes the result, and every page, deterministic.
      - lines: "25-27"
        note: >
          OFFSET pages only work over the same deterministic order. The skipped
          rows are still computed, so deep pages get slower; keyset pagination
          (indexes module) avoids that.
      - lines: "31-35"
        note: >
          FETCH FIRST is the SQL-standard spelling (LIMIT is PostgreSQL's and
          MySQL's); WITH TIES (PG 13) also returns rows that tie with the
          last one. Ties have no order among themselves, so the outer ORDER BY
          fixes the display.
  - id: turkish-order
    file: examples/03_turkish_order.sql
    title: "Sorting Turkish names: code points vs tr-x-icu"
    runtime: sql
    check: exact
    annotations:
      - lines: "8"
        note: >
          StackLab's local databases use the builtin C.UTF-8 collation, which
          orders by code point, as Go's string comparison does. Fast and the
          same on every OS, but Çorum lands after Zonguldak and Ağrı after
          Ankara.
      - lines: "12"
        note: >
          tr-x-icu is ICU's Turkish collation: Ç after C, ğ after g, I before
          İ. An index on name is built in the column's collation and can't
          serve this ORDER BY; declare the column COLLATE "tr-x-icu", or add an
          index with that collation, if this is the order you sort by.
      - lines: "16-18"
        note: >
          Collation decides comparisons, not just sorting: "before J" is 4
          cities by code point and 6 by Turkish rules, because Ç and İ come
          before J in the Turkish alphabet.
use_cases:
  - "List screens: filter, sort by a user-chosen column plus a unique tie-breaker, LIMIT for the page size"
  - "Leaderboards and \"top N\" widgets: ORDER BY ... DESC NULLS LAST, FETCH FIRST n ROWS WITH TIES"
  - "Turkish-facing lists (cities, customer names) sorted with COLLATE \"tr-x-icu\""
strengths:
  - "WHERE runs next to the data and can use indexes; a Go or TS .filter() runs after every row has crossed the network"
  - "NULLS FIRST/LAST and FETCH ... WITH TIES state edge cases declaratively that hand-written comparators usually get wrong"
  - "Besides the database default, collation is a per-column or per-query choice, like passing a Turkish Intl.Collator to sort()"
weaknesses:
  - "Written order (SELECT ... FROM ... WHERE) differs from evaluation order, which is why aliases fail in WHERE"
  - "No order without ORDER BY: tests on small tables can pass while relying on an order nothing promises"
  - "OFFSET pagination recomputes every skipped row, so deep pages get slower; keyset pagination needs more care"
pitfalls:
  - title: "SELECT * in application queries"
    detail: "Every column travels to the app, a new column changes every result, and an index-only scan would need every column in the index. Name the columns."
  - title: "Mixing AND and OR without parentheses"
    detail: "a OR b AND c means a OR (b AND c). Parenthesize, or use IN for alternatives on one column."
    bad: "WHERE status = 'new' OR status = 'paid' AND total > 100"
    good: "WHERE status IN ('new', 'paid') AND total > 100"
  - title: "Paging without a unique tie-breaker"
    detail: "ORDER BY created_at LIMIT 20 OFFSET 20 can repeat or skip rows when timestamps tie. End the ORDER BY with a unique column."
    good: "ORDER BY created_at DESC, id DESC LIMIT 20"
  - title: "DESC puts NULLs on top"
    detail: "NULLS FIRST is the default for DESC. Say NULLS LAST when missing values belong at the bottom."
quiz:
  - q: "SELECT price * qty AS total FROM items WHERE total > 100 fails with 'column \"total\" does not exist'. Why?"
    options:
      - "Aliases need double quotes"
      - "WHERE is evaluated before the SELECT list, so the alias doesn't exist yet"
      - "Computed columns can't be filtered"
    answer: 1
    why: "Logical order is FROM, WHERE, SELECT, ORDER BY, LIMIT. ORDER BY can use the alias; WHERE can't."
  - q: "How does PostgreSQL read WHERE a = 1 OR a = 2 AND b = 3?"
    options:
      - "(a = 1 OR a = 2) AND b = 3"
      - "a = 1 OR (a = 2 AND b = 3)"
      - "It's a syntax error without parentheses"
    answer: 1
    why: "AND binds tighter than OR, as && does over || in Go and JS."
  - q: "ORDER BY rating DESC on a column with some NULLs puts the NULL rows…"
    options:
      - "first"
      - "last"
      - "nowhere: NULL rows are skipped"
    answer: 0
    why: "NULL sorts as larger than any value, and NULLS FIRST is the default for DESC."
  - q: "With ORDER BY created_at LIMIT 20 OFFSET n, some rows show up on two pages and others on none. The fix?"
    options:
      - "Add a unique tie-breaker: ORDER BY created_at, id"
      - "Use FETCH FIRST instead of LIMIT"
      - "Add an index on created_at"
    answer: 0
    why: "Rows with equal created_at have no defined order, so each page query may arrange them differently. A unique last key fixes the order."
further_reading:
  - title: "PostgreSQL docs: SELECT"
    url: "https://www.postgresql.org/docs/current/sql-select.html"
  - title: "PostgreSQL docs: Sorting Rows (ORDER BY)"
    url: "https://www.postgresql.org/docs/current/queries-order.html"
  - title: "PostgreSQL docs: LIMIT and OFFSET"
    url: "https://www.postgresql.org/docs/current/queries-limit.html"
  - title: "PostgreSQL docs: Collation Support"
    url: "https://www.postgresql.org/docs/current/collation.html"
---

## Concept

A `SELECT` is written in one order and evaluated in another:

1. `FROM`: where the rows come from
2. `WHERE`: keep the rows that satisfy the condition
3. `SELECT` list: compute the output columns (and their aliases)
4. `ORDER BY`: sort, by output columns, aliases or any expression
5. `LIMIT` / `OFFSET` (or `FETCH FIRST`): cut the result

(`GROUP BY` and `HAVING` slot in between 2 and 3; they get their own lesson.) Most
surprises in this lesson follow from that order: aliases don't exist in `WHERE`, and
nothing is ordered until step 4. A result without `ORDER BY` has no order you may rely
on, so `LIMIT` without `ORDER BY` returns *some* rows, not the first ones.

Sorting text depends on a collation. StackLab's local-mode databases use the builtin
`C.UTF-8` collation, which compares Unicode code points: fast, identical on every OS, and not
alphabetical for Turkish. `COLLATE "tr-x-icu"` asks for Turkish rules where you need them.

## Coming from C/C++/Java/Python/JS

- **JS arrays**: `WHERE` is `.filter()`, the `SELECT` list is `.map()`, `ORDER BY` is
  `.sort()`, `LIMIT/OFFSET` is `.slice()`. The difference is where they run: next to
  the data, possibly through an index, before anything crosses the network.
- **Comparators**: `ORDER BY points DESC NULLS LAST, id` is Java's
  `Comparator.comparing(...).reversed().thenComparing(...)` or Go's
  `cmp.Or(cmp.Compare(b.points, a.points), cmp.Compare(a.id, b.id))`, with NULL
  placement built in. Like Go's `slices.SortFunc`, SQL's sort is not stable: rows that
  compare equal come out in any order.
- **String order**: Go compares strings byte by byte and JS's `<` compares UTF-16 code
  units; for Turkish letters both give the same order as `C.UTF-8`, Ç after Z. The
  equivalents of `COLLATE "tr-x-icu"` are `Intl.Collator('tr')` and Java's
  `Collator.getInstance(Locale.forLanguageTag("tr"))`.
- **Python**: `sorted(rows, key=...)[:10]` sorts everything, then slices. PostgreSQL
  takes `LIMIT` into account when planning and can stop early, for example by reading
  just the first entries of a matching index.

## Design notes

Give every query an explicit `ORDER BY` that ends in a unique column whenever the order
is visible to someone: a UI list, an export, a page. It costs little and turns
"usually right" into "always right". For Turkish-facing data, decide the collation in
the schema (`name text COLLATE "tr-x-icu"`) when it's the only order you'll ever sort
by, so indexes and comparisons follow it; keep the code-point default when the column
holds identifiers, codes or e-mails, where linguistic order has no meaning and code
point order is cheaper.
