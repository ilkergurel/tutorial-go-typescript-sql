---
id: sql.modifying.returning-upsert
title: "RETURNING, upsert with ON CONFLICT, MERGE"
track: sql
module: modifying
order: 2
level: intermediate
minutes: 12
since: pg9.5
prerequisites: [sql.modifying.insert-update, sql.querying.null-logic]
compare_with: [ts.services.advanced, sql.modern.16-to-18]
tags: [returning, upsert, on-conflict, merge]
summary: >
  RETURNING reads back what you wrote in the same statement. ON CONFLICT makes
  an insert an upsert; MERGE expresses insert/update/delete in one pass with a
  join condition.
examples:
  - id: upsert-returning
    file: examples/01_upsert.sql
    title: Upsert stock levels and read back the affected rows
    runtime: sql
    check: exact
    annotations:
      - lines: "7-11"
        note: >
          One statement: insert the new sku, refresh the existing one. The
          conflict target (sku) must be a unique constraint — a conflict the
          database never sees can't be upserted.
      - lines: "10"
        note: >
          qty = stock.qty + EXCLUDED.qty: existing row plus incoming amount.
          EXCLUDED is the row that *would have* been inserted.
      - lines: "11"
        note: >
          RETURNING fires once per affected row, after the update or insert —
          including OLD/NEW in PG 18 for both sides of the change.
      - lines: "17-22"
        note: >
          MERGE joins source to target and picks an action per row. Use it when
          the decision is "match by X, then insert/update/delete"; plain
          ON CONFLICT when it's just insert-or-update.
use_cases:
  - "Idempotent ingest: re-running a load must not duplicate rows"
  - "Counters and stock: qty = qty + n without read-modify-write races (with FOR UPDATE or in one statement)"
  - "API responses that echo the written rows without a second round-trip"
strengths:
  - "Atomic: no window where a concurrent reader sees the old value between your UPDATE and INSERT"
  - "RETURNING eliminates the follow-up SELECT and its race"
  - "MERGE covers insert+update+delete in one declarative pass (PG 17 adds MERGE ... RETURNING)"
weaknesses:
  - "ON CONFLICT DO NOTHING hides *why* nothing happened; you still need RETURNING to know"
  - "MERGE requires PG 15+; ON CONFLICT remains the portable workhorse"
  - "Uperts on hot rows serialize on the row lock — batch bursts can queue"
pitfalls:
  - title: "ON CONFLICT without a conflict target"
    detail: "DO UPDATE needs (sku) — without it the statement can't know which unique constraint to arbitrate."
  - title: "Assuming RETURNING shows the pre-image"
    detail: "It returns the new row. PG 18's RETURNING OLD/NEW gives both, explicitly."
quiz:
  - q: "What does EXCLUDED refer to in ON CONFLICT DO UPDATE?"
    options: ["The existing row", "The row that would have been inserted", "The deleted row"]
    answer: 1
    why: "EXCLUDED carries the proposed insert, so you can combine it with the current row."
  - q: "RETURNING in an UPDATE gives you…"
    options: ["the old values", "the new values", "both by default"]
    answer: 1
    why: "Default is the post-image; PG 18 adds RETURNING OLD.* / NEW.* for both sides."
further_reading:
  - title: "PostgreSQL docs: INSERT ... ON CONFLICT"
    url: "https://www.postgresql.org/docs/current/sql-insert.html"
  - title: "PostgreSQL docs: MERGE"
    url: "https://www.postgresql.org/docs/current/sql-merge.html"
---

## Concept

`INSERT ... ON CONFLICT` turns a would-be duplicate into an update (`DO UPDATE`) or a no-op
(`DO NOTHING`); `RETURNING` streams the affected rows back out of the same statement.
`MERGE` (PG 15+) joins a source and picks per-row actions.

## Coming from C/C++/Java/Python/JS

- App code: the SELECT-then-INSERT-else-UPDATE dance is a race; ON CONFLICT does it in one
  atomic statement.
- Go/TS: RETURNING replaces the "insert, then query by id" round trip — one statement, one
  network hop, no interleaving window.

## Design notes

Upserts are the idiomatic answer to *idempotency*: replaying an event stream or a CSV load
must converge, not duplicate. But decide conflict behavior deliberately — DO NOTHING can
mask bugs where the same key means different things. When the logic grows beyond
insert-or-update (deletes, conditions across tables), that's MERGE's shape.
