---
id: go.postgres.querying
title: "Querying and scanning: CollectRows, RowToStructByName, NULL handling"
track: go
module: postgres
order: 2
level: intermediate
minutes: 13
since: go1.25
prerequisites: [go.postgres.pgx, go.composite.structs]
compare_with: [sql.querying.null-logic, ts.everyday-types.null, go.errors.is-as-join]
tags: [postgres, pgx, scanning, nullability, generics]
summary: >
  pgx.CollectRows with RowToStructByName turns rows into typed structs and does
  the close-and-check bookkeeping for you. Give every nullable column a type that
  can say NULL (*string, pgtype.Text), because pgx refuses to scan NULL into a zero value.
examples:
  - id: collect-rows
    file: examples/01_collect_rows.go
    title: CollectRows, RowTo and CollectOneRow, and how strict name mapping is
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-19"
        note: >
          Fields match columns by name, ignoring case and, for untagged fields,
          underscores (pgx 5.5.2+), so BirthYear finds birth_year. A db:"col"
          tag pins the name, db:"-" skips the field, unexported fields are
          ignored.
      - lines: "29-33"
        note: >
          Exec with no arguments uses PostgreSQL's simple protocol, which takes
          several statements in one string: handy for setup. With arguments,
          pgx sends one statement and the values separately, which is what
          makes parameters injection-proof (next lesson).
      - lines: "38-41"
        note: >
          CollectRows loops, scans, closes the rows and checks rows.Err(): the
          four steps a hand-written loop gets wrong. Ignoring Query's error is
          the documented style here, because CollectRows returns the same error.
      - lines: "43-45"
        note: >
          RowTo[T] covers single-column results. CollectRows always returns a
          non-nil slice, so zero rows encode to JSON as [] rather than null.
      - lines: "47-49"
        note: >
          No row is an error, not a zero User. pgx.ErrNoRows wraps
          sql.ErrNoRows (pgx 5.7+), so either check works. Translate it to your
          own ErrNotFound at the repository boundary, as the errors module did
          with *pgconn.PgError.
      - lines: "51-57"
        note: >
          Name mapping is strict: a column without a field fails, and so does a
          field without a column. With SELECT *, an ALTER TABLE ... ADD COLUMN
          breaks binaries already deployed. RowToStructByNameLax only forgives
          missing columns, not extra ones; list the columns.
  - id: nulls
    file: examples/02_nulls.go
    title: NULL in, NULL out, and the aggregate that returns NULL
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-18"
        note: >
          Nullability goes into the type. *string reads naturally and encodes
          to JSON null; pgtype.Text is what sqlc generates for pgx.
          sql.NullString and sql.Null[string] (Go 1.22) work too, because pgx
          accepts any sql.Scanner.
      - lines: "34-36"
        note: >
          A nil pointer, an untyped nil or pgtype.Text{} is sent as NULL. The
          empty string is a value: Gül is known to have no phone, Ömer's is
          unknown, and the database keeps the difference.
      - lines: "41-43"
        note: >
          NULL into a plain string is an error that names the column. pgx won't
          invent "" for you; JDBC's getString would have handed you null, and
          getInt a silent 0.
      - lines: "45-56"
        note: >
          The two rows keep "" and NULL apart. A NULL pgtype.Text still has
          String == "", so check Valid first; with *string, forgetting the nil
          check panics on dereference instead of printing a fake value.
      - lines: "58-63"
        note: >
          Except for count, aggregates over zero rows return NULL: sum of
          nothing is NULL, not 0. Here coalesce is honest, because no orders
          really means a quantity of 0.
use_cases:
  - "CollectRows + RowToStructByName: list endpoints, reports, anything that maps rows onto a struct"
  - "RowTo[T]: single-column results such as ids or names"
  - "QueryRow / CollectOneRow: lookups by key, where no row becomes a domain-level not found"
  - "ForEachRow: large results processed row by row without building a slice"
strengths:
  - "Unlike a JDBC ResultSet loop, CollectRows closes the rows and checks the error in one call, so the two classic mistakes can't happen"
  - "Mapping mistakes are loud: an extra or missing column is an error, unlike encoding/json, which silently leaves unknown fields at their zero value"
  - "NULL cannot become a zero value by accident: scanning it into string or int64 fails and names the column"
  - "Generics give typed results ([]User) without an ORM or code generation"
weaknesses:
  - "Names are matched at run time: a typo in a db tag or a column alias is a runtime error, which sqlc (lesson 5) moves to generation time"
  - "RowToStructByName matches columns by reflection (cached per struct and column list); a hand-written Scan, which sqlc generates, skips even that"
  - "Three spellings of a nullable value (*T, pgtype.T, sql.Null[T]) invite a codebase that mixes all of them"
pitfalls:
  - title: "SELECT * into a struct"
    detail: "Adding a column to the table breaks every RowToStructByName over SELECT * until the struct catches up. List the columns you map."
    bad: |
      rows, _ := conn.Query(ctx, "SELECT * FROM users")
      users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
    good: |
      rows, _ := conn.Query(ctx, "SELECT id, name, city, birth_year FROM users")
      users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
  - title: "coalesce to make NULL go away"
    detail: "coalesce(phone, '') merges \"unknown\" with \"known to be empty\". Use it where NULL really means zero (an aggregate over no rows); elsewhere scan into a nullable type."
  - title: "Reading pgtype.Text.String without Valid"
    detail: "A NULL pgtype.Text has String == \"\". Check Valid first, or use *string so a NULL can't pass for a value."
  - title: "A hand-written loop without rows.Err()"
    detail: "Next returns false at the end and on a network error alike. Without the Err check, a failure halfway through looks like a short result."
    bad: |
      for rows.Next() {
          var u User
          if err := rows.Scan(&u.ID, &u.Name); err != nil {
              return nil, err
          }
          users = append(users, u)
      }
      return users, nil
    good: |
      users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
  - title: "Building an IN list by hand"
    detail: "Pass a Go slice as one parameter and compare with = any($1). IN ($1) compares against the whole array, and string-joining ids into the SQL is the injection the next lesson is about."
    bad: |
      q := "SELECT name FROM users WHERE id IN (" + strings.Join(ids, ",") + ")"
    good: |
      rows, _ := conn.Query(ctx, "SELECT name FROM users WHERE id = any($1)", ids)
quiz:
  - q: "A query uses SELECT * with RowToStructByName[User]. A colleague adds a nullable column to the table. What happens to the binary already deployed?"
    options: ["The new column is ignored", "Scanning fails: the struct has no field for the new column", "The column lands in the nearest matching field"]
    answer: 1
    why: "Every column must have a field (and every field a column); extra columns are an error even with the Lax variant."
  - q: "Which Go type honestly represents a nullable text column?"
    options: ["string: NULL becomes \"\"", "*string or pgtype.Text", "[]byte"]
    answer: 1
    why: "Both can say NULL. Scanning NULL into string is an error in pgx, and \"\" would conflate unknown with empty anyway."
  - q: "SELECT max(created_at) FROM orders WHERE customer_id = $1, for a customer with no orders, scanned into a time.Time. What happens?"
    options: ["The zero time.Time", "An error: cannot scan NULL into *time.Time", "pgx.ErrNoRows"]
    answer: 1
    why: "An aggregate without GROUP BY always returns one row; over zero input rows its value is NULL. Scan into *time.Time or use coalesce."
  - q: "CollectOneRow found no rows. Which checks are true?"
    options: ["Only errors.Is(err, pgx.ErrNoRows)", "Both errors.Is(err, pgx.ErrNoRows) and errors.Is(err, sql.ErrNoRows)", "err is nil and the value is the zero struct"]
    answer: 1
    why: "Since pgx 5.7, pgx.ErrNoRows unwraps to sql.ErrNoRows."
further_reading:
  - title: "package pgx: CollectRows, RowToStructByName, ForEachRow"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#CollectRows"
  - title: "package pgtype: PostgreSQL types and their Go mappings"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype"
  - title: "package database/sql: Null[T]"
    url: "https://pkg.go.dev/database/sql#Null"
  - title: "PostgreSQL: Aggregate functions (NULL over no rows)"
    url: "https://www.postgresql.org/docs/current/functions-aggregate.html"
---

## Concept

`Query` returns `pgx.Rows`, a cursor over the result that holds a connection until it is
closed. Don't loop over it by hand; hand it to a collector together with a function that
turns one row into a value:

- `pgx.CollectRows(rows, fn)` returns `[]T`; `CollectOneRow` returns the first `T` or
  `pgx.ErrNoRows`; `CollectExactlyOneRow` also fails with `ErrTooManyRows`.
- `fn` is usually `pgx.RowTo[T]` (one column), `pgx.RowToStructByName[T]` (columns to
  fields by name) or `RowToStructByPos[T]` (by position). The `RowToAddrOf...` variants
  return `*T`.
- `ForEachRow` scans into variables you provide and calls a function per row: no slice.

All of them close the rows and check `rows.Err()`. `QueryRow(...).Scan(...)` remains the
shortest way to read one row into variables.

Go has no null string or null int, so a nullable column needs a type that can hold
NULL: a pointer (`*string`, nil for NULL), a `pgtype` value (`pgtype.Text`, with `Valid`),
or `sql.Null[T]`. Scanning NULL into a plain `string` or `int64` is an error, which is the
point: the struct you scan into should say what the schema says.

## Coming from C/C++/Java/Python/JS

- Java: JDBC's `getInt` returns 0 for NULL unless you remember `wasNull()`; Go makes you
  pick a NULL-capable type up front, and fails loudly if you didn't. `CollectRows` with
  `RowToStructByName` is roughly Spring's `BeanPropertyRowMapper` without the framework.
- Python: psycopg hands back `None` for NULL in any column. Go strings can't be `nil`, so
  `*string` or `pgtype.Text` plays `None`'s role.
- JS/TS: node-postgres puts `null` in the row object and TypeScript believes whatever type
  you declared. pgx checks every value as it scans.
- C: libpq's `PQgetisnull` is the `Valid` flag of `pgtype.Text`, minus the struct.

## Design notes

Treat the struct as the contract with the schema: `NOT NULL` columns get plain types,
nullable ones get `*T` (or `pgtype.T` if you follow sqlc's output), and the query lists its
columns. Pick one nullable spelling per codebase; pointers are plain Go and encode to JSON
`null`, as does `pgtype.Text`.

`coalesce` is for places where NULL really means zero, such as the sum of no rows.
Turning an unknown phone number into `""` loses information you can't get back.

Keep `pgx.ErrNoRows` inside the repository: return your own `ErrNotFound` (wrapped with
`%w`) so callers never import pgx, which is the translation rule from the errors module.

Text comparison is exact: `WHERE city = $1` matches `İzmir` byte for byte. For
case-insensitive Turkish matching, remember this database's builtin `C.UTF-8` locale
lowercases `I` to `i`, never `ı`; Turkish rules need an explicit ICU collation such as
`lower(city COLLATE "tr-x-icu")`, which the `sql.search` module covers.
