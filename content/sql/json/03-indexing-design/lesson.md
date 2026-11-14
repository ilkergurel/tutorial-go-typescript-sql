---
id: sql.json.indexing-design
title: "Indexing jsonb, building JSON responses in SQL, when a real column wins"
track: sql
module: json
order: 3
level: advanced
minutes: 15
since: pg12
prerequisites: [sql.json.jsonb-operators, sql.indexes.gin-brgin, sql.constraints.generated]
compare_with: [sql.modeling.anti-patterns, sql.aggregation.agg-funcs, go.http.json-validation, ts.services.zod]
tags: [json, jsonb, indexes, expression-index, gin, generated-columns, json-responses, schema-design]
summary: >
  GIN serves containment and jsonpath matches; ranges and sorting on one key
  need a B-tree on the exact expression the query writes. json keeps the key
  order you build, jsonb normalizes it. A key that every row has and every query
  uses wants to be a real, typed, constrained column.
examples:
  - id: expression-index
    file: examples/01_expression_index.sql
    title: "GIN for @? and a B-tree expression index for ranges and ORDER BY ... LIMIT"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-11"
        note: >
          Priority 1 is on 1 % of the tickets, the kvkk label on 2 %: selective
          conditions, the case where an index pays off.
      - lines: "14"
        note: >
          jsonb_path_ops, the smaller GIN operator class from the GIN lesson:
          it serves @>, @? and @@, but not the key-existence operators.
      - lines: "15"
        note: >
          A B-tree on an expression. Anything that isn't a plain column or a
          function call needs its own parentheses, hence the doubled ones. The
          cast makes it an index of integers, so ranges and sorting compare
          numbers, not text.
      - lines: "16"
        note: >
          The docs: a new expression index gets statistics only after ANALYZE.
          Until then the planner has no idea how common priority 1 is and falls
          back to default estimates.
      - lines: "19-20"
        note: >
          The jsonpath filter is answered from the GIN index. The plan prints the
          path in canonical form ($."labels"[*]?(@ == "kvkk")): a jsonpath
          value is stored parsed, like jsonb.
      - lines: "23-24"
        note: >
          The 200 priority-1 tickets come from tickets_priority. GIN couldn't
          help here: <= isn't one of its jsonb operators.
      - lines: "25-29"
        note: >
          An Index Scan in index order feeds the LIMIT directly: no Sort, and
          the first five entries read are the answer. For a deterministic top
          five, add a tie-breaker and index (((doc ->> 'priority')::int), id).
      - lines: "32-33"
        note: >
          (doc -> 'priority')::int casts jsonb to int (allowed since PG 11) and
          gives the same numbers, but the planner matches expressions, not
          meanings: Seq Scan. Keep the indexed expression in one place, such as
          a view, or promote the key to a column (example 3).
  - id: responses
    file: examples/02_responses.sql
    title: "Building JSON responses: json vs jsonb key order, whole rows, timestamps, big ids"
    runtime: sql
    check: exact
    annotations:
      - lines: "23-26"
        note: >
          json_build_object returns json: keys in call order, printed as
          "key" : value with spaces around the colon. jsonb_build_object returns
          jsonb: keys stored shorter first (id, code, owner, title) and printed
          in jsonb's canonical "key": value form. Both are valid JSON; choose
          json when a client or a snapshot test depends on key order. Turkish
          text goes out as UTF-8, not as \u escapes.
      - lines: "32"
        note: >
          to_jsonb(row) names keys after columns, so renaming a column renames
          an API field. - 'id' drops a key; jsonb_strip_nulls drops REQ-2's null
          owner. For public APIs list the fields explicitly: with to_jsonb, a
          column added later, say password_hash, would leak.
      - lines: "40"
        note: >
          timestamptz becomes an ISO 8601 string in the session time zone,
          "2026-11-13T09:00:00+03:00": what JavaScript's Date and Go's time.Time
          (RFC 3339) parse.
      - lines: "42"
        note: >
          9007199254740993 is exact in jsonb, whose numbers are numeric. A
          browser's JSON.parse returns 9007199254740992: JavaScript numbers are
          doubles, exact only up to 2^53 - 1. Send bigint ids as strings, or keep
          them below that limit.
      - lines: "41-45"
        note: >
          The subquery-per-list shape from the aggregation lesson: each list is
          aggregated on its own, and coalesce turns "no tests" into [] instead
          of NULL.
  - id: promote-column
    file: examples/03_promote_column.sql
    title: "Promoting a key to a STORED generated column with NOT NULL and a foreign key"
    runtime: sql
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          customer_id and total are in every order; gift_note and coupon only in
          some. That split is the design question of this lesson.
      - lines: "16-19"
        note: >
          A STORED generated column computes its value on every write of doc and
          stores it: a real int column that can be NOT NULL, carry a foreign key
          and be indexed. doc stays the single source of truth, and the column
          can never disagree with it. Adding a stored generated column rewrites
          the table, so on a big table plan it like any rewriting migration.
      - lines: "18"
        note: >
          PostgreSQL 18 makes a generated column VIRTUAL unless you say STORED,
          and a virtual column can't be indexed or carry a foreign key (both
          fail with SQLSTATE 0A000). Here STORED is not optional.
      - lines: "22-26"
        note: >
          The join uses a typed, indexed, constrained column; total still comes
          out of the document. If every report sums totals, total is the next
          key to promote.
      - lines: "29"
        note: >
          What remains in the document is genuinely optional data: notes and
          coupons that vary per order.
      - lines: "35"
        note: >
          SQLSTATE 23503: the foreign key runs on write. A missing customer_id
          would fail NOT NULL (23502) and a non-numeric one the cast (22P02),
          also at INSERT time. jsonb alone would have accepted all three.
use_cases:
  - "GIN (jsonb_path_ops): filtering event payloads or product attributes by varying keys with @> or jsonpath"
  - "B-tree expression index: one hot key used for ranges, sorting or top-N, while the rest stays in the document"
  - "JSON built in SQL: API responses with nested lists, CSV-free exports, snapshots for an audit log"
  - "STORED generated columns: promoting stable keys of ingested documents (customer, type, created time) to typed, constrained, indexed columns"
strengths:
  - "One GIN index serves containment and jsonpath filters on any key, without an index per attribute"
  - "Expression indexes give one key B-tree powers (ranges, ORDER BY ... LIMIT) and planner statistics, without changing the table"
  - "SQL-built JSON is escaped correctly and arrives in one round trip; the Go or TS layer can pass it through"
  - "A generated column turns a document field into a real column (types, NOT NULL, foreign keys, statistics) while the document stays the source"
weaknesses:
  - "GIN can't do ranges or ordering, and an expression index serves only the exact expression it was built on"
  - "Statistics on keys inside a document exist only for indexed or declared expressions; everything else is estimated with defaults"
  - "json and jsonb render keys differently, and to_jsonb(row) couples API field names to column names"
  - "Promotion costs a table rewrite for a stored generated column, and every promoted key is one more place a schema change must reach"
pitfalls:
  - title: "Writing the expression differently from the index"
    detail: "(doc -> 'k')::int and (doc ->> 'k')::int mean the same thing, but only the indexed spelling uses the index. Put the expression in one view or function, or make it a column."
    bad: "CREATE INDEX ON tickets (((doc ->> 'priority')::int)); ... WHERE (doc -> 'priority')::int <= 1"
    good: "CREATE INDEX ON tickets (((doc ->> 'priority')::int)); ... WHERE (doc ->> 'priority')::int <= 1"
  - title: "A generated column without STORED in PostgreSQL 18"
    detail: "Since PG 18 the default is VIRTUAL, computed on read, and a virtual column can't have an index or a foreign key. Say STORED when the column exists to be indexed or referenced."
    bad: "ADD COLUMN customer_id int GENERATED ALWAYS AS ((doc ->> 'customer_id')::int)"
    good: "ADD COLUMN customer_id int GENERATED ALWAYS AS ((doc ->> 'customer_id')::int) STORED"
  - title: "bigint ids as JSON numbers for JavaScript clients"
    detail: "Ids above 2^53 - 1 lose precision in JSON.parse without any error. Send them as strings (t.id::text), or use ids that stay below the limit."
  - title: "to_jsonb(row) as a public API"
    detail: "Every column, including ones added later, becomes a field. Build the object from an explicit list of keys."
    bad: "SELECT to_jsonb(u) FROM users u"
    good: "SELECT jsonb_build_object('id', u.id::text, 'name', u.name) FROM users u"
  - title: "Expecting jsonb to keep the key order you built"
    detail: "jsonb stores keys shorter-first and then bytewise, so id comes before code whatever you wrote. Use the json variants (json_build_object, json_agg) when order matters to a consumer."
quiz:
  - q: "There is a GIN index on `doc` (jsonb_path_ops). Which condition can it serve?"
    options: ["(doc ->> 'priority')::int <= 1", "doc @? '$.labels[*] ? (@ == \"kvkk\")'", "doc ? 'labels'"]
    answer: 1
    why: "jsonb_path_ops supports @>, @? and @@. Ranges are B-tree work, and key existence needs the default jsonb_ops class."
  - q: "The index is on `((doc ->> 'priority')::int)`. Does `WHERE (doc -> 'priority')::int = 1` use it?"
    options: ["Yes, the values are identical", "No, the planner matches the indexed expression, not its meaning", "Only after ANALYZE"]
    answer: 1
    why: "Index matching is syntactic on the expression tree; -> plus a jsonb-to-int cast is a different expression."
  - q: "`jsonb_build_object('title', 'x', 'id', 1)` prints as?"
    options: ["{\"title\": \"x\", \"id\": 1}", "{\"id\": 1, \"title\": \"x\"}", "{\"title\" : \"x\", \"id\" : 1}"]
    answer: 1
    why: "jsonb stores keys shorter first; the third form is what json_build_object would print."
  - q: "Every order has a customer_id that reports join on and that must reference customers. Where should it live?"
    options: ["In the jsonb document, with a GIN index", "In a column (or a STORED generated column) with NOT NULL and a foreign key", "In a VIRTUAL generated column with a foreign key"]
    answer: 1
    why: "Only a stored column can be NOT NULL, indexed and referenced by a foreign key; PG 18 rejects foreign keys and indexes on virtual generated columns."
further_reading:
  - title: "PostgreSQL docs: jsonb Indexing"
    url: "https://www.postgresql.org/docs/current/datatype-json.html#JSON-INDEXING"
  - title: "PostgreSQL docs: Indexes on Expressions"
    url: "https://www.postgresql.org/docs/current/indexes-expressional.html"
  - title: "PostgreSQL docs: JSON Creation Functions"
    url: "https://www.postgresql.org/docs/current/functions-json.html#FUNCTIONS-JSON-CREATION-TABLE"
  - title: "PostgreSQL docs: Generated Columns"
    url: "https://www.postgresql.org/docs/current/ddl-generated-columns.html"
  - title: "PostgreSQL docs: CREATE STATISTICS (statistics on a single expression)"
    url: "https://www.postgresql.org/docs/current/sql-createstatistics.html"
  - title: "MDN: Number.MAX_SAFE_INTEGER"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER"
---

## Concept

Three decisions come with every `jsonb` column.

**How to index it.** The GIN lesson covered the whole-document index: `jsonb_ops` (all
keys and values, serves `@>`, `?`, `?|`, `?&`, `@?`, `@@`) and the smaller
`jsonb_path_ops` (`@>`, `@?`, `@@`). GIN finds documents that *contain* something. It
can't answer ranges or sorting, so a key used with `<`, `BETWEEN` or `ORDER BY` needs a
**B-tree expression index** on exactly the expression the query writes, such as
`((doc ->> 'priority')::int)`. An expression index also gives the planner statistics
for that expression after `ANALYZE`; `CREATE STATISTICS ... ON (expression)` (PG 14)
gives the statistics without the index.

**How to build JSON for clients.** `json_build_object`/`jsonb_build_object`, `to_jsonb`
on a row, and the aggregates from the aggregation lesson produce the response document
in SQL. `json` output keeps your key order; `jsonb` output is canonical: keys shorter
first, one space after the colon. Dates become ISO 8601 strings, numbers stay exact
numerics, which JavaScript can't always represent.

**When a column wins.** If every row has the key and queries filter, join, sort or
aggregate on it, or it must be `NOT NULL`, unique or a foreign key, it should be a
column. A `STORED` generated column extracts it from the document on every write, so
ingested payloads stay intact and the column can't drift from them.

## Coming from C/C++/Java/Python/JS

- **JS/TS:** `JSON.parse` turns every number into a double. Ids above
  `Number.MAX_SAFE_INTEGER` (2^53 - 1) come back rounded, without an error, so bigint ids
  travel as strings. On the way in, a jsonb column is `unknown` until zod has checked it;
  a promoted column is a type the database already checked.
- **Go:** `encoding/json` decodes a JSON number into `float64` when the target is `any`,
  with the same precision problem; a struct field of type `int64`, or the `,string` tag
  option, keeps it exact. pgx can scan a SQL-built JSON column straight into
  `json.RawMessage` and the handler can write it out unchanged.
- **Java:** a jsonb column is like a `Map<String, Object>` field in an entity: flexible,
  and invisible to every constraint and index you'd put on a real field. Promoting a key
  is the same refactoring as replacing a map entry with a typed field.
- **Document databases:** there you index paths inside documents. Postgres gives you
  that (GIN, expression indexes) and also lets you move a hot path into a typed,
  constrained column without leaving the table.

## How it works

The planner considers an expression index when the query contains the indexed
expression itself; an equivalent spelling doesn't count, as the last plan of example 1
shows: `(doc -> 'priority')::int` missed the index that `(doc ->> 'priority')::int` used.
In return, the docs point out, the search is as fast as on a plain column, because the
expression's values are stored in the index and not recomputed (they are computed on
every insert and non-HOT update instead). It also explains why the
generated column is the robust solution: every query writes `customer_id`, and there is
nothing to spell differently.

Without statistics on an expression, the planner falls back to fixed guesses. The
`CREATE STATISTICS` docs give 0.5 % selectivity for equality and range conditions in
that case. A key whose values are very skewed (one tenant with half the rows) is
exactly where that guess goes wrong, and where an expression index or a column, both of
which `ANALYZE` measures, fixes the plans.

## Design notes

Structured where it's stable, JSON where it isn't. Walk each key of a document through
three questions: does every row have it, do queries filter, join, sort or aggregate on
it, and does it need a constraint? Any yes makes it a candidate column. What stays in
the document is the long tail: optional attributes, integration-specific fields, the raw
payload kept for replay.

When the document is the record (an order received from a shop, a webhook you must be
able to replay), promote with a `STORED` generated column and leave the document
untouched. When the application owns the data, make it an ordinary column, backfill it,
and remove the key from the document, so there is one source of truth.

For responses, let SQL build the shape when the response is a straightforward
projection of the data: one round trip, correct escaping, and no N+1 queries. Choose
`json` when key order is part of the contract, list keys explicitly instead of
`to_jsonb(row)` for anything public, and send bigint ids as strings to JavaScript
clients.
