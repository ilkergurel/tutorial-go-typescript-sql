---
id: sql.querying.null-logic
title: "NULL and three-valued logic"
track: sql
module: querying
order: 2
level: basic
minutes: 10
since: pg9.0
prerequisites: [sql.querying.select]
compare_with: [ts.everyday-types.null, design.paradigms.generics-absence, go.errors.errors-as-values]
tags: ["null", "three-valued-logic", "predicates"]
summary: >
  NULL means "unknown", not "empty". Any comparison with NULL yields unknown,
  which WHERE treats as false — the root of most NULL surprises, and the reason
  IS DISTINCT FROM and COALESCE exist.
examples:
  - id: three-values
    file: examples/01_three_values.sql
    title: x <> 'a' doesn't match NULL; IS DISTINCT FROM does
    runtime: sql
    check: exact
    annotations:
      - lines: "10"
        note: >
          brin's email is NULL, and NULL <> 'cay@example.com' evaluates to
          unknown, not true. WHERE keeps only true — so NULL rows vanish.
      - lines: "13"
        note: >
          IS DISTINCT FROM treats NULL as a value you can compare: NULL is
          distinct from 'cay@example.com', so brin appears. It's the
          null-safe inequality.
      - lines: "16"
        note: >
          COALESCE substitutes a display value for NULL. Don't use it to
          "fix" data in UPDATEs — store what you mean, render what you mean.
use_cases:
  - "Audit queries where 'not X' must include missing values (IS DISTINCT FROM / IS NULL)"
  - "Display defaults (COALESCE(nickname, name))"
  - "Deduplication guards (NULLIF in unique-safe upserts)"
strengths:
  - "Three-valued logic is mathematically consistent: unknown propagates honestly through AND/OR/NOT"
  - "IS DISTINCT FROM and IS NULL give you exact, index-friendly null handling"
  - "CHECK constraints treat unknown as pass, so partial data isn't rejected by mistake"
weaknesses:
  - "Every predicate silently returns unknown on NULL: aggregates skip NULLs, NOT IN has its famous trap"
  - "NULL = NULL is not true, so joins and DISTINCT behave in ways SQL newcomers don't predict"
pitfalls:
  - title: "NOT IN with a subquery that returns NULL"
    detail: "x NOT IN (1, 2, NULL) is never true: unknown compares kill the whole list. Use NOT EXISTS."
  - title: "COUNT(*) vs COUNT(col)"
    detail: "COUNT(col) skips NULLs; COUNT(*) counts rows. Mixing them misreports totals."
  - title: "Writing WHERE col = NULL"
    detail: "Never true. NULL predicates are IS NULL / IS NOT NULL."
quiz:
  - q: "SELECT 1 WHERE NULL <> 1 returns?"
    options: ["1", "no rows", "NULL"]
    answer: 1
    why: "NULL <> 1 is unknown, and WHERE only passes true."
  - q: "Which is the null-safe version of a <> b?"
    options: ["a != b OR a IS NULL", "a IS DISTINCT FROM b", "COALESCE(a,0) <> COALESCE(b,0)"]
    answer: 1
    why: "IS DISTINCT FROM compares with NULL-as-a-value in one expression, index-friendly and readable."
further_reading:
  - title: "PostgreSQL docs: Comparison Functions (IS DISTINCT FROM)"
    url: "https://www.postgresql.org/docs/current/functions-comparison.html"
  - title: "PostgreSQL docs: NULLs"
    url: "https://www.postgresql.org/docs/current/tutorial-nulls.html"
---

## Concept

SQL has three logical values: true, false, and unknown. NULL compared to anything is
unknown, and WHERE-style filters keep only true. `IS NULL`, `IS DISTINCT FROM`
and `COALESCE` exist to handle unknowns deliberately.

## Coming from C/C++/Java/Python/JS

- TS: null and undefined are values you can ===; SQL NULL is the *absence of a value*, and
  it isn't equal to itself. `NULL = NULL` is unknown, not true.
- Go: a NULLable column ≈ a pointer you must nil-check; the database forces the check in
  every predicate instead of panicking later.
- C: not a sentinel like NULL pointer dereference — it's a third truth value in the logic.

## Design notes

Schema design decides where NULL is meaningful: a missing value that means "not yet known"
earns NULL; one that means "none" deserves a real value (empty array, false, 0). Constraints
(NOT NULL) then encode the decision, so queries can rely on it. The NOT IN trap is an
anti-join lesson in disguise — covered properly with EXISTS in the joins module.
