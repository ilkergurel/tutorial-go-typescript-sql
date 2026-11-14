---
id: sql.files-movement.copy
title: "COPY ... FROM/TO vs \\copy"
track: sql
module: files-movement
order: 1
level: intermediate
minutes: 14
since: pg15
prerequisites: [sql.modifying.insert-update, sql.basics.psql]
compare_with: [go.postgres.bulk, go.files-io.csv-json, ts.node.fs-streams, fullstack.tracelite.step-07-import]
tags: [copy, psql, csv, bulk-load, encoding, windows-1254, pgx]
summary: >
  COPY moves rows in bulk between a table or query and a stream. With a file
  name the server opens the file on its own disk, which takes a privileged role;
  with STDIN/STDOUT the data rides the client connection, and that is all psql's
  \copy and pgx's CopyFrom/CopyTo do.
examples:
  - id: server-side
    file: examples/01_server_side.sql
    title: "TO STDOUT goes to the client; a file name goes to the server's disk"
    runtime: sql
    check: exact
    annotations:
      - lines: "10"
        note: >
          The rows leave as COPY data messages after the statement, not as a
          result set. psql would print a header line and two CSV rows here;
          this runner, like a plain query call in any driver, never asks for
          them and keeps only the tag. Drivers have a separate API for it:
          pgx's PgConn().CopyTo, psycopg's cursor.copy(), PgJDBC's CopyManager.
      - lines: "13"
        note: >
          COPY (query) TO accepts a SELECT, VALUES, or INSERT/UPDATE/DELETE/MERGE
          with RETURNING. COPY table TO works only on plain tables and (since
          PG 18) populated materialized views; for a view or a partitioned
          table, wrap it in a SELECT.
      - lines: "17-20"
        note: >
          USAGE asks whether the role's privileges apply to you right now.
          The playground role has none of the three on purpose: the docs warn
          they "could be used to gain superuser-level access", and TO PROGRAM
          would let lesson SQL run shell commands on this machine.
      - lines: "22"
        note: >
          The privilege check comes first, so the path is never even looked at.
          With the role, the server would look for C:/Users/ayse/... on its own
          disk, which is your laptop only if the server runs there. psql's
          \copy requirements FROM 'C:/Users/ayse/Downloads/requirements.csv'
          WITH (FORMAT csv, HEADER) reads the file on the client and needs no
          role.
  - id: copy-in
    file: examples/02_copy_in.go
    title: "COPY ... FROM STDIN from Go: CSV, a BOM, and a windows-1254 file"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "26-34"
        note: >
          PgConn().CopyFrom sends the statement, then copies the io.Reader to
          the server in chunks; the server parses CSV, not the client. Any
          reader works: an os.File, an upload's request body, a gzip.Reader.
          pgx.CopyFrom (the Go/Postgres module) is the other API: Go values,
          binary format.
      - lines: "27"
        note: >
          HEADER MATCH (PG 15) checks that the header names the table's columns,
          in order, instead of skipping the first line unseen. A file whose
          columns were reordered in Excel then fails loudly instead of loading
          one column's values into another.
      - lines: "35-38"
        note: >
          Two CSV rules decide NULL vs empty: an unquoted empty field is NULL,
          a quoted one ("") is the empty string. The ';' inside quotes is data.
          The output shows REQ-2's title as 'Rapor; PDF', its priority as NULL,
          and REQ-3's title as ''.
      - lines: "39"
        note: >
          Excel's "CSV UTF-8" starts with a byte order mark, and COPY treats it
          as data: the error says got "code", expected "code" because an
          invisible U+FEFF precedes the first. HEADER true would discard it
          along with the header line; with no header it would end up inside
          REQ-4's code.
      - lines: "40-42"
        note: >
          0xDD is İ and 0xFE is ş in windows-1254, Turkish Excel's "CSV (Comma
          delimited)" encoding; neither starts valid UTF-8, so the load fails at
          the first one. ENCODING 'WIN1254' makes the server convert while it
          reads. Without the option COPY assumes client_encoding, which
          defaults to the database encoding: UTF8 here.
  - id: copy-out
    file: examples/03_copy_out.go
    title: "COPY ... TO STDOUT from Go: text, CSV, and a file Turkish Excel opens"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "29-33"
        note: >
          CopyTo writes each piece of COPY data to the io.Writer as it
          arrives, so memory stays flat whether the export to an
          http.ResponseWriter has three rows or fifty million. COPY has no
          ORDER BY of its own; the query form adds one.
      - lines: "34"
        note: >
          The text format, COPY's default: tab-separated, NULL as \N, and
          backslash escapes for tab, newline and backslash itself. It is
          unambiguous and what pg_dump uses, but no spreadsheet reads it.
      - lines: "35"
        note: >
          CSV quotes only what needs it: the field holding a tab and quotes,
          with the quotes doubled, and TC-3's empty string as "" so it reads
          back as '' rather than NULL. FORCE_QUOTE (col) or * quotes every
          non-NULL value if a consumer insists.
      - lines: "36-38"
        note: >
          For Turkish Excel: a BOM written by Go before the data, ';' as the
          delimiter and a decimal comma made in SQL. For a legacy consumer that
          wants windows-1254, drop the BOM and add ENCODING 'WIN1254'; the
          server then converts on the way out.
use_cases:
  - "Loading a CSV export from another system or from Excel: \\copy in psql, PgConn().CopyFrom in a Go service"
  - "Streaming a large CSV download straight from a query to an HTTP response with CopyTo"
  - "Seeding a database and moving tables between databases (COPY ... TO STDOUT piped into COPY ... FROM STDIN)"
  - "Server-side COPY from a file for a DBA on the database host, where the file already lives on the server"
strengths:
  - "One statement for the whole load, with no per-row parse, plan or round trip; the docs call it almost always faster than INSERT, even prepared and batched"
  - "The server parses CSV, quoting, NULLs and encodings itself, so a client can stream an uploaded file through without parsing it"
  - "Unlike MySQL's LOAD DATA LOCAL INFILE, client-side loading needs no server switch: STDIN is part of the protocol, and \\copy is just psql using it"
  - "COPY (query) TO exports any result, including joins and RETURNING output, not only whole tables"
weaknesses:
  - "All or nothing: by default one bad row aborts the whole load and reports only the first error"
  - "CSV in PostgreSQL's dialect only: one delimiter character, no decimal-comma numbers, no DD.MM.YYYY dates unless DateStyle says so"
  - "Server-side files need roles the docs warn can lead to superuser access, and the file has to be on the database host"
  - "Plain query APIs can't use it: the client needs the COPY sub-protocol (psql, pgx, psycopg, PgJDBC's CopyManager)"
pitfalls:
  - title: "COPY FROM 'C:\\...' typed into a GUI client or a Go program"
    detail: "The server opens that path, on the server's machine. Either it fails with \"permission denied to COPY from a file\" or, as a superuser, it reads whatever happens to be at that path on the server. Read the file on the client side instead."
    bad: "COPY requirements FROM 'C:/Users/ayse/Downloads/requirements.csv' WITH (FORMAT csv, HEADER);"
    good: "\\copy requirements FROM 'C:/Users/ayse/Downloads/requirements.csv' WITH (FORMAT csv, HEADER)"
  - title: "\\copy is one line"
    detail: "psql takes the entire rest of the line as \\copy's arguments, without variable interpolation. For a long export query, write the SQL COPY ... TO STDOUT over several lines and end it with \\g file."
    good: "COPY (SELECT code, title FROM requirements ORDER BY code)\n  TO STDOUT WITH (FORMAT csv, HEADER)\n\\g requirements.csv"
  - title: "Empty Excel cells arrive as NULL"
    detail: "Excel writes an empty cell as nothing between two delimiters, which CSV COPY reads as NULL, so a NOT NULL text column rejects the row. FORCE_NOT_NULL (column) reads unquoted empty values as empty strings instead."
  - title: "A windows-1254 file read as UTF-8"
    detail: "Excel's plain \"CSV\" save on Turkish Windows is windows-1254. COPY fails with invalid byte sequence for encoding \"UTF8\" at the first İ, ş or ğ; ENCODING 'WIN1254' fixes it without converting the file first."
  - title: "Granting pg_read_server_files to the application role"
    detail: "It lets the role read any file the PostgreSQL service account can read, including server configuration. Stream through the connection instead; the app never needs it."
quiz:
  - q: "You run psql on your laptop against a database server elsewhere and execute COPY t FROM '/home/me/data.csv' WITH (FORMAT csv). What happens?"
    options:
      - "psql reads /home/me/data.csv from the laptop and sends it"
      - "The server looks for /home/me/data.csv on its own filesystem, and only if your role may read server files"
      - "It fails because COPY needs STDIN"
    answer: 1
    why: "Files named in SQL COPY are opened by the server process. \\copy is the client-side variant."
  - q: "COPY ... FROM STDIN WITH (FORMAT csv) reads the line REQ-2;;\"\" with DELIMITER ';' into three text columns. What are the second and third values?"
    options: ["'' and ''", "NULL and ''", "NULL and NULL"]
    answer: 1
    why: "In CSV, an unquoted empty field is NULL and a quoted empty field is the empty string."
  - q: "Which of these needs no special role or superuser rights?"
    options:
      - "COPY t TO '/tmp/t.csv'"
      - "COPY t TO PROGRAM 'gzip > /tmp/t.csv.gz'"
      - "\\copy t TO 't.csv' WITH (FORMAT csv)"
    answer: 2
    why: "\\copy runs COPY t TO STDOUT and psql writes the file with your own OS permissions; the other two act on the server."
  - q: "A Turkish user's CSV fails with: invalid byte sequence for encoding \"UTF8\": 0xdd 0x7a. What's the cheapest correct fix?"
    options:
      - "Add ENCODING 'WIN1254' to the COPY options"
      - "Change the database encoding to WIN1254"
      - "Remove every non-ASCII character first"
    answer: 0
    why: "0xDD is İ in windows-1254. COPY converts from the ENCODING option to the database encoding while it reads."
further_reading:
  - title: "PostgreSQL docs: COPY"
    url: "https://www.postgresql.org/docs/current/sql-copy.html"
  - title: "PostgreSQL docs: psql (\\copy)"
    url: "https://www.postgresql.org/docs/current/app-psql.html"
  - title: "PostgreSQL docs: Predefined Roles"
    url: "https://www.postgresql.org/docs/current/predefined-roles.html"
  - title: "PostgreSQL docs: Populating a Database"
    url: "https://www.postgresql.org/docs/current/populate.html"
  - title: "PostgreSQL docs: Character Set Support"
    url: "https://www.postgresql.org/docs/current/multibyte.html"
  - title: "package pgconn: PgConn.CopyFrom and CopyTo"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgConn.CopyFrom"
---

## Concept

`COPY` moves many rows in one statement between a table (or, outbound, a query) and a
stream in one of three formats: `text` (the default: tab-separated, `\N` for NULL),
`csv`, or `binary`. Where the stream comes from or goes to is the part that matters:

- **`FROM STDIN` / `TO STDOUT`**: the data travels over your client connection. Any role
  that may `INSERT` into (or `SELECT` from) the table can do this.
- **`FROM 'file'` / `TO 'file'`**: the *server process* opens a path on the *server's*
  filesystem, as the operating-system account PostgreSQL runs under. That needs
  superuser or the predefined role `pg_read_server_files` / `pg_write_server_files`.
- **`PROGRAM 'command'`**: the server runs a shell command and reads or writes its
  pipe. Superuser or `pg_execute_server_program`.

psql's `\copy` is not a different command. It sends `COPY ... FROM STDIN` (or
`TO STDOUT`) and pipes a local file through the connection, with your own file
permissions. pgx's `PgConn().CopyFrom` and `CopyTo` do exactly the same with an
`io.Reader` or `io.Writer`, which is why the loading and exporting examples here are Go
programs: this app's SQL runner, like a plain query call in any driver, doesn't speak
the copy sub-protocol. It shows only the command tag for `TO STDOUT`, and a
`COPY ... FROM STDIN` in it would wait for data until the run times out.

The options you'll use most: `HEADER` (`true` skips the first line, `MATCH` checks it
against the column names, PG 15), `DELIMITER`, `NULL`, `ENCODING`, `FORCE_QUOTE`,
`FORCE_NULL` and `FORCE_NOT_NULL` (CSV only), a column list, and `WHERE` on
`COPY FROM` to filter rows on the way in. `ON_ERROR` comes in the next lesson.

## Coming from C/C++/Java/Python/JS

- **MySQL**: `LOAD DATA INFILE` vs `LOAD DATA LOCAL INFILE` is the same server-file vs
  client-file split as `COPY ... FROM 'file'` vs `\copy`. SQL Server has it as
  `BULK INSERT` (server) vs the `bcp` tool (client).
- **Java**: PgJDBC's `CopyManager.copyIn(sql, reader)` is `PgConn().CopyFrom`; JDBC's
  `addBatch` is still one INSERT per row.
- **Python**: psycopg 3's `with cur.copy("COPY t FROM STDIN") as copy: copy.write(...)`
  is the same conversation; `executemany` is not.
- **C**: libpq's `PQputCopyData` and `PQgetCopyData` are the C API for it.
- **Node**: `pg-copy-streams` turns COPY into a Node stream you can `pipeline()` a file
  or an upload into.

## How it works

`COPY ... FROM STDIN` is a short conversation inside the normal protocol. The client
sends the statement; the server answers "send me copy data" and waits; the client sends
the bytes in chunks of any size (they don't have to align with rows), then "done" or
"fail"; the server answers with the tag, `COPY 3`. `TO STDOUT` is the mirror image.
A client that sends the statement but doesn't take part gets only the tag, or, for
`FROM STDIN`, a server waiting for data that never comes.

Parsing happens on the server, after encoding conversion: input is read in the
`ENCODING` option's character set, or the session's `client_encoding` without it, and
converted to the database encoding, even when the server reads the file itself. That
is why a windows-1254 file fails under `client_encoding` UTF8 (the default in a UTF-8
database), and why one option fixes it.

Rows are inserted as they are parsed, with the table's constraints and triggers applied.
If row 90,001 fails, the statement fails and the 90,000 rows before it stay behind as
dead rows that take space until `VACUUM`. `binary` is the fastest format and the one
pgx's `CopyFrom` uses, but the docs call it less portable across architectures and
PostgreSQL versions, and it can't read into a column of a different type.

## Design notes

Decide by where the file lives. On your machine: `\copy`, or a driver's
`COPY ... FROM STDIN`. Already on the database server, and you're the DBA:
server-side `COPY` saves a network hop, at the price of a role the docs say "could be
used to gain superuser-level access". Application roles never need it, and code that
streams through the connection works wherever the database runs.

In a Go service, pick the API by what you hold. Rows already parsed into Go values:
`pgx.CopyFrom` with binary encoding. A CSV upload you trust to be well formed:
`PgConn().CopyFrom(ctx, r.Body, ...)` streams it through without parsing it in Go.
Exports: `CopyTo(ctx, w, "COPY (query) TO STDOUT WITH (FORMAT csv, HEADER)")` into an
`http.ResponseWriter` is a streaming download in a few lines.

Don't COPY files from people straight into the real table. One malformed row aborts the
load and reports a single error, and the format rules (decimal commas, DD.MM.YYYY,
`Geçti` vs `KALDI`) are richer than any COPY option. Load into a staging table and
validate there, which is the next lesson. For speed, the docs' advice for big loads:
create indexes after loading a fresh table, consider dropping foreign keys during huge
loads, and `ANALYZE` afterwards.

The Excel side was observed, not documented: on 2026-11-14, Excel for Microsoft 365 (build
16.0.20527.20196), with its decimal separator set to `,` as on Turkish Windows, wrote "CSV
UTF-8" starting with `EF BB BF`, wrote an empty cell as nothing between two separators
(`Kaya, Ltd.;;99,9`), and wrote "CSV (Comma delimited)" in the Windows ANSI code page,
windows-1252 on that en-US machine. Turkish Windows's ANSI code page is 1254
([code page identifiers](https://learn.microsoft.com/en-us/windows/win32/intl/code-page-identifiers)).
The PostgreSQL side is in the COPY docs (with the default settings a NULL "is written as an
unquoted empty string", and "Reading values follows similar rules"), and the examples show
that rule, the BOM kept as data and ENCODING 'WIN1254' at work on 18.7.

