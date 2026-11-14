---
id: sql.joins.many-to-many
title: "Many-to-many via junction tables; semi-joins (EXISTS); anti-joins (NOT IN NULL trap)"
track: sql
module: joins
order: 2
level: intermediate
minutes: 14
since: pg9.5
prerequisites: [sql.joins.inner-outer, sql.querying.null-logic]
compare_with: [fullstack.tracelite.step-02-schema, sql.modeling.anti-patterns, ts.functions.higher-order]
tags: [joins, many-to-many, junction-table, exists, semi-join, anti-join, not-in]
summary: >
  A many-to-many relationship is a table of links. EXISTS asks "is there at
  least one?" without repeating rows, and NOT EXISTS asks "is there none?"
  safely, while NOT IN returns nothing once its subquery yields a NULL.
examples:
  - id: junction
    file: examples/01_junction.sql
    title: "The trace-link table: composite key, cascades, idempotent linking"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-11"
        note: >
          The junction table needs no id of its own: the pair is the identity.
          The composite primary key forbids duplicate links, makes both
          columns NOT NULL and builds an index starting with requirement_id.
          Attributes of the link itself (who linked, when, which kind) go here
          too.
      - lines: "8-9"
        note: >
          CASCADE on both sides: a link means nothing once either end is gone.
          Compare the RESTRICT on test runs in the foreign keys lesson, where
          the child was evidence.
      - lines: "12-13"
        note: >
          A multicolumn B-tree is most efficient for conditions on its leading
          column, so the key index serves "tests of this requirement". The
          reverse question, "requirements of this test", gets its own index.
      - lines: "20-21"
        note: >
          ON CONFLICT DO NOTHING (from the modifying-data module) skips the
          duplicate: INSERT 0 0. Linking becomes idempotent, so a retried API
          request doesn't fail.
      - lines: "23-28"
        note: >
          Two joins walk the relationship: requirement to link to test case.
          Each requirement appears once per linked test; REQ-EXPORT has no
          links and drops out of the inner joins.
      - lines: "30-32"
        note: >
          Deleting test 11 removes its links to both requirements in the same
          statement: (1, 11) and (2, 11) are gone.
  - id: semi-join
    file: examples/02_semi_join.sql
    title: "Semi-joins with EXISTS, and 'verified' as EXISTS plus NOT EXISTS"
    runtime: sql
    check: exact
    annotations:
      - lines: "2"
        note: >
          last_passed NULL means "never run". A nullable boolean has three
          states, and every query on it has to handle the third (line 33).
      - lines: "12-17"
        note: >
          The join repeats a requirement once per link. SELECT DISTINCT would
          hide that, but it still builds every pair first, and it stops
          working the moment you also select a column from the link side.
      - lines: "19-23"
        note: >
          EXISTS returns each requirement at most once and can stop at the
          first matching link. The select list inside is ignored; SELECT 1 is
          the convention. r.id IN (SELECT requirement_id FROM trace_links) is
          the same semi-join, and PG 18 plans both identically here.
      - lines: "25-35"
        note: >
          "Every linked test passed" is written as "no linked test failed to
          pass", plus EXISTS so a requirement without tests isn't verified
          vacuously. REQ-RESET is out because test 12 never ran.
      - lines: "33"
        note: >
          IS NOT TRUE counts NULL as not passed. With last_passed = false,
          test 12's NULL would match nothing and REQ-RESET would wrongly count
          as verified.
  - id: not-in-trap
    file: examples/03_not_in_trap.sql
    title: "The NOT IN NULL trap, and NOT EXISTS as the fix"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-7"
        note: >
          A legitimate nullable foreign key: ad-hoc runs belong to no test
          case. Nothing about the schema hints at the trap below.
      - lines: "12-15"
        note: >
          While the subquery returns only 10, 10 and 12, NOT IN is correct:
          Locked account (11) never ran.
      - lines: "19-22"
        note: >
          The same query, after one NULL arrived: 11 NOT IN (10, 10, 12, NULL)
          is 11 <> 10 AND ... AND 11 <> NULL, which is NULL, and WHERE keeps
          only true. Every row fails the same way: no rows, no error, no
          warning.
      - lines: "24-25"
        note: >
          The trap in isolation. Without the NULL the answer is true; with it,
          NULL.
      - lines: "27-30"
        note: >
          NOT EXISTS asks whether a matching row exists, and the NULL row
          simply doesn't match. EXPLAIN shows it as an Anti Join; NOT IN can't
          be planned that way because of its NULL semantics and runs as a
          filter over a hashed SubPlan.
      - lines: "32-36"
        note: >
          LEFT JOIN plus IS NULL is the same anti-join. Test a column that is
          never NULL in real rows (r.id, the primary key); testing a nullable
          column would also keep matched rows that happen to hold NULL there.
use_cases:
  - "Trace links between requirements and test cases, tags on articles, users in groups, products in categories"
  - "EXISTS: 'has at least one' filters, such as requirements with tests or customers with an open order"
  - "NOT EXISTS: coverage gaps, never-run tests, orphan detection, and 'all X satisfy Y' written as 'no X violates Y'"
strengths:
  - "The junction table's composite primary key makes links unique and indexed, with no surrogate id"
  - "EXISTS never duplicates outer rows and may stop at the first match, unlike JOIN plus DISTINCT"
  - "NOT EXISTS is NULL-safe and is planned as an anti-join"
weaknesses:
  - "Every many-to-many traversal costs two joins, and the reverse direction needs its own index"
  - "NOT IN (subquery) is only correct while the subquery can never yield NULL, and it is not planned as an anti-join"
  - "EXISTS can't return columns of the inner table; when you need them, you need a join or LATERAL"
pitfalls:
  - title: "NOT IN over a nullable column"
    detail: "One NULL in the subquery result makes NOT IN NULL for every row, and the query returns nothing. Use NOT EXISTS, which also plans better."
    bad: "WHERE id NOT IN (SELECT test_case_id FROM test_runs)"
    good: "WHERE NOT EXISTS (SELECT 1 FROM test_runs r WHERE r.test_case_id = t.id)"
  - title: "JOIN plus DISTINCT for an existence question"
    detail: "It builds every pair and then removes duplicates; EXISTS asks the actual question and returns each outer row once."
  - title: "A junction table without a unique pair"
    detail: "A surrogate id with no UNIQUE (requirement_id, test_case_id) lets the same link be inserted twice, and every count over links is then wrong. Make the pair the primary key or at least unique."
  - title: "Checking a nullable column in LEFT JOIN ... IS NULL"
    detail: "WHERE r.test_case_id IS NULL can't tell 'no match' from 'matched a row whose column is NULL'. Check a NOT NULL column of the right side, usually its primary key."
quiz:
  - q: "test_runs.test_case_id contains one NULL. What does `SELECT ... FROM test_cases WHERE id NOT IN (SELECT test_case_id FROM test_runs)` return?"
    options: ["The test cases that never ran", "No rows", "An error"]
    answer: 1
    why: "For every row, x NOT IN (..., NULL) is either false or NULL, never true, so WHERE keeps nothing."
  - q: "A requirement has five linked tests. How many times does it appear in `WHERE EXISTS (SELECT 1 FROM trace_links l WHERE l.requirement_id = r.id)`?"
    options: ["Once", "Five times", "Zero times"]
    answer: 0
    why: "EXISTS is a semi-join: it only asks whether a match exists, so each outer row appears at most once."
  - q: "What is the result of `SELECT 11 NOT IN (10, NULL)`?"
    options: ["true", "false", "NULL"]
    answer: 2
    why: "11 <> 10 is true, 11 <> NULL is NULL, and true AND NULL is NULL."
  - q: "Why does the junction table use `PRIMARY KEY (requirement_id, test_case_id)`?"
    options: ["Every table needs an integer id", "It forbids duplicate links and indexes lookups by requirement", "It makes deletes cascade"]
    answer: 1
    why: "The pair is the link's identity; the key enforces uniqueness and its index leads with requirement_id. Cascading comes from the foreign keys."
further_reading:
  - title: "PostgreSQL docs: Subquery Expressions (EXISTS, IN, NOT IN)"
    url: "https://www.postgresql.org/docs/current/functions-subquery.html"
  - title: "PostgreSQL docs: Constraints (foreign keys, with a many-to-many example)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Multicolumn Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-multicolumn.html"
---

## Concept

Relational tables have no "list of ids" column that the database can check. A
many-to-many relationship (a requirement is verified by many test cases, a test case
covers many requirements) is a table of its own with one row per link: a **junction**
table. Its primary key is the pair, and it has a foreign key to each side.

Two query shapes go with it. A **semi-join** returns rows of A that have at least one
match in B, each at most once: `EXISTS` or `IN (subquery)`. An **anti-join** returns rows
of A with no match in B: `NOT EXISTS`, or `LEFT JOIN ... WHERE b.key IS NULL`. `NOT IN`
looks like an anti-join and isn't one when NULLs are involved.

## Coming from C/C++/Java/Python/JS

- Java/JPA: `@ManyToMany` generates exactly this join table and hides it. Mapping it as
  its own entity is what lets you add attributes to the link.
- Go/TS: in code you would keep a slice or `Set` of ids on one side. In a table that is
  the comma-separated-list (or id array) anti-pattern: no foreign key can check it.
- JS: `EXISTS` is `tests.some(...)`, `NOT EXISTS` is `!tests.some(...)`, and the
  "verified" query is `tests.length > 0 && tests.every(t => t.lastPassed === true)`.
  `Array.prototype.includes` is two-valued, so JS has no equivalent of the NOT IN trap.

## Design notes

Prefer EXISTS and NOT EXISTS for "has any" and "has none" questions, and reserve joins
for when you need columns from both sides. Treat `NOT IN (subquery)` as a review
finding unless the column is declared NOT NULL; even then NOT EXISTS says the same thing
and is planned as an anti-join. `NOT IN` over a literal list (`status NOT IN ('done',
'dropped')`) is fine as long as no NULL can appear in the list.

Name the junction table after the relationship (`trace_links`) once the link has
meaning of its own, and give it the index for the reverse direction from the start.
