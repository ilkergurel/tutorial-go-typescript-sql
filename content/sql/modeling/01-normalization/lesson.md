---
id: sql.modeling.normalization
title: "From entities to tables: normalization 1NF to BCNF by fixing a messy spreadsheet"
track: sql
module: modeling
order: 1
level: intermediate
minutes: 14
since: pg14
prerequisites: [sql.constraints.fk-check, sql.aggregation.group-by]
compare_with: [sql.modeling.denormalize, sql.joins.many-to-many, fullstack.tracelite.step-02-schema]
tags: [modeling, normalization, functional-dependency, 1nf, 2nf, 3nf, bcnf, anomalies]
summary: >
  Normalization is "one fact, one place". Find the functional dependencies in
  the data, give each determinant a table keyed by it, and the keys and foreign
  keys you declare are those dependencies, enforced on every write.
examples:
  - id: spreadsheet
    file: examples/01_spreadsheet.sql
    title: "A test log spreadsheet as a table, and its three anomalies"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-12"
        note: >
          No key and every column nullable: the shape of a spreadsheet. Its
          real key is (test_code, run_on), one run per test per day. A
          normalized table holds only columns that depend on its whole key,
          and here four columns don't.
      - lines: "14-17"
        note: >
          REQ-1's title is stored in three rows and Mehmet's e-mail in two.
          Every repetition is a copy that can disagree with the others, and
          row 3 already does.
      - lines: "21-24"
        note: >
          A functional dependency check: group by the determinant and count
          the distinct values that should depend on it. Run it against any
          import before you design the tables. Every row it returns is a
          contradiction a person has to resolve.
      - lines: "28-29"
        note: >
          Without a key, nothing rejects a half-empty row. count(*) now says 5
          and count(run_on) says 4, so every aggregate over the sheet has to
          know which rows aren't really runs.
      - lines: "33-34"
        note: >
          REQ-2 lived only inside TC-3's run row. Deleting a run shouldn't
          delete a requirement, but in this shape it does, and nothing
          reports it.
  - id: decompose
    file: examples/02_decompose.sql
    title: "Decompose to 3NF: one table per determinant, then rebuild the sheet with joins"
    runtime: sql
    check: exact
    setup: |
      CREATE TABLE sheet (
        test_code text, run_on date, req_code text, req_title text,
        tester text, tester_email text, result text, tags text
      );
      INSERT INTO sheet VALUES
        ('TC-1', '2026-11-02', 'REQ-1', 'Pay by card',    'Ayşe',   'ayse@example.com',   'passed', 'smoke, payment'),
        ('TC-2', '2026-11-02', 'REQ-1', 'Pay by card',    'Mehmet', 'mehmet@example.com', 'failed', 'payment'),
        ('TC-1', '2026-11-09', 'REQ-1', 'Card payment',   'Ayşe',   'ayse@example.com',   'passed', 'smoke, payment'),
        ('TC-3', '2026-11-09', 'REQ-2', 'Invoice e-mail', 'Mehmet', 'mehmet@example.com', 'passed', 'email');
    annotations:
      - lines: "3-6"
        note: >
          The title depended on the run only through req_code (a transitive
          dependency, which 3NF forbids). Now req_code is the key of a table
          that holds the title once. Codes as primary keys keep this example
          short; the keys lesson explains why a real schema adds a surrogate
          id and keeps UNIQUE (code).
      - lines: "7-10"
        note: >
          test_code determines req_code, but test_code is only part of the
          run's key (test_code, run_on). Storing req_code per run was the 2NF
          violation. It now lives once per test case, and the foreign key is
          the link between the two tables.
      - lines: "11-14"
        note: >
          tester determines the e-mail: the second 3NF fix. UNIQUE (email)
          also declares the reverse, one address per tester, which the
          spreadsheet couldn't express at all.
      - lines: "15-21"
        note: >
          What's left needs the whole key: who ran this test on this day, and
          the result. The composite primary key is the sheet's implicit key,
          now enforced, and the CHECK closes the result vocabulary.
      - lines: "22-26"
        note: >
          1NF: 'smoke, payment' was two values in one field. One row per tag
          can be matched exactly, counted and referenced. The anti-patterns
          lesson shows what the comma-separated version breaks.
      - lines: "29-32"
        note: >
          Normalizing doesn't resolve contradictions, it exposes them.
          DISTINCT ON keeps the first row per req_code in ORDER BY order, the
          newest run's title. A plain SELECT DISTINCT req_code, req_title
          would produce two REQ-1 rows and fail on the primary key.
      - lines: "36-38"
        note: >
          string_to_table (PG 14) splits on the comma, trim drops the space,
          and DISTINCT removes TC-1's tags, which the sheet repeated for each
          of its two runs.
      - lines: "41-45"
        note: >
          Joining along the foreign keys gives the sheet's columns back: the
          decomposition is lossless, because each split was on a determinant
          that became the key of its new table. Every run now shows the one
          current title.
      - lines: "50"
        note: >
          The dependency req_code -> req_title is now a primary key, so a
          second title is a duplicate key: SQLSTATE 23505.
  - id: bcnf
    file: examples/03_bcnf.sql
    title: "Beyond 3NF: a dependency whose determinant is not a key"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-8"
        note: >
          Two candidate keys: (test_code, lab) and (test_code, tester). Every
          column belongs to one of them, and 3NF only restricts columns that
          belong to none, so this table is in 3NF.
      - lines: "12-18"
        note: >
          BCNF's rule is stricter: every determinant must be a candidate key.
          tester determines lab but isn't a key here, so the fact is repeated
          per assignment, and the update anomaly is back.
      - lines: "21-25"
        note: >
          testers is keyed by the determinant. UNIQUE (name, lab) follows
          from the primary key, but a foreign key must point at a unique
          constraint or index on exactly its columns; without it the FK below
          fails with SQLSTATE 42830.
      - lines: "26-32"
        note: >
          The textbook BCNF split is assignments (test_code, tester), with no
          lab. That loses "one tester per test per lab", which then spans two
          tables and can't be a key. Keeping lab as a copy tied to testers by
          a composite foreign key keeps both rules declarable: deliberate,
          controlled redundancy, the subject of the next lesson.
      - lines: "38-39"
        note: >
          ON UPDATE CASCADE carries the changed (name, lab) pair into every
          assignment, in the same statement. A move to 'ios' would fail with
          23505 instead: TC-1 on iOS is Mehmet's, and the primary key still
          holds.
      - lines: "42"
        note: >
          Zeynep works in android, so (Zeynep, ios) matches no tester row:
          SQLSTATE 23503. The copy can't drift from its source.
use_cases:
  - "Designing tables from a spreadsheet, a CSV export or a legacy table: list the dependencies first, then derive the tables"
  - "Cleaning an import: the GROUP BY ... HAVING count(DISTINCT ...) > 1 check lists every contradiction to resolve before loading"
  - "Reviewing a schema: a column that depends on part of a key, or on another non-key column, is an update anomaly waiting to happen"
  - "Deciding where a new column goes: next to the key that determines it"
strengths:
  - "A rename or an e-mail change is a one-row UPDATE; in a flat table or a document store it is a multi-row update that a forgotten WHERE or a partial failure leaves inconsistent"
  - "The dependencies become PRIMARY KEY, UNIQUE and REFERENCES declarations checked on every write, instead of invariants each Go or TS service has to remember"
  - "Things exist on their own: a requirement without tests or a tester without runs is a row, not a NULL-padded line"
  - "Narrow, keyed tables are what B-tree indexes, joins and the planner's per-column statistics are built for"
weaknesses:
  - "Reads need joins: the spreadsheet comes back only by joining four tables, which reports and ORMs pay for on every read"
  - "BCNF can cost a rule: a dependency that ends up spanning two tables can't be a key and needs a composite foreign key, a trigger or application code"
  - "Normalization knows only the current value: a normalized title is today's title, so values that must be frozen (prices on invoices) need a deliberate copy"
  - "Over-splitting, such as a table per attribute, adds joins without removing any redundancy; stop where the dependencies stop"
pitfalls:
  - title: "Loading with SELECT DISTINCT and trusting the result"
    detail: "DISTINCT over (code, title) gives one row per distinct pair, so a contradiction becomes a duplicate key, or, with no key declared, a duplicate entity. Run the dependency check first and resolve what it finds."
    bad: "INSERT INTO requirements SELECT DISTINCT req_code, req_title FROM sheet;"
    good: "SELECT req_code FROM sheet GROUP BY req_code HAVING count(DISTINCT req_title) > 1;  -- resolve these, then load"
  - title: "Adding an id column and calling it normalized"
    detail: "A surrogate key makes every row distinct, duplicates included, and leaves the dependencies among the other columns untouched. Every anomaly from example 1 survives it."
    bad: "ALTER TABLE sheet ADD COLUMN id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY;"
    good: "requirements (code PRIMARY KEY, title), test_runs PRIMARY KEY (test_code, run_on), ...: the keys the dependencies imply"
  - title: "Normalizing away a snapshot"
    detail: "The price on an order line or the title a test had when it ran may look like copies, but they are different facts: the value at that time. Replacing them with a join to the current value rewrites history."
  - title: "Treating BCNF as the goal"
    detail: "A decomposition that loses a business rule is worse than a controlled copy. Keep the copy and tie it to its source with a composite foreign key, as in example 3."
quiz:
  - q: "The sheet's key is (test_code, run_on), and req_code depends on test_code alone. Which normal form does that violate?"
    options: ["1NF", "2NF", "3NF only", "None: it's fine"]
    answer: 1
    why: "A non-key column that depends on part of a composite key is a partial dependency, which is exactly what 2NF forbids."
  - q: "How do you check whether tester -> tester_email holds in imported data?"
    options:
      - "SELECT DISTINCT tester, tester_email FROM sheet"
      - "GROUP BY tester HAVING count(DISTINCT tester_email) > 1 returns no rows"
      - "Add UNIQUE (tester_email)"
    answer: 1
    why: "The dependency holds when every tester has exactly one e-mail, so group by the determinant and look for more than one. UNIQUE (tester_email) checks the reverse direction."
  - q: "lab_assignments in example 3 is in 3NF but not in BCNF. Why?"
    options:
      - "tester determines lab, and tester is not a candidate key"
      - "It has a composite primary key"
      - "lab could be NULL"
    answer: 0
    why: "BCNF requires every determinant to be a candidate key. 3NF lets this pass because lab is part of a candidate key, (test_code, lab)."
  - q: "Decomposing on a dependency X -> Y produces what?"
    options:
      - "A table keyed by X that holds Y, and X as a foreign key where Y used to be"
      - "A CHECK constraint relating X and Y"
      - "An index on Y"
    answer: 0
    why: "The determinant becomes a key, so each X maps to one Y by construction, and the original table keeps only the reference."
further_reading:
  - title: "PostgreSQL docs: Constraints (primary keys, unique, foreign keys)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Foreign Keys (tutorial)"
    url: "https://www.postgresql.org/docs/current/tutorial-fk.html"
  - title: "PostgreSQL docs: SELECT, DISTINCT ON"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-DISTINCT"
  - title: "PostgreSQL docs: String Functions (string_to_table)"
    url: "https://www.postgresql.org/docs/current/functions-string.html"
---

## Concept

A table models one kind of thing, and every non-key column states a fact about that
thing. Normalization is the discipline that keeps it that way, phrased in terms of
**functional dependencies**: `X -> Y` holds when each value of `X` comes with exactly one
value of `Y` (a requirement code has one title; a tester has one e-mail address). `X` is
the **determinant**.

Data in which a determinant is not a key stores the same fact many times, and that
causes three **anomalies**:

- **Update**: a fact changed in one copy but not the others (REQ-1 has two titles).
- **Insertion**: a fact can't be recorded without an unrelated one (a requirement with
  no test run yet).
- **Deletion**: removing one fact removes another (the only run of TC-3 was the only
  record of REQ-2).

The cure is always the same move: give the determinant its own table, keyed by it, and
replace the copies with a foreign key. The normal forms name which dependencies are
left to remove.

## Coming from C/C++/Java/Python/JS

- **Any OO language:** a class with `customerName` and `customerEmail` fields on every
  `Order` is the same smell as the spreadsheet; you would extract a `Customer` and hold
  a reference. Normalization is that refactoring, with a test (the dependency check)
  for when it's needed.
- **Java/JPA:** entities with `@ManyToOne` already produce normalized tables. The trap is
  the other direction: an `@Embedded` value object copied into many rows is
  denormalized, which is fine for a snapshot and a bug for shared data.
- **Document databases and JSON APIs:** a nested document per order is the
  spreadsheet again. It reads in one go, and every write to a shared value has to find
  all the copies.
- **Python/pandas:** `df.groupby('req_code')['req_title'].nunique() > 1` is the
  dependency check of example 1, and `drop_duplicates` is the `SELECT DISTINCT` you
  shouldn't trust blindly.

## How it works

The normal forms, as a checklist for each table (a candidate key is any minimal set of
columns that identifies a row):

1. **1NF**: every field holds one value; no lists in a cell, no `tag1, tag2, tag3`
   columns.
2. **2NF**: no non-key column depends on just part of a composite key (req_code on
   test_code alone).
3. **3NF**: no non-key column depends on another non-key column (req_title on req_code,
   tester_email on tester). The usual summary: every non-key column depends on the key,
   the whole key, and nothing but the key.
4. **BCNF**: every determinant is a candidate key, key columns included (tester -> lab
   in example 3).

The procedure that gets you there:

1. Name the entity each row is about and its real key.
2. List the dependencies; confirm them on real data with the `GROUP BY ... HAVING
   count(DISTINCT ...) > 1` check.
3. For each dependency whose determinant isn't a key, move the dependent columns to a
   table keyed by the determinant, and reference it.
4. Declare everything: the primary keys, UNIQUE constraints and foreign keys you end up
   with are the dependencies, and from then on the database rejects every write that
   would break one.

Splitting on a determinant that becomes a key is always lossless: joining back gives
exactly the original rows. Going from 3NF to BCNF, however, can lose a dependency, as in
example 3.

## Design notes

Aim for 3NF by default, check for BCNF, and treat every remaining copy as a decision you
can name. In practice the dependencies come from the domain, not from the data you
happen to have: four rows can't prove that a tester has one lab, only that no
counterexample exists yet. Ask whoever owns the rules.

Normalized does not mean "everything gets a surrogate id". Example 2 uses codes as keys
for brevity; a real TraceLite schema keys requirements by `id`, declares
`UNIQUE (project_id, code)`, and references the id, because codes get renamed. The
dependencies, and therefore the tables, are the same either way.

When a split would lose a rule, as in example 3, keep the redundant column and make the
database maintain it: a composite foreign key with `ON UPDATE CASCADE` turns a copy
into something that can't drift. The next lesson covers deliberate denormalization in
general, including the copies that aren't redundant at all, such as a price at the
time of sale.
