---
id: sql.modern.16-to-18
title: "Highlights from PG 16 to 18: uuidv7, virtual generated columns, OLD/NEW in RETURNING, temporal constraints, skip scan, async I/O, JSON_TABLE, MERGE ... RETURNING"
track: sql
module: modern
order: 1
level: intermediate
minutes: 14
since: pg18
prerequisites: [sql.modifying.returning-upsert, sql.indexes.plans]
compare_with: [sql.tables-types.ids, sql.constraints.generated, sql.modeling.temporal, sql.subqueries-ctes.modifying, sql.json.path-table, sql.indexes.gin-brgin, go.timeline.1-21-to-1-27]
tags: [postgresql-16, postgresql-17, postgresql-18, uuidv7, generated-columns, temporal, merge, json_table, skip-scan, aio]
summary: >
  Three releases changed a lot of standing advice: UUID keys, generated
  columns, audit queries, non-overlap constraints, composite index design,
  JSON import and MERGE. A tour with small runnable proofs, each feature tagged
  with its version and linked to the lesson that covers it in depth.
examples:
  - id: pg18-tables
    file: examples/01_pg18_tables.sql
    title: "PG 18 table definitions: temporal keys, uuidv7, virtual columns, NOT NULL as constraints"
    runtime: sql
    check: exact
    annotations:
      - lines: "3"
        note: >
          btree_gist gives int a GiST equality operator class. For key columns
          that aren't ranges, the docs call adding it the expected way to use
          WITHOUT OVERLAPS.
      - lines: "6-11"
        note: >
          PG 18 temporal primary key: requirement_id may repeat as long as the
          valid ranges don't overlap. It is enforced like the exclusion
          constraint you had to write by hand before 18 (requirement_id WITH
          =, valid WITH &&), and foreign keys can now reference it.
      - lines: "17"
        note: >
          uuidv7() (PG 18) puts a timestamp in the leading bits, so new keys
          arrive roughly in order at the end of the B-tree instead of at random
          places. The old advice, gen_random_uuid() or the uuid-ossp
          extension, gives random v4 keys (sql.tables-types.ids).
      - lines: "21"
        note: >
          No keyword now means VIRTUAL (PG 18): computed when read, no storage.
          Before 18 the STORED keyword was required, so old DDL can't silently
          change meaning. Virtual columns may use built-in functions and types
          only.
      - lines: "23-24"
        note: >
          PG 18 temporal foreign key: the rows with the same requirement_id
          must together cover the whole ran period. The referenced key must be
          declared WITHOUT OVERLAPS, and ON DELETE CASCADE and friends aren't
          supported here.
      - lines: "28"
        note: >
          This run spans the switch between two versions. Neither covers it
          alone; together they do, and that is what PERIOD checks.
      - lines: "30-32"
        note: >
          uuid_extract_version (PG 17) reads the version back; duration_s is
          computed on read.
      - lines: "36-42"
        note: >
          The temporal key's index is GiST, the uuid key's is B-tree. Also new
          in 18: every NOT NULL is a named row in pg_constraint (contype n),
          and ALTER TABLE can add one as NOT VALID.
      - lines: "43-44"
        note: >
          attgenerated is 'v' for virtual, 's' for stored.
      - lines: "47"
        note: >
          SQLSTATE 23P01, exclusion_violation: the docs describe WITHOUT
          OVERLAPS as enforced by an EXCLUDE constraint rather than a unique
          one, and the error says so. Catch 23P01 in Go, not 23505.
  - id: json-merge
    file: examples/02_json_merge.sql
    title: "IS JSON (16), JSON_TABLE (17), MERGE ... RETURNING (17) with old/new (18)"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-7"
        note: >
          PG 16 IS JSON: validate text before casting it instead of catching a
          cast error. It can also test for ARRAY or SCALAR, and WITH UNIQUE
          KEYS.
      - lines: "17-21"
        note: >
          PG 17 JSON_TABLE: the SQL-standard way to turn a JSON document into
          typed rows. Before 17 you combined jsonb_array_elements or
          jsonb_to_recordset with ->> per field (sql.json.path-table).
      - lines: "25-37"
        note: >
          One statement: parse the CI report, merge it into test_cases and
          return what changed. A MERGE with RETURNING can sit in WITH like any
          data-modifying statement, which is how its output gets sorted.
      - lines: "32"
        note: >
          TC-1 matches with an unchanged status: no action, so no RETURNING
          row.
      - lines: "34"
        note: >
          PG 17 WHEN NOT MATCHED BY SOURCE acts on target rows the source
          doesn't mention. Before 17 that took a second statement.
      - lines: "35"
        note: >
          merge_action() (PG 17) names the action behind each row; old and new
          (PG 18) give both sides, with old NULL for the insert. Before 17 a
          MERGE returned only its row count.
  - id: planner-io
    file: examples/03_planner_io.sql
    title: "Skip scan, EXPLAIN's new defaults, and the async I/O settings (PG 18)"
    runtime: sql
    check: exact
    annotations:
      - lines: "8"
        note: >
          status leads the index and the query never mentions it. Before 18 a
          B-tree could serve this query only by reading the whole index, so
          the advice was "filter column first, or a second index".
      - lines: "10"
        note: >
          271 pages of index, for comparison with what the scan touches below.
      - lines: "13"
        note: >
          The answer. Running the query once first also keeps one-time catalog
          lookups out of the EXPLAIN output below.
      - lines: "16-17"
        note: >
          Index Searches: 7 (a PG 18 EXPLAIN line): the scan jumped from one
          status value to the next, touching 23 index buffers instead of 271
          pages. Buffers appears without asking for it, and row counts print
          with two decimals: both new in 18.
      - lines: "20-22"
        note: >
          PG 18 asynchronous I/O lets a backend queue several reads at once for
          sequential scans, bitmap heap scans and VACUUM. worker (the default)
          hands them to io_workers background processes; io_uring needs a
          Linux build with liburing; sync means no AIO. The
          effective_io_concurrency default rose to 16. A tuned server shows
          other values.
use_cases:
  - "Reviewing old advice before applying it: Stack Overflow answers, ORM defaults and blog posts written for PostgreSQL 9 to 15"
  - "Planning an upgrade: which features you gain, and which incompatibilities (the release notes' Migration section) you must handle"
  - "Choosing a minimum supported server version for software you ship, such as TraceLite on a customer's machine"
strengths:
  - "Several features replace extensions or hand-written workarounds: uuidv7 instead of uuid-ossp, WITHOUT OVERLAPS instead of a hand-written EXCLUDE, JSON_TABLE instead of jsonb_to_recordset chains"
  - "RETURNING old/new and MERGE ... RETURNING make audit-style writes one statement, without triggers"
  - "Skip scan and async I/O speed up existing queries without changing them"
  - "One major version a year, five years of support for each, and fix-only minor releases at least every three months make upgrades plannable"
weaknesses:
  - "Features depend on the server version, not on your client: code using uuidv7() fails on a PG 17 server with no compile-time warning"
  - "Defaults moved: generated columns are virtual unless you say STORED, and EXPLAIN ANALYZE prints BUFFERS, so old expected outputs and scripts can differ"
  - "Some new features are first versions: temporal foreign keys support no CASCADE or SET NULL actions, virtual columns can't use user-defined functions"
  - "Managed services and Linux distributions trail the newest release, so 'available in PostgreSQL' and 'available on your server' differ"
pitfalls:
  - title: "Following advice without its version"
    detail: "'Use uuid-ossp for UUIDs', 'generated columns are always stored', 'MERGE can't return rows', 'a composite index is useless without its first column': each was true for some versions. Check the release notes for when it changed."
  - title: "Using a feature newer than the oldest server you support"
    detail: "Check at startup or in a migration, and fail with a clear message instead of a syntax error halfway through. server_version_num is an integer: 180007 for 18.7."
    good: "SELECT current_setting('server_version_num')::int >= 180000 AS has_pg18;"
  - title: "Catching unique violations from a temporal key"
    detail: "A WITHOUT OVERLAPS conflict is an exclusion violation, SQLSTATE 23P01, not unique_violation 23505. Error mapping written for plain keys misses it and returns a 500."
  - title: "Assuming a virtual column is indexed or stored"
    detail: "Virtual is the default now. If you need to index the value or read it often from a large table, say STORED explicitly, as before 18."
    bad: "price_gross int GENERATED ALWAYS AS (price_net * 120 / 100)"
    good: "price_gross int GENERATED ALWAYS AS (price_net * 120 / 100) STORED"
quiz:
  - q: "On PG 18, `CREATE TABLE t (a int, b int GENERATED ALWAYS AS (a * 2))` creates b as..."
    options: ["A stored generated column", "A virtual generated column", "A syntax error: STORED is required"]
    answer: 1
    why: "PG 18 made VIRTUAL the default. Before 18, STORED was required, so the same statement failed."
  - q: "Which version added RETURNING to MERGE, and which added old/new in RETURNING?"
    options: ["Both PG 16", "MERGE ... RETURNING in 17, old/new in 18", "Both PG 18"]
    answer: 1
    why: "PG 17 added MERGE ... RETURNING with merge_action(); PG 18 added the old and new aliases for INSERT, UPDATE, DELETE and MERGE."
  - q: "An index on (status, ran_on); the query filters only on ran_on. What changed in PG 18?"
    options: ["Nothing, the leading column must be constrained", "The B-tree can skip from one status value to the next instead of reading the whole index", "PostgreSQL creates a second index automatically"]
    answer: 1
    why: "Skip scan lets multicolumn B-tree indexes be used without conditions on the leading columns; it pays off when those columns have few distinct values."
  - q: "Inserting an overlapping period into a table with `PRIMARY KEY (id, valid WITHOUT OVERLAPS)` fails with which SQLSTATE?"
    options: ["23505 unique_violation", "23P01 exclusion_violation", "23514 check_violation"]
    answer: 1
    why: "WITHOUT OVERLAPS is enforced like an EXCLUDE constraint on a GiST index, so a conflict is an exclusion violation."
further_reading:
  - title: "PostgreSQL 18 release notes"
    url: "https://www.postgresql.org/docs/release/18.0/"
  - title: "PostgreSQL 17 release notes"
    url: "https://www.postgresql.org/docs/release/17.0/"
  - title: "PostgreSQL 16 release notes"
    url: "https://www.postgresql.org/docs/release/16.0/"
  - title: "PostgreSQL docs: CREATE TABLE (WITHOUT OVERLAPS, PERIOD, VIRTUAL)"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
  - title: "PostgreSQL docs: SQL/JSON functions, JSON_TABLE"
    url: "https://www.postgresql.org/docs/current/functions-json.html"
  - title: "PostgreSQL docs: Asynchronous I/O settings (io_method)"
    url: "https://www.postgresql.org/docs/current/runtime-config-resource.html"
  - title: "PostgreSQL versioning policy"
    url: "https://www.postgresql.org/support/versioning/"
---

## Concept

PostgreSQL ships one major version a year. Much of what you'll read online predates the
last three, and some of it is now wrong. The headline changes, with the old advice they
retire:

**PostgreSQL 16 (2023)**

- SQL/JSON constructors (`JSON_OBJECT`, `JSON_ARRAY`, `JSON_ARRAYAGG`,
  `JSON_OBJECTAGG`) and the `IS JSON` predicate.
- `ANY_VALUE()`; integer literals in hex, octal and binary (`0x1F`) and underscores in
  numbers (`1_000_000`); `pg_input_is_valid()` to test a cast without an error.
- `pg_stat_io` for I/O statistics; parallel `FULL` and right outer hash joins.

**PostgreSQL 17 (2024)**

- `JSON_TABLE` and `JSON_EXISTS`/`JSON_QUERY`/`JSON_VALUE`: "parse JSON in the app" is
  no longer the only clean option.
- `MERGE ... RETURNING` with `merge_action()`, `WHEN NOT MATCHED BY SOURCE`, and `MERGE`
  on updatable views.
- `COPY ... ON_ERROR ignore`, `transaction_timeout`, incremental backups, the
  `MAINTAIN` privilege, and the builtin `C.UTF-8` collation provider (the one local
  mode's playground database uses).

**PostgreSQL 18 (2025)**

- `uuidv7()`: time-ordered UUID keys without an extension.
- Generated columns are **virtual** by default.
- `RETURNING old.*, new.*` in `INSERT`, `UPDATE`, `DELETE` and `MERGE`.
- Temporal constraints: `PRIMARY KEY`/`UNIQUE ... WITHOUT OVERLAPS` and foreign keys
  with `PERIOD`.
- B-tree **skip scan**; an **asynchronous I/O** subsystem; `EXPLAIN ANALYZE` includes
  `BUFFERS` by default.
- `NOT NULL` constraints stored in `pg_constraint` (nameable, addable `NOT VALID`);
  `CHECK` and foreign keys can be `NOT ENFORCED`; `pg_upgrade` keeps planner
  statistics; `casefold()`; MD5 passwords deprecated; data checksums on by default in
  new clusters.

The examples prove the ones you'll touch in everyday schema and query work; the linked
lessons go deeper.

## Coming from C/C++/Java/Python/JS

- **Language versions:** like checking whether advice targets Java 8 or Java 21, or
  Go before 1.22's loop variable change, check which PostgreSQL version a piece of SQL
  advice assumes. The go.timeline lesson does the same tour for Go.
- **Runtime, not compile time:** a Go program built with a newer toolchain still runs;
  SQL that uses `uuidv7()` fails on a PG 17 server at execution. The database is the
  runtime, and its version is a deployment fact your code should check.
- **Release cadence:** one major version a year, each supported for five years, with
  fix-only minor releases at least every three months. Minor upgrades (18.6 to 18.7) are
  binary replacements; major upgrades go through `pg_upgrade`, dump and restore, or
  logical replication.

## Design notes

Decide a minimum server version deliberately and write it down, as this curriculum does
with `since`. For an application you host, take the newest major version your platform
offers; for software installed on someone else's machine (TraceLite's packaging step),
support what they can realistically run and check `server_version_num` at startup.

Read the "Migration to Version N" section of each release's notes before upgrading: it
lists the incompatibilities, such as the virtual default for generated columns, MD5
deprecation warnings, `AFTER` triggers running as the queuing role, and `VACUUM` and
`ANALYZE` processing inheritance children by default in 18.

Adopt new features where they remove code or extensions: `uuidv7()` for new
UUID-keyed tables, `WITHOUT OVERLAPS` for validity periods, `RETURNING old/new` for
audit-style writes, `JSON_TABLE` for imports. Don't rewrite working code just because a
newer form exists, and keep the old idiom in mind when you read other people's schemas.
