---
id: sql.constraints.keys
title: "NOT NULL, DEFAULT, primary keys and UNIQUE NULLS NOT DISTINCT"
track: sql
module: constraints
order: 1
level: basic
minutes: 12
since: pg15
prerequisites: [sql.tables-types.ids]
compare_with: [go.basics.variables, ts.everyday-types.null, sql.querying.null-logic]
tags: [constraints, not-null, default, primary-key, unique]
summary: >
  Constraints are invariants the database checks on every write, from every
  client. NOT NULL and DEFAULT shape columns, a primary key names the row, and
  UNIQUE (with NULLS NOT DISTINCT since PG 15) guards the real-world identity a
  surrogate key can't.
examples:
  - id: not-null-default
    file: examples/01_not_null_default.sql
    title: "DEFAULT fills omitted columns; an explicit NULL is not omitted"
    runtime: sql
    check: exact
    annotations:
      - lines: "3"
        note: >
          PRIMARY KEY means NOT NULL plus UNIQUE plus "this is the row's
          identity": one per table, and the default target of foreign keys.
          The identity column generates it (previous module).
      - lines: "5-6"
        note: >
          NOT NULL DEFAULT is the usual pair. The default spares every INSERT
          from naming the column; NOT NULL stops a later UPDATE from setting
          it to "no value".
      - lines: "7"
        note: >
          Leave a column nullable only when NULL has a meaning you can name.
          Here it does: the ticket is unassigned.
      - lines: "11-12"
        note: >
          Both rows get status 'open'. The DEFAULT keyword asks for the default
          explicitly, which helps in a multi-row VALUES list where only some
          rows carry a value.
      - lines: "17"
        note: >
          The ORM trap: a client that always lists every column sends NULL for
          "not set", so the DEFAULT never fires. Omit the column or write
          DEFAULT. The error carries SQLSTATE 23502 (not_null_violation).
  - id: surrogate-natural
    file: examples/02_surrogate_natural.sql
    title: "Surrogate keys identify rows; UNIQUE keeps the real-world identity"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-6"
        note: >
          A natural key is data that already identifies the thing. It reads
          well wherever it is referenced: users.currency below says TRY with
          no join. The price is paid when it changes, because every referencing
          row must change too. ISO codes almost never change; e-mail addresses
          do.
      - lines: "13-14"
        note: >
          The identity column makes every row distinct, so the same person
          signs up twice. A surrogate key answers "which row", never "which
          person".
      - lines: "17-21"
        note: >
          The common shape: a meaningless, immutable id for references, and
          the natural key declared unique next to it.
      - lines: "23"
        note: >
          A UNIQUE constraint takes plain columns only. lower(email) is an
          expression, so it becomes a unique index, which enforces the same
          rule. lower() uses the database's case rules: under C.UTF-8, I lowers
          to i, never to ı (see the pitfalls).
      - lines: "26-28"
        note: >
          The e-mail address changed and the id did not, so anything pointing
          at users.id is untouched. With email as the primary key, every
          referencing row would need the update.
      - lines: "30"
        note: >
          'Ayse.Yilmaz@Example.com' and the stored address have the same
          lower(), so the index rejects it (SQLSTATE 23505, unique_violation).
          The error names the index: give constraints and indexes names your
          error mapping can recognise.
  - id: nulls-not-distinct
    file: examples/03_nulls_not_distinct.sql
    title: "UNIQUE NULLS NOT DISTINCT (PG 15) for optional key columns"
    runtime: sql
    check: exact
    annotations:
      - lines: "6-11"
        note: >
          By default a unique constraint treats NULLs as distinct, so two
          "global" theme rows coexist and every lookup of the global default
          becomes a coin toss. The SQL standard leaves this choice to each
          database, and they differ.
      - lines: "21"
        note: >
          NULLS NOT DISTINCT makes (NULL, 'theme') equal to (NULL, 'theme') for
          this constraint only; NULL = NULL in a query is still NULL. Before
          PG 15 the workaround was a second, partial unique index:
          (key) WHERE tenant_id IS NULL.
      - lines: "24"
        note: >
          The second global theme row now fails with the same SQLSTATE 23505
          as any other duplicate.
use_cases:
  - "NOT NULL DEFAULT for status fields, flags and counters, so no code path can leave them unset"
  - "Surrogate identity key plus UNIQUE on the natural key: users (email), products (sku), requirements (project_id, code)"
  - "Natural primary keys for small, stable, standard codes: ISO 4217 currencies, ISO 3166 countries"
  - "NULLS NOT DISTINCT for 'global default plus per-tenant override' tables and other keys with an optional part"
strengths:
  - "Unlike a check in a Go handler or a zod schema, a constraint also covers psql sessions, migration scripts and the second service that writes the same table"
  - "Uniqueness holds under concurrency: two transactions inserting the same e-mail can't both commit, which an application-side SELECT-then-INSERT can't promise"
  - "Violations come back with a SQLSTATE (23502, 23505) and the constraint name, so the API can map them to a precise 409 or 422 in one place"
weaknesses:
  - "Every PRIMARY KEY and UNIQUE is a B-tree index that each INSERT and key UPDATE must maintain"
  - "A DEFAULT applies only when the column is omitted; clients that always send every column bypass it"
  - "Case-insensitive or partial uniqueness needs a unique index, which can't be attached as a table constraint with ADD CONSTRAINT ... USING INDEX"
  - "A natural primary key that changes has to be rewritten in every referencing row"
pitfalls:
  - title: "Sending NULL and expecting the DEFAULT"
    detail: "Only an omitted column or the DEFAULT keyword gets the default. An explicit NULL is stored as NULL, or rejected by NOT NULL."
    bad: "INSERT INTO tickets (title, status) VALUES ('x', NULL);"
    good: "INSERT INTO tickets (title) VALUES ('x');  -- or VALUES ('x', DEFAULT)"
  - title: "Treating the surrogate key as the uniqueness rule"
    detail: "An identity primary key makes every row distinct, duplicates included. Declare UNIQUE on whatever identifies the thing in the real world."
  - title: "Case-insensitive keys under the wrong case rules"
    detail: "lower() follows the database's case rules. StackLab's databases use C.UTF-8, which maps I to i, so lower('IŞIK') is 'işik' and never collides with 'ışık'. When Turkish rules define identity (user names, product names), index the Turkish lowercase. For e-mail addresses the default rules are what you want."
    bad: "CREATE UNIQUE INDEX ON users (lower(username));"
    good: "CREATE UNIQUE INDEX ON users (lower(username COLLATE \"tr-x-icu\"));"
  - title: "Expecting UNIQUE to reject a second NULL"
    detail: "Under the default NULLS DISTINCT, any number of rows may hold NULL in a unique column. Use NOT NULL, or NULLS NOT DISTINCT on PG 15+."
quiz:
  - q: "status is `text NOT NULL DEFAULT 'open'`. What does `INSERT INTO tickets (title, status) VALUES ('x', NULL)` do?"
    options: ["Stores 'open'", "Stores NULL", "Fails with a not-null violation", "Stores an empty string"]
    answer: 2
    why: "An explicit NULL is a value, so the DEFAULT doesn't apply and NOT NULL rejects it. Only an omitted column or the DEFAULT keyword gets the default."
  - q: "users has `id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY` and `email text NOT NULL`, nothing else. What stops two rows with the same e-mail?"
    options: ["The primary key", "The identity column", "NOT NULL", "Nothing"]
    answer: 3
    why: "The primary key is on id, which is different for every row. Only a UNIQUE constraint or unique index on email (or lower(email)) prevents the duplicate."
  - q: "A table has `UNIQUE (tenant_id, key)` without NULLS NOT DISTINCT. How many rows with (NULL, 'theme') can it hold?"
    options: ["One", "Two", "Any number"]
    answer: 2
    why: "NULLs are distinct by default, so the constraint never considers two of these rows equal."
further_reading:
  - title: "PostgreSQL docs: Constraints"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Default Values"
    url: "https://www.postgresql.org/docs/current/ddl-default.html"
  - title: "PostgreSQL docs: Unique Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-unique.html"
  - title: "PostgreSQL docs: CREATE TABLE"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
---

## Concept

A constraint is a rule the database checks on every INSERT and UPDATE, whoever sends
them. The four in this lesson cover most of a table's shape:

- **NOT NULL**: the column always has a value.
- **DEFAULT**: not a constraint strictly, but its partner: the value used when an INSERT
  leaves the column out.
- **PRIMARY KEY**: the row's identity. Unique and not null, one per table, and what
  foreign keys point at by default.
- **UNIQUE**: no two rows share these values. NULLs count as distinct unless you say
  `NULLS NOT DISTINCT` (PG 15+).

A key is either **natural** (data that already identifies the thing: an ISO currency
code, an e-mail address) or **surrogate** (a generated number or UUID with no meaning).
The usual design is a surrogate primary key for references plus a UNIQUE constraint on
the natural key, because a surrogate key by itself never stops a duplicate.

## Coming from C/C++/Java/Python/JS

- Java: Bean Validation's `@NotNull` runs in the JVM, for that one application. `NOT NULL`
  runs in the database, for every client, including psql and the next service someone
  writes.
- Go: a struct field always holds a value; an unset `int` is `0` and is sent as `0`. To
  get the column's DEFAULT, the INSERT has to leave the column out. To store NULL, the
  field needs a pointer or a nullable type.
- TypeScript: a nullable column is `string | null` under `strictNullChecks`, and NOT NULL
  is the database's non-optional property. SQL has no `undefined`: a value is either
  sent or the column is omitted.
- Python/JS: a UNIQUE constraint behaves like a set keyed on its columns, except that
  NULL never equals NULL unless you ask for `NULLS NOT DISTINCT`.

## Design notes

Application validation exists for good error messages; the constraint is the guarantee.
Two app instances can both "check that the e-mail is free" and both insert, while a
unique index lets exactly one of them commit. Map the SQLSTATE (23505
`unique_violation`, 23502 `not_null_violation`) to an HTTP status in one place and
let the constraint name pick the message.

Pick natural primary keys only when the value is small, standard and practically never
changes. Everything people can edit (e-mail addresses, requirement codes like REQ-12,
user names) gets a surrogate id plus UNIQUE. A requirements table, for example, keys
on `id` and declares `UNIQUE (project_id, code)`, so renaming REQ-12 touches one row
and every trace link to it survives.

PostgreSQL 18 records NOT NULL as a constraint in `pg_constraint`, with a name such as
`tickets_status_not_null`, and like a CHECK it can now be added `NOT VALID` and
validated later. What NOT NULL enforces is unchanged.
