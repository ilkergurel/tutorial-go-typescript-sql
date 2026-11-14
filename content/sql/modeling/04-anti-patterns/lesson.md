---
id: sql.modeling.anti-patterns
title: "Anti-patterns: EAV, polymorphic foreign keys, comma-separated lists, jsonb for everything"
track: sql
module: modeling
order: 4
level: advanced
minutes: 14
since: pg9.6
prerequisites: [sql.modeling.normalization, sql.tables-types.jsonb-arrays]
compare_with: [sql.joins.many-to-many, sql.json.indexing-design, design.paradigms.polymorphism, design.patterns.anti-patterns]
tags: [modeling, anti-patterns, eav, polymorphic-association, exclusive-arc, junction-table, jsonb]
summary: >
  EAV tables, polymorphic target_type/target_id pairs, lists in strings and
  jsonb-only tables all move the schema out of the database's sight. Each one
  gives up types, NOT NULL, foreign keys or statistics, and every reader pays
  for the checks the writer skipped.
examples:
  - id: eav
    file: examples/01_eav.sql
    title: "Entity-attribute-value: everything fits, nothing is checked"
    runtime: sql
    check: exact
    annotations:
      - lines: "4-9"
        note: >
          EAV turns columns into rows, so the schema becomes data. The
          database can no longer say what a test case has: one text column
          holds every value, so there are no types, no NOT NULL for a
          required attribute, no CHECK, no DEFAULT, and no foreign key from a
          value.
      - lines: "12-14"
        note: >
          'high' isn't a number and 'prority' isn't an attribute, and both
          are accepted. With columns, the first is a type error at INSERT and
          the second a "column does not exist" error in the query that
          misspells it.
      - lines: "17-23"
        note: >
          Reading an entity back is a pivot: one FILTER aggregate per
          attribute, every value still text. TC-3 has no priority and nothing
          says why. Filtering on two attributes takes a join or subquery per
          attribute.
      - lines: "27-33"
        note: >
          The attributes every test case has become ordinary columns:
          priority a smallint with a range check, automated a boolean with a
          default. Adding one is an ALTER TABLE, which is the point: the
          change is reviewed and visible.
      - lines: "32"
        note: >
          What really varies per customer goes into one jsonb column. It is
          still unchecked inside (the jsonb lesson shows what to constrain),
          but it is one column next to typed ones, not the whole model.
      - lines: "38"
        note: >
          The typed column rejects 'high' at the INSERT, SQLSTATE 22P02: at
          write time, once, where EAV accepted it and left the problem to
          every reader.
  - id: polymorphic
    file: examples/02_polymorphic.sql
    title: "Polymorphic target_type/target_id, and the exclusive arc that replaces it"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          Rails' polymorphic belongs_to, Django's GenericForeignKey and
          Hibernate's @Any produce this shape. A foreign key references one
          table, and target_id points into a different one per row, so
          nothing checks it.
      - lines: "17-18"
        note: >
          Both mistakes are accepted: a comment on a test case that never
          existed, and one whose test case was just deleted. There is no
          foreign key to attach ON DELETE CASCADE to either.
      - lines: "21-25"
        note: >
          Every read joins every possible target conditionally. A third
          target type means changing every such query.
      - lines: "28-35"
        note: >
          An exclusive arc: one nullable foreign key per target type, each
          with real integrity and its own ON DELETE action, and a CHECK that
          exactly one is set. num_nonnulls (PG 9.6) counts its non-null
          arguments, so none or both fails with SQLSTATE 23514.
      - lines: "36-37"
        note: >
          Each reference column needs its own index, for "comments on this
          test case" and for the cascade. That cost grows per target type,
          which is why many targets call for another design (Design notes).
      - lines: "40-41"
        note: >
          Deleting test case 1 removes its comment in the same statement:
          CASCADE works because the reference is real.
      - lines: "43"
        note: >
          Test case 9 doesn't exist: SQLSTATE 23503. The polymorphic table
          accepted the same row on line 17.
  - id: lists
    file: examples/03_lists.sql
    title: "Ids in a string, ids in a document, ids in a junction table"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-6"
        note: >
          A list in a string, the textbook 1NF violation. It is still common
          wherever a multi-select is saved as one field.
      - lines: "9"
        note: >
          LIKE '%1%' matches the 1 in 12, so TC-3 is reported as covering
          REQ-1. Delimiter tricks such as ',' || req_ids || ',' LIKE '%,1,%'
          fix this query and break on TC-3's space after the comma, and no
          B-tree index helps a leading wildcard.
      - lines: "12-17"
        note: >
          jsonb parses the list into numbers, and @> '1' is exact: by a
          special rule of jsonb containment, an array contains a primitive
          value that is one of its elements. A GIN index on doc would serve
          the equivalent doc @> '{"req_ids": [1]}'.
      - lines: "21-24"
        note: >
          Integrity is now a query that unpacks every document, run after the
          fact. Nothing stopped 99 on the way in, and nothing will stop
          REQ-12 from being deleted while documents still point at it.
      - lines: "27-33"
        note: >
          The junction table makes each link a row with a foreign key on both
          sides. The primary key forbids duplicate links, and the index on
          requirement_id serves the reverse lookup (the many-to-many lesson).
      - lines: "37"
        note: >
          "Which tests cover REQ-1" is an indexed equality, exact by
          construction.
      - lines: "39"
        note: >
          SQLSTATE 23503 at write time. Deleting REQ-12 while links exist is
          refused too, unless you declare ON DELETE CASCADE.
use_cases:
  - "Custom fields per customer: typed columns for the common ones, one jsonb column for the long tail, CHECK constraints on the keys you rely on"
  - "Comments, attachments or watchers on two or three entity types: an exclusive arc; on many types: a table per target or a shared parent table"
  - "Many-to-many links such as test cases to requirements: a junction table with two foreign keys"
  - "Where jsonb is right: payloads stored as received (webhooks, imports) and attributes that genuinely differ from row to row"
strengths:
  - "Typed columns reject bad values once, at write time, instead of every reader casting text and failing at read time"
  - "Real foreign keys refuse dangling references and run ON DELETE actions for every client, which an ORM-level polymorphic association can't promise for psql or a second service"
  - "A junction table answers both directions with indexed equality lookups; a list in a string needs parsing on every row"
  - "The corrected designs show up in the schema, so \\d, sqlc, ORMs and new team members all see the model"
weaknesses:
  - "A new attribute is a migration (ALTER TABLE ADD COLUMN) instead of an INSERT; jsonb stays the escape hatch for what you can't predict"
  - "An exclusive arc adds a column, an index and a CHECK term per target type, so it stops scaling after a handful of targets"
  - "Reading a list back from a junction table costs a join, and an ORM may hide that join behind lazy loading"
  - "A jsonb column inside a typed table still needs CHECK constraints or validation for whatever the code relies on"
pitfalls:
  - title: "EAV for flexibility"
    detail: "Every value becomes text, required attributes can't be enforced, and each attribute in a filter is another join. Since jsonb exists, there is little reason left to build EAV in PostgreSQL."
    bad: "CREATE TABLE attrs (entity_id int, attr text, value text, PRIMARY KEY (entity_id, attr));"
    good: "priority smallint NOT NULL CHECK (priority BETWEEN 1 AND 4), extra jsonb NOT NULL DEFAULT '{}'"
  - title: "A type column plus an id column"
    detail: "No foreign key, no cascade, orphans after every delete in a target table. Use one real reference per target type, or another of the designs below."
    bad: "target_type text NOT NULL, target_id int NOT NULL"
    good: "requirement_id int REFERENCES requirements, test_case_id int REFERENCES test_cases, CHECK (num_nonnulls(requirement_id, test_case_id) = 1)"
  - title: "Searching a list with LIKE"
    detail: "Substring matching finds 1 inside 12 and can't use a B-tree index. Store one row per element."
    bad: "WHERE req_ids LIKE '%1%'"
    good: "WHERE requirement_id = 1  -- on a junction table"
  - title: "jsonb as the whole row"
    detail: "A table of (id, data jsonb) gives up types, NOT NULL, foreign keys and per-field statistics: without an expression index or CREATE STATISTICS on the expression (PG 14), the planner uses default estimates for filters like data ->> 'status' = 'open'. Every update also rewrites and row-locks the whole document."
quiz:
  - q: "How would you make priority required for every test case in example 1's EAV table?"
    options: ["NOT NULL on the value column", "A CHECK on attr", "No constraint can: a missing attribute is a missing row"]
    answer: 2
    why: "Constraints check the rows that exist. An attribute nobody inserted has no row to check; only a trigger or application code could notice."
  - q: "Why can't comments_poly declare a foreign key on target_id?"
    options:
      - "target_id is an int"
      - "A foreign key references exactly one table, and target_id points into a different table depending on target_type"
      - "Foreign keys can't be NOT NULL"
    answer: 1
    why: "REFERENCES names one table. A value whose target table depends on another column is outside what a foreign key can express."
  - q: "Which row does CHECK (num_nonnulls(requirement_id, test_case_id) = 1) reject?"
    options: ["requirement_id 1, test_case_id NULL", "requirement_id NULL, test_case_id 1", "requirement_id 1, test_case_id 1"]
    answer: 2
    why: "Two non-null arguments make num_nonnulls return 2. The arc allows exactly one target."
  - q: "req_ids is text holding '1,12' or '12, 99'. Which query reliably finds the tests for requirement 1?"
    options: ["WHERE req_ids LIKE '%1%'", "WHERE ',' || req_ids || ',' LIKE '%,1,%'", "Neither: store the links in a junction table and query WHERE requirement_id = 1"]
    answer: 2
    why: "The first matches 12, the second breaks on spaces and other formatting. Only one row per link makes the lookup exact and indexable."
further_reading:
  - title: "PostgreSQL docs: Constraints (foreign keys, CHECK)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: JSON Types (containment, document structure)"
    url: "https://www.postgresql.org/docs/current/datatype-json.html"
  - title: "PostgreSQL docs: Comparison Functions (num_nonnulls)"
    url: "https://www.postgresql.org/docs/current/functions-comparison.html"
  - title: "PostgreSQL docs: CREATE STATISTICS (expression statistics)"
    url: "https://www.postgresql.org/docs/current/sql-createstatistics.html"
---

## Concept

The four designs in this lesson share one move: they take structure the database could
check and hide it inside data.

- **Entity-attribute-value (EAV)**: a table of `(entity_id, attr, value)`, so any
  attribute can be added without a migration. Every value is text, and the list of
  attributes lives in the rows.
- **Polymorphic association**: `target_type` plus `target_id`, so one comments table can
  point at requirements or test cases. A foreign key can't point at "one of several
  tables", so nothing checks the reference.
- **Lists in a string**: `'1,12'` in a text column. Matching, counting and integrity all
  require parsing.
- **jsonb for everything**: a table of `(id, data jsonb)`. Parsed and indexable, but
  without types, NOT NULL, foreign keys or per-field statistics.

Each is chosen for flexibility, and each makes every write unchecked and every read
responsible for the checks. The test to apply to any design: can the database reject a
wrong value or a dangling reference at write time? If not, all readers inherit the job.

## Coming from C/C++/Java/Python/JS

- **Go and TS:** EAV and jsonb-only tables are `map[string]any` and
  `Record<string, unknown>` persisted. You wouldn't model a domain type that way in
  code; a struct or an interface with real fields is the column list.
- **Rails, Django, Hibernate:** polymorphic `belongs_to`, `GenericForeignKey` and `@Any`
  make the polymorphic pattern one line of code. The ORM enforces the reference only
  for writes it performs; the database sees an unconstrained integer.
- **WordPress and Magento:** `wp_postmeta` and Magento's product attributes are the
  well-known EAV schemas. They exist because plugins define attributes at runtime, and
  they pay for it in pivot queries and caches. An application with a known domain
  doesn't have that excuse.
- **MongoDB:** a jsonb-only table is a document store inside PostgreSQL. You keep
  transactions and joins, but give up exactly the parts that made a relational
  database worth choosing.

## Design notes

For custom fields, use typed columns for every attribute the code knows about and one
`jsonb` column for the rest. Add CHECK constraints for the jsonb keys the code depends
on; when a key earns several checks, promote it to a column.

For references to several kinds of parent, choose by the number of target types:

- **Exclusive arc** (example 2): two or three targets. Each gets a real foreign key and
  index, and one CHECK keeps exactly one set.
- **A table per target** (`requirement_comments`, `test_case_comments`): any number of
  targets, simplest constraints, and a `UNION ALL` view when you need all comments.
- **A shared parent table**: an `items (id)` table that requirements and test cases each
  reference one-to-one, and comments reference `items`. One foreign key on the comment
  side, at the price of an extra table and join.

For lists, a junction table is the default. `text[]` is acceptable for plain labels that
reference nothing (the jsonb and arrays lesson); ids in a string or a document never
are.

jsonb is the right tool for data whose shape you don't control or can't predict. When
every row has the field, or you filter, join or sum on it, it is a column.
