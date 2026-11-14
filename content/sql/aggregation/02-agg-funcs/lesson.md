---
id: sql.aggregation.agg-funcs
title: "string_agg, array_agg, jsonb_agg"
track: sql
module: aggregation
order: 2
level: intermediate
minutes: 13
since: pg9.5
prerequisites: [sql.aggregation.group-by, sql.tables-types.jsonb-arrays]
compare_with: [sql.json.jsonb-operators, sql.joins.lateral, go.files-io.csv-json]
tags: [aggregation, string_agg, array_agg, jsonb_agg, jsonb_object_agg, json, collation, fan-out]
summary: >
  Aggregates can build values: string_agg joins text, array_agg collects an
  array, jsonb_agg and jsonb_object_agg build JSON. Put ORDER BY inside the
  call, handle the NULLs a LEFT JOIN brings in, and aggregate each child list
  on its own to avoid join fan-out.
examples:
  - id: lists
    file: examples/01_lists.sql
    title: "string_agg and array_agg: element order, NULLs, DISTINCT, Turkish sorting"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-19"
        note: >
          string_agg skips NULL inputs (Giriş has no owner); array_agg keeps
          them as NULL elements. ORDER BY inside the call orders the elements;
          the query's own ORDER BY only sorts the result rows. Without it the
          element order is unspecified.
      - lines: "20-21"
        note: >
          DISTINCT removes duplicates before collecting, and FILTER drops the
          NULL that array_agg would otherwise keep. Can owns two tests and
          appears once.
      - lines: "28-29"
        note: >
          The elements are compared in the column's collation. This database's
          default, C.UTF-8, compares code points, so Ç (U+00C7), İ (U+0130) and
          Ş (U+015E) sort after every ASCII letter and Giriş comes first.
          tr-x-icu gives the Turkish alphabet: C, Ç, ..., G, ..., I, İ, ..., S, Ş.
      - lines: "34-35"
        note: >
          With no input rows array_agg returns NULL, not an empty array, and so
          do string_agg and jsonb_agg. coalesce supplies the empty value a
          client can iterate over.
      - lines: "40"
        note: >
          SQLSTATE 42P10. Can has two ids, so "the distinct owners ordered by
          id" isn't defined. Order by the aggregated value, or remove
          duplicates in a subquery first.
  - id: json
    file: examples/02_json.sql
    title: "jsonb_agg and jsonb_build_object: a nested document per row, and the empty-list trap"
    runtime: sql
    check: exact
    annotations:
      - lines: "12-13"
        note: >
          REQ-EXPORT's padding row has t.id and t.title NULL, but the object
          built from them is not NULL: [{"id": null, "title": null}].
          jsonb_agg_strict (PG 16) skips NULL inputs and wouldn't catch this
          either.
      - lines: "15"
        note: >
          jsonb_build_object takes alternating keys and values. Each test row
          becomes one object, and jsonb_agg collects them in t.id order.
      - lines: "26-28"
        note: >
          FILTER (WHERE t.id IS NOT NULL) drops the padding row, so the
          aggregate sees no input and returns NULL, and coalesce turns that
          into []. The same two steps apply to array_agg with '{}'.
      - lines: "23-29"
        note: >
          The whole response document comes out of the database; the Go or TS
          layer can pass it through. jsonb doesn't preserve key order, so keys
          appear in jsonb's storage order, not the order of the call. Use the
          json variants if the text order matters.
      - lines: "36-37"
        note: >
          jsonb_object_agg(key, value) builds an object; keys must not be NULL,
          and with jsonb a duplicate key keeps its last value. Aggregates can't
          be nested, so jsonb_object_agg(status, count(*)) is an error; count
          in a grouped subquery first.
  - id: two-lists
    file: examples/03_two_lists.sql
    title: "Two child lists: join fan-out, and one subquery per list"
    runtime: sql
    check: exact
    annotations:
      - lines: "18-25"
        note: >
          Each test row pairs with each tag row of the same requirement, and
          the aggregates see all 6 pairs: count says 6, every tag appears twice.
          REQ-EXPORT gets {NULL} from its padding row. count(DISTINCT t.id) and
          array_agg(DISTINCT g.tag) would hide the damage but still build the
          product, and array_agg(DISTINCT t.title) would also merge two
          different tests that happen to share a title.
      - lines: "29-35"
        note: >
          ARRAY(subquery) is a correlated subquery per requirement, so each list
          is built from its own table with its own ORDER BY, and nothing
          multiplies. It returns {} when the subquery finds no rows. For JSON,
          the same shape is (SELECT coalesce(jsonb_agg(...), '[]') FROM ...
          WHERE ... = r.id), or a LATERAL subquery per child.
use_cases:
  - "API list endpoints that return parent rows with nested children as JSON, in one query instead of 1 + N"
  - "Exports and reports with a readable cell such as 'Can, Zeynep' (string_agg) or a tag list (array_agg)"
  - "Lookup maps sent to a frontend: jsonb_object_agg(code, label) gives {\"REQ-1\": \"Login\", ...}"
  - "Collecting ids to pass on in one parameter: array_agg(id) feeding = ANY($1) in a later query"
strengths:
  - "One query returns the response shape, so the ORM-style 1 + N queries (one per parent for its children) disappear"
  - "ORDER BY inside the aggregate gives deterministic lists with the collation you choose, including Turkish order"
  - "jsonb_build_object and jsonb_agg produce valid JSON with correct escaping; there is no string concatenation to get wrong"
  - "string_agg skips NULLs, unlike JS's join (empty string) or Java's String.join (the text \"null\")"
weaknesses:
  - "NULL behaviour differs per function: string_agg skips NULLs, array_agg and jsonb_agg keep them, and all return NULL for no rows"
  - "Over a LEFT JOIN, the padding row ends up in the list ({NULL}, or an object of nulls) unless you FILTER it out"
  - "Every collected value is built in memory and a single value is capped at 1 GB, so whole-table exports belong in COPY or a streamed cursor, not one jsonb_agg"
  - "JSON built in SQL is untyped to the application; Go or TS still needs a schema check (struct tags, zod) on the way in"
pitfalls:
  - title: "ORDER BY in the wrong place"
    detail: "The query's ORDER BY sorts result rows, not the elements of an aggregate. Inside the call it goes after all arguments: string_agg(title ORDER BY title, ', ') parses as a one-argument string_agg and fails with function string_agg(text) does not exist."
    bad: "SELECT string_agg(title ORDER BY title, ', ') FROM test_cases"
    good: "SELECT string_agg(title, ', ' ORDER BY title) FROM test_cases"
  - title: "jsonb_agg over a LEFT JOIN"
    detail: "A parent without children yields [{\"id\": null, ...}], or [null] when you aggregate a bare column. Filter the padding row out and coalesce the empty case."
    bad: "jsonb_agg(jsonb_build_object('id', t.id))"
    good: "coalesce(jsonb_agg(jsonb_build_object('id', t.id)) FILTER (WHERE t.id IS NOT NULL), '[]')"
  - title: "Two child lists from two joins"
    detail: "Joining tests and tags to the same requirement multiplies them, and every list and count is inflated. DISTINCT hides it at a cost. Aggregate each child in its own subquery."
    bad: "FROM requirements r LEFT JOIN test_cases t ON ... LEFT JOIN tags g ON ... GROUP BY r.id"
    good: "SELECT r.code, ARRAY(SELECT ... FROM test_cases t WHERE t.requirement_id = r.id ORDER BY ...), ARRAY(SELECT ... FROM tags g WHERE g.requirement_id = r.id ORDER BY ...) FROM requirements r"
  - title: "Code-point order for Turkish names"
    detail: "Under C.UTF-8, Çağla, İrem and Şule sort after Zeynep. Name the collation in the aggregate's ORDER BY, or declare the column with it."
    bad: "string_agg(name, ', ' ORDER BY name)"
    good: "string_agg(name, ', ' ORDER BY name COLLATE \"tr-x-icu\")"
quiz:
  - q: "owner holds 'Zeynep', NULL and 'Can' (in id order). What does `string_agg(owner, ', ' ORDER BY id)` return?"
    options: ["Zeynep, , Can", "Zeynep, Can", "NULL"]
    answer: 1
    why: "string_agg concatenates the non-NULL inputs only; the NULL contributes neither a value nor a delimiter."
  - q: "REQ-EXPORT has no tests. What does `jsonb_agg(jsonb_build_object('id', t.id))` give for it after `requirements r LEFT JOIN test_cases t ... GROUP BY r.id`?"
    options: ["NULL", "[]", "[{\"id\": null}]"]
    answer: 2
    why: "The LEFT JOIN supplies one padding row, and jsonb_build_object turns its NULLs into an object, which jsonb_agg collects."
  - q: "A requirement has 2 tests and 3 tags. Joined to both tables and grouped by requirement, what does count(t.id) return?"
    options: ["2", "3", "6"]
    answer: 2
    why: "Each test row pairs with each tag row of the same requirement, so the group holds 2 x 3 rows, all with a non-NULL t.id."
  - q: "Which call returns the titles in alphabetical order?"
    options: ["string_agg(title, ', ') with ORDER BY title at the end of the query", "string_agg(title, ', ' ORDER BY title)", "string_agg(title ORDER BY title, ', ')"]
    answer: 1
    why: "Element order is set inside the call, after all arguments. The query-level ORDER BY sorts rows, and the third form is a call to a one-argument string_agg that doesn't exist."
further_reading:
  - title: "PostgreSQL docs: Aggregate Functions"
    url: "https://www.postgresql.org/docs/current/functions-aggregate.html"
  - title: "PostgreSQL docs: Aggregate Expressions (ORDER BY inside the call)"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-AGGREGATES"
  - title: "PostgreSQL docs: JSON creation functions (jsonb_build_object)"
    url: "https://www.postgresql.org/docs/current/functions-json.html#FUNCTIONS-JSON-CREATION-TABLE"
  - title: "PostgreSQL docs: JSON Types (json vs jsonb, key order, duplicate keys)"
    url: "https://www.postgresql.org/docs/current/datatype-json.html"
  - title: "PostgreSQL docs: Array Constructors (ARRAY(subquery))"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SQL-SYNTAX-ARRAY-CONSTRUCTORS"
---

## Concept

Not every aggregate reduces a group to a number. Some collect it:

- `string_agg(value, delimiter)` joins text,
- `array_agg(value)` builds an array,
- `jsonb_agg(value)` builds a JSON array, and `jsonb_object_agg(key, value)` a JSON object.

They follow the same rules as `count`: one result per group, computed after `WHERE`. What
is new is that their result depends on the **order** of the input, and that order is
unspecified unless you write `ORDER BY` inside the call.

Together with `jsonb_build_object`, they turn a parent-child relationship into a nested
document in one query: one row per requirement, its tests as a JSON array. That is the
shape an API returns, and it replaces the "one query for the parents, one per parent for
the children" pattern ORMs fall into.

Two things go wrong regularly. NULLs: `string_agg` skips them, `array_agg` and
`jsonb_agg` keep them, and a LEFT JOIN supplies one for every parent without children.
Fan-out: joining two child tables to the same parent multiplies their rows before any
aggregate sees them.

## Coming from C/C++/Java/Python/JS

- **Python:** `', '.join(sorted(names))` is `string_agg(name, ', ' ORDER BY name)`,
  except that `join` raises `TypeError` on `None` where `string_agg` skips it. `sorted()`
  on `str` compares code points, like C.UTF-8 here; Turkish order needs a collation
  library, as it needs `COLLATE "tr-x-icu"` here.
- **JS:** `[a, null, b].join(', ')` turns the null into an empty string (`"a, , b"`);
  `Intl.Collator('tr')` is the browser's Turkish collation.
- **Java:** `Collectors.joining(", ")` appends the text `null` for null elements;
  `Collator.getInstance(Locale.forLanguageTag("tr"))` sorts Turkish.
- **Go:** `strings.Join(slices.Sorted(...), ", ")`, and `golang.org/x/text/collate` for
  Turkish order. A `jsonb_agg` result scans into `json.RawMessage` and can go straight
  into the HTTP response.
- **ORMs:** eager-loading two collections through joins hits the same fan-out as
  example 3, which is why ORMs usually load each collection with a query of its own.

## Design notes

Build JSON in SQL when the response is a projection of the query: list endpoints,
exports, dashboards. The application passes the bytes through and stays thin. Build
objects in Go or TS when the data feeds domain logic that wants typed values. Either way,
make empty lists `[]`, never `null`: `coalesce(... FILTER (WHERE child.id IS NOT NULL),
'[]')`.

Give each child list its own subquery (`ARRAY(...)`, a scalar `(SELECT jsonb_agg(...)
...)`, or `LATERAL`). One `GROUP BY` over two child joins is wrong by construction, and
`DISTINCT` inside the aggregates only hides it.

Prefer `jsonb_agg` and `jsonb_build_object` unless you need the exact text (key order,
duplicate keys) that the `json` variants keep. For Turkish text, put the collation where
the ordering happens (`ORDER BY name COLLATE "tr-x-icu"` inside the aggregate) or on the
column, if Turkish order is what the column means everywhere.
