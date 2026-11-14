---
id: sql.files-movement.dump-generate
title: "Exporting results; pg_dump/pg_restore; generating test data with generate_series"
track: sql
module: files-movement
order: 3
level: intermediate
minutes: 15
since: pg17
prerequisites: [sql.files-movement.copy, sql.modifying.insert-update]
compare_with: [go.testing.integration, go.postgres.bulk, sql.security-ops.backup-migrations, fullstack.tracelite.step-08-export]
tags: [generate_series, test-data, setseed, md5, pg_dump, pg_restore, export, copy]
summary: >
  generate_series plus INSERT ... SELECT seeds 100,000 realistic, reproducible rows
  in one statement. pg_dump exports a database as a consistent snapshot, as a SQL
  script for psql or an archive for pg_restore; query results leave through COPY.
examples:
  - id: seed
    file: examples/01_seed.sql
    title: "100,000 test runs in one statement, skewed, correlated and reproducible"
    runtime: sql
    check: exact
    annotations:
      - lines: "16-19"
        note: >
          generate_series is a row source: every expression on i becomes a
          column. Indexing an array with [1 + i % 5] cycles through sample
          values; the 1 + is there because SQL arrays start at 1.
      - lines: "23-34"
        note: >
          One INSERT ... SELECT, one round trip, INSERT 0 100000. A client loop
          would send 100,000 statements; even batched, it moves data the
          server could have made itself.
      - lines: "25"
        note: >
          u is spread evenly over [0, 1); squaring it piles values up near 0,
          so low ids get most runs: REQ-001 has 7,273, REQ-003 2,236. Real
          data is skewed like this, and plans and index choices change with
          it; i % 200 would give every requirement exactly 500.
      - lines: "26"
        note: >
          One run every 257 seconds from 1 January, about ten months. The +03
          in the literal fixes the instant whatever the session's TimeZone is.
      - lines: "27-28"
        note: >
          Both columns come from v, so they are correlated: the slowest 10% of
          runs are the failures (average 24.78 s against 4.13), the way
          timeouts look in real test results. The fourth power gives a long
          tail of slow runs.
      - lines: "29-30"
        note: >
          md5 makes realistic-looking strings: a 10-digit hex "commit" shared
          by 25 consecutive runs. Every fourth tester is NULL, an automated
          run, so queries meet NULLs the way they will in production.
      - lines: "32-34"
        note: >
          The md5 of the row number, read as bytes, gives two numbers between 0
          and 1 that look random and are identical on every run, on every
          server: a reproducible stand-in for random(), with no seed to manage.
      - lines: "36-42"
        note: >
          Check what you generated: counts, distinct values, ratios, ranges.
          AT TIME ZONE 'Europe/Istanbul' turns the timestamptz into Turkish
          local time before taking the date, independent of the session.
  - id: seeded-random
    file: examples/02_seeded_random.sql
    title: "random() with setseed, and the date trap in generate_series"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          setseed takes a value between -1 and 1 and restarts the session's
          generator, so the same statements after the same seed fill identical
          tables. random(min, max) is PG 17; before that, floor(random() * 6)
          + 1.
      - lines: "11-13"
        note: >
          Without a new seed the generator just continues. Reorder two
          statements in a seed script and every table after them changes.
      - lines: "15-19"
        note: >
          Comparing tables keeps this output stable. The values themselves are
          not promised across versions: PG 15's release notes say its new
          random() algorithm gives different results "even for the same seed
          value".
      - lines: "21-23"
        note: >
          There is no date version of generate_series, so dates resolve to the
          timestamptz one: days are stepped in the session's time zone, and
          you get timestamps back. Cast to date, or add an integer series to a
          date: date '2026-11-01' + i.
  - id: stream-copy
    file: examples/03_stream_copy.go
    title: "The text format pg_dump uses, and a table streamed between two connections"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "16-26"
        note: >
          Both connections use the same URL so the example is self-contained;
          the pattern is for two different databases: production to a
          reporting database, an old server to a new one.
      - lines: "37-41"
        note: >
          COPY's text format: tab-separated, \N for NULL, escapes for tabs and
          newlines inside values. A plain-format pg_dump stores table data in
          this form.
      - lines: "44-53"
        note: >
          io.Pipe connects CopyTo's io.Writer to CopyFrom's io.Reader. Rows flow
          from one server to the other as they are read, with no temporary file:
          what pg_dump -t runs src | psql dst does, inside one program.
      - lines: "48"
        note: >
          A failed source reaches the reader as an error rather than EOF;
          pgconn then aborts the COPY on the destination, so a half-read table
          is never loaded as if it were complete.
      - lines: "55-58"
        note: >
          EXCEPT ALL in both directions compares the tables as multisets on
          the server, without fetching 50,000 rows into Go.
use_cases:
  - "Seeding a development or test database with enough skewed, realistic data to make EXPLAIN plans meaningful"
  - "Reproducible benchmark data: the same script gives the same rows on every machine and every run"
  - "Copying a database or a single table to another server: pg_dump/pg_restore, or COPY streamed between connections"
  - "Exporting a report as CSV from psql (\\copy, --csv) or from code (CopyTo)"
strengths:
  - "100,000 rows in one statement, made where they are stored, instead of a Faker loop in Python or Java inserting row by row"
  - "Deterministic generators (arithmetic, md5) need no seed and can't drift between versions or sessions"
  - "pg_dump takes a consistent snapshot without blocking readers or writers, unlike stopping the application to copy files"
  - "The custom format keeps a table of contents: pg_restore can restore one table, reorder items, or work in parallel, which a mysqldump-style SQL script can't"
weaknesses:
  - "Realism is hand-made: names, distributions and correlations are only as good as the expressions you write"
  - "pg_dump is a logical copy of one database: no roles or tablespaces (pg_dumpall --globals-only has those), no point-in-time recovery, slow to restore for large databases"
  - "Statistics are not dumped by default in PG 18, so a restored database needs ANALYZE before its plans mean anything"
  - "A dump can't be loaded into an older major version without editing, and pg_dump refuses to dump a server newer than itself"
pitfalls:
  - title: "Seeding from a client loop"
    detail: "100,000 single-row INSERTs are 100,000 statements. Generate the rows in SQL, or use CopyFrom when the data really comes from the program."
    bad: "for i := range 100_000 { conn.Exec(ctx, \"INSERT INTO test_runs VALUES ($1, ...)\", i, ...) }"
    good: "INSERT INTO test_runs SELECT i, ... FROM generate_series(1, 100000) AS i;"
  - title: "setseed through a connection pool"
    detail: "The seed belongs to one session. With pgxpool, SELECT setseed(...) and the INSERT that follows can run on different connections. Put them in one transaction, or on one acquired connection."
  - title: "Looking at plans right after seeding"
    detail: "Until ANALYZE runs (yourself, or autovacuum eventually), the planner guesses row counts. The docs recommend ANALYZE after any bulk load."
    good: "ANALYZE test_runs;"
  - title: "Restoring a custom-format dump with psql"
    detail: "-Fc writes a compressed archive, not SQL. pg_restore reads it; pg_restore -f - turns it into a script if you need one."
    bad: "psql -d tracelite_copy -f tracelite.dump"
    good: "pg_restore -d tracelite_copy tracelite.dump"
  - title: "Restoring into a database whose roles don't exist"
    detail: "The dump sets object owners; on another server those roles may be missing and every ALTER ... OWNER fails. Create the roles first (pg_dumpall --globals-only), or restore with --no-owner so the restoring user owns everything."
quiz:
  - q: "Which pg_dump formats can pg_restore restore in parallel with -j?"
    options: ["Plain SQL", "Custom and directory", "Any format"]
    answer: 1
    why: "The pg_restore docs: only the custom and directory archive formats are supported with -j, and the input must be a regular file or directory."
  - q: "A seed script calls setseed(0.5) and then fills tables with random(). Does it produce the same data on PostgreSQL 14 and 18?"
    options:
      - "Yes, the seed fixes the sequence"
      - "No: PG 15 changed random()'s algorithm, so the same seed gives different values"
      - "Only on the same operating system"
    answer: 1
    why: "The PG 15 migration notes say random() results differ from prior versions even for the same seed. md5- or arithmetic-based generators don't have this problem."
  - q: "What does generate_series(date '2026-11-01', date '2026-11-03', interval '1 day') return?"
    options: ["date values", "timestamp without time zone", "timestamp with time zone"]
    answer: 2
    why: "There is no date variant; the dates are converted for the timestamptz variant, which steps in the session's time zone."
  - q: "pg_dump -Fc -d tracelite -f tracelite.dump made an archive. How do you read the SQL it contains without a database?"
    options:
      - "psql -f tracelite.dump"
      - "pg_restore -f - tracelite.dump"
      - "Unzip it: it is gzip-compressed SQL"
    answer: 1
    why: "Without -d, pg_restore writes the script it would run; -f - sends it to standard output."
further_reading:
  - title: "PostgreSQL docs: Set Returning Functions (generate_series)"
    url: "https://www.postgresql.org/docs/current/functions-srf.html"
  - title: "PostgreSQL docs: Mathematical Functions (random, setseed)"
    url: "https://www.postgresql.org/docs/current/functions-math.html"
  - title: "PostgreSQL docs: SQL Dump"
    url: "https://www.postgresql.org/docs/current/backup-dump.html"
  - title: "PostgreSQL docs: pg_dump"
    url: "https://www.postgresql.org/docs/current/app-pgdump.html"
  - title: "PostgreSQL docs: pg_restore"
    url: "https://www.postgresql.org/docs/current/app-pgrestore.html"
  - title: "PostgreSQL docs: Populating a Database (ANALYZE afterwards)"
    url: "https://www.postgresql.org/docs/current/populate.html"
---

## Concept

**Generating data.** `generate_series(1, 100000)` is a table of 100,000 integers, and
`INSERT ... SELECT` turns any query into rows. Every column is an expression on the
counter: arithmetic for ids, dates and cycles, arrays for sample values, `md5` for
strings and for numbers that look random but come out the same every time. Real data
is skewed, correlated and has NULLs, and test data that has none of the three gives
plans and bugs that production never sees.

**Exporting results.** A query result becomes a file through COPY, as in the first
lesson. From psql there are three ways:

```
\copy (SELECT code, title FROM requirements ORDER BY code) TO 'requirements.csv' WITH (FORMAT csv, HEADER)
psql -X -d tracelite --csv -c "SELECT code, title FROM requirements ORDER BY code" -o requirements.csv
COPY (SELECT ...) TO STDOUT WITH (FORMAT csv, HEADER) \g requirements.csv
```

The third allows a query over several lines. From code, it's `CopyTo` into a file or an
HTTP response.

**pg_dump and pg_restore.** `pg_dump` exports one database: definitions and data, as
a consistent snapshot, without blocking readers or writers. The default plain format
is an SQL script that psql replays. The custom format (`-Fc`, compressed) and the
directory format (`-Fd`) are archives that `pg_restore` reads selectively,
in any order, and in parallel. Roles and tablespaces belong to the whole cluster, so
they come from `pg_dumpall --globals-only`. The usual commands:

```
# plain SQL script, replayed by psql into a database created from template0
pg_dump -d tracelite -f tracelite.sql
createdb -T template0 tracelite_copy
psql -X -d tracelite_copy --set ON_ERROR_STOP=on -f tracelite.sql

# custom format: restore everything with 4 jobs, or look inside first
pg_dump -Fc -d tracelite -f tracelite.dump
pg_restore -d tracelite_copy -j 4 tracelite.dump
pg_restore -l -f toc.list tracelite.dump
pg_restore -L toc.list -d tracelite_copy tracelite.dump
pg_restore -f - --schema-only tracelite.dump

# one table's rows as INSERT statements, for another database product
pg_dump -d tracelite -t test_runs --data-only --inserts -f test_runs.sql
```

`pg_restore -l` lists the archive's table of contents; edit the list (delete lines, or
comment them out with `;`) and `-L` restores just those items, in that order. On
Windows the tools live in `C:\Program Files\PostgreSQL\18\bin`. Pass `-f` instead of
redirecting with `>` in PowerShell: Microsoft's about_Redirection page for Windows
PowerShell 5.1 says redirected byte streams are treated as strings, which "results in
corrupted data". Reproduced here with Windows PowerShell 5.1 and pg_dump 18.7:
`pg_dump -Fc ... > bad.dump` wrote a UTF-16 file (it starts with the bytes FF FE) that
`pg_restore -l` rejects as "not a valid tar archive", while `-f good.dump` works. Text
suffers too: a TOC list written with `pg_restore -l ... > toc.list` came out UTF-16, and
`pg_restore -L` warned "line ignored", restored nothing and still exited with 0.

## Coming from C/C++/Java/Python/JS

- **Faker** (Python), **Datafaker** (Java), **@faker-js** (JS): great names and
  addresses, but a loop in the client inserting row by row. `generate_series` makes the
  rows where they are stored; arrays of sample values cover most needs, and a
  Faker-made CSV loaded with COPY covers the rest.
- **Go**: seeding test databases with `for i := range n { db.Exec(...) }` is the habit to
  drop. One SQL statement, or `CopyFrom` with `CopyFromSlice` (the Go/Postgres
  module) when the program computes the values.
- **MySQL**: `mysqldump` writes an SQL script like pg_dump's plain format. The custom
  format adds what a script can't: a table of contents for selective, reordered,
  parallel restores.
- **SQL Server**: `BACKUP DATABASE` is a physical backup. pg_dump is logical, closer
  to generating scripts; PostgreSQL's physical backups (`pg_basebackup`, WAL archiving)
  are in the security and operations module.
- **Shell**: `generate_series` is `seq` inside the database, and a plain dump piped into
  psql (`pg_dump -h host1 db | psql -X -h host2 db`, straight from the docs) is the same
  idea as the Go pipe in the third example.

## How it works

Three ways to make values from a row number, from most to least reproducible:

- **Arithmetic on i**: `i % 5`, `i * interval '257 seconds'`, `i / 25`. Exact, but
  regular: cycles show up as patterns, and `i % 200` gives every group the same count.
- **Hashing i**: `md5(i::text)` read as bytes (`decode(..., 'hex')`, `get_byte`). It
  looks random, involves no state, and gives the same values on every server and
  version, since md5 is a fixed algorithm.
- **random() with setseed()**: random() is volatile (evaluated per row) and keeps its
  state per session. After `setseed`, the docs promise the same sequence again within
  the session; across versions they don't, and PG 15 did change it.

A plain-format dump is the schema as `CREATE` statements, then each table's rows as a
`COPY ... FROM stdin;` command followed by the rows in COPY's text format and a line
holding only `\.`, then indexes and the constraints not written inline (primary keys,
unique and foreign keys; NOT NULL and CHECK stay in `CREATE TABLE`). That inline data is
why psql, which knows to feed it to the COPY, replays dumps, and why a driver can't
simply execute the file. pg_dump 18.7 also wraps the script in psql's `\restrict` and
`\unrestrict` meta-commands (the `--restrict-key` option). The docs describe the parts:
data is dumped as COPY unless `--inserts`, `--section` splits pre-data, data and
post-data (indexes, triggers, most constraints), and the COPY page says psql ends in-line
data at `\.`. A dump of a small schema from PostgreSQL 18.7 showed exactly this order,
with `\N` for NULL and `\t` for a tab inside a value.

pg_dump reads everything from one snapshot, taken when it starts, which is how it
stays consistent while others write; only operations that need an exclusive lock, such
as most forms of `ALTER TABLE`, have to wait for it. Statistics are not dumped by default in PG 18 (`--statistics` adds them), so run
`ANALYZE` after a restore. A dump loads into newer major versions; pg_dump refuses to
dump a server newer than itself, so after upgrading a server, upgrade the client tools
too.

## Design notes

Keep seed scripts in the repository as SQL: one `INSERT ... SELECT FROM
generate_series` per table, the row count in one place, and deterministic generators,
so tests and benchmarks on different machines see the same data. Make the data look
like production in shape (skew, NULL rates, correlations, Turkish text) rather than
copying production rows, which drags personal data into places it doesn't belong.
Run `ANALYZE` after seeding, before you read a single `EXPLAIN`.

Use pg_dump for moving a database to another server or version, for a schema snapshot
(`--schema-only`) to diff in review, and for pulling one table out. The docs say it is
"generally not the right choice" for regular backups of production databases:
physical backups with WAL archiving restore faster and to any point in time. For
moving one table between live databases from code, the streamed COPY of the third
example avoids temporary files altogether.
