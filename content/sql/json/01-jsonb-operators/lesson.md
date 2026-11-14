---
id: sql.json.jsonb-operators
title: "json vs jsonb; the operators ->, ->>, @>, ?"
track: sql
module: json
order: 1
level: intermediate
minutes: 14
since: pg14
prerequisites: [sql.tables-types.jsonb-arrays, sql.querying.null-logic]
compare_with: [sql.indexes.gin-brgin, sql.aggregation.agg-funcs, go.files-io.csv-json, ts.services.zod]
tags: [json, jsonb, operators, containment, existence, json-null, jsonb_set, subscripting]
summary: >
  -> walks a jsonb document and keeps jsonb, ->> ends the walk with text; a JSON
  null and a missing key are different things. @> tests whether a document
  contains a shape, ? whether a key or string element exists. ||, -, #-,
  jsonb_set and subscripts change documents, with traps around NULL.
examples:
  - id: navigate
    file: examples/01_navigate.sql
    title: "Navigating a document: ->, ->>, #>>, array indexes, and JSON null vs SQL NULL"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-19"
        note: >
          -> returns jsonb: number_jsonb is the JSON number 108, still of type
          jsonb. ->> returns text: main, without the quotes. Walk with -> and
          use ->> only at the last step, when you want the value.
      - lines: "20"
        note: >
          #>> takes the whole path as a text array. Event 2 has no agent, and
          the docs' rule applies: extraction operators return NULL instead of
          failing when the document doesn't have the requested structure.
      - lines: "21-23"
        note: >
          JSON array positions are 0-based, unlike SQL arrays (1-based), and -1
          is the last element. Event 2's empty tags array has no element 0,
          which is NULL again rather than an error.
      - lines: "29-33"
        note: >
          Event 1 has "retry_of": null, event 2 has no such key. -> returns the
          JSON value null for the first (a real jsonb value, so IS NULL is
          false) and SQL NULL for the second. ->> turns both into SQL NULL.
          Only ? and jsonb_typeof tell them apart.
      - lines: "39"
        note: >
          '9' > '100' as text, because the character 9 sorts after 1, so event 2
          passes a filter it should fail. The same happens in ORDER BY on a
          ->> value.
      - lines: "40-41"
        note: >
          Two correct forms: cast the text (an expression index can serve
          this one, see the indexing lesson), or compare jsonb with jsonb,
          which compares numbers numerically.
      - lines: "46"
        note: >
          SQLSTATE 42883: there is no text = integer operator. Compare with a
          string ('108'), cast, or ask with containment:
          payload @> '{"build": {"number": 108}}'.
  - id: containment
    file: examples/02_containment.sql
    title: "@> containment and the ? existence operators, and what json lacks"
    runtime: sql
    check: exact
    annotations:
      - lines: "16-17"
        note: >
          Containment is anchored at the root: the pattern's shape must match
          the document's from the top. {"branch": "main"} would need a
          top-level branch key, and no event has one.
      - lines: "18"
        note: >
          Array containment ignores order and duplicates: ["smoke", "nightly"]
          is contained in ["nightly", "smoke"]. Event 2 has only smoke.
      - lines: "19-20"
        note: >
          A string is not a one-element array: {"tags": "smoke"} matches
          nothing. The docs' exception, an array containing a bare primitive,
          shows up when you compare the array itself, as
          payload -> 'tags' @> '"smoke"' does.
      - lines: "21"
        note: >
          A pattern element matches any array element that contains it: some
          result has status failed. A pattern object with two keys must find
          both in the same element; status from one result and test from
          another don't combine.
      - lines: "27-28"
        note: >
          ? looks at top-level keys only. branch lives under build, so
          nested_key is false everywhere; payload -> 'build' ? 'branch' would
          be true.
      - lines: "29-31"
        note: >
          On an array, ? tests for string elements; ?| means any of the list,
          ?& all of it. Strings only: '[1, 2]'::jsonb ? '1' is false.
      - lines: "36-38"
        note: >
          jsonb compares numbers as numbers and objects key by key, so the two
          documents are equal. It still prints 120.0 as written: JSON numbers
          are stored as numeric, which keeps trailing zeros.
      - lines: "41"
        note: >
          SQLSTATE 42883. json stores the text and offers only the extraction
          operators (->, ->>, #>, #>>): no containment, existence, equality or
          GIN index. Cast to jsonb, or store jsonb in the first place.
  - id: modify
    file: examples/03_modify.sql
    title: "Changing documents: shallow ||, subscripts, jsonb_set, - and #-, and the NULL trap"
    runtime: sql
    check: exact
    annotations:
      - lines: "10"
        note: >
          || merges top-level keys, and the right side's value wins whole:
          notify lost email. The docs say it outright: || does not operate
          recursively.
      - lines: "14"
        note: >
          Subscript assignment (PG 14) creates or replaces one nested value. The
          right-hand side must be jsonb, hence the quoted 'true'.
      - lines: "16"
        note: >
          jsonb_set does the same with a text[] path, and adds the key because
          create_if_missing defaults to true. All earlier steps of the path must
          exist, or the document comes back unchanged.
      - lines: "20"
        note: >
          - removes a top-level key, #- removes the value at a path. Both are
          ordinary operators, applied left to right here.
      - lines: "25-27"
        note: >
          to_jsonb(NULL) is SQL NULL, and jsonb_set returns NULL when given one.
          In an UPDATE that sets the whole column to NULL, or fails if it is
          NOT NULL. jsonb_set_lax (PG 13) writes a JSON null by default, or
          deletes the key, raises an error or returns the document unchanged.
use_cases:
  - "Reading fields from stored webhook payloads, third-party API responses and event logs"
  - "Filtering semi-structured rows by shape: events on branch main with any failed result (@>)"
  - "Per-project settings and feature flags: test for a key (?), change one nested value (subscripts, jsonb_set)"
  - "String lists inside documents, such as tags: ?| for any of, ?& for all of"
strengths:
  - "@> says 'has this nested shape' in one operator that a GIN index can serve, where Go or Java code would decode and walk the tree"
  - "Extraction never throws on a missing key or a different shape; it returns NULL, which suits documents whose structure varies"
  - "jsonb comparisons are semantic: key order, whitespace and 120 vs 120.0 don't matter, unlike comparing JSON text"
  - "Partial updates (subscripts, jsonb_set, -, #-) happen inside one UPDATE, without a read-modify-write round trip through the application"
weaknesses:
  - "->> returns text, so numbers and dates need casts, and a forgotten cast compares strings without any error"
  - "A missing key, a JSON null and a mistyped path all come back as NULL through ->>, so a typo in a key name fails silently"
  - "|| is shallow and jsonb_set returns NULL for a NULL value: two easy ways to lose data in an UPDATE"
  - "Changing one key writes a new version of the whole row and locks it, so big, frequently updated documents contend"
pitfalls:
  - title: "Comparing ->> output as text"
    detail: "'9' > '100' is true for strings. Cast the extracted text, or compare jsonb values."
    bad: "WHERE payload -> 'build' ->> 'number' > '100'"
    good: "WHERE (payload -> 'build' ->> 'number')::int > 100"
  - title: "Testing a JSON null with -> ... IS NULL"
    detail: "-> returns the JSON value null, which is not SQL NULL. If null and missing mean the same to you, test the ->> result; if they differ, use ? or jsonb_typeof."
    bad: "WHERE payload -> 'retry_of' IS NULL"
    good: "WHERE payload ->> 'retry_of' IS NULL"
  - title: "Deep merge with ||"
    detail: "The right-hand object replaces the left one's nested objects whole. Set the nested value instead."
    bad: "UPDATE project_settings SET prefs = prefs || '{\"notify\": {\"slack\": true}}'"
    good: "UPDATE project_settings SET prefs['notify']['slack'] = 'true'"
  - title: "jsonb_set with a value that may be NULL"
    detail: "A NULL parameter makes jsonb_set return NULL, and the UPDATE erases the document (or fails on NOT NULL). Use jsonb_set_lax and say what NULL should mean."
    bad: "SET prefs = jsonb_set(prefs, '{theme}', to_jsonb($1::text))"
    good: "SET prefs = jsonb_set_lax(prefs, '{theme}', to_jsonb($1::text), true, 'delete_key')"
  - title: "Casting a JSON null on older servers"
    detail: "Since PostgreSQL 18, (payload -> 'retry_of')::int returns NULL for a JSON null; earlier versions raise an error. Code that must also run on older servers should cast the ->> text instead."
quiz:
  - q: "payload is {\"retry_of\": null}. What does `payload -> 'retry_of' IS NULL` return?"
    options: ["true", "false", "NULL"]
    answer: 1
    why: "-> returns the JSON value null, a non-NULL jsonb value. ->> would return SQL NULL."
  - q: "What does `'{\"tags\": [\"a\", \"b\"]}'::jsonb @> '{\"tags\": \"b\"}'` return?"
    options: ["true, because b is in the array", "false, because inside an object a string doesn't match an array", "an error"]
    answer: 1
    why: "Containment compares structure from the root; the primitive-in-array exception only applies when the array itself is compared, as in payload -> 'tags' @> '\"b\"'."
  - q: "build.number is 9 in one row and 108 in another. Which condition selects only 108?"
    options: ["payload -> 'build' ->> 'number' > '100'", "(payload -> 'build' ->> 'number')::int > 100", "payload -> 'number' > '100'"]
    answer: 1
    why: "The first compares text ('9' > '100' is true); the third looks for a top-level number key, which doesn't exist."
  - q: "prefs is {\"notify\": {\"email\": true, \"slack\": false}}. What is `prefs || '{\"notify\": {\"slack\": true}}'`?"
    options: ["{\"notify\": {\"email\": true, \"slack\": true}}", "{\"notify\": {\"slack\": true}}", "An error: duplicate key notify"]
    answer: 1
    why: "|| merges only the top level; the right side's notify object replaces the left one whole."
further_reading:
  - title: "PostgreSQL docs: json and jsonb operators"
    url: "https://www.postgresql.org/docs/current/functions-json.html#FUNCTIONS-JSON-OP-TABLE"
  - title: "PostgreSQL docs: Additional jsonb operators"
    url: "https://www.postgresql.org/docs/current/functions-json.html#FUNCTIONS-JSONB-OP-TABLE"
  - title: "PostgreSQL docs: jsonb Containment and Existence"
    url: "https://www.postgresql.org/docs/current/datatype-json.html#JSON-CONTAINMENT"
  - title: "PostgreSQL docs: jsonb Subscripting"
    url: "https://www.postgresql.org/docs/current/datatype-json.html#JSONB-SUBSCRIPTING"
  - title: "PostgreSQL docs: JSON processing functions (jsonb_set, jsonb_set_lax)"
    url: "https://www.postgresql.org/docs/current/functions-json.html#FUNCTIONS-JSON-PROCESSING"
---

## Concept

The jsonb lesson in the types module showed the storage difference: `json` keeps an exact copy of the
input text, `jsonb` stores it parsed (keys deduplicated and reordered, whitespace gone).
For querying, the consequence is that `json` only has the four extraction operators;
everything else in this lesson is `jsonb`-only. Use `json` when the exact text matters,
for example a webhook body whose signature was computed over its bytes, and `jsonb` for
everything you query.

The operators, by job:

- **Navigate:** `->` takes a key or a 0-based array index (negative counts from the
  end) and returns `jsonb`; `->>` does the same and returns `text`; `#>` and `#>>` take a
  whole path as `text[]`. A path that doesn't exist yields NULL, never an error.
- **Null:** a key holding JSON `null` is not the same as a missing key. `->` returns
  the jsonb value `null` for the first and SQL NULL for the second; `->>` returns SQL
  NULL for both.
- **Test:** `@>` asks whether the left document contains the right one's shape (`<@`
  is the reverse). `?`, `?|` and `?&` ask whether a string exists as a top-level key or
  as a string element of an array.
- **Change:** `||` merges or concatenates (shallowly), `-` and `#-` delete, `jsonb_set`
  and subscript assignment (`doc['a']['b'] = ...`, PG 14) replace one nested value.

## Coming from C/C++/Java/Python/JS

- **JS:** `payload #>> '{build,agent,os}'` is `payload?.build?.agent?.os`, with NULL for
  `undefined`. JSON null versus a missing key is JS's `null` versus `undefined`, and
  `payload ? 'retry_of'` is `'retry_of' in payload`.
- **Python:** `->` chains are `d.get('build', {}).get('agent')`; `?` is `key in d`;
  `@>` is a recursive "is a subset of" check that you would otherwise write by hand.
- **Go:** decoding into a struct, a `*int` field is nil both for `null` and for a
  missing key, so Go can't tell them apart without `json.RawMessage` or a map lookup;
  SQL can, with `?`. pgx uses `$1` placeholders, so the `?` operators cause no trouble.
- **Java:** JDBC uses `?` as the parameter placeholder. With PgJDBC, write the operators
  as `??` in a `PreparedStatement`, or use `@>`, which has no question mark.
- **Everyone:** `->>` hands you text. Comparisons and sorting on it are text operations
  until you cast, whereas a JSON decoder in your language would have given you 108 as a
  number.

## Design notes

Filter with operators an index can serve and extract in the `SELECT` list. `WHERE
payload @> '{"build": {"branch": "main"}}'` and `WHERE payload -> 'build' ->> 'branch' =
'main'` ask the same question, but only the first can use a GIN index on `payload` (the
GIN lesson); the second needs an expression index of its own (the indexing lesson in this
module).

Decide once what "absent" means. If a JSON `null` and a missing key mean the same thing,
strip nulls on write (`jsonb_strip_nulls`) or always test the `->>` result; if they
differ, test with `?` and say so in a comment.

Update narrowly: subscripts or `jsonb_set` for nested values, `jsonb_set_lax` when the
new value comes from a parameter that may be NULL, and `||` only for top-level keys.
Every one of these writes a new version of the whole row and takes its row lock. A key
that several writers update independently, or that every query filters on, has outgrown
the document; the indexing lesson shows how to promote it to a column.

PostgreSQL 18 changed one detail: casting a JSON `null` to a scalar type, as in
`(payload -> 'retry_of')::int`, now returns NULL; earlier versions raise an error.
