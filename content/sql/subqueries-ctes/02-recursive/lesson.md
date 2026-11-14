---
id: sql.subqueries-ctes.recursive
title: "Recursive CTEs: requirement hierarchies and trace chains, with the CYCLE clause"
track: sql
module: subqueries-ctes
order: 2
level: advanced
minutes: 15
since: pg14
prerequisites: [sql.subqueries-ctes.scalar, sql.joins.many-to-many]
compare_with: [go.iterators.writing, fullstack.tracelite.step-02-schema]
tags: [cte, with-recursive, recursion, tree, graph, cycle, search, hierarchy, traceability]
summary: >
  WITH RECURSIVE runs a start query, then repeats a step query on the rows the
  previous round produced until a round returns nothing: enough to walk
  requirement trees down and up, and trace graphs. SEARCH orders the result
  depth- or breadth-first, and CYCLE (PG 14) stops paths that revisit a node.
examples:
  - id: tree
    file: examples/01_tree.sql
    title: "A requirement tree: walking down with SEARCH DEPTH FIRST, and walking up"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-16"
        note: >
          The non-recursive term runs once and produces the starting rows,
          here just AUTH at depth 0.
      - lines: "17"
        note: >
          UNION ALL keeps every row. A tree has no duplicates to discard, so the
          cheaper ALL is right; graphs are another matter (example 2).
      - lines: "18-20"
        note: >
          In each round, subtree stands for the rows the previous round
          produced, not everything so far. Round 1 finds AUTH's children, round
          2 their children, round 3 finds nothing and the query stops. Each
          round joins on parent_id, so a real table wants an index on it.
      - lines: "21"
        note: >
          SEARCH DEPTH FIRST BY code (PG 14) adds a column, ord, that sorts
          every child right after its parent and siblings by code. Without it
          the rows come out breadth-first, which the docs call an
          implementation detail it is "perhaps unsound" to rely on.
      - lines: "27-34"
        note: >
          Walking up is the same shape with the join reversed: each round finds
          the parent of the last row. It stops after AUTH, whose NULL parent_id
          matches nothing. The primary key index serves this direction.
  - id: trace-graph
    file: examples/02_trace_graph.sql
    title: "A trace graph with a cycle: the naive loop, CYCLE, and UNION"
    runtime: sql
    check: exact
    annotations:
      - lines: "16-27"
        note: >
          Without line 22 this query never ends: SYS-2 leads to SYS-3, SYS-3
          back to SYS-2, and every lap produces rows that differ in hops. With
          the guard it ends at 6 hops, having reached SYS-2 three times and
          TC-10 four times. A depth limit bounds the damage; it doesn't detect
          the cycle.
      - lines: "36"
        note: >
          CYCLE code SET is_cycle USING path adds two columns. path records the
          codes visited on the way to each row; when a step reaches a code
          already on its path, that row is emitted with is_cycle = true and not
          extended. Here SYS-2 at hop 3 closes the loop and the walk ends.
      - lines: "39"
        note: >
          path is an array of row values, so ORDER BY path lists every route as
          a depth-first outline. TC-10 appears twice, once per route: in a
          graph, reachable through several paths is normal.
      - lines: "43-48"
        note: >
          UNION discards rows that duplicate any earlier result row. With only
          the node itself in each row, a revisited node is a duplicate and the
          loop runs dry. Add hops or a path to the row and that protection is
          gone, which is why the CYCLE version needs CYCLE.
  - id: rollup-coverage
    file: examples/03_rollup_coverage.sql
    title: "Coverage per top-level requirement: carry the root down, then aggregate"
    runtime: sql
    check: exact
    annotations:
      - lines: "21-27"
        note: >
          Each round copies root_id from parent to child, so every requirement
          ends up tagged with its top-level ancestor. Anything derivable from
          the parent can be carried the same way: depth, a path, an inherited
          owner.
      - lines: "29-30"
        note: >
          Test 10 is linked to AUTH-LOGIN and to AUTH-LOGIN-2FA, both inside
          AUTH's subtree: 4 links but 3 distinct tests. Rolling up over a
          hierarchy needs DISTINCT, or a test is counted once per requirement
          it covers.
      - lines: "31"
        note: >
          FILTER from the GROUP BY lesson: failing tests per root, in the same
          pass as the totals.
      - lines: "34-35"
        note: >
          LEFT JOINs keep EXPORT even though nothing below it has tests; its
          counts are 0, because count skips the NULL padding.
use_cases:
  - "Requirement and document hierarchies: everything under a requirement, the breadcrumb of ancestors, outlines in reading order"
  - "Traceability chains: from a stakeholder need through system requirements to the tests that verify it, and back"
  - "Org charts, category trees, threaded comments, bill-of-materials explosions"
  - "Impact analysis on dependency graphs: what is affected if this component changes"
strengths:
  - "The whole walk is one query and one round trip, instead of one query per level from Go or TS"
  - "The tree stays a plain parent_id column with a foreign key, so writes are simple and integrity is enforced"
  - "SEARCH and CYCLE (PG 14) are standard SQL and replace hand-written path arrays and ANY() checks"
  - "Values can be carried down the walk (root, depth, path), so roll-ups need no second pass"
weaknesses:
  - "UNION ALL over cyclic data never terminates; safety has to be written into every graph query"
  - "Each round is a join, so deep trees cost one join per level, and an unindexed parent_id means one table scan per level"
  - "Row order is undefined without SEARCH or an explicit ORDER BY on a path"
  - "Read-heavy, deep hierarchies can be cheaper with a materialized path (the ltree extension) or a closure table, at the cost of more complex writes"
pitfalls:
  - title: "UNION ALL over a graph without a cycle check"
    detail: "One bad link (A to B, B to A) makes the query run until a timeout or a resource limit stops it. Add CYCLE, or use UNION when each row is just the node."
    bad: "WITH RECURSIVE reach AS (SELECT 'STK-1'::text AS code, 0 AS hops UNION ALL SELECT l.to_code, r.hops + 1 FROM reach r JOIN trace_links l ON l.from_code = r.code) SELECT * FROM reach"
    good: "WITH RECURSIVE reach AS (... same ...) CYCLE code SET is_cycle USING path SELECT * FROM reach"
  - title: "Expecting UNION to stop a walk that carries a depth"
    detail: "UNION only drops rows that are entirely equal. (SYS-2, 1) and (SYS-2, 3) differ, so a cycle repeats forever. The docs make the same point for depth and path columns."
  - title: "Relying on the output order"
    detail: "Rows currently come out breadth-first, but the docs call that an implementation detail. Use SEARCH DEPTH FIRST or SEARCH BREADTH FIRST and ORDER BY the column it creates."
  - title: "LIMIT as a loop guard"
    detail: "A LIMIT in the outer query does stop runaway recursion in PostgreSQL, because only the rows fetched are computed. The docs recommend it for testing only: other systems might work differently."
  - title: "Double counting in roll-ups"
    detail: "A test linked to several requirements in one subtree is reached once per link. Count DISTINCT test ids, or deduplicate before aggregating."
    bad: "count(l.test_case_id)"
    good: "count(DISTINCT l.test_case_id)"
quiz:
  - q: "During one round of the recursive term, what does the self-reference (subtree in example 1) contain?"
    options: ["Every row produced so far", "Only the rows the previous round produced", "The whole requirements table"]
    answer: 1
    why: "The working table holds the previous round's new rows; each round's output replaces it, and the query stops when a round adds nothing."
  - q: "The trace links contain a cycle. Which version is guaranteed to terminate?"
    options: ["UNION ALL, rows of (code, hops)", "UNION, rows of (code, hops)", "UNION ALL with CYCLE code SET is_cycle USING path"]
    answer: 2
    why: "Rows with a growing hops value are never equal, so neither UNION nor UNION ALL stops. CYCLE checks the code against the path and stops extending a path that revisits it."
  - q: "Why does example 3 count DISTINCT test ids per root?"
    options: ["count(*) would count NULLs", "A test linked to two requirements in the same subtree would otherwise count twice", "DISTINCT makes the recursion stop"]
    answer: 1
    why: "The walk reaches the test once per linked requirement. Distinct ids count each test once per root."
  - q: "What does SEARCH DEPTH FIRST BY code SET ord give you?"
    options: ["A column to ORDER BY so each child follows its parent, siblings by code", "A guarantee that the CTE's output is already sorted", "Cycle detection"]
    answer: 0
    why: "SEARCH only adds the ordering column; you still ORDER BY it. Cycle detection is CYCLE's job."
further_reading:
  - title: "PostgreSQL docs: Recursive Queries"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-RECURSIVE"
  - title: "PostgreSQL docs: Search Order (SEARCH)"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-SEARCH"
  - title: "PostgreSQL docs: Cycle Detection (CYCLE)"
    url: "https://www.postgresql.org/docs/current/queries-with.html#QUERIES-WITH-CYCLE"
  - title: "PostgreSQL docs: SELECT, WITH clause (SEARCH and CYCLE syntax)"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-WITH"
---

## Concept

`WITH RECURSIVE` defines a CTE in two parts joined by `UNION ALL` (or `UNION`): a
**non-recursive term** that produces the starting rows, and a **recursive term** that
refers to the CTE itself. Despite the name, it is a loop, not recursion: the start rows
form the first round; each further round runs the recursive term against the previous
round's rows only; all rounds together are the result; the loop stops when a round
produces no rows.

That loop walks a **tree** stored as `parent_id`: down (children of the last round) or
up (the parent of the last row). It also walks a **graph**, such as trace links from
stakeholder needs through system requirements to test cases. Graphs bring two things
trees don't: a node reachable by several paths, and cycles. A cycle makes a `UNION ALL`
walk run forever.

PG 14 added two standard clauses after the CTE: `SEARCH DEPTH FIRST | BREADTH FIRST BY
col SET ordcol` adds a column to sort by, and `CYCLE col SET is_cycle USING path` tracks
the path and stops extending a path when it revisits a node.

## Coming from C/C++/Java/Python/JS

- **Any language:** this is the breadth-first frontier loop you have written many
  times: `frontier = [start]; while frontier: frontier = children_of(frontier);
  result += frontier`. The working table is `frontier`. `CYCLE` is a per-path `visited`
  check; `UNION` is a global `visited` set over whole rows.
- **Go and TS services:** without this, you either query once per level (N round trips)
  or load the whole table and build a `map[int][]int` of children in memory. The
  recursive CTE runs the loop next to the data and returns only the result.
- **Java/JPA:** a self-referencing `@OneToMany children` collection is lazy by default,
  so walking down issues one query per node whose children you touch. A native query
  with `WITH RECURSIVE` replaces the walk.
- **Python:** `networkx.descendants(G, 'STK-1')` returns the set of reachable nodes,
  which is what the `UNION` version of example 2 computes.

## How it works

Example 1, round by round:

1. Start: `AUTH` (depth 0).
2. Children of `AUTH`: `AUTH-LOGIN`, `AUTH-RESET` (depth 1).
3. Children of those: `AUTH-LOGIN-2FA`, `AUTH-LOGIN-LOCKOUT` (depth 2).
4. Children of those: none. Stop.

With `UNION` instead of `UNION ALL`, each round also drops rows equal to any row already
produced; that is what lets the node-only walk in example 2 terminate. `CYCLE` is
documented as a rewrite of a hand-written pattern: carry an array of visited keys,
mark a row whose key is already in it, and don't extend marked rows. `SEARCH` is a
similar rewrite that carries a sort key.

## Design notes

For hierarchies, start with an adjacency list (`parent_id REFERENCES` the same table)
and recursive CTEs: writes stay trivial, the foreign key keeps the tree honest, and
reads are one query. Index `parent_id` for downward walks. Consider a materialized path
(the `ltree` extension) or a closure table only when deep, read-heavy trees show up in
`EXPLAIN ANALYZE`.

For graphs, make every traversal cycle-safe, even if the application is supposed to
prevent cycles: one bad link must not turn a page load into a timeout. Use `UNION` when
you need the set of reachable nodes, and `CYCLE` when you need paths, depths or
per-path data.

Carry what you need down the walk (root id, depth, path) and aggregate afterwards, and
remember that in a graph, or a roll-up over a tree with many-to-many links, the same
leaf arrives once per path: count distinct ids.
