---
id: sql.modeling.temporal
title: "History and temporal data: PG 18 WITHOUT OVERLAPS, history tables, a bitemporal introduction"
track: sql
module: modeling
order: 3
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.constraints.generated, sql.modeling.denormalize]
compare_with: [sql.modifying.soft-delete, sql.server-side.triggers, fullstack.tracelite.step-13-audit, design.paradigms.immutability]
tags: [modeling, temporal, without-overlaps, period, ranges, history, bitemporal, pg18]
summary: >
  A temporal table stores each fact with the period it was true. PG 18's
  WITHOUT OVERLAPS keys and PERIOD foreign keys enforce "one at a time" and
  "only while the parent exists"; a second period, the time the database knew
  it, makes past reports reproducible.
examples:
  - id: temporal-key
    file: examples/01_temporal_key.sql
    title: "A temporal primary key: one owner per requirement at any moment"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          The key compares requirement_id with = and valid_at with &&, both
          in one GiST index, and GiST has no built-in = for int or text.
          Without btree_gist the CREATE TABLE fails with SQLSTATE 42704 (no
          default operator class for access method "gist"). The docs call
          btree_gist the expected way to use this feature.
      - lines: "10"
        note: >
          WITHOUT OVERLAPS on the last column means: no two rows with the
          same requirement_id whose periods overlap. The docs describe it as
          behaving like EXCLUDE USING gist (requirement_id WITH =, valid_at
          WITH &&). The column must be a range or multirange, and empty
          ranges are rejected (SQLSTATE 23514).
      - lines: "13-15"
        note: >
          Half-open [start, end) ranges let one period end exactly where the
          next begins. No upper bound means "still true": the current row is
          the one where upper_inf(valid_at) holds.
      - lines: "18-21"
        note: >
          @> answers "what was true on that day" in one predicate. A
          multicolumn GiST index can serve conditions on any of its columns,
          so the key's own index helps here too.
      - lines: "25-28"
        note: >
          PG 18 has no temporal UPDATE (the standard's FOR PORTION OF), so
          changing a fact from a date onwards is your code: shorten the
          current row, then insert the new one, in one transaction. The key
          is checked as each row is written, so the reverse order fails.
      - lines: "35-39"
        note: >
          The primary key is backed by a GiST index named after the
          constraint. A key made only of equality columns would be a B-tree.
      - lines: "42"
        note: >
          Selin's May to July overlaps Ayşe's period and Mehmet's. The error
          is an exclusion violation, SQLSTATE 23P01, not 23505 like an
          ordinary duplicate key: error mapping that only knows 23505 will
          let it through as a 500.
  - id: temporal-fk
    file: examples/02_temporal_fk.sql
    title: "A PERIOD foreign key, and a temporal join"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-14"
        note: >
          In a temporal table id alone is no longer unique; (id, period) is.
          The rename is a new version row, so both names are kept along with
          when each applied.
      - lines: "22-23"
        note: >
          PERIOD makes the last column special: the reference holds when the
          referenced rows with that id, taken together, cover the whole
          referencing period. The other side needs a WITHOUT OVERLAPS key.
          Only NO ACTION is supported; CASCADE, SET NULL, SET DEFAULT and
          RESTRICT are rejected for PERIOD foreign keys (SQLSTATE 0A000).
      - lines: "26-27"
        note: >
          No single project row covers March to September; Checkout and
          Payments together do, so the reference holds. Splitting a version,
          as a rename does, never breaks the memberships that span it.
      - lines: "32-35"
        note: >
          The temporal join: match on the key and on overlap (&&), then
          intersect the periods with *. The result is temporal too: Ayşe was
          on Checkout until July and on Payments after, which neither table
          says on its own.
      - lines: "38"
        note: >
          Project 2 ends on 1 April, so April to July is uncovered: SQLSTATE
          23503, the ordinary foreign key error. The other direction is
          guarded as well: shortening or deleting a project version that a
          membership still needs fails with the same code.
  - id: bitemporal
    file: examples/03_bitemporal.sql
    title: "Bitemporal: what was true, and when the database knew it"
    runtime: sql
    check: exact
    annotations:
      - lines: "9-10"
        note: >
          Two time axes. valid_at (application time) is when the price holds
          in the world, and the business may set it in the past or future.
          recorded_at (system time) is when the database held the belief, and
          only the system sets it. It is a daterange with literal dates here
          so the output is reproducible; a real table uses tstzrange starting
          at now().
      - lines: "12"
        note: >
          WITHOUT OVERLAPS takes a single range column. With two axes the rule
          is "no two rows for one SKU overlap in both periods at once", which
          is EXCLUDE with two && operators.
      - lines: "20-24"
        note: >
          Nothing is deleted, and the only change to an old row is the upper
          bound of recorded_at. The belief "129.00, open-ended" was what the
          database said from January to March, and it stays queryable.
      - lines: "27-29"
        note: >
          A correction is the same move. The typo row keeps its recorded
          period, 1 to 20 March: exactly when an invoice could have used it.
      - lines: "32-36"
        note: >
          Three answers for the same day, all correct: 129.00 as known in
          February, 149.00 as known on 15 March (what an invoice printed then
          shows), 139.00 as known today. Reproducing an old report means
          filtering recorded_at by the report's date.
      - lines: "39"
        note: >
          This row overlaps the current 139.00 row in valid_at (from 15
          March) and in recorded_at (from 20 March): SQLSTATE 23P01. A
          correction has to close the old belief first, as lines 27-28 do.
use_cases:
  - "Ownership, assignments and memberships over time: who was responsible for REQ-12 when the defect shipped"
  - "Price lists, tax rates and tariffs with future-dated changes: enter next month's prices today"
  - "Contracts, employment and insurance cover, where corrections are backdated and old reports must stay reproducible (bitemporal)"
  - "Audit questions such as 'what did the screen show on 15 March' (system time, history tables)"
strengths:
  - "A temporal key makes 'one owner at a time' a declaration; in Go or TS the same rule is a SELECT-then-INSERT that races"
  - "A PERIOD foreign key checks coverage against all referenced versions together, which a hand-written trigger would need range aggregation and locking to get right"
  - "Range operators make as-of filters (@>), overlap tests (&&) and temporal joins (*) one predicate each, and GiST indexes serve them"
  - "Bitemporal rows reproduce past reports exactly, without restoring a backup or replaying a log"
weaknesses:
  - "No temporal UPDATE or DELETE in PG 18: splitting a period takes two or three statements of your own, in the right order"
  - "PERIOD foreign keys support only NO ACTION, so retiring a parent means closing or deleting the children first"
  - "Temporal keys need btree_gist and a GiST index, which is larger and slower to maintain than a B-tree"
  - "Tables grow with every change, and every query needs a time predicate; forgetting it silently returns all versions"
pitfalls:
  - title: "Querying a temporal table without a time predicate"
    detail: "Every version comes back, so project 1 appears twice and joins multiply rows. Say which moment you mean, or read through a view of current rows."
    bad: "SELECT name FROM projects WHERE id = 1;"
    good: "SELECT name FROM projects WHERE id = 1 AND valid_at @> current_date;"
  - title: "Closed upper bounds"
    detail: "daterange is stored in the canonical [) form, so '[2026-01-01,2026-06-01]' becomes [2026-01-01,2026-06-02) and overlaps a row starting on 1 June. Write half-open ranges; daterange(a, b) builds [a, b) by default."
  - title: "Inserting the new version before closing the old one"
    detail: "The key sees two open periods and fails with 23P01. Shorten the current row first, then insert, inside one transaction."
  - title: "Mapping only 23505 to 409 Conflict"
    detail: "Violations of WITHOUT OVERLAPS keys and EXCLUDE constraints are SQLSTATE 23P01 (exclusion_violation). Map it next to 23505, or the API answers a conflict with a 500."
  - title: "History that outlives the erasure duty"
    detail: "History tables and closed versions keep personal data after the current row is anonymized. KVKK Article 7 erasure and anonymization requests have to reach the history too."
quiz:
  - q: "projects has PRIMARY KEY (id, valid_at WITHOUT OVERLAPS) with rows for id 1 valid [2026-01-01,2026-07-01) and [2026-07-01,). A membership references project 1 with PERIOD for [2026-03-01,2026-09-01). Is it accepted?"
    options: ["No: no single project row covers it", "Yes: the two rows together cover it", "Only with ON UPDATE CASCADE"]
    answer: 1
    why: "A PERIOD reference is satisfied when the combined periods of the matching referenced rows cover the referencing period completely."
  - q: "Which SQLSTATE does an overlapping insert into a WITHOUT OVERLAPS primary key raise?"
    options: ["23505 unique_violation", "23P01 exclusion_violation", "23514 check_violation"]
    answer: 1
    why: "The key is enforced like an exclusion constraint, so it reports exclusion_violation. 23514 is what an empty range gets."
  - q: "The owner of requirement 2 changes on 1 October. How do you record that in PG 18?"
    options:
      - "UPDATE ... FOR PORTION OF valid_at FROM '2026-10-01'"
      - "Shorten the current row to end on 1 October, then insert the new owner from 1 October"
      - "Insert the new owner from 1 October, then shorten the current row"
    answer: 1
    why: "PG 18 has no FOR PORTION OF. The key is checked per row, so the old period must end before the new one is inserted."
  - q: "In a bitemporal table, what does recorded_at answer?"
    options: ["When the fact is true in the world", "When the database held that belief", "When the row was last read"]
    answer: 1
    why: "recorded_at is system time. valid_at is application time, which the business can set in the past or future."
further_reading:
  - title: "PostgreSQL docs: CREATE TABLE (WITHOUT OVERLAPS, PERIOD)"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
  - title: "PostgreSQL docs: Range Types"
    url: "https://www.postgresql.org/docs/current/rangetypes.html"
  - title: "PostgreSQL docs: Range Functions and Operators"
    url: "https://www.postgresql.org/docs/current/functions-range.html"
  - title: "PostgreSQL docs: btree_gist"
    url: "https://www.postgresql.org/docs/current/btree-gist.html"
  - title: "PostgreSQL 18 release notes (temporal constraints)"
    url: "https://www.postgresql.org/docs/18/release-18.html"
  - title: "PostgreSQL docs: Unsupported SQL features (T180, T181)"
    url: "https://www.postgresql.org/docs/current/unsupported-features-sql-standard.html"
---

## Concept

An ordinary table holds the current state: an UPDATE overwrites the past. When the past
matters, a schema can keep time along one or two axes:

- **Application time** (valid time): when a fact is true in the world. Each row carries
  a period, typically a `daterange` or `tstzrange` named `valid_at`, and a change of
  owner, name or price is a new row with a new period. The business sets these
  periods, including in the past and the future.
- **System time** (transaction time): when the database held a row. Only the system
  sets it. The classic form is a **history table**: the current table stays as it is,
  and every UPDATE or DELETE copies the old version, stamped with its system period,
  into a `_history` table.
- **Bitemporal**: both at once, for domains where facts are corrected after the fact
  and you must still answer "what did we believe then".

PostgreSQL 18 adds the constraints application time needs: `PRIMARY KEY` and `UNIQUE`
with `WITHOUT OVERLAPS` on a range column, and foreign keys with `PERIOD`. The SQL
standard's system-versioned tables (`FOR SYSTEM_TIME AS OF`, feature T180) and temporal
`UPDATE ... FOR PORTION OF` are not part of PG 18.

## Coming from C/C++/Java/Python/JS

- **Java:** `effectiveFrom`/`effectiveTo` fields with a service that checks for
  overlaps is the hand-written version of example 1, and it races under concurrency.
  Hibernate Envers is a history-table implementation (audit tables filled by the ORM),
  so it misses writes that bypass the ORM; a database trigger doesn't.
- **Go and TS:** there's no temporal library to reach for, which is fine: keep the
  rules in the database (keys, PERIOD foreign keys) and the period splitting in one
  store function that runs its statements in a transaction.
- **Event sourcing:** an append-only event log is system time taken to the extreme.
  A bitemporal table is its relational cousin: you never lose a belief, but you can
  still query state with plain SQL instead of replaying events.
- **Python/JS dates:** a range `[start, end)` is the same half-open convention as
  `range(a, b)` and `Array.prototype.slice`: adjacent periods share a boundary
  without overlapping.

## How it works

`WITHOUT OVERLAPS` turns the key into an exclusion constraint over a GiST index: equal
on the leading columns and overlapping (`&&`) on the period is a conflict, reported as
SQLSTATE 23P01. Empty periods are rejected. Scalar key columns need btree_gist.

A `PERIOD` foreign key compares the leading columns for equality and requires the
referenced rows' periods, taken together, to cover the referencing period. Deleting or
shortening a referenced version is checked the same way (NO ACTION is the only
action PG 18 accepts for it).

Changing a temporal fact from a date onwards is a split you write yourself, inside a
transaction:

1. Shorten the row that is current at that date: `upper = change date`.
2. Insert the new version from the change date.
3. If the change ends before the old period did, insert the remainder of the old
   version after it.

A history table for system time is usually maintained by an `AFTER UPDATE OR DELETE`
trigger that inserts the old row with `tstzrange(previous change, now())`. The triggers
lesson and TraceLite's audit step build one; the design questions are the same as here.

## Design notes

Pick the axis from the question the business asks. "Who owned REQ-12 on 15 March?" is
application time: add `valid_at` and a `WITHOUT OVERLAPS` key. "What did this record look
like before Mehmet edited it?" is system time: a history table, filled by a trigger so
no writer can skip it. "What did the invoice of 15 March say, given that we corrected the
price on 20 March?" is bitemporal, and rare enough that you should be sure before paying
for it.

Keep reads of the present cheap. Most queries want current rows only; give them a view
(`WHERE upper_inf(valid_at)` or `valid_at @> current_date`), or keep a plain current
table and put only the past in a history table. A soft-delete flag is the degenerate
case of all this: one transition, no period.

History is personal data too. Anonymizing a customer's current row leaves their old
versions in history tables and closed periods, so erasure under KVKK Article 7 has to
cover them, and a retention policy should say how long history is kept at all.
