---
id: sql.modifying.insert-update
title: "Multi-row INSERT, INSERT ... SELECT, UPDATE ... FROM, DELETE ... USING"
track: sql
module: modifying
order: 1
level: basic
minutes: 14
since: pg10
prerequisites: [sql.querying.select, sql.basics.set-based]
compare_with: [go.postgres.bulk, sql.files-movement.csv-staging, sql.subqueries-ctes.modifying]
tags: [insert, update, delete, unnest, bulk]
summary: >
  Write sets, not rows: many rows per INSERT, rows produced by a query, and
  UPDATE and DELETE driven by a join. One statement, one round trip, all rows
  or none.
examples:
  - id: multi-row
    file: examples/01_multi_row.sql
    title: "Many rows per INSERT: VALUES lists and unnest over arrays"
    runtime: sql
    check: exact
    annotations:
      - lines: "11-15"
        note: >
          Three rows, one statement: one round trip, one parse, and all or
          nothing, since a CHECK failure in any row inserts none of them. The
          tag reads INSERT 0 3; the 0 is a leftover OID field and is always 0
          now.
      - lines: "18-20"
        note: >
          DEFAULT works per value, so one statement can mix explicit values
          and column defaults. Columns left out of the list get their default
          (or NULL) too.
      - lines: "24-28"
        note: >
          unnest with several arrays in FROM zips them into rows. From Go you
          pass three slices as parameters, so the SQL text is the same for 3
          rows or 3,000: nothing to build, nothing to inject. Arrays of
          different lengths are padded with NULLs, so check lengths first.
  - id: insert-select
    file: examples/02_insert_select.sql
    title: "INSERT ... SELECT: convert and load from a staging table"
    runtime: sql
    check: exact
    annotations:
      - lines: "11-15"
        note: >
          Load raw text first, convert in SQL second. A bad row then fails in
          a query you can inspect and rerun, not halfway through a load.
      - lines: "19-25"
        note: >
          The rows to insert are a query's result: zero, two or a million of
          them, inserted by one statement. The tag counts what the query
          produced (INSERT 0 2), not what staging held.
      - lines: "22"
        note: >
          Turkish number format: drop the thousands dot, then turn the decimal
          comma into a point, before casting. Cast first and '1.250,00' fails;
          worse, a thousands-only value such as '1.250' would cast happily to
          1.25.
      - lines: "25"
        note: >
          NOT EXISTS skips skus already on sale, so rerunning the load adds
          nothing. Two loads running at the same moment could still both
          insert; ON CONFLICT (next lesson) closes that gap.
  - id: update-delete-join
    file: examples/03_update_delete_join.sql
    title: "UPDATE ... FROM, DELETE ... USING, and the subquery trap"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          The target is joined to price_list, and each joined row becomes one
          update. XYZ-99 matches no product and changes nothing. UPDATE 2
          counts target rows.
      - lines: "16-17"
        note: >
          The join must give at most one source row per target row. With two
          price_list rows for TEA-01, only one would be applied, and the docs
          say which one is not predictable. Here sku is the primary key.
      - lines: "20-22"
        note: >
          USING is DELETE's FROM: join to the list, delete what matches. WHERE
          p.sku IN (SELECT sku FROM discontinued) is the standard spelling
          and equally fine here.
      - lines: "28-29"
        note: >
          A SET with a subquery and no WHERE updates every row; where the
          subquery finds nothing, the value is NULL. BAK-01 has no new price,
          and only NOT NULL stops it from losing its price. Add WHERE EXISTS
          (...) or use FROM.
use_cases:
  - "Inserting a batch from Go or TS in one statement: unnest over one array parameter per column"
  - "Imports: COPY or INSERT raw rows into a staging table, then INSERT ... SELECT with conversion and filtering"
  - "Applying a list to a table: new prices with UPDATE ... FROM, removals with DELETE ... USING"
strengths:
  - "One statement per batch is one round trip and atomic, unlike a loop of single-row INSERTs that leaves half the rows written if it fails outside a transaction"
  - "A join in UPDATE or DELETE states \"apply this list\" directly, where an ORM typically loads each row into memory and saves it back"
  - "Array parameters keep the SQL text constant whatever the batch size, so the driver can reuse one prepared statement"
weaknesses:
  - "UPDATE ... FROM and DELETE ... USING are PostgreSQL extensions; the portable subquery forms are wordier and easy to get wrong"
  - "One huge statement holds its row locks until commit and makes one big transaction; very large backfills need batching"
  - "Binding one parameter per value hits the protocol's 16-bit parameter count on big multi-row VALUES; arrays need one parameter per column"
pitfalls:
  - title: "A source with duplicate keys in UPDATE ... FROM"
    detail: "Each target row is updated at most once, with one of its matching source rows chosen unpredictably. Make the source unique first (a key, GROUP BY, or DISTINCT ON)."
  - title: "Forgetting the join condition"
    detail: "UPDATE ... FROM without a WHERE linking the tables joins every target row to every source row: all rows are updated, each with an arbitrary source row."
    bad: "UPDATE products p SET price = l.new_price FROM price_list l;"
    good: "UPDATE products p SET price = l.new_price FROM price_list l WHERE l.sku = p.sku;"
  - title: "Running a big UPDATE or DELETE blind"
    detail: "psql has no undo. Run it inside BEGIN, check the count in the command tag (UPDATE 3, not UPDATE 30000), then COMMIT or ROLLBACK."
  - title: "Building multi-row INSERT text in code"
    detail: "Concatenating values into SQL invites injection and produces a new statement text per batch size. Pass arrays to unnest, or use COPY (pgx's CopyFrom) for very large loads."
quiz:
  - q: "price_list has two rows for TEA-01 (235 and 240). What does UPDATE products p SET price = l.new_price FROM price_list l WHERE l.sku = p.sku do to TEA-01?"
    options:
      - "Fails with a duplicate error"
      - "Applies both, so the last one wins"
      - "Applies one of them; which one is not predictable"
    answer: 2
    why: "A target row is updated at most once per statement, and the docs say which joined row gets used is not readily predictable."
  - q: "Which statement sets price to NULL for every product missing from price_list?"
    options:
      - "UPDATE products p SET price = l.new_price FROM price_list l WHERE l.sku = p.sku"
      - "UPDATE products SET price = (SELECT l.new_price FROM price_list l WHERE l.sku = products.sku)"
      - "DELETE FROM products p USING price_list l WHERE l.sku = p.sku"
    answer: 1
    why: "Without a WHERE, the subquery form updates every row, and a subquery that finds nothing yields NULL."
  - q: "INSERT INTO t (a) SELECT ... reports INSERT 0 2. What is the 0?"
    options:
      - "The number of rows that failed"
      - "A legacy OID field, always 0 today"
      - "The id of the first inserted row"
    answer: 1
    why: "The tag is INSERT oid count; tables WITH OIDS are gone, so oid is always 0."
  - q: "A Go service must insert 5,000 rows in one statement without building SQL text. What fits?"
    options:
      - "INSERT ... SELECT * FROM unnest($1::text[], $2::numeric[])"
      - "A loop of 5,000 single-row INSERTs inside one transaction"
      - "fmt.Sprintf one VALUES list with all the rows"
    answer: 0
    why: "One parameter per column carries every row; the text never changes. The loop is 5,000 round trips; Sprintf invites injection."
further_reading:
  - title: "PostgreSQL docs: INSERT"
    url: "https://www.postgresql.org/docs/current/sql-insert.html"
  - title: "PostgreSQL docs: UPDATE"
    url: "https://www.postgresql.org/docs/current/sql-update.html"
  - title: "PostgreSQL docs: DELETE"
    url: "https://www.postgresql.org/docs/current/sql-delete.html"
  - title: "PostgreSQL docs: Array Functions (unnest)"
    url: "https://www.postgresql.org/docs/current/functions-array.html"
---

## Concept

`INSERT`, `UPDATE` and `DELETE` take sets, like `SELECT` returns them:

- **Multi-row `INSERT ... VALUES`**: many rows in one statement.
- **`INSERT ... SELECT`**: the rows to insert are the result of a query (a staging
  table, another table, `unnest` over arrays sent by the app).
- **`UPDATE ... FROM`**: join the target to other tables; each target row that joins
  is updated using the joined values.
- **`DELETE ... USING`**: the same join, deciding what to delete.

Each is one statement: atomic, one round trip, and reported by a command tag with a
count (`INSERT 0 3`, `UPDATE 2`, `DELETE 2`), which drivers expose as rows affected
(`CommandTag.RowsAffected()` in pgx).

## Coming from C/C++/Java/Python/JS

- **ORMs** (JPA, Django, Prisma): `for p in products: p.price = new[p.sku]; p.save()` is
  N selects and N updates. `UPDATE ... FROM` is the same intent in one statement, and
  the database does the matching.
- **JDBC's `addBatch()`**: batching saves round trips but still executes the statement
  N times. `unnest` over arrays is one execution with one parameter per column.
- **Go**: `for _, it := range items { tx.Exec(...) }` is the habit to break. Pass one
  slice per column (`[]string` for the skus, and so on) and let `unnest` turn them into
  rows. For tens of thousands of rows and more, pgx's `CopyFrom` (the Go/Postgres
  module) is faster still.
- **SQL Server and MySQL** also allow joins in `UPDATE` and `DELETE`, each with its own
  syntax. PostgreSQL's is `FROM` for `UPDATE` and `USING` for `DELETE`; neither is
  standard SQL.

## Design notes

Prefer a two-step shape for anything that comes from outside: land raw rows in a staging
table, then move them with one `INSERT ... SELECT` (or `UPDATE ... FROM`) that converts,
validates and filters. The conversion lives in a query you can rerun and inspect instead
of in a loop. Keep the source side of every join unique on the join key, because
PostgreSQL won't stop you when it isn't. When an insert must also cope with rows that
already exist, the next lesson replaces `NOT EXISTS` with `ON CONFLICT`, which is safe
under concurrency.
