---
id: sql.tables-types.jsonb-arrays
title: "jsonb, arrays, enums vs lookup tables vs CHECK"
track: sql
module: tables-types
order: 4
level: intermediate
minutes: 14
since: pg9.4
prerequisites: [sql.tables-types.text-numbers]
compare_with: [sql.json.jsonb-operators, sql.modeling.anti-patterns, sql.joins.many-to-many, ts.everyday-types.unions-enums]
tags: [types, jsonb, json, arrays, enum, lookup-table, check]
summary: >
  jsonb suits attributes that really vary per row, arrays suit small lists read with
  the row; neither checks types or references. For a fixed set of values pick CHECK
  (simplest), an enum (ordered, compact, values never removable) or a lookup table
  (values that carry data or change often).
examples:
  - id: jsonb
    file: examples/01_jsonb.sql
    title: "json vs jsonb, querying attributes, and what jsonb doesn't check"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          jsonb parses the document once on input: whitespace, key order and
          duplicate keys are gone (only the last "a" survives). The docs advise
          jsonb for most applications; plain json only when the exact text must
          be kept.
      - lines: "9-10"
        note: >
          The dividing line of this lesson. price exists for every product and is
          filtered, summed and constrained, so it is a column with a type. Tea has
          an origin, a cable has a length: those vary by category, so they go in
          jsonb.
      - lines: "20-23"
        note: >
          -> returns jsonb (1000 is a JSON number), ->> returns text (Rize, not
          "Rize"). @> is containment and ? tests for a key; both can be served by
          a GIN index on attrs, which the JSON module covers.
      - lines: "26-27"
        note: >
          A string where a number belongs, accepted without complaint. A CHECK
          such as jsonb_typeof(attrs -> 'weight_g') = 'number' would stop it; if
          you find yourself writing many such checks, the field wants to be a
          column.
      - lines: "30-32"
        note: >
          SQLSTATE 22P02 at read time, far from the insert that caused it, and the
          whole report fails. This is the real cost of schemaless data: the
          validation you skipped on write is paid by every reader.
  - id: arrays
    file: examples/02_arrays.sql
    title: "Arrays: 1-based indexing, membership, unnest, and no foreign keys"
    runtime: sql
    check: exact
    annotations:
      - lines: "4"
        note: >
          text[] is an array of text. Declared sizes such as text[3] are accepted
          and ignored, per the docs. NOT NULL DEFAULT '{}' keeps "no tags" as an
          empty array, so code never has to tell NULL and {} apart.
      - lines: "15-16"
        note: >
          Indexing starts at 1. tags[0] and tags[3] are NULL, not an error: like
          JS's undefined, unlike Python's IndexError or Go's panic. For the empty
          array, array_length(tags, 1) is NULL while cardinality(tags) is 0, so
          WHERE array_length(tags, 1) = 0 never matches.
      - lines: "21-22"
        note: >
          'sql' = ANY (tags) reads naturally for one value; tags @> ARRAY['go',
          'sql'] means "has all of these". A GIN index on tags can serve @> but
          not = ANY, so write tags @> ARRAY['sql'] on large tables.
      - lines: "25-28"
        note: >
          unnest expands each row's array into rows, and the function in FROM can
          refer to the table before it. This is how arrays rejoin set-based SQL:
          as soon as you aggregate over elements, you are treating them as rows.
      - lines: "30-32"
        note: >
          'golang' next to 'go' is now a second tag, and nothing can stop it: a
          foreign key can't point into an array. The docs' own tip: arrays are not
          sets, and searching for elements can be a sign of misdesign. When tags
          need integrity or data of their own, use a junction table.
  - id: enum-lookup-check
    file: examples/03_enum_lookup_check.sql
    title: "One status, three designs: enum, lookup table, CHECK"
    runtime: sql
    check: exact
    annotations:
      - lines: "2"
        note: >
          An enum is a type: 4 bytes per value, labels case-sensitive, and two
          different enum types can't even be compared. Values can be added and
          renamed, but never removed or reordered without recreating the type.
      - lines: "5-10"
        note: >
          A lookup table holds the values as rows. With the code itself as the
          primary key, tickets.as_lookup stays readable without a join; the join
          is only needed for the label and position.
      - lines: "17"
        note: >
          A CHECK keeps the column plain text and the allowed list in the table's
          definition. Easiest to read, but only this table knows the list.
      - lines: "25-27"
        note: >
          enum_order follows the declaration: open, in_progress, done. text_order
          is alphabetical: done, in_progress, open. If "sort by status" should
          mean workflow order, text needs a CASE or a position column.
      - lines: "30-32"
        note: >
          Labels and order are data: translating a label or moving a status is an
          UPDATE, not a migration. Neither an enum nor a CHECK can attach a
          Turkish label to a value.
      - lines: "35"
        note: >
          In a migration the type already exists, and the docs' rule applies: a
          value added inside a transaction block can't be used until that
          transaction commits. (This sandbox created the type in the same
          transaction, a case where Postgres allows the new value at once.)
      - lines: "37-40"
        note: >
          Changing a CHECK means dropping and re-adding it, and the new one scans
          every row under an exclusive lock. On a large table, add it NOT VALID
          and validate separately, as the ALTER TABLE lesson shows.
      - lines: "44"
        note: >
          SQLSTATE 23503: rows still say 'open'. Retiring a lookup value is
          explicit (migrate the rows, or add an active flag); for an enum there is
          no statement for it at all.
use_cases:
  - "jsonb: per-category product attributes, per-integration settings, raw webhook payloads kept for replay"
  - "arrays: tags or labels read and written with their row, small value lists without an identity of their own"
  - "CHECK (x IN ...): short, stable lists owned by one table"
  - "enum: stable, ordered states shared by several tables (severity, workflow stage)"
  - "lookup table: values with labels, translations, sort order or flags, or values that change at business speed"
strengths:
  - "jsonb gives per-row flexibility without an entity-attribute-value table, and GIN indexes make @> and ? fast"
  - "An array keeps a short list inside the row: one read, no join, and GIN-indexable containment"
  - "enum values are 4 bytes, typo-proof, and sort in a meaningful order"
  - "A lookup table is the only option where values carry data and change with DML instead of DDL"
weaknesses:
  - "Nothing inside jsonb or an array is type-checked or reference-checked unless you add CHECK constraints"
  - "Changing one key or element writes a new version of the whole row and locks it, which the docs flag for large documents"
  - "Enum values can't be removed or reordered without recreating the type, and an added value can't be used before it commits"
  - "A lookup table costs a foreign-key check on every write and a join for every label"
pitfalls:
  - title: "jsonb for core fields"
    detail: "Fields that every row has and that queries filter, join or sum on lose types, NOT NULL, foreign keys and planner statistics inside jsonb."
    bad: "SELECT sum((data ->> 'price')::numeric) FROM orders"
    good: "price numeric(12, 2) NOT NULL as a column; jsonb for the leftovers"
  - title: "Arrays of foreign keys"
    detail: "author_ids int[] can't be checked against authors, and deleting an author leaves dangling ids. A junction table with two foreign keys can."
    bad: "articles.author_ids int[]"
    good: "article_authors (article_id REFERENCES articles, author_id REFERENCES authors, PRIMARY KEY (article_id, author_id))"
  - title: "Testing an empty array with array_length"
    detail: "array_length of an empty array is NULL, so the comparison is never true."
    bad: "WHERE array_length(tags, 1) = 0"
    good: "WHERE cardinality(tags) = 0"
  - title: "Adding an enum value and using it in one migration"
    detail: "Migration tools usually wrap each migration in a transaction; the new value can't be used before ALTER TYPE commits. Put ADD VALUE and the UPDATE that uses it in separate migrations."
  - title: "Comparing or grouping json values"
    detail: "The json type has no equality operator, so =, DISTINCT and GROUP BY on json fail with 'operator does not exist: json = json'. jsonb has one."
  - title: "Appending a literal to an array with ||"
    detail: "tags || 'golang' reads the literal as an array and fails with malformed array literal. Use array_append(tags, 'golang') or ARRAY['golang']."
quiz:
  - q: "An object in a jsonb input has the same key twice. What is stored?"
    options: ["Both pairs, in input order", "Only the last value for that key", "Only the first value", "Nothing: the input is rejected"]
    answer: 1
    why: "jsonb doesn't keep duplicate keys; only the last value is kept. (json keeps the exact text, duplicates included.)"
  - q: "tags is text[] holding {go,sql}. What is tags[0]?"
    options: ["'go'", "NULL", "An error"]
    answer: 1
    why: "Arrays are 1-based and an out-of-bounds subscript returns NULL instead of raising an error."
  - q: "Ticket statuses need a Turkish label each, and support staff add new ones every month. Which design fits?"
    options: ["enum", "CHECK constraint", "Lookup table plus a foreign key"]
    answer: 2
    why: "Only a lookup table lets values carry data (the label) and be added or retired with plain DML."
  - q: "Why can't one migration transaction run ALTER TYPE status ADD VALUE 'archived' and then set rows to 'archived'?"
    options: ["A new enum value can't be used until the ALTER TYPE commits", "ADD VALUE is not allowed in a transaction block", "Enum columns can't be updated"]
    answer: 0
    why: "Since PostgreSQL 12 ADD VALUE may run in a transaction block, but the docs say the new value can't be used until the transaction has committed."
further_reading:
  - title: "PostgreSQL docs: JSON Types"
    url: "https://www.postgresql.org/docs/current/datatype-json.html"
  - title: "PostgreSQL docs: Arrays"
    url: "https://www.postgresql.org/docs/current/arrays.html"
  - title: "PostgreSQL docs: Enumerated Types"
    url: "https://www.postgresql.org/docs/current/datatype-enum.html"
  - title: "PostgreSQL docs: ALTER TYPE (adding enum values)"
    url: "https://www.postgresql.org/docs/current/sql-altertype.html"
---

## Concept

Relational columns give you types, constraints and statistics per field. Three features
relax that, each for a specific shape of data:

- **`jsonb`**: a parsed JSON document per row, for attributes that differ from row to
  row. Queryable (`->`, `->>`, `@>`, `?`) and indexable, but nothing inside is checked.
- **Arrays** (`text[]`, `int[]`): an ordered list inside the row, for small lists you
  read and write with it. 1-based, and no foreign keys into them.
- **A fixed set of values** (a status, a severity) can be declared three ways: a
  `CHECK (x IN (...))` on a text column, an `enum` type, or a lookup table referenced by
  a foreign key. They differ in what changing the set costs.

## Coming from C/C++/Java/Python/JS

- **TypeScript:** `'open' | 'in_progress' | 'done'` is the same idea as an enum or a CHECK,
  enforced at a different layer; a `jsonb` column is `Record<string, unknown>` and needs
  validating (zod) when it comes back, exactly like a `fetch` response.
- **Java:** Java enums are ordered like Postgres enums, but Java enums can carry fields.
  Postgres enums can't; that's what the lookup table is for.
- **Go:** Go has no enum type, only typed constants, so the database enum is stricter than
  your Go code. `jsonb` maps naturally to `json.RawMessage` or a struct via
  `encoding/json`.
- **Python/JS:** arrays here start at 1, and out-of-range reads return NULL, not an
  `IndexError`.
- **Document databases:** `jsonb` gives you a document per column inside a relational row,
  so you can keep transactions, joins and constraints for everything that is stable.

## Design notes

Structured where it's stable, JSON where it isn't. If every row has the field, or you
filter, join or sum on it, it is a column. `jsonb` earns its place for the long tail of
optional attributes and for payloads you store as received. The docs also suggest keeping
documents to a predictable structure and a manageable size, since every update rewrites
and locks the whole row.

Arrays are a denormalization: fine for tags that are just strings, wrong as soon as the
elements are references to other rows or need their own data.

For value sets, start with CHECK: it's plain text, visible in the table definition, and
changing it is ordinary DDL. Choose an enum when the order matters and the set is stable
across many tables. Choose a lookup table when values carry data (labels, translations,
positions, an active flag) or when people outside engineering add them.
