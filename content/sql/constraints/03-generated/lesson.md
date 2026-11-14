---
id: sql.constraints.generated
title: "Generated columns (virtual by default in PG 18, vs STORED) and exclusion constraints"
track: sql
module: constraints
order: 3
level: advanced
minutes: 14
since: pg18
prerequisites: [sql.constraints.fk-check, sql.tables-types.time]
compare_with: [sql.modeling.temporal, sql.modern.16-to-18, sql.indexes.composite]
tags: [generated-columns, virtual, stored, exclusion, gist, ranges]
summary: >
  A generated column derives its value from the rest of the row: virtual (the
  PG 18 default) computes it on read, STORED on write. Exclusion constraints
  generalize UNIQUE to other operators, so "no two bookings overlap" becomes a
  declaration.
examples:
  - id: virtual-vs-stored
    file: examples/01_generated.sql
    title: "A virtual and a stored generated column side by side"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          No keyword means VIRTUAL in PG 18: no storage, computed whenever the
          column is read, like a column of a view. PG 12 to 17 had only stored
          generated columns and required the STORED keyword, so DDL written
          without it is PG 18+ only.
      - lines: "9-10"
        note: >
          STORED computes the value on INSERT and UPDATE and writes it to disk:
          it costs space and write time, reads are free, and it can be indexed.
          Either kind may use only immutable functions and the current row: no
          subqueries, no other generated columns.
      - lines: "13-16"
        note: >
          The INSERT never mentions the generated columns, and the UPDATE of
          vat_pct changes gross_cents with no trigger. Writing one yourself
          fails with SQLSTATE 428C9: cannot insert a non-DEFAULT value.
      - lines: "18-20"
        note: >
          lower('Iğdır') gives 'iğdır' under the database's C.UTF-8 rules, not
          the Turkish 'ığdır'. lower(name COLLATE "tr-x-icu") is allowed in a
          generation expression and gives 'ığdır'. A STORED column bakes the
          rules in at write time. Also note gross_cents: integer division
          truncates, so put the rounding rule into the expression.
      - lines: "22-26"
        note: >
          pg_attribute.attgenerated tells the kinds apart: 'v' virtual,
          's' stored, empty for ordinary columns.
      - lines: "28-29"
        note: >
          The stored column takes a normal B-tree index. PG 18 rejects an index
          on a virtual column (SQLSTATE 0A000), and also a UNIQUE constraint
          on one. To index such a value, make it STORED or index the expression
          itself.
  - id: exclusion-ranges
    file: examples/02_exclusion_ranges.sql
    title: "EXCLUDE ... WITH &&: no two bookings overlap"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          A real schema would use tstzrange (see the time zones lesson). tsrange
          keeps the printed output independent of the session's time zone.
      - lines: "6"
        note: >
          For any two rows, during && during must not be true. A B-tree can
          only answer equality and ordering, so overlap needs a GiST index,
          which the constraint creates. Range types bring their own GiST
          support: no extension needed.
      - lines: "9-12"
        note: >
          Half-open ranges [start, end) make back-to-back bookings legal: one
          ends at 10:00, the next starts at 10:00, and they share no instant.
          With a closed upper bound, 10:00 would belong to both.
      - lines: "17-18"
        note: >
          11:00 to 12:00 overlaps Mehmet's booking, so the insert fails with
          SQLSTATE 23P01 (exclusion_violation). A UNIQUE constraint can't
          express this: the two ranges are not equal.
  - id: per-room
    file: examples/03_per_room.sql
    title: "Per room, live bookings only: = and && with btree_gist"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-3"
        note: >
          btree_gist adds GiST support for scalar types such as text and int.
          Without it the CREATE TABLE fails: data type text has no default
          operator class for access method "gist". CREATE EXTENSION is
          transactional, so here it is rolled back with everything else; a
          real database installs it once, in a migration.
      - lines: "10-12"
        note: >
          Two rows conflict only when room = room AND during && during. The
          WHERE clause makes the constraint partial: cancelled bookings don't
          take part, which no UNIQUE constraint can express.
      - lines: "15-17"
        note: >
          Same time, different rooms: the = comparison is false, so there is
          no conflict.
      - lines: "18-20"
        note: >
          Cancelling booking 1 takes it out of the constraint, so booking 3
          can claim an overlapping slot in Galata.
      - lines: "26-27"
        note: >
          10:00 to 11:00 overlaps booking 3 (09:30 to 10:30) in the same room,
          so the insert fails with SQLSTATE 23P01.
use_cases:
  - "STORED: normalized lookup keys such as lower(name) or a slug that you filter or sort on and must index"
  - "VIRTUAL: derived values read now and then, such as gross from net and VAT, display names or unit conversions"
  - "EXCLUDE: room and resource booking, non-overlapping validity periods for prices or tax rates, one active assignment per person at a time"
strengths:
  - "Unlike a computed property on a Go struct or a TS class, the value is the same for every reader: reports, other services and psql"
  - "A virtual column costs no storage, and adding one never rewrites the table (adding a STORED one does)"
  - "An exclusion constraint replaces the application's 'look for an overlap, then insert', which races: two concurrent overlapping inserts can't both commit"
weaknesses:
  - "PG 18 can't index virtual columns or put UNIQUE on them; anything you search by needs STORED or an expression index"
  - "Virtual columns may use only built-in functions and types, and are recomputed on every read"
  - "Changing a STORED column's expression (ALTER COLUMN ... SET EXPRESSION) rewrites its existing data"
  - "Exclusion constraints need a GiST index, btree_gist for = on scalar columns, and can't be arbiters for ON CONFLICT DO UPDATE"
pitfalls:
  - title: "Indexing a virtual column"
    detail: "In PG 18 a generated column without a keyword is virtual, and CREATE INDEX on it fails with 'indexes on virtual generated columns are not supported'. Decide the kind when you create the column."
    bad: "name_key text GENERATED ALWAYS AS (lower(name))  -- then CREATE INDEX ... (name_key)"
    good: "name_key text GENERATED ALWAYS AS (lower(name)) STORED"
  - title: "Closed ranges for back-to-back slots"
    detail: "'[09:00,10:00]' and '[10:00,11:00)' overlap at 10:00, so the second booking is rejected. Use half-open '[)' ranges, which is also what tsrange(a, b) builds by default."
  - title: "Forgetting btree_gist"
    detail: "EXCLUDE USING gist (room WITH =, ...) on a text or int column needs the btree_gist extension. It is trusted, so the database owner can install it without superuser rights."
  - title: "Case rules frozen into a stored key"
    detail: "A STORED lower(name) is computed with the rules in force at write time. Under C.UTF-8, 'IĞDIR' becomes 'iğdir'. If Turkish rules matter, write lower(name COLLATE \"tr-x-icu\") in the expression."
quiz:
  - q: "In PG 18, what does `total int GENERATED ALWAYS AS (price * qty)` (no keyword) create?"
    options: ["A stored generated column", "A virtual generated column", "A syntax error"]
    answer: 1
    why: "Virtual is the default in PG 18. PG 12 to 17 required the STORED keyword, so they reject this line."
  - q: "Which generated column can you index in PG 18?"
    options: ["Virtual", "Stored", "Both", "Neither"]
    answer: 1
    why: "Indexes on virtual generated columns are not supported in PG 18; a stored one indexes like any other column."
  - q: "Bookings '[09:00,10:00)' and '[10:00,11:00)' in the same room, with EXCLUDE USING gist (room WITH =, during WITH &&). What happens to the second insert?"
    options: ["It fails: the ranges touch at 10:00", "It succeeds: half-open ranges that only touch don't overlap"]
    answer: 1
    why: "The first range excludes 10:00 itself, so && is false and the constraint is satisfied."
  - q: "Why does `EXCLUDE USING gist (room WITH =, during WITH &&)` with room text need btree_gist?"
    options: ["GiST has no built-in operator class for = on text", "Range types need it", "Every exclusion constraint needs it"]
    answer: 0
    why: "Ranges have GiST support built in; btree_gist supplies GiST operator classes for scalar types like text and int."
further_reading:
  - title: "PostgreSQL docs: Generated Columns"
    url: "https://www.postgresql.org/docs/current/ddl-generated-columns.html"
  - title: "PostgreSQL docs: Constraints (exclusion constraints)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Range Types (constraints on ranges)"
    url: "https://www.postgresql.org/docs/current/rangetypes.html"
  - title: "PostgreSQL docs: btree_gist"
    url: "https://www.postgresql.org/docs/current/btree-gist.html"
---

## Concept

A **generated column** is `GENERATED ALWAYS AS (expression)`, computed from other columns
of the same row and never written directly. PG 12 introduced the **STORED** kind,
computed on write and kept on disk. PG 18 adds **VIRTUAL**, computed on read with no
storage, and makes it the default. The expression may use only immutable functions and
the current row.

An **exclusion constraint** generalizes UNIQUE: for any two rows, comparing them with the
listed operators must not return true for all of them. UNIQUE is the special case where
every operator is `=`. With `&&` (overlaps) on a range column, "no double booking"
becomes a constraint, and a `WHERE` clause can limit it to some rows.

## Coming from C/C++/Java/Python/JS

- Python/TS: a virtual column is a `@property` or getter, computed on access. A stored
  column is a field that the database recomputes on every write, so it can never go stale.
- Java/Go: a computed field on an entity or struct only exists in that one program. A
  generated column gives the same value to every reader, including SQL reports.
- Overlap checks in application code (load the bookings of that room, compare intervals,
  insert) race under concurrency, much like an unlocked check-then-act on a shared map.
  The exclusion constraint is checked through its index and holds across transactions.

## Design notes

Choose the kind by how the value is used. Virtual suits values that are cheap to compute
and rarely filtered on; it adds nothing to writes or disk. Stored suits values you index,
search or sort by, or that are expensive to compute, at the price of storage and write
time.

PG 18 also adds temporal keys: `PRIMARY KEY (room, during WITHOUT OVERLAPS)` behaves like
`EXCLUDE USING gist (room WITH =, during WITH &&)` and needs btree_gist for the same
reason. Reach for plain EXCLUDE when you need what a key can't express: a `WHERE`
clause, as with cancelled bookings, or operators other than `=` and `&&`. The temporal
data lesson covers WITHOUT OVERLAPS and PERIOD foreign keys.
