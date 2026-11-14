---
id: sql.modifying.soft-delete
title: "Soft-delete trade-offs"
track: sql
module: modifying
order: 3
level: intermediate
minutes: 13
since: pg10
prerequisites: [sql.modifying.returning-upsert, sql.querying.null-logic]
compare_with: [sql.modeling.temporal, sql.security-ops.rls, design.paradigms.immutability]
tags: [soft-delete, partial-index, views, foreign-keys, privacy]
summary: >
  A deleted_at column turns DELETE into UPDATE and keeps the row. In exchange,
  every query, unique constraint and foreign key has to learn what "deleted"
  means, and the data is still there when the law says it must go.
examples:
  - id: flag
    file: examples/01_flag.sql
    title: "deleted_at instead of DELETE, and the first two things it breaks"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          A nullable timestamptz rather than a boolean: it records when, and
          NULL means live. Being NULL-based, the filter is IS NULL; a
          deleted_at = NULL test would match nothing.
      - lines: "10-11"
        note: >
          Soft delete is an UPDATE: it writes a new row version, fires UPDATE
          triggers rather than DELETE ones, and leaves every byte of personal
          data in place.
      - lines: "15-16"
        note: >
          The table's default answer is now wrong. Each query, report, export
          and admin screen must add the filter, and the one that forgets
          shows deleted accounts without any error.
      - lines: "20"
        note: >
          UNIQUE (email) covers deleted rows too, so the old account blocks
          the new one (SQLSTATE 23505). The next example narrows it.
  - id: partial-unique
    file: examples/02_partial_unique.sql
    title: "A partial unique index and a view of live rows"
    runtime: sql
    check: exact
    annotations:
      - lines: "7"
        note: >
          A partial index holds only rows matching its WHERE, so uniqueness
          applies among live rows only. It's an index, not a constraint:
          ON CONFLICT must repeat the predicate to use it
          (ON CONFLICT (email) WHERE deleted_at IS NULL), and a foreign key
          can't reference it.
      - lines: "10-11"
        note: >
          The view puts "live" in one place. The app reads live_users and can
          no longer forget the filter. A simple view like this is also
          updatable, so UPDATEs through it work too.
      - lines: "17"
        note: >
          Selin's old row has deleted_at set, so it isn't in the index, and the
          new row doesn't collide with it.
      - lines: "23"
        note: >
          Among live rows the rule still holds. The message says "unique
          constraint" but names the partial index, users_email_live: no
          constraint exists, only the index.
  - id: foreign-keys
    file: examples/03_foreign_keys.sql
    title: "Foreign keys don't know about deleted_at"
    runtime: sql
    check: exact
    annotations:
      - lines: "19-20"
        note: >
          A foreign key checks that the parent row exists, and a soft-deleted
          row exists. CASCADE never fires, and order 13 is accepted for a
          customer the app considers gone.
      - lines: "23-27"
        note: >
          The report counts Can's three orders as if nothing happened. Whether
          that's right (revenue happened) or wrong (active customers) is a
          decision every join now has to make explicitly.
      - lines: "31-33"
        note: >
          Anonymizing keeps the row, so orders and invoices stay intact, but
          removes what identifies the person. That is often the honest version
          of "delete my account" when records must be retained.
      - lines: "36-37"
        note: >
          A real DELETE is what ON DELETE CASCADE was declared for: Can's
          orders go with him, in the same statement.
use_cases:
  - "Undo windows: a user can restore something for 30 days before a purge job deletes it for real"
  - "Records that others reference and that must outlive deactivation (customers on invoices), usually combined with anonymization"
  - "Sync to offline or mobile clients, which must learn that a row was deleted (a tombstone) instead of simply never seeing it again"
strengths:
  - "Restoring is an UPDATE, while undoing a hard DELETE means restoring a backup"
  - "Child rows keep a valid parent, so history (orders, invoices) stays joinable without an archive"
  - "Partial unique and partial regular indexes (WHERE deleted_at IS NULL) keep uniqueness and hot-path indexes about live rows only"
weaknesses:
  - "Every query must filter; the forgotten filter fails silently by showing deleted data, unlike a missing row, which fails loudly"
  - "Constraints stop meaning what they say: UNIQUE needs a partial index, and foreign keys and ON DELETE actions ignore the flag"
  - "Deleted rows never leave: tables and non-partial indexes keep growing, and personal data stays stored"
pitfalls:
  - title: "Treating the flag as erasure"
    detail: "GDPR, and in Turkey KVKK (Law No. 6698, Article 7), can oblige you to erase, destroy or anonymize personal data. A deleted_at flag hides the data; it doesn't remove it from the table, the backups or the replicas. Plan a purge or an anonymizing UPDATE."
  - title: "A plain UNIQUE on a soft-deleted table"
    detail: "Deleted rows keep their values, so re-registering with the same e-mail fails. Use a partial unique index over live rows."
    bad: "email text NOT NULL UNIQUE"
    good: "CREATE UNIQUE INDEX users_email_live ON users (email) WHERE deleted_at IS NULL;"
  - title: "Filtering in some queries but not others"
    detail: "Point the application at a view (live_users) or, for stronger guarantees, row-level security, so forgetting the filter is impossible rather than unlikely."
  - title: "Partial indexes and parameters"
    detail: "The planner uses a partial index only when the query's WHERE provably implies the predicate at planning time. Write deleted_at IS NULL literally (or read through the view); a parameter like deleted_at IS NOT DISTINCT FROM $1 won't match."
quiz:
  - q: "users has UNIQUE (email) and a deleted_at column. Selin deletes her account (deleted_at set) and signs up again. What happens?"
    options:
      - "The sign-up works: deleted rows are ignored by constraints"
      - "Unique violation: the constraint still sees her old row"
      - "The old row is replaced"
    answer: 1
    why: "Constraints know nothing about the flag. A partial unique index WHERE deleted_at IS NULL limits uniqueness to live rows."
  - q: "orders references customers ON DELETE CASCADE. A customer is soft-deleted. What happens to the orders?"
    options:
      - "Deleted, by the cascade"
      - "Nothing: the parent row still exists, so no foreign key action fires"
      - "They are soft-deleted too"
    answer: 1
    why: "Foreign key actions react to DELETE and UPDATE of the key, not to other columns changing."
  - q: "A customer asks you to erase their personal data under KVKK. Their orders must be kept for tax reasons. What fits?"
    options:
      - "Set deleted_at; the data is hidden from the app"
      - "Keep the row and replace the personal columns with NULL or placeholders (anonymize)"
      - "DELETE the customer and let CASCADE remove the orders"
    answer: 1
    why: "The flag hides but retains the data; a cascade destroys records you must keep. Anonymizing keeps the references and removes the person."
further_reading:
  - title: "PostgreSQL docs: Partial Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-partial.html"
  - title: "PostgreSQL docs: Views (CREATE VIEW, updatable views)"
    url: "https://www.postgresql.org/docs/current/sql-createview.html"
  - title: "PostgreSQL docs: Constraints (foreign keys and ON DELETE)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL docs: Row Security Policies"
    url: "https://www.postgresql.org/docs/current/ddl-rowsecurity.html"
---

## Concept

Soft delete replaces `DELETE` with `UPDATE ... SET deleted_at = now()`. The row stays;
"deleted" becomes a convention that every reader has to honor. Teams pick it for undo,
for audit, and to keep history (orders, invoices) attached to a customer who left.

The price is that the database's own guarantees no longer line up with what the
application means:

- **Queries** see deleted rows unless each one filters them out.
- **Unique constraints** count deleted rows; you need a partial unique index.
- **Foreign keys** check only that the parent row exists. `ON DELETE` actions never
  fire, and new children can attach to a "deleted" parent.
- **Storage and privacy**: the data, personal data included, never leaves.

## Coming from C/C++/Java/Python/JS

- **ORMs**: Rails' paranoia/discard, Django packages, Hibernate's `@SQLDelete` and
  `@SQLRestriction` add the filter for you, but only to queries that go through the ORM.
  Raw SQL, reports, BI tools and psql still see every row. A view or row-level security
  enforces the rule in the database, where all of them pass.
- **Go and TS**: there is no ORM magic to lean on with pgx or a query builder, so the
  filter lives in each hand-written query. That is the strongest argument for a view
  such as `live_users` from day one.
- **Garbage collection**: soft-deleted rows are like objects you never release because
  something might still point to them. A retention policy and a purge job are the GC.

## Design notes

First decide what "deleted" means in your domain. Often it's really a state
(`closed`, `cancelled`, `archived`) that deserves a `status` column and its own rules,
not a generic flag. When it is deletion, the usual alternatives are:

- **Hard delete plus archive**: move the row to an `_archive` table in the same
  statement (`WITH gone AS (DELETE ... RETURNING *) INSERT INTO ... SELECT * FROM gone`)
  so the live table keeps honest constraints.
- **Anonymize**: keep the row for its references, wipe the personal columns. This is
  how a "delete my account" request can coexist with records you must retain.
- **History tables** (modeling module) when you need what was true at any point in time,
  not just "gone or not".

If you do soft-delete, make it complete: a partial unique index, a view for live rows,
indexes with `WHERE deleted_at IS NULL` on hot paths, an explicit decision per foreign
key, and a purge job with a retention period. Privacy law sets the hard limit: KVKK
Article 7 requires personal data to be erased, destroyed or anonymized once the
reasons for processing it no longer exist, and a flag does none of those.
