---
id: fullstack.tracelite.step-13-audit
title: "Step 13: Audit history: database trigger vs application events, and the trade-offs"
track: fullstack
module: tracelite
order: 13
level: advanced
minutes: 15
since: pg18
prerequisites: [fullstack.tracelite.step-12-search, sql.server-side.triggers, sql.json.jsonb-operators, fullstack.tracelite.step-09-realtime]
compare_with: [design.architecture.business-rules, sql.modeling.denormalize, sql.modeling.temporal, go.postgres.transactions]
tags: [tracelite, audit, triggers, jsonb, set_config, current_setting, connection-pool, outbox, copy, design-trade-offs]
summary: >
  A row trigger records every change from every writer as a jsonb diff, with the user
  named per transaction through set_config. Application events record intent but only
  from code that remembers to write them. TraceLite keeps the trigger and lets the
  service label its changes.
examples:
  - id: trigger-audit
    file: examples/01_trigger_audit.sql
    title: "Migration 006: one trigger function, the actor from set_config, diffs as jsonb"
    runtime: sql
    check: exact
    annotations:
      - lines: "12-23"
        note: >
          One table for the history of requirements, test cases and links.
          Test runs aren't audited: they are insert-only evidence already, and
          step 2's RESTRICT keeps them.
      - lines: "14-15"
        note: >
          now() is the transaction's start time, the same for every row it
          writes. xact (pg_current_xact_id, PG 13) groups those rows, so the
          UI can show "one save changed these four things".
      - lines: "18"
        note: >
          No foreign key, unlike everything else in TraceLite. Step 2's
          question, whether the child is part of the parent or a record about
          it, has the second answer here: the record of a deleted project must
          survive the delete. row_id has no foreign key either, which is why
          the history at the end outlives AUTH-RESET.
      - lines: "29-30"
        note: >
          to_jsonb(OLD) and to_jsonb(NEW) make one function work for every
          table. The timestamps are dropped: the audit row has its own, and an
          updated_at in every diff would hide what really changed.
      - lines: "33-37"
        note: >
          The diff keeps only the members whose values differ, each as [old,
          new]. A save that changed nothing but updated_at produces no row, so
          the history shows edits, not clicks on Save.
      - lines: "39"
        note: >
          The user comes from a setting the application puts in the
          transaction. current_setting with missing_ok true returns NULL if it
          was never set, but '' once any transaction on this connection has set
          it (example 3), hence nullif. Without a user, the database login
          stands in, marked db:.
      - lines: "40"
        note: >
          The bridge between the two designs: the service may name what it is
          doing (move, import), and the trigger stores that label next to the
          facts.
      - lines: "47-50"
        note: >
          AFTER ROW triggers on the audited tables. On the same table and
          event, triggers fire in alphabetical order of their names, so
          requirements_audit runs before step 9's requirements_notify; both
          belong to the same transaction.
      - lines: "50"
        note: >
          trace_links has no id; TG_ARGV names the column that identifies the
          row in the history, and a link belongs to its requirement's.
      - lines: "54"
        note: >
          set_config(name, value, is_local true) is SET LOCAL as a function:
          it takes the value as a parameter, and it ends with the transaction.
          Store.WithTx sends it first in every transaction (example 3). This
          runner keeps the whole script in one transaction, so each "request"
          below sets its own values.
      - lines: "63-64"
        note: >
          A move, labelled by the service. Moving is just a parent_id change to
          the database; the label is what makes "moved under AUTH-LOGIN"
          readable in the history.
      - lines: "67"
        note: >
          The same title saved again: at most updated_at differs, which the
          function ignores, so it returned without a row. Three UPDATEs, two
          entries in the history.
      - lines: "70-71"
        note: >
          One DELETE, and the foreign key's cascade removed the link too. Both
          rows are recorded under Ayşe's name: a cascade runs as ordinary row
          deletes, triggers included.
      - lines: "74-75"
        note: >
          The detail page's history: the full row for an insert or a delete,
          [old, new] pairs for updates. The link's rows appear because their
          row_id is the requirement, and all of it is still there after the
          requirement is gone.
  - id: two-designs
    file: examples/02_two_designs.sql
    title: "The same changes, recorded by a trigger and by application events: what each one misses"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-7"
        note: >
          The rows exist before the trigger does, so both designs start with
          zero records.
      - lines: "10-21"
        note: >
          Example 1's function without the label and the timestamp handling:
          the user, the code, and for an UPDATE the members that changed, as
          [old, new].
      - lines: "24-29"
        note: >
          The application-events design: the service inserts one row per use
          case, in the same transaction as the change (step 11's outbox idea),
          with names a person can read: kind, the code at the time, and the
          parts of the change that matter.
      - lines: "32-34"
        note: >
          After each step, mark stores how many records each design holds. A
          step that a design missed leaves its number where it was.
      - lines: "37-41"
        note: >
          A use case through the service: both designs record it, the trigger
          as a diff, the service as an event.
      - lines: "44-46"
        note: >
          A fix in psql, or a migration's data fix, is plain DML: the trigger
          records both rows, under the login role; the service never ran, so
          its log has nothing.
      - lines: "49"
        note: >
          ADD COLUMN with a constant default fires nothing at all: since PG 11
          the default is kept in the catalog, and no row is written.
      - lines: "53-57"
        note: >
          Both designs live in the transaction, so both lose the attempt when
          it rolls back. Recording failed attempts means writing outside the
          transaction, which then also records things that never happened.
      - lines: "60"
        note: >
          TRUNCATE removes rows without firing ON DELETE row triggers. An
          ON TRUNCATE statement trigger could record that it happened, not what
          was removed.
      - lines: "63"
        note: >
          The tally: the trigger missed only what bypasses DML (DDL,
          TRUNCATE); the service missed every write that didn't go through it.
      - lines: "64-65"
        note: >
          The same move, twice. The trigger knows the fact (parent_id went from
          one uuid to another); the event knows the meaning (moved from AUTH
          to AUTH-LOGIN), with codes, which stay readable after the requirement
          is deleted.
  - id: actor
    file: examples/03_actor.go
    title: "Naming the user through a connection pool: session settings leak, transaction settings don't"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "18-26"
        note: >
          The one place where the user reaches the database. Every store
          method that writes runs inside WithTx, so every audited write
          carries a user; a single autocommitted statement outside it would
          not.
      - lines: "21"
        note: >
          A bind parameter, so a user name can't change the SQL. SET has no
          parameters; people who use it build the statement with Sprintf.
      - lines: "32"
        note: >
          One connection makes the reuse certain. With a pool of 4 under load,
          the same thing happens to some requests, which is worse: the wrong
          name appears only sometimes.
      - lines: "33"
        note: >
          The trigger in this example reports what it would record as a notice
          (the schema is at the end); pgx hands every notice to this function,
          so each record prints under the request that made it.
      - lines: "39-44"
        note: >
          After WithTx commits, the setting is gone: the next write, from a
          path that named no one, is recorded as db:playground, a visible gap
          instead of a wrong name.
      - lines: "45-47"
        note: >
          The setting now exists on this connection with the value '', not
          NULL, which is why the trigger wraps it in nullif.
      - lines: "49-52"
        note: >
          The tempting version: SET names the user for the session. The
          setting stays on the connection after Ayşe's request returns it to
          the pool, and the next request's change is booked to her. Undoing it
          on every release (pgxpool has an AfterRelease hook) is a rule that
          must never be forgotten; a transaction-local setting needs no
          cleanup.
use_cases:
  - "Change history on a detail page: who changed the title, when, from what to what"
  - "Compliance questions (who approved this requirement, who removed that link) that must be answered for writes from psql, imports and migrations too"
  - "Attributing changes in a pooled-connection application without a database role per user"
  - "Deciding, for any cross-cutting record (audit, notifications, search indexing), whether the database or the application should produce it"
strengths:
  - "The trigger sees every row written by INSERT, UPDATE, DELETE, COPY and cascades, from any client: a forgotten code path shows up as a db: entry instead of a hole"
  - "Diffs are computed from OLD and NEW, so they are exact; an application event says what the code believed it changed"
  - "set_config(…, true) scopes the user to one transaction, which is the right lifetime with a connection pool; SET would leak to the next borrower"
  - "Audit rows commit and roll back with the change, so the log never claims a change that didn't happen"
weaknesses:
  - "A row trigger runs once per row: a 10,000-row import writes 10,000 audit rows in 10,000 function calls; statement-level triggers with transition tables would batch them"
  - "The trigger records facts, not intent: a move is a parent_id change between two uuids until the service labels it"
  - "DDL, TRUNCATE, disabled triggers and session_replication_role = replica bypass it, and a table owner can disable it"
  - "Nothing records attempts that rolled back, in either design"
pitfalls:
  - title: "SET instead of set_config(…, true) on a pooled connection"
    detail: "A session-level setting outlives the request and is seen by the next one that borrows the connection, so its changes are attributed to the previous user. Set the user transaction-locally, first thing in every transaction."
    bad: "conn.Exec(ctx, fmt.Sprintf(\"SET tracelite.actor = '%s'\", user))"
    good: "tx.Exec(ctx, \"SELECT set_config('tracelite.actor', $1, true)\", user)"
  - title: "Testing current_setting(…, true) only for NULL"
    detail: "Once a custom setting has been set on a connection, it exists there with the value '' after the transaction ends, and missing_ok no longer returns NULL. Treat '' as unset."
    bad: "coalesce(current_setting('tracelite.actor', true), 'unknown')"
    good: "coalesce(nullif(current_setting('tracelite.actor', true), ''), 'db:' || session_user)"
  - title: "Writes outside the transaction helper"
    detail: "A single statement sent through the pool, outside WithTx, runs in its own transaction where nobody set the user. Route every write through the helper, and treat db: entries in the log as bugs to find."
  - title: "Auditing with BEFORE triggers"
    detail: "A BEFORE trigger runs before constraints are checked and before later BEFORE triggers may change the row, so it can record values that are never stored. Audit in AFTER triggers, which see the row as written."
  - title: "Copying secrets into the log"
    detail: "to_jsonb(NEW) copies every column. Don't attach the generic trigger to tables like webhooks, whose secret would end up in the history; or remove such members in the function."
quiz:
  - q: "TraceLite's pool has 4 connections. A middleware runs SET tracelite.actor = <user> once per request. What goes wrong?"
    options: ["SET fails inside a pool", "A later request on the same connection that doesn't set the user is recorded under the previous user's name", "Nothing: SET is per request"]
    answer: 1
    why: "SET lasts for the session, which outlives the request; set_config(…, true) ends with the transaction."
  - q: "A bulk load COPYs rows straight into requirements. Which design records them?"
    options: ["Only the trigger: COPY FROM fires row triggers", "Only application events", "Neither: COPY bypasses both"]
    answer: 0
    why: "COPY FROM invokes the table's triggers and constraints; with application events, the loader would need to write its own event."
  - q: "A migration runs ALTER TABLE requirements ADD COLUMN priority int NOT NULL DEFAULT 3. What does the audit trigger record?"
    options: ["One UPDATE per row", "Nothing: no row is updated", "One row for the whole table"]
    answer: 1
    why: "DDL fires no row triggers, and a constant default is stored in the catalog rather than written into each row."
  - q: "Why does the audit_log have no foreign key to projects?"
    options: ["Foreign keys slow down triggers", "The history must survive the project's deletion", "uuid columns can't have foreign keys"]
    answer: 1
    why: "It is a record about the project, not part of it; with CASCADE it would vanish, with RESTRICT no project could ever be deleted."
further_reading:
  - title: "PostgreSQL docs: Configuration settings functions (set_config, current_setting)"
    url: "https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADMIN-SET"
  - title: "PostgreSQL docs: Overview of trigger behavior (ordering, row vs statement)"
    url: "https://www.postgresql.org/docs/current/trigger-definition.html"
  - title: "PostgreSQL docs: CREATE TRIGGER (transition tables)"
    url: "https://www.postgresql.org/docs/current/sql-createtrigger.html"
  - title: "PostgreSQL docs: COPY (triggers and constraints)"
    url: "https://www.postgresql.org/docs/current/sql-copy.html"
  - title: "PostgreSQL docs: TRUNCATE"
    url: "https://www.postgresql.org/docs/current/sql-truncate.html"
---

## Concept

The question "who changed this requirement, and what did it say before?" has two honest
answers.

**A database trigger** (migration 006). An `AFTER INSERT OR UPDATE OR DELETE … FOR EACH
ROW` trigger on `requirements`, `test_cases` and `trace_links` writes one `audit_log`
row per changed row, with the diff as jsonb. The database doesn't know TraceLite's
users, so the application tells it: the first statement of every transaction is
`set_config('tracelite.actor', $1, true)`, and the trigger reads it back with
`current_setting`.

**Application events.** The service writes a `change_events` row for each use case
(`requirement.moved`, `requirement.renamed`) in the same transaction as the change, with
the user it already knows and the data a person wants to read.

They fail differently:

| | Trigger | Application events |
|---|---|---|
| Writes through the service | yes, as diffs | yes, as named use cases |
| Rows removed by a foreign key's `ON DELETE CASCADE` | yes, each one | only what the use case names |
| psql fixes, bulk SQL, migrations' data fixes | yes, user shown as `db:<role>` | no |
| COPY, and step 7's import (COPY to staging, then INSERT … SELECT) | yes: COPY FROM fires row triggers | only if the importer writes one |
| DDL (`ADD COLUMN … DEFAULT`), `TRUNCATE` | no | no |
| Rolled-back transactions | no (rolled back with them) | no (same) |
| Intent ("moved under AUTH-LOGIN") | only if the service labels it | yes |

TraceLite keeps the trigger as the record of what happened, because a history that
misses whole categories of writes can't answer an auditor. It borrows the one strength
of events: the service may label a change (`set_config('tracelite.action', 'move',
true)`), and the trigger stores the label next to the diff.

## Coming from C/C++/Java/Python/JS

- **Java/Hibernate Envers, Spring Data auditing**: entity listeners write revisions,
  which is the application-events design generated for you. They see what goes through
  the ORM, and nothing else. `@CreatedBy` comes from an `AuditorAware` bean, usually
  reading Spring Security's thread-local context; here the user travels in the
  `context.Context` and then in the transaction.
- **Python/django-simple-history, Rails/PaperTrail**: same model, same blind spots
  (`queryset.update()` in Django skips save signals, `update_all` in Rails skips
  callbacks). A trigger can't be skipped by choosing a different API call.
- **JS/Prisma middleware or extensions**: the same again, per client instance.
- **Change data capture (Debezium, logical decoding)**: reads the WAL, so it sees
  every committed change, like the trigger, without slowing the writer. It runs as a
  separate service and still can't name the application user unless the application
  writes it somewhere: the same problem `set_config` solves here.

## How it works

`migrations/006_audit.sql` is example 1's middle part. The store side, in
`internal/store/store.go`:

```go
// WithTx: step 3's helper; the actor comes from the request's context.
return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, "SELECT set_config('tracelite.actor', $1, true)", ActorFrom(ctx)); err != nil {
		return err
	}
	return fn(s.q.WithTx(tx))
})
```

`ActorFrom` reads what a middleware put into the request's context. In a single-user
local install that is the owner's name from configuration (step 15); with
authentication, the signed-in user. A use case that wants a label sets
`tracelite.action` the same way inside its closure (`move` for moves, `import` for step
7's importer).

Bulk loads need nothing extra. PostgreSQL's COPY reference says that `COPY FROM` invokes
the destination table's triggers and check constraints, so a pgx `CopyFrom` inside
`WithTx` is recorded row by row under the loading user. Step 7's importer copies into a
staging table and then runs `INSERT … SELECT` into `requirements`, which the trigger sees
like any other insert.

The history endpoint, in `openapi.yaml`:

```yaml
  /api/requirements/{requirementId}/history:
    parameters: [{ $ref: "#/components/parameters/RequirementId" }]
    get:
      operationId: getRequirementHistory
      parameters: [{ $ref: "#/components/parameters/Limit" }, { $ref: "#/components/parameters/Cursor" }]
      responses:
        "200":
          description: Newest first, keyset-paged by id
          content:
            application/json:
              schema:
                type: object
                required: [items, nextCursor]
                properties:
                  items:
                    type: array
                    items:
                      type: object
                      required: [at, actor, action, table, op, changes]
                      properties:
                        at: { type: string, format: date-time }
                        actor: { type: string }
                        action: { type: string }
                        table: { type: string, enum: [requirements, test_cases, trace_links] }
                        op: { type: string, enum: [INSERT, UPDATE, DELETE] }
                        changes: { type: object }   # the full row, or {column: [old, new]}
                  nextCursor: { type: [string, "null"] }
        "404": { $ref: "#/components/responses/Problem" }
```

It reads `audit_log` by `row_id` through `audit_log_row`, so it works for deleted
requirements too, as long as the client still has the id. The keys inside `changes` are
column names, snake_case, because they describe the table and not the wire format; the
detail page maps the few it shows (`title`, `status`, `parent_id`) to Turkish labels.

```mermaid
sequenceDiagram
  participant H as Handler
  participant S as Store.WithTx
  participant P as PostgreSQL
  H->>S: MoveRequirement(ctx with actor "mehmet")
  S->>P: BEGIN
  S->>P: SELECT set_config('tracelite.actor', 'mehmet', true)
  S->>P: SELECT set_config('tracelite.action', 'move', true)
  S->>P: UPDATE requirements SET parent_id = ...
  P->>P: requirements_audit: INSERT INTO audit_log (actor mehmet, action move, diff)
  P->>P: requirements_notify (step 9): pg_notify, queued until commit
  S->>P: COMMIT (both, or neither)
```

## Design notes

**Same choice as step 9, different weights.** Step 9 chose a trigger for change
notifications because every writer must notify, and a lost or extra notification costs
only a refetch. For the audit trail the "every writer" argument is stronger still (a
history with holes is worse than none, because it is believed), and the cost of the
trigger's weakness, missing intent, is softened by the action label. If TraceLite's
history were mainly a product feature ("Ayşe moved this under AUTH-LOGIN") rather than a
record, application events would win, and a trigger could still backstop them.

**Row triggers and bulk writes.** A row trigger costs one function call and one insert
per changed row. For TraceLite's imports (thousands of rows, rarely) that is acceptable.
At larger scale, statement-level triggers with transition tables (`REFERENCING OLD TABLE
AS … NEW TABLE AS …`) write the whole statement's audit rows in one `INSERT … SELECT`;
the price is one trigger per event, because a trigger with transition tables can't be
attached to several events at once.

**What the trigger can't stop.** A table owner can `ALTER TABLE … DISABLE TRIGGER`, and
`session_replication_role = replica` disables ordinary triggers for a session (it needs
superuser or a granted SET privilege). The application's role owns the tables (it runs
the migrations), so an attacker with its password could bypass the audit. A stricter
setup migrates with an owner role, runs the application as a role without ownership, and
grants it only INSERT on `audit_log`: then the application can't rewrite history either.
TraceLite, installed on one machine for one team, doesn't go that far; it says so in its
installation notes.

**Retention.** Audit rows are small, but they are never updated and rarely read.
Requirements change slowly enough that years of history fit easily; if a project ever
needed limits, a range partition per year (sql.modeling.temporal) lets old years be
detached and archived without a `DELETE`.
