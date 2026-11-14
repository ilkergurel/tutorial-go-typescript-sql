---
id: sql.joins.inner-outer
title: "Inner, left/right/full outer, cross and self joins"
track: sql
module: joins
order: 1
level: basic
minutes: 13
since: pg7.1
prerequisites: [sql.querying.null-logic, sql.constraints.fk-check]
compare_with: [go.composite.slices-and-maps, ts.functions.higher-order, sql.joins.many-to-many]
tags: [joins, inner-join, outer-join, cross-join, self-join]
summary: >
  A join pairs rows from two tables by a condition. INNER keeps matched pairs;
  LEFT, RIGHT and FULL also keep unmatched rows, padded with NULLs; CROSS pairs
  everything; a self join pairs a table with itself.
examples:
  - id: inner-left
    file: examples/01_inner_left.sql
    title: "INNER vs LEFT, and where the filter on the optional side goes"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          A nullable foreign key: a test case may exist before anyone links it.
          A NULL key never matches anything, not even another NULL, so the
          smoke test can't take part in a join on requirement_id.
      - lines: "16-20"
        note: >
          JOIN means INNER JOIN. A row without a partner disappears from either
          side: Export CSV has no tests, the smoke test has no requirement.
      - lines: "22-26"
        note: >
          LEFT JOIN keeps every row of the left table and fills the right
          side's columns with NULL when nothing matched. Add WHERE t.id IS NULL
          and this becomes the "requirements without tests" report. To list
          unlinked tests instead, put test_cases on the left.
      - lines: "28-33"
        note: >
          WHERE runs after the join. On the padded rows t.automated is NULL,
          NULL is not true, so those rows go, and so does Password reset,
          whose only test is manual. The LEFT JOIN has quietly become an INNER
          JOIN.
      - lines: "35-39"
        note: >
          In ON, the condition decides what counts as a match before the
          padding happens, so every requirement stays. Rule of thumb:
          conditions on the optional side go in ON, filters on the preserved
          side go in WHERE. For inner joins the two placements are equivalent.
  - id: full-cross
    file: examples/02_full_cross.sql
    title: "FULL for reconciliation, CROSS + LEFT for expected-versus-actual"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-11"
        note: >
          FULL JOIN keeps unmatched rows from both sides: logout was planned
          but never reported, signup was reported but never planned. coalesce
          picks whichever side has the key, so the sort works for every row.
      - lines: "27-32"
        note: >
          CROSS JOIN produces every browser with every locale (2 x 2 = 4 rows):
          the runs that should exist. LEFT JOIN the runs that do exist, and a
          NULL marks firefox/tr-TR as never tested. The same shape finds
          missing days in a time series.
      - lines: "30"
        note: >
          A CROSS JOIN of N and M rows yields N x M rows. Here that is the
          point; when it happens by accident (a comma join with a forgotten
          WHERE), it is the classic exploding result.
  - id: self-join
    file: examples/03_self_join.sql
    title: "Self joins: parent lookup and duplicate pairs, with Turkish case rules"
    runtime: sql
    check: exact
    annotations:
      - lines: "4"
        note: >
          parent_id references the same table. One level up is a self join;
          walking the whole tree needs a recursive CTE (subqueries and CTEs
          module).
      - lines: "16-20"
        note: >
          The aliases c and p give one table two roles. LEFT keeps the root
          requirements, whose parent is NULL.
      - lines: "22-26"
        note: >
          To compare rows of one table with each other, join it to itself.
          a.id < b.id keeps each unordered pair once and never pairs a row with
          itself; a.id <> b.id would list every pair twice.
      - lines: "28-34"
        note: >
          Under the database's C.UTF-8 rules lower('KAYIT FORMU') is
          'kayit formu', with a dotted i, so it never equals 'kayıt formu'.
          With COLLATE "tr-x-icu", I lowers to ı and the duplicate is found.
use_cases:
  - "INNER: the everyday 'rows with their parent' query, such as test cases with their requirement"
  - "LEFT ... WHERE right.id IS NULL: coverage gaps, orphans, customers without orders"
  - "FULL: reconciling two sources such as a test plan and CI results, or a ledger and a bank statement"
  - "CROSS + LEFT: expected-versus-actual matrices (browser x locale, day x sensor)"
  - "Self join: parent and child in one table, duplicate detection, comparing a row with its neighbour"
strengths:
  - "Declarative: you state the condition, and the planner chooses a nested loop, hash join or merge join from table statistics, where Go or TS code would hard-wire one strategy"
  - "Outer joins make missing data visible as NULL-padded rows instead of rows that silently aren't there"
  - "A join condition can be any boolean expression: ranges, lower(), inequalities, not just key equality"
weaknesses:
  - "NULL-padded rows meet three-valued logic: a WHERE on the optional side undoes the outer join"
  - "Joining a parent to two independent child tables multiplies rows (2 tests x 3 comments = 6 rows per requirement) and inflates counts and sums"
  - "RIGHT JOIN reads backwards; swapping the tables gives the same result as a LEFT JOIN, which most code standardizes on"
pitfalls:
  - title: "Filtering the optional side in WHERE"
    detail: "The NULL-padded rows fail the condition and vanish, turning the outer join into an inner one. Move the condition into ON."
    bad: "FROM requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id WHERE t.automated"
    good: "FROM requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id AND t.automated"
  - title: "Fan-out from two one-to-many joins"
    detail: "requirements JOIN test_cases JOIN comments returns tests x comments rows per requirement, so count(*) and sum() come out multiplied. Aggregate each child separately (subqueries, or LATERAL in this module's last lesson)."
  - title: "Comma joins hide a missing condition"
    detail: "FROM a, b with a forgotten WHERE clause is a cross join. Explicit JOIN ... ON keeps the condition next to the table it belongs to, and leaving out ON is a syntax error."
    bad: "FROM requirements r, test_cases t  -- WHERE forgotten"
    good: "FROM requirements r JOIN test_cases t ON t.requirement_id = r.id"
  - title: "Expecting NULL keys to match"
    detail: "NULL = NULL is unknown, so rows with a NULL key never join, not even to each other. Model 'not linked' deliberately and use an outer join where it matters."
quiz:
  - q: "`requirements r LEFT JOIN test_cases t ON t.requirement_id = r.id WHERE t.automated`: what happens to requirements that have no tests?"
    options: ["They appear with NULLs", "They disappear", "The query fails"]
    answer: 1
    why: "Their padded t.automated is NULL, WHERE keeps only true, so they are filtered out after the join."
  - q: "Which join returns unmatched rows from both tables?"
    options: ["INNER JOIN", "LEFT JOIN", "FULL JOIN", "CROSS JOIN"]
    answer: 2
    why: "FULL JOIN pads whichever side is missing, so rows that exist on only one side appear either way."
  - q: "browsers has 4 rows and locales 5. How many rows does browsers CROSS JOIN locales return?"
    options: ["5", "9", "20"]
    answer: 2
    why: "A cross join is the Cartesian product: 4 x 5 = 20."
  - q: "In a self join looking for duplicate titles, why `a.id < b.id` rather than `a.id <> b.id`?"
    options: ["It is faster to compare", "It reports each pair once instead of twice", "<> doesn't work on integers"]
    answer: 1
    why: "<> keeps both (x, y) and (y, x); < keeps only one ordering and still excludes pairing a row with itself."
further_reading:
  - title: "PostgreSQL docs: Joined Tables (table expressions)"
    url: "https://www.postgresql.org/docs/current/queries-table-expressions.html"
  - title: "PostgreSQL tutorial: Joins Between Tables"
    url: "https://www.postgresql.org/docs/current/tutorial-join.html"
---

## Concept

A join combines two tables into pairs of rows for which a condition holds. The kinds
differ only in what happens to rows that find no partner:

| Join | Matched pairs | Unmatched left rows | Unmatched right rows |
|---|---|---|---|
| `[INNER] JOIN` | kept | dropped | dropped |
| `LEFT [OUTER] JOIN` | kept | kept, right side NULL | dropped |
| `RIGHT [OUTER] JOIN` | kept | dropped | kept, left side NULL |
| `FULL [OUTER] JOIN` | kept | kept | kept |
| `CROSS JOIN` | every combination, no condition | | |

A self join is not a separate kind: it is any of the above with the same table on both
sides under two aliases.

## Coming from C/C++/Java/Python/JS

- Python/pandas: `merge(how="inner" | "left" | "right" | "outer" | "cross")` maps
  one-to-one onto these joins, including the NaN padding (NULL here).
- Go/TS: a join is the nested loop or the map lookup you would write by hand (build a
  map by `requirementId`, then probe it per row). The planner picks between those
  strategies from table sizes; you only state the condition.
- JS: an inner join is a `flatMap` that emits one pair per match; a left join is the same
  `flatMap` emitting `{ ...r, test: null }` when the match list is empty.
- Java/JPA: forgetting to join is what produces the N+1 query problem. One join is one
  round trip.

## Design notes

Before writing an outer join, decide which side may be missing and put the preserved
table first, then use LEFT JOIN consistently; mixing LEFT and RIGHT in one query is hard
to read. Conditions on the optional side belong in ON, filters on the preserved side in
WHERE.

Keep join conditions on keys the schema declares (foreign key to primary key) wherever
possible: they are indexed and they can't drift. Expression joins such as `lower(a.title)
= lower(b.title)` are legitimate for analysis and clean-up, and the Turkish example shows
why their collation has to be chosen on purpose.
