---
id: sql.server-side.triggers
title: "Triggers: an audit trail and updated_at; why business logic mostly belongs in the app"
track: sql
module: server-side
order: 3
level: advanced
minutes: 15
since: pg11
prerequisites: [sql.server-side.functions]
compare_with: [sql.subqueries-ctes.modifying, sql.modifying.soft-delete, fullstack.tracelite.step-13-audit, design.architecture.business-rules]
tags: [triggers, plpgsql, audit, updated-at, transition-tables, soft-delete]
summary: >
  A trigger runs a function on every INSERT, UPDATE or DELETE, whoever issues
  it: ideal for mechanical invariants such as updated_at and an audit trail.
  The same property makes it hidden control flow, so business decisions belong
  in the application, where callers can see and test them.
examples:
  - id: updated-at
    file: examples/01_updated_at.sql
    title: "updated_at with a BEFORE ROW trigger, skipping no-op updates"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-21"
        note: >
          A BEFORE ROW trigger receives the proposed row as NEW and returns the
          row to store. Assigning NEW.updated_at changes what gets written: no
          second UPDATE, no recursion.
      - lines: "18"
        note: >
          now() is the transaction's start time, the same for every row the
          transaction touches. That suits "changed together", and it is why
          the final SELECT can compare with now(). Use clock_timestamp() if
          you need the moment of each write.
      - lines: "23-27"
        note: >
          WHEN is evaluated before the function is called. OLD.* IS DISTINCT
          FROM NEW.* (the docs' own idiom) skips updates that change nothing,
          so re-saving an unchanged form doesn't bump updated_at.
      - lines: "29"
        note: >
          A real change: the trigger stamps the row.
      - lines: "30"
        note: >
          Still UPDATE 1, because the row was rewritten, but no column
          differs, so WHEN skipped the trigger and the old stamp stays.
      - lines: "31"
        note: >
          A client trying to set updated_at itself: the row now differs, the
          trigger fires and overwrites the value. psql sessions, migrations and
          every service get the same rule, which no ORM hook can promise.
  - id: audit
    file: examples/02_audit.sql
    title: "A generic audit trail: one trigger function, a jsonb diff per change"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-10"
        note: >
          An append-only log. at and by_role come from their defaults, inside
          the same transaction as the change they describe.
      - lines: "13-30"
        note: >
          One function for every audited table: TG_OP says what happened and
          TG_TABLE_NAME where. to_jsonb turns a row into a document, so the log
          doesn't have to mirror each table's columns.
      - lines: "17-18"
        note: >
          OLD is NULL for INSERT and NEW is NULL for DELETE (PG 11+), so each
          is read only where it exists.
      - lines: "22-26"
        note: >
          For updates, only the keys whose values differ, as [old, new] pairs:
          the answer to "who changed the status, and from what?" without
          diffing two full snapshots later.
      - lines: "28"
        note: >
          An AFTER ROW trigger's return value is ignored. AFTER rather than
          BEFORE: by then every BEFORE trigger has had its say, so the log
          records the row as actually stored.
      - lines: "32-34"
        note: >
          FOR EACH ROW: once per affected row. For bulk changes, a FOR EACH
          STATEMENT trigger with REFERENCING OLD TABLE / NEW TABLE (transition
          tables) sees all affected rows at once and is cheaper.
      - lines: "36-38"
        note: >
          Nothing in these statements mentions auditing. That is the point,
          and the cost: the behaviour lives in the schema, not in the code
          that issues the write.
      - lines: "41-43"
        note: >
          The audit row is written in the same transaction as the change, so
          undoing the change removes its audit row too: the log never records
          a change that didn't happen.
      - lines: "45"
        note: >
          Four entries; the update is reduced to the one field that changed,
          and the rolled-back rename left no trace.
  - id: hidden-flow
    file: examples/03_hidden_flow.sql
    title: "Hidden control flow: a DELETE that deletes nothing, an UPDATE that updates more"
    runtime: sql
    check: exact
    annotations:
      - lines: "16-23"
        note: >
          A BEFORE ROW trigger that returns NULL cancels the operation for that
          row. This one marks the row as deleted first, so every DELETE on
          requirements quietly becomes an UPDATE.
      - lines: "26-38"
        note: >
          An AFTER ROW trigger that writes to another table: every test case
          update now also updates its requirement, a second write the caller
          never asked for.
      - lines: "41"
        note: >
          DELETE 0: the command tag counts rows actually deleted. A Go handler
          that maps RowsAffected() == 0 to 404 Not Found now reports a missing
          row for one that exists and was just changed.
      - lines: "45"
        note: >
          UPDATE 3 and three NOTICEs: the trigger ran once per row, and each
          run updated the same requirement row again. Row-level AFTER triggers
          fire at the end of the statement, so each run already saw all three
          test cases passed.
      - lines: "46"
        note: >
          REQ-LOGIN flipped to passing as a side effect of a statement about
          test cases.
      - lines: "49-52"
        note: >
          The catalog (or \d in psql) is the only place this logic is visible.
          A code review of the Go handler that runs the DELETE shows none of
          it.
use_cases:
  - "updated_at maintained for every writer: API, psql, migrations, bulk jobs"
  - "An audit trail that no code path can skip, written in the same transaction as the change"
  - "Keeping a denormalized value consistent when a generated column can't express it (it needs another table)"
  - "Announcing changes with NOTIFY or the tcn extension (next lesson)"
  - "INSTEAD OF triggers that make a complex view writable"
strengths:
  - "Unlike JPA @PreUpdate or Django's save() and signals, a trigger also fires for raw SQL, bulk updates and other services; Django's QuerySet.update() skips save() entirely"
  - "Runs in the writing transaction: if the trigger fails, the change fails with it, and a rolled-back change takes its audit rows along"
  - "No extra round trips: the work happens where the row already is"
  - "Statement-level triggers with transition tables process a bulk change in one set-based pass"
weaknesses:
  - "Hidden control flow: the statement you read is not everything that runs, and command tags can lie (DELETE 0 for a soft delete)"
  - "Cost per row on bulk operations: a row trigger on a 1-million-row UPDATE runs its function a million times"
  - "Cascades are unbounded: triggers fire triggers, and avoiding infinite recursion is your job"
  - "Logic in PL/pgSQL is harder to unit-test, debug and log than the same logic in Go or TS"
pitfalls:
  - title: "Returning NULL from a BEFORE trigger by accident"
    detail: "A BEFORE ROW trigger that ends with RETURN NULL skips the row without an error: the statement succeeds with a lower row count. Return NEW (OLD in a DELETE trigger) unless skipping is the point."
    bad: "BEGIN\n  NEW.updated_at := now();\n  RETURN NULL;\nEND"
    good: "BEGIN\n  NEW.updated_at := now();\n  RETURN NEW;\nEND"
  - title: "Updating the same table from an AFTER trigger"
    detail: "An AFTER UPDATE trigger that runs UPDATE on its own table fires itself again, until the stack depth limit stops it. To change the row being written, use a BEFORE trigger and assign to NEW; if you must write the same table, guard with a WHEN condition or pg_trigger_depth()."
    bad: "-- AFTER UPDATE ... FOR EACH ROW:\nUPDATE requirements SET updated_at = now() WHERE id = NEW.id;"
    good: "-- BEFORE UPDATE ... FOR EACH ROW:\nNEW.updated_at := now();\nRETURN NEW;"
  - title: "Auditing current_user behind a connection pool"
    detail: "The app connects as one role, so every audit row says the same name. Pass the end user per transaction and read it in the trigger; current_setting's second argument returns NULL instead of failing when the setting is absent (psql, migrations)."
    good: "-- in the transaction, from Go:\nSET LOCAL app.user_id = '42';\n-- in the trigger function:\ncurrent_setting('app.user_id', true)"
  - title: "Business workflows in triggers"
    detail: "Status transitions with rules, roll-ups that encode policy, anything that sends e-mail or calls a service: these are decisions, and decisions belong where callers can see, test and log them. A trigger can't call an HTTP API safely anyway: it runs inside the transaction, before anyone knows whether it will commit."
  - title: "Several triggers for one event"
    detail: "Triggers of the same kind on the same event fire in alphabetical order by name, and each BEFORE trigger gets the row as the previous one returned it. Renaming a trigger can change behaviour. Prefer one trigger per purpose and event, with names like requirements_updated_at."
quiz:
  - q: "A BEFORE UPDATE ... FOR EACH ROW trigger function returns NULL. What happens to that row?"
    options: ["It's updated with all columns set to NULL", "It isn't updated, and the command tag doesn't count it", "The statement fails"]
    answer: 1
    why: "Returning NULL from a row-level BEFORE trigger skips the operation for that row, silently."
  - q: "When do row-level AFTER triggers (not constraint triggers) run?"
    options: ["Immediately after each row is written", "At the end of the statement, once per affected row", "At COMMIT"]
    answer: 1
    why: "They are queued per row and fired when the statement finishes; only constraint triggers can be deferred to commit."
  - q: "Someone runs `UPDATE requirements SET status = 'approved'` in psql, bypassing the API. What still sets updated_at?"
    options: ["The ORM's pre-update hook", "A BEFORE UPDATE trigger", "Both"]
    answer: 1
    why: "ORM hooks run only in the application that defines them; a trigger fires for every writer."
  - q: "An audit trigger inserts into audit_log, and later in the same transaction an error causes a ROLLBACK. What does audit_log contain from that transaction?"
    options: ["The audit rows, flagged as failed", "Nothing", "Only the rows written before the error"]
    answer: 1
    why: "The trigger runs in the same transaction as the change; rolling back undoes both."
further_reading:
  - title: "PostgreSQL docs: Overview of Trigger Behavior"
    url: "https://www.postgresql.org/docs/current/trigger-definition.html"
  - title: "PostgreSQL docs: CREATE TRIGGER (WHEN, REFERENCING, firing order)"
    url: "https://www.postgresql.org/docs/current/sql-createtrigger.html"
  - title: "PostgreSQL docs: Trigger Functions in PL/pgSQL (audit examples)"
    url: "https://www.postgresql.org/docs/current/plpgsql-trigger.html"
  - title: "PostgreSQL docs: System Information Functions (pg_trigger_depth)"
    url: "https://www.postgresql.org/docs/current/functions-info.html"
---

## Concept

A **trigger** binds a function to `INSERT`, `UPDATE`, `DELETE` or `TRUNCATE` on a table.
Two choices define it:

- **Timing:** `BEFORE` triggers run before the row is written and can change it (assign
  to `NEW`) or cancel it (return `NULL`). `AFTER` triggers see the row as stored and can
  only act elsewhere or raise an error. `INSTEAD OF` triggers replace the operation on a
  view.
- **Level:** `FOR EACH ROW` runs once per affected row; `FOR EACH STATEMENT` runs once
  per statement, even when it touches no rows, and can see all affected rows through
  transition tables.

The trigger runs in the writing transaction, for every writer: the API, psql, a migration,
another service. That is exactly what you want for **mechanical invariants**, rules about
the data that involve no business decision: an `updated_at` stamp, an audit trail, a
derived value kept in sync. It is exactly what you don't want for **business logic**: a
workflow hidden in the schema runs where nobody reading the calling code can see it.

## Coming from C/C++/Java/Python/JS

- **ORM lifecycle hooks** (JPA `@PreUpdate`, Django `save()` and signals, Sequelize
  hooks) are the same idea in the application, and they only run when the write goes
  through the ORM. Django's `QuerySet.update()` is documented to skip `save()` and the
  `pre_save`/`post_save` signals; a trigger can't be skipped that way.
- **Observers and event listeners:** a trigger is a synchronous listener inside the
  transaction. If it throws, the change it listens to is rolled back.
- **AOP and Python decorators:** a cross-cutting concern invisible at the call site, with
  the same trade-off: less repetition, less visibility.
- **Go with pgx:** there are no hooks at all; the SQL you send is what runs. Triggers
  bring back exactly the invisible part that Go programmers tend to value not having.

## How it works

For one `UPDATE`, the order is: `BEFORE STATEMENT` triggers, then for each row the
`BEFORE ROW` triggers (in alphabetical order by name, each receiving the row as the
previous one returned it), the row write, and finally, at the end of the statement, the
queued `AFTER ROW` triggers and the `AFTER STATEMENT` triggers. Constraint triggers
(`CREATE CONSTRAINT TRIGGER ... DEFERRABLE`) can be deferred to commit.

Inside a PL/pgSQL trigger function, `NEW` and `OLD` hold the rows (NULL where they don't
apply), and `TG_OP`, `TG_TABLE_NAME`, `TG_WHEN`, `TG_LEVEL` and `TG_ARGV` describe the
call, which is how one function serves many tables. Statements the function runs fire
their own triggers; the depth is unlimited, and `pg_trigger_depth()` tells a trigger how
deeply it is nested.

Since PostgreSQL 18, `AFTER` triggers run as the role that was active when the event was
queued, not the role active when they fire (at the end of the statement or at commit).
It matters only if a transaction switches roles in between.

## Design notes

Draw the line by asking what the trigger knows. **Invariants about data** (timestamps,
audit rows, a counter that must match its rows) belong in triggers or, better, in
declarative features when one fits: a generated column for derived values, a `CHECK`,
foreign key or exclusion constraint for rules. **Decisions about the domain** (may this
requirement be approved, who gets notified, what "passing" means) belong in the
application, in code that callers can read and tests can cover. Example 3's roll-up is
the classic mistake: the policy for "passing" ended up in the schema, firing three times
per statement.

Keep triggers boring: one purpose each, named `<table>_<purpose>`, no external side
effects, created in the same migration as the table, and listed in review. For audit,
the explicit alternative is a data-modifying CTE with `RETURNING old.*, new.*` (PG 18,
sql.subqueries-ctes.modifying) for the statements that need it, or the application writing
an audit or outbox row in the same transaction. TraceLite step 13 builds both and compares
them.
