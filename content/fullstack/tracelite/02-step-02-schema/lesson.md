---
id: fullstack.tracelite.step-02-schema
title: "Step 2: Schema: hierarchical requirements, test cases, trace links, test runs"
track: fullstack
module: tracelite
order: 2
level: intermediate
minutes: 15
since: pg18
prerequisites: [fullstack.tracelite.step-01-architecture, sql.tables-types.ids, sql.constraints.fk-check, sql.joins.many-to-many, sql.subqueries-ctes.recursive]
compare_with: [sql.modeling.normalization, go.postgres.migrations-sqlc, sql.indexes.stats-pagination, sql.security-ops.backup-migrations]
tags: [tracelite, schema, migrations, uuidv7, foreign-keys, composite-keys, recursive-cte, indexes, constraints]
summary: >
  TraceLite's first migration: requirements as a parent_id tree, test cases, trace links
  and test runs, with uuidv7 keys, named constraints the Go store can map, composite
  foreign keys that keep everything inside one project, and an index for every query.
examples:
  - id: migration
    file: examples/01_migration.sql
    title: "Migration 001's first two tables, verbatim, and each constraint refusing a row"
    runtime: sql
    check: exact
    annotations:
      - lines: "2"
        note: >
          goose's header (go.postgres.migrations-sqlc). The file is
          migrations/001_init.sql; this example runs its first two tables
          verbatim, and How it works shows all of it. Later steps add 002,
          003, ... and never edit this one once it has run anywhere.
      - lines: "4"
        note: >
          uuidv7() (PG 18) puts a millisecond timestamp in the leading bits, so
          new keys land at the right edge of the B-tree like a sequence would.
          Unlike bigint identity values, ids are safe to show in URLs, can be
          minted outside the database (the importer in step 7), and travel as
          JSON strings without the 2^53 problem.
      - lines: "10-11"
        note: >
          A project owns its requirements: CASCADE removes them with it.
          parent_id NULL means top level. The tree is an adjacency list, the
          simplest model to write to.
      - lines: "12-13"
        note: >
          Codes are identifiers, not prose: ASCII capitals, digits, single
          hyphens. COLLATE "C" compares bytes, so ORDER BY code, the keyset
          cursor and Go's string comparison agree on every server, whatever
          locale the database was created with.
      - lines: "14"
        note: >
          char_length counts characters, not bytes; the Go service counts runes
          with utf8.RuneCountInString to match. A Turkish title of 200
          characters can be 400 bytes (line 53).
      - lines: "16-17"
        note: >
          text plus a CHECK instead of CREATE TYPE ... AS ENUM: a value can be
          added or retired by replacing the constraint in a migration, while an
          enum value can be added and renamed but never dropped.
      - lines: "20-24"
        note: >
          Every constraint has a chosen name, because the Go store maps
          violations by constraint name: requirements_code_key becomes 409,
          requirements_parent_fkey becomes 422 on #/parentId. Generated names
          would change if a column were renamed.
      - lines: "21-24"
        note: >
          The parent must be in the same project. The pair (project_id, id) is
          already unique because id is, but a foreign key can only reference
          columns with a unique constraint, hence line 21. MATCH SIMPLE, the
          default, skips the check when parent_id is NULL, which is what lets
          top-level rows exist.
      - lines: "22"
        note: >
          A row can't be its own parent. Longer loops (A under B under A) are
          not something a constraint can see; example 2 shows the check the
          store runs instead.
      - lines: "26"
        note: >
          PostgreSQL indexes the referenced side of a foreign key, never the
          referencing side. This index serves the tree walk (children of a
          node) and the check that runs when a parent is deleted.
      - lines: "31-42"
        note: >
          A lesson helper, not part of the migration. The EXCEPTION block runs
          the INSERT in a subtransaction, so a refused row is rolled back and
          the script continues; GET STACKED DIAGNOSTICS reads the same fields a
          *pgconn.PgError carries in Go: Code and ConstraintName.
      - lines: "49-50"
        note: >
          Lower case is refused rather than silently upper-cased: one canonical
          form, enforced where every writer passes. Line 50 is what
          "auth-login".toLocaleUpperCase("tr") produces: dotted capital İ, not
          ASCII, so it fails the same CHECK.
      - lines: "52"
        note: >
          AUTH-LOGIN exists, but in project Ödeme. The composite key
          (Raporlama, AUTH-LOGIN's id) matches no row, so the database, not
          the application, rejects the cross-project parent.
      - lines: "53-54"
        note: >
          ş is two bytes in UTF-8. The limit is 200 characters, so 400 bytes
          pass and 201 characters don't. A byte-based check in Go (len(title))
          would have refused the first row.
      - lines: "60"
        note: >
          Project names are prose, so their order depends on the collation:
          byte order puts Ödeme after Raporlama. The explicit COLLATE "C"
          keeps this output the same on servers created with different
          locales; a Turkish UI sorts with a Turkish collation instead.
  - id: tree
    file: examples/02_tree.sql
    title: "The requirement queries: the tree, a keyset page, the move check, deletes"
    runtime: sql
    check: exact
    annotations:
      - lines: "15-25"
        note: >
          Seeding by code, one INSERT per level: each level joins the codes of
          the level above. Step 7's CSV import resolves parent codes the same
          way, in bulk.
      - lines: "30-37"
        note: >
          The tree endpoint's query. It starts at the project's roots, joins
          children on parent_id round by round (the requirements_parent index),
          and SEARCH DEPTH FIRST BY code gives each child a place right after
          its parent. The API returns this flat, ordered list with depth; the UI
          nests it (step 6).
      - lines: "42-43"
        note: >
          GENERIC_PLAN (PG 16) plans a statement that still has $1, $2, $3 in it:
          exactly what the Go store prepares. The unique constraint's index on
          (project_id, code) both filters and delivers the order, so there is
          no Sort node and the scan stops after LIMIT rows, on every page.
      - lines: "44-45"
        note: >
          One row more than the page size: the third row only says that a next
          page exists. The API returns two rows and a cursor holding the last
          code, AUTH-LOGIN-LOCKOUT.
      - lines: "47-55"
        note: >
          A move is legal unless the new parent is the moved row or one of its
          descendants. Walking up from the new parent costs one primary-key
          lookup per level, cheaper than collecting the moved row's whole
          subtree. CYCLE makes the walk finish even if bad data already
          contains a loop.
      - lines: "59-62"
        note: >
          The foreign key is satisfied (every parent exists) and so is the
          CHECK (no row is its own parent), yet AUTH now hangs below its own
          grandchild. The loop has no root, so the tree query silently loses
          five requirements. This is why the store runs the move check, inside
          a transaction (step 3).
      - lines: "65-66"
        note: >
          The cascade removes Raporlama's REPORT and REPORT-PDF in the same
          statement. The parent foreign key is checked at the end of that
          statement, when no orphan remains.
      - lines: "67"
        note: >
          AUTH-LOGIN still has two children, so the default NO ACTION refuses:
          SQLSTATE 23503 on requirements_parent_fkey, which the API reports as
          409 Conflict. Deleting a subtree is a deliberate operation, not a side
          effect.
  - id: links-runs
    file: examples/03_links_runs.sql
    title: "Trace links, latest runs, coverage per requirement, and evidence that survives a delete"
    runtime: sql
    check: exact
    annotations:
      - lines: "8-19"
        note: >
          The link carries project_id so that both foreign keys can be composite:
          a link between a requirement and a test case of different projects
          can't exist. The primary key's index starts with (project_id,
          requirement_id): the matrix of a project and the tests of a
          requirement. Line 19 serves the reverse direction and the check when a
          test case is deleted.
      - lines: "22"
        note: >
          Test evidence doesn't disappear as a side effect: a test case with runs
          can't be deleted, and neither can its project, even through the
          cascade (line 66). Otherwise RESTRICT and the default NO ACTION
          differ only in that NO ACTION can be deferred until later in the
          transaction.
      - lines: "28"
        note: >
          (test_case_id, finished_at DESC, id DESC) answers "latest run of this
          test case" with one index probe, and pages a test case's runs newest
          first. id breaks ties between runs that finished in the same
          microsecond.
      - lines: "37-40"
        note: >
          Linking is idempotent: ON CONFLICT DO NOTHING turns a repeated PUT into
          INSERT 0 0 instead of an error. project_id is taken from the
          requirement, and the composite foreign key checks the test case
          against it.
      - lines: "46-50"
        note: >
          TC-1 failed, then passed; TC-3 passed, then failed. Only the latest
          run of each test case counts for coverage, so the order of
          finished_at matters, not the number of runs.
      - lines: "54"
        note: >
          The trap: bool_and ignores NULL, and TC-4 has never run, so its latest
          result is NULL. AUTH-LOCKOUT's naive_all_passed is true although one of
          its two tests never ran.
      - lines: "55-58"
        note: >
          TraceLite's coverage vocabulary, used from step 6 on: uncovered (no
          linked test), failing (a latest run failed), passing (every linked
          test's latest run passed), incomplete (anything else: never run, or
          last skipped). Counting the passes and comparing with count(*) treats
          NULL as not passed.
      - lines: "60-62"
        note: >
          The join uses the primary key's leading columns, and LATERAL fetches
          one row per link through test_runs_latest: an index probe each, not a
          sort of all runs.
      - lines: "66"
        note: >
          The cascade reaches test_cases, and test_runs_test_case_fkey refuses.
          RESTRICT raises SQLSTATE 23001 (restrict_violation), not the 23503
          that NO ACTION raises (example 2's last statement), so the store maps
          both. The whole DELETE fails: no requirement or link is gone either.
use_cases:
  - "Any hierarchy users edit: requirements, documents and sections, categories, org units, bills of materials"
  - "Many-to-many links that must stay inside a tenant, project or workspace: carry the scope column and make both foreign keys composite"
  - "Evidence tables (test runs, audit records, payments) that must survive deletes elsewhere: ON DELETE RESTRICT"
  - "Public identifiers that appear in URLs and are created by importers or other services: uuid DEFAULT uuidv7()"
strengths:
  - "Integrity lives in the database: an import script, a psql session and the API all hit the same CHECKs and keys, unlike validation kept only in Go or in an ORM's model classes"
  - "Composite foreign keys make cross-project links impossible without a trigger or a lookup in application code"
  - "Named constraints give the store a stable vocabulary: 23505 on requirements_code_key means duplicate code, whatever the message text"
  - "The adjacency list keeps writes to one row (insert, move, delete), while recursive CTEs answer subtree, ancestor and cycle questions in one round trip"
weaknesses:
  - "No constraint can prevent a loop of two or more rows; the move check must run in every writer, in a transaction that serializes moves"
  - "uuid keys are 16 bytes against bigint's 8, in every index and foreign key, and are awkward to read in logs; codes are what people quote"
  - "Codes in byte order sort REQ-10 before REQ-2; numbered codes need zero padding or an ICU collation with numeric ordering"
  - "Carrying project_id in trace_links duplicates a fact the requirement already knows, and the extra unique constraints cost an index each"
pitfalls:
  - title: "Expecting UNIQUE (project_id, id) to be redundant"
    detail: "It is redundant for uniqueness, since id alone is unique, but a composite foreign key can only reference columns covered by a unique constraint or primary key. Without it, CREATE TABLE fails."
  - title: "Forgetting the index on the referencing column"
    detail: "PostgreSQL indexes primary keys and unique constraints, not foreign key columns. Deleting a parent then scans the child table, and the tree walk does one sequential scan per level."
    bad: "parent_id uuid REFERENCES requirements"
    good: "parent_id uuid REFERENCES requirements; CREATE INDEX requirements_parent ON requirements (parent_id);"
  - title: "bool_and over a LEFT JOIN"
    detail: "Aggregates skip NULLs, so a test that never ran doesn't spoil bool_and. Count the matches and compare with the count of links instead."
    bad: "bool_and(last.result = 'passed')"
    good: "count(*) FILTER (WHERE last.result = 'passed') = count(*)"
  - title: "Upper-casing codes with the user's locale"
    detail: "In a Turkish locale, toLocaleUpperCase turns i into İ (U+0130), which the code CHECK rejects. Normalize identifiers with locale-independent toUpperCase() in TS or strings.ToUpper in Go."
  - title: "Relying on uuidv7 order for display"
    detail: "uuidv7 values sort by creation time to the millisecond, but two sessions in the same millisecond interleave randomly. Order lists by a real column (code, finished_at) and use id only as a tiebreaker."
quiz:
  - q: "Why can trace_links reference requirements (project_id, id) at all?"
    options: ["Because id is the primary key", "Because the UNIQUE (project_id, id) constraint exists", "Because both columns are NOT NULL"]
    answer: 1
    why: "A foreign key must reference columns covered by a primary key or unique constraint. The primary key covers id alone, so the pair needs its own unique constraint."
  - q: "UPDATE requirements SET parent_id = <id of its grandchild> WHERE code = 'AUTH'. What does PostgreSQL do with this schema?"
    options: ["Rejects it with requirements_not_own_parent", "Rejects it with requirements_parent_fkey", "Accepts it; the store's recursive check must catch it"]
    answer: 2
    why: "The parent exists in the same project and isn't the row itself, so every constraint holds. Only a walk up the ancestors finds the loop."
  - q: "A requirement has two children. DELETE FROM requirements WHERE id = <it>, and then DELETE FROM projects WHERE id = <its project>. Which succeed?"
    options: ["Both", "Only the project delete", "Neither"]
    answer: 1
    why: "Deleting the parent alone leaves children pointing at nothing and fails with 23503. The project cascade deletes parent and children in one statement, and the check at the end of it finds no orphans (provided no test runs block it)."
  - q: "TC-4 is linked to a requirement and has no runs. Which coverage does TraceLite report if the requirement's other test passed?"
    options: ["passing", "incomplete", "uncovered"]
    answer: 1
    why: "A linked test without a passing latest run makes the requirement incomplete; bool_and would have said passing because it skips the NULL."
further_reading:
  - title: "PostgreSQL docs: Constraints (foreign keys, MATCH SIMPLE, RESTRICT vs NO ACTION)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: CREATE TABLE"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
  - title: "PostgreSQL docs: UUID functions (uuidv7)"
    url: "https://www.postgresql.org/docs/current/functions-uuid.html"
  - title: "PostgreSQL docs: WITH queries, SEARCH and CYCLE"
    url: "https://www.postgresql.org/docs/current/queries-with.html"
  - title: "PostgreSQL docs: EXPLAIN (GENERIC_PLAN)"
    url: "https://www.postgresql.org/docs/current/sql-explain.html"
  - title: "PostgreSQL docs: Collation support"
    url: "https://www.postgresql.org/docs/current/collation.html"
---

## Concept

TraceLite stores four things, all inside a **project**:

- **requirements**, a tree: each row has an optional `parent_id` pointing at another
  requirement of the same project. A human-readable `code` (`AUTH-LOGIN`) is unique per
  project; the `id` is what URLs and foreign keys use.
- **test_cases**, a flat list per project, with the same kind of `code` (`TC-1`).
- **trace_links**, the many-to-many relation "this test case verifies this requirement".
  A test can verify several requirements, and a requirement can be verified by several
  tests.
- **test_runs**, the evidence: one row per execution of a test case, with a `result` of
  `passed`, `failed` or `skipped`. Only the latest run of a test case (by `finished_at`,
  then `id`) counts for coverage.

The schema enforces everything that a single row or a single foreign key can express:
code format, lengths in characters, allowed values, uniqueness per project, and "both
ends of every reference live in the same project". What it can't express (no loops in
the tree) is checked by one recursive query in the store, inside a transaction.

## Coming from C/C++/Java/Python/JS

- **JPA/Hibernate**: `@ManyToOne` to the same entity plus `@OneToMany(mappedBy)` gives the
  same adjacency list, but walking it loads one level per query. Here the walk is one
  `WITH RECURSIVE` statement, and the tree is never loaded into memory to be checked.
- **Django/Rails**: model validations (`validates :code, uniqueness: true`) run in the
  application and race under concurrency; a unique constraint can't. TraceLite keeps
  the Go checks for friendly messages and lets the database have the last word.
- **ORM-generated migrations**: Django and Prisma diff models to produce DDL. Here the
  migration is SQL you write and review, and constraint names are chosen, because Go
  code switches on them.
- **C/C++**: a `parent` pointer in a struct with no owner of the invariant. The foreign
  key is the invariant "every parent pointer points at a live node of the same project",
  enforced on every write.

## How it works

The canonical migration, `migrations/001_init.sql`. Example 1 runs its first two tables
verbatim, example 3 its last two; later steps add tables and columns in migrations 002
and up, never by editing this file.

```sql
-- +goose Up
CREATE TABLE projects (
  id         uuid PRIMARY KEY DEFAULT uuidv7(),
  name       text NOT NULL CONSTRAINT projects_name_length CHECK (char_length(name) BETWEEN 1 AND 200),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE requirements (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  project_id  uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
  parent_id   uuid,  -- NULL: top level
  code        text COLLATE "C" NOT NULL CONSTRAINT requirements_code_format
                CHECK (code ~ '^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$' AND char_length(code) <= 40),
  title       text NOT NULL CONSTRAINT requirements_title_length CHECK (char_length(title) BETWEEN 1 AND 200),
  description text NOT NULL DEFAULT '',
  status      text NOT NULL DEFAULT 'draft'
                CONSTRAINT requirements_status_valid CHECK (status IN ('draft', 'approved', 'obsolete')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT requirements_code_key UNIQUE (project_id, code),
  CONSTRAINT requirements_in_project UNIQUE (project_id, id),  -- target of same-project FKs
  CONSTRAINT requirements_not_own_parent CHECK (parent_id <> id),
  CONSTRAINT requirements_parent_fkey FOREIGN KEY (project_id, parent_id)
    REFERENCES requirements (project_id, id)
);
CREATE INDEX requirements_parent ON requirements (parent_id);
CREATE TABLE test_cases (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  project_id  uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
  code        text COLLATE "C" NOT NULL CONSTRAINT test_cases_code_format
                CHECK (code ~ '^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$' AND char_length(code) <= 40),
  title       text NOT NULL CONSTRAINT test_cases_title_length CHECK (char_length(title) BETWEEN 1 AND 200),
  description text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT test_cases_code_key UNIQUE (project_id, code),
  CONSTRAINT test_cases_in_project UNIQUE (project_id, id)
);
CREATE TABLE trace_links (  -- "this test case verifies this requirement"
  project_id     uuid NOT NULL,
  requirement_id uuid NOT NULL,
  test_case_id   uuid NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, requirement_id, test_case_id),
  CONSTRAINT trace_links_requirement_fkey FOREIGN KEY (project_id, requirement_id)
    REFERENCES requirements (project_id, id) ON DELETE CASCADE,
  CONSTRAINT trace_links_test_case_fkey FOREIGN KEY (project_id, test_case_id)
    REFERENCES test_cases (project_id, id) ON DELETE CASCADE
);
CREATE INDEX trace_links_test_case ON trace_links (project_id, test_case_id);
CREATE TABLE test_runs (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  test_case_id uuid NOT NULL CONSTRAINT test_runs_test_case_fkey REFERENCES test_cases ON DELETE RESTRICT,
  result       text NOT NULL CONSTRAINT test_runs_result_valid CHECK (result IN ('passed', 'failed', 'skipped')),
  finished_at  timestamptz NOT NULL DEFAULT now(),
  duration_ms  integer CONSTRAINT test_runs_duration_ms_valid CHECK (duration_ms >= 0),
  notes        text NOT NULL DEFAULT ''
);
CREATE INDEX test_runs_latest ON test_runs (test_case_id, finished_at DESC, id DESC);
```

Which query each index serves:

- `requirements_code_key (project_id, code)`: the flat list, paged by code (keyset), and
  lookups by code (imports, links by code).
- `requirements_parent (parent_id)`: children of a node (the tree walk) and the
  foreign-key check when a requirement is deleted.
- `trace_links` primary key `(project_id, requirement_id, test_case_id)`: the trace
  matrix of a project and the tests of one requirement. `trace_links_test_case`: the
  requirements of one test, and the check when a test case is deleted.
- `test_runs_latest`: the latest run per test case, and a test case's runs newest first.

Columns that are never absent are `NOT NULL` with a default (`description` is `''`, not
`NULL`), so the API never has to tell "missing" from "empty". The only nullable columns
are `parent_id` (top level) and `duration_ms` (a skipped run has none). Rows have no
`updated_at` trigger: the store's `UPDATE` statements set `updated_at = now()`
themselves; step 13 compares triggers with application events for the audit trail.

## Design notes

**Keys.** `uuid DEFAULT uuidv7()` rather than `bigint GENERATED ALWAYS AS IDENTITY`,
following the ids lesson's rule for ids that leave the database: they appear in URLs and
JSON, the step 7 importer can assign them before rows exist, and they don't reveal how
many requirements a project has. Version 7 keeps inserts at the end of the index. People
never type ids; they type codes, which is why `code` exists separately and is unique per
project, not globally. Go reads uuid columns into `uuid.UUID` from the standard library
(Go 1.27); TypeScript sees branded strings (step 5).

**One project, enforced.** Carrying `project_id` into `trace_links` and making both
foreign keys composite costs a column and two unique constraints; the alternative, a
trigger or a check in Go before every insert, is code that must run in every writer and
races under concurrency. Same for `parent_id`: the composite key makes "parent in
another project" impossible, and the Go service only translates the error.

**What the database can't do.** No constraint sees a loop of two or more rows.
Rather than a trigger that walks the tree on every update, TraceLite has exactly one
operation that changes `parent_id` after creation (moving a requirement) and runs the
walk there, in a transaction that serializes moves per project (step 3).

**Deletes.** A requirement with children can't be deleted (409 in the API); a project
cascades through its requirements, test cases and links, but stops at test runs. That
asymmetry is deliberate: structure can be rebuilt, evidence can't. Steps that add tables
(jobs, audit) decide their own `ON DELETE` with the same question: is the child part of
the parent, or a record about it?

**Alternatives rejected.** `ltree` or a closure table would make subtree reads cheaper,
at the cost of rewriting paths or closure rows on every move; TraceLite's trees are small
and recursive CTEs are enough. A PostgreSQL `ENUM` for `status` and `result` would give
sqlc a Go type for free, but values can't be removed later; `text` plus a named `CHECK`
is changed by a migration like any other constraint.
