---
id: sql.basics.psql
title: "psql essentials: \\d, \\dt, \\x, \\timing, \\e"
track: sql
module: basics
order: 2
level: basic
minutes: 13
since: pg10
prerequisites: [sql.basics.mental-model]
compare_with: [go.tooling.go-command, ts.node.cli-server, sql.security-ops.monitoring]
tags: [psql, catalog, information-schema, tooling]
summary: >
  psql's backslash commands are answered by psql itself, mostly by sending
  catalog queries it writes for you. Learn the dozen you'll use daily, and the
  SQL behind \d, which you can reuse anywhere psql isn't.
examples:
  - id: list
    file: examples/01_list.sql
    title: "\\d without a pattern, as the SQL psql sends"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-10"
        note: >
          The identity columns bring sequences with them, and \d lists them
          next to the tables: GENERATED ... AS IDENTITY is backed by a
          sequence the table owns.
      - lines: "12"
        note: >
          COMMENT ON stores documentation in the catalog. \dt+ and \d+ show it
          in their Description column, so a colleague browsing in psql sees
          it without opening your repository.
      - lines: "16-30"
        note: >
          The core of what psql runs for \d, trimmed. Start psql with -E (or
          \set ECHO_HIDDEN on) and it prints the full query before every
          meta-command: the quickest way to learn the catalogs.
      - lines: "29"
        note: >
          pg_table_is_visible is true when the table's schema is on the
          search path and no earlier schema has the same name. It is why \dt
          lists nothing for tables in schemas outside your path; \dt *.*
          shows every schema.
  - id: describe
    file: examples/02_describe.sql
    title: "\\d orders: columns, constraints, and the portable way"
    runtime: sql
    check: exact
    annotations:
      - lines: "10-17"
        note: >
          format_type prints the names psql shows, with modifiers:
          character varying(80), timestamp with time zone. pg_get_expr turns
          a stored default back into SQL text. Identity columns have no
          default expression; their attidentity flag says how values arrive.
      - lines: "21"
        note: >
          Positive attnum skips system columns (ctid, xmin, ...). A dropped
          column only becomes invisible; its slot stays in pg_attribute,
          flagged attisdropped, until the table is rewritten.
      - lines: "26-29"
        note: >
          pg_get_constraintdef reconstructs the definition rather than
          returning what you typed, hence (total >= (0)::numeric). Since
          PG 18 NOT NULL is stored here too, as contype 'n' constraints with
          their own names.
      - lines: "33-36"
        note: >
          information_schema is the SQL standard's catalog: the same query
          works on other databases. The price: data_type drops the (80) and
          (10,2) into separate columns, and every Postgres-specific type
          collapses to ARRAY or USER-DEFINED.
  - id: names
    file: examples/03_names.sql
    title: "Stored names: folding, quotes, and a Turkish letter"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          Unquoted names fold to lower case; quoted ones are stored exactly.
          The fold covers ASCII letters only: in this UTF-8 database Şubeler
          keeps its capital Ş, so şubeler (with ş) is a different name.
      - lines: "7-8"
        note: >
          Identifiers are limited to 63 bytes, not characters, and each
          Turkish letter outside ASCII takes two bytes in UTF-8. This
          55-character name is one byte too long, so the server cuts it and
          says so only in a NOTICE.
      - lines: "10-11"
        note: >
          length counts characters, octet_length bytes: 7 and 8 for Şubeler.
          The long name came back at 63 bytes, without its final m.
      - lines: "18-22"
        note: >
          to_regclass parses its argument as SQL would, folding PAYMENTS to
          payments and keeping "Invoices" as written, and returns NULL
          instead of raising an error when nothing matches.
      - lines: "25"
        note: >
          Unquoted, Invoices means invoices, which doesn't exist. psql patterns
          follow the same rule: \dt Invoices finds nothing; \dt "Invoices"
          finds the table.
use_cases:
  - "Inspecting a server you reached over SSH, where there is no GUI: \\dn, \\dt app.*, \\d app.orders"
  - "Running migration and seed scripts: psql -v ON_ERROR_STOP=1 -f migrate.sql"
  - "Learning the catalogs: psql -E prints the SQL behind every meta-command, ready to reuse in Go or TS tooling"
strengths:
  - "Ships with every PostgreSQL install and needs nothing but a terminal, unlike GUI clients"
  - "Meta-commands are thin wrappers over catalog queries you can see (-E) and copy, not a black box"
  - "\\x auto switches to one-field-per-line only when a row is wider than the screen; \\gx does it for a single query"
weaknesses:
  - "Meta-commands exist only in psql: pgx, node-postgres and every GUI need the catalog queries instead"
  - "Its output is formatted for humans, so scripts should query the catalogs or information_schema rather than parse \\d"
  - "On Windows, the console code page can garble Turkish characters until it matches what psql expects"
pitfalls:
  - title: "\\dt shows nothing, yet the app works"
    detail: "Without a pattern, \\d commands list only objects visible on your search_path. The app's tables are probably in another schema: \\dn lists schemas, \\dt app.* or \\dt *.* finds them."
  - title: "NULL and the empty string look identical"
    detail: "By default psql prints nothing for NULL. Set a visible marker once in your psqlrc."
    good: "\\pset null '(null)'"
  - title: "A missing semicolon"
    detail: "SQL is sent only at ';' (or \\g). Until then psql keeps buffering, and the prompt changes from playground=> to playground-> to say so. Backslash commands, by contrast, end at the newline."
  - title: "Turkish characters garbled in the Windows console"
    detail: "psql warns at startup when it detects a problematic console code page. The docs' fix: switch the console to your Windows code page (cmd.exe /c chcp 1254 on Turkish Windows, 1252 on Western European ones) and set the console font to Lucida Console."
quiz:
  - q: "\\dt lists no tables, but the application reads and writes them fine. What's the most likely reason?"
    options:
      - "psql lacks privileges to see them"
      - "They live in a schema that isn't on your search_path; try \\dt *.*"
      - "\\dt only shows tables created in the current session"
    answer: 1
    why: "Without a pattern, \\d commands show only objects visible on the search path. *.* matches every schema."
  - q: "What does starting psql with -E do?"
    options:
      - "Runs every statement in its own transaction"
      - "Prints the catalog queries that backslash commands send"
      - "Turns on expanded output"
    answer: 1
    why: "-E is the same as \\set ECHO_HIDDEN on: psql shows each hidden query before running it. Expanded output is -x."
  - q: "After CREATE TABLE \"Invoices\" (id int), which psql command lists it?"
    options:
      - "\\dt Invoices"
      - "\\dt invoices"
      - "\\dt \"Invoices\""
    answer: 2
    why: "Patterns fold to lower case like unquoted SQL names; double quotes stop the folding."
  - q: "Why can't your Go code send \\d orders through pgx?"
    options:
      - "pgx strips backslashes from queries"
      - "Meta-commands are interpreted by psql and never reach the server; send the catalog query instead"
      - "It needs superuser rights"
    answer: 1
    why: "The server only understands SQL. psql turns \\d into catalog queries before anything is sent."
further_reading:
  - title: "PostgreSQL docs: psql"
    url: "https://www.postgresql.org/docs/current/app-psql.html"
  - title: "PostgreSQL docs: System Catalogs"
    url: "https://www.postgresql.org/docs/current/catalogs.html"
  - title: "PostgreSQL docs: The Information Schema"
    url: "https://www.postgresql.org/docs/current/information-schema.html"
  - title: "PostgreSQL docs: System Information Functions"
    url: "https://www.postgresql.org/docs/current/functions-info.html"
---

## Concept

psql reads two kinds of input. SQL is buffered until a `;` and then sent to the server.
Backslash commands (meta-commands) end at the newline and are handled by psql itself;
many of them are just catalog queries psql writes for you. That is why StackLab's runner
can't execute `\d`: it only speaks SQL to the server, as your Go code does. The examples
run the SQL those commands stand for instead.

The working set:

| Command | What it does |
|---|---|
| `\l`, `\c db` | list databases; reconnect to another one (a new connection) |
| `\dn`, `\dt`, `\dv`, `\di`, `\df`, `\du` | list schemas, tables, views, indexes, functions, roles |
| `\d name`, `\d+ name` | describe one object; `+` adds sizes, comments and more |
| `\x`, `\x auto`, `\gx` | expanded output (one field per line): toggle, only when too wide, or for one query |
| `\timing` | toggle printing how long each statement took |
| `\e`, `\ef fn` | edit the query buffer (or a function) in your editor |
| `\i file.sql` | run a file as if you'd typed it |
| `\?`, `\h CREATE TABLE` | help for meta-commands; SQL syntax help |

Patterns work on every `\d` command: `\dt app.*` lists tables in schema `app`, `\dt *.*`
lists tables in all schemas, and an `S` (`\dtS`) includes system objects.

## Coming from C/C++/Java/Python/JS

- **Python's or Node's REPL**: psql is that for SQL, with `\d` in place of `dir()` and
  `help()`, and `\i file.sql` to load a script into the session.
- **The `go` command**: psql ships with the server the way `go` ships with Go. GUI
  clients are optional; psql is always there, including on a server you reach over SSH.
- **Reflection** (Java's `Class.getDeclaredFields`, Go's `reflect`): the system catalogs
  are the database describing itself in tables, and `\d` is a pretty-printer over them.
  `information_schema` is the portable, standardized layer on top, with less detail.
- **Windows**: psql is a console program, and the console has its own code page. If psql
  warns about it at startup, fix that first (see the pitfall), or Turkish letters may
  come out garbled.

## How it works

`\timing` prints how long each statement took as psql saw it, so the number includes the
network and transferring the result, but not printing it. The docs say only "how long
each SQL statement takes"; psql's source (src/bin/psql/common.c, function
ExecQueryAndProcessResults) starts the clock before sending the query and stops it once
the result has arrived, before printing. `EXPLAIN ANALYZE`, covered in the indexes
module, works the other way round: its Execution Time is measured inside the server and,
per the docs, includes neither planning (reported separately) nor sending rows to the
client. When `\timing` is much larger, the gap is planning plus getting the rows to you.
On PostgreSQL 18.7 here, 300,000 generated rows sent to `\o NUL` took 135 to 184 ms by
`\timing`, while `EXPLAIN ANALYZE` of the same query reported 61 to 81 ms.

`\e` opens the current query buffer (or, if it's empty, the last query) in an editor:
`PSQL_EDITOR`, `EDITOR` or `VISUAL`, whichever is set first, else `vi` on Unix and
`notepad.exe` on Windows. When you save and quit, any complete statement in the buffer,
one ending in `;`, runs at once. Your personal settings go in `~/.psqlrc`, or
`%APPDATA%\postgresql\psqlrc.conf` on Windows; `\x auto`, `\timing on` and a
`\pset null` marker are common first lines.

Patterns follow SQL's identifier rules: unquoted letters fold to lower case, double
quotes keep them. In SQL itself, in a UTF-8 database, the server's fold only touches
ASCII letters, as the `Şubeler` example shows. The docs say only that unquoted names "are
always folded to lower case"; the server's source (src/backend/parser/scansup.c, function
downcase_identifier) lowers A to Z and leaves every byte of a multibyte character alone.
With Turkish table names in SQL, quote them or keep them all lower case.

psql's own fold for patterns is cruder. It goes byte by byte through the C library's
`isupper`/`tolower` in the client's locale (src/fe_utils/string_utils.c, function
patternToSQLRegex), which can change a byte inside a UTF-8 letter. Observed with psql 18.7
on StackLab's Windows machine (locale en-US, code page 1252; console code page 437 or 65001;
client encoding UTF8): `\dt Şubeler`, and even `\dt şubeler`, fail with `invalid byte
sequence for encoding "UTF8"`, while `\dt "Şubeler"` and `\dt *ubeler` find the table.
In a pattern, double-quote any name with a non-ASCII letter, lower case included.

## Design notes

psql's split between SQL and meta-commands is the useful lesson here: the database
describes itself in ordinary tables, and every tool, from psql to pgAdmin to your own
health check, is just querying them. When you need schema facts in code (a migration
check, a test asserting an index exists), copy the query `psql -E` shows you, or use
`information_schema` if it must also run on another database. Don't scrape `\d` output:
it is formatted for people and changes between versions.
