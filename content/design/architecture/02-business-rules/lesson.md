---
id: design.architecture.business-rules
title: "Where business rules live: app vs database"
track: design
module: architecture
order: 2
level: advanced
minutes: 15
since: pg11
prerequisites: [sql.constraints.fk-check, sql.transactions.isolation-levels, go.postgres.transactions]
compare_with: [sql.transactions.row-locks, sql.transactions.serializable, sql.server-side.triggers, ts.architecture.explicit-states, go.postgres.repository-testing]
tags: [architecture, business-rules, constraints, triggers, concurrency, turkish]
summary: >
  Invariants about data belong in the database, where every writer obeys them and
  concurrency can't slip past them; policies that need the outside world belong in the
  app. Rules written twice drift, and "in the database" is only safe if it is a
  constraint or holds a lock.
examples:
  - id: check-then-insert
    file: examples/01_check_then_insert.sql
    title: "A read-then-write rule races, whether the app or a trigger does the reading"
    runtime: sql-multisession
    setup: |
      CREATE TABLE courses (id int PRIMARY KEY, seats int NOT NULL);
      CREATE TABLE bookings (
          course_id int NOT NULL REFERENCES courses,
          student   text NOT NULL,
          PRIMARY KEY (course_id, student)
      );
      INSERT INTO courses VALUES (1, 1), (2, 1);
    check: exact
    annotations:
      - lines: "5-12"
        note: >
          The application-side check: both sessions count 0 bookings, both decide there
          is a seat, both insert. Nothing conflicts, because they insert different rows,
          so both commit and course 1 is overbooked.
      - lines: "15-24"
        note: >
          The same rule moved into the database as a trigger. It is still a read
          followed by a write, and the read sees only committed rows plus its own.
      - lines: "27-30"
        note: >
          B's trigger counts 0 because A's row is not committed yet; under READ
          COMMITTED it is invisible to B's statement. Course 2 is overbooked exactly like
          course 1. Moving code into the database did not make it a constraint.
      - lines: "32-33"
        note: >
          Sequentially the trigger refuses the third booking (P0001 is RAISE EXCEPTION's
          SQLSTATE). This is why such rules pass every single-user test and fail in
          production.
      - lines: "35"
        note: >
          Two bookings each for one seat. PostgreSQL's docs warn about CHECK constraints
          that read other rows for the same reason: they appear to work in simple tests.
  - id: constraint-decides
    file: examples/02_constraint_decides.sql
    title: "A counter on the parent row plus a CHECK: the database decides, and serializes"
    runtime: sql-multisession
    setup: |
      CREATE TABLE courses (
          id     int PRIMARY KEY,
          seats  int NOT NULL,
          booked int NOT NULL DEFAULT 0,
          CONSTRAINT courses_not_overbooked CHECK (booked <= seats)
      );
      CREATE TABLE bookings (
          course_id int NOT NULL REFERENCES courses,
          student   text NOT NULL,
          PRIMARY KEY (course_id, student)
      );
      INSERT INTO courses VALUES (1, 1);
    check: exact
    annotations:
      - lines: "1-2"
        note: >
          The cross-row rule becomes a single-row one: courses.booked is maintained with
          every booking, and CHECK (booked <= seats), in the setup, states the rule where
          a CHECK can enforce it.
      - lines: "6-7"
        note: >
          Incrementing the counter is a write to one row, so it takes that row's lock.
          The increment is relative (booked + 1): the database computes it, not the app.
      - lines: "8-9"
        note: >
          B's increment waits for A's lock. When A commits, B re-reads the row (booked is
          now 1), computes 2, and the CHECK refuses it with SQLSTATE 23514 and the
          constraint's name, which the app maps to "course full" in one place.
      - lines: "12-13"
        note: >
          One seat, one booking, and a counter that agrees. The race is gone because the
          rule is checked on the row that every competing writer must lock.
  - id: one-definition
    file: examples/03_one_definition.go
    title: "One rule, four implementations, three answers; then one home for it"
    runtime: go
    check: exact
    annotations:
      - lines: "17-18"
        note: >
          A rule as simple as "unique, ignoring case" depends on whose case mapping you
          use. JavaScript's toLowerCase applies Unicode's full mapping, so "İsmail"
          becomes a 7-character string; Go and this database's lower() do not.
      - lines: "27-36"
        note: >
          Go's strings.ToLower and lower() in this C.UTF-8 database agree with each
          other and disagree with Turkish: IŞIL becomes işil, a different name from
          ışıl. ToLowerSpecial(unicode.TurkishCase) and the ICU tr collation give the
          Turkish answer.
      - lines: "38-44"
        note: >
          The rule gets one home: a unique index on the Turkish-aware expression. Every
          writer, including psql and the next service, now obeys the same definition.
      - lines: "45-47"
        note: >
          IŞIL and Işıl are the same name as ışıl and are refused; işil, with a dotted
          i, is a different Turkish name and is accepted.
      - lines: "52-59"
        note: >
          The app asks the home instead of re-implementing it: insert, and translate the
          violation by constraint name into a domain error. A pre-check for nicer UX
          should run the same SQL expression, never a Go copy of it.
use_cases:
  - "Database: invariants about stored data (NOT NULL, ranges, uniqueness, references, no overlaps, quotas kept as counters), because every writer and every concurrent transaction must obey them"
  - "Application: policies that need the outside world (payment authorization, emails, feature flags, the current user's permissions), change often, or need rich error messages"
  - "Both: the database enforces, the app pre-validates for UX with the same definition, and maps constraint violations to domain errors"
  - "Triggers: things that must happen on every write path (audit rows, outbox events, updated_at), not decisions"
strengths:
  - "A constraint holds for every client, including psql sessions, migrations, batch jobs and services you haven't written yet; no application check can say that"
  - "Constraints are concurrency-safe by construction: unique indexes, exclusion constraints and single-row CHECKs are enforced under the locks PostgreSQL takes anyway"
  - "Constraint violations come back with a SQLSTATE and the constraint's name, so the app can map them to precise errors in one place"
  - "Application rules are easy to unit-test, to version with the code, to explain in an error message, and to change without a migration"
weaknesses:
  - "A CHECK constraint sees only the row being written; cross-row rules need a counter, an exclusion constraint, a lock or SERIALIZABLE"
  - "Rules in the database are written in SQL or PL/pgSQL, tested only against a real PostgreSQL, and deployed by migration rather than with the code"
  - "Database errors are terse; without a mapping layer, users see 'violates check constraint' instead of 'the course is full'"
  - "A rule enforced in both places must be kept identical, or the two will disagree on edge cases (the Turkish İ/ı example)"
pitfalls:
  - title: "Check-then-act in application code"
    detail: "SELECT to check, then INSERT or UPDATE, is a race under READ COMMITTED. Make the database decide (a constraint, a conditional UPDATE ... WHERE booked < seats), lock the parent row first (SELECT ... FOR UPDATE), or run the transaction SERIALIZABLE and retry on 40001."
    bad: |
      n := countBookings(ctx, tx, course)  // two requests both see n = 0
      if n < seats { insertBooking(ctx, tx, course, student) }
    good: |
      tag, _ := tx.Exec(ctx, `UPDATE courses SET booked = booked + 1
          WHERE id = $1 AND booked < seats`, course)
      if tag.RowsAffected() == 0 { return ErrCourseFull }
  - title: "Assuming a trigger is a constraint"
    detail: "A trigger that counts other rows runs under the same snapshot rules as application code, so it races the same way. Use it only with a lock (for example on the parent row) or under SERIALIZABLE."
  - title: "Duplicating a rule in Go, TypeScript and SQL"
    detail: "Three implementations drift: case mapping, rounding, time zones, trimming. Keep one authoritative definition and have the other layers call or mirror it from a single source (a shared schema, a generated validator, an endpoint), with tests that compare them."
  - title: "Business decisions in triggers"
    detail: "Sending email, calling services or applying pricing policy from a trigger hides behaviour from the code that caused it and makes it run inside the transaction. Keep triggers for audit and outbox rows (sql.server-side.triggers)."
quiz:
  - q: "Why did the BEFORE INSERT trigger let both bookings for course 2 through?"
    options: ["Triggers don't run inside transactions", "B's statement could not see A's uncommitted row, so both counts were 0", "The trigger was created after the inserts", "RAISE EXCEPTION is only a warning"]
    answer: 1
    why: "Under READ COMMITTED each statement sees rows committed before it started, plus its own. A trigger that reads other rows inherits that view and races like application code."
  - q: "What made the counter + CHECK version safe?"
    options: ["CHECK constraints read other tables", "Both sessions had to update the same course row, so the second waited, re-read it and failed the CHECK", "The bookings primary key", "SERIALIZABLE isolation"]
    answer: 1
    why: "The rule was moved onto a row every competing writer must lock. The blocked UPDATE re-evaluated booked + 1 against the committed value and the CHECK refused 2 > 1."
  - q: "Which rule belongs in the application rather than the database?"
    options: ["An order's total is never negative", "No two bookings for one room overlap", "A refund needs the payment provider's approval", "Every booking references an existing course"]
    answer: 2
    why: "It depends on an external system and its response. The other three are invariants about stored data that a constraint (CHECK, EXCLUDE, FOREIGN KEY) enforces for every writer."
  - q: "Under Turkish rules, which sign-up is a different user from 'ışıl'?"
    options: ["IŞIL", "Işıl", "işil", "ıŞıL"]
    answer: 2
    why: "Turkish has dotted and dotless i as separate letters: I/ı and İ/i. 'işil' has a dotted i, so it is a different name; IŞIL, Işıl and ıŞıL all lower-case to ışıl."
further_reading:
  - title: "PostgreSQL: Constraints (the note on CHECK constraints that reference other rows)"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html"
  - title: "PostgreSQL: Data consistency checks at the application level"
    url: "https://www.postgresql.org/docs/current/applevel-consistency.html"
  - title: "PostgreSQL: Collation support (ICU collations)"
    url: "https://www.postgresql.org/docs/current/collation.html"
  - title: "package strings: ToLowerSpecial"
    url: "https://pkg.go.dev/strings#ToLowerSpecial"
  - title: "MDN: String.prototype.toLocaleLowerCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleLowerCase"
---

## Concept

A business rule is a statement that must stay true ("a course is never overbooked") or a
decision the system makes ("refunds over 1,000 TL need approval"). Where each one lives
is the most consequential line you draw, because it decides who can break it.

Split rules into two kinds:

- **Invariants about stored data.** Facts that must hold for every row at every moment,
  whoever writes: required fields, ranges, uniqueness, references, no overlapping
  bookings, a quota. They belong in the **database**, as constraints. Every client obeys
  them, psql and migrations included, and PostgreSQL enforces them correctly under
  concurrency.
- **Policies and workflows.** Rules that need the outside world (a payment provider,
  the current user, a feature flag, the clock in a time zone), that change with the
  business, or whose failure needs a careful explanation. They belong in the
  **application**, as code you can unit-test and deploy with the rest
  (ts.architecture.explicit-states shows a pure core for exactly this).

Two traps sit between them. First, **"in the database" is not the property that
matters; being a constraint is.** A trigger that counts rows is still a read followed by
a write, and races exactly like application code (example 1). PostgreSQL's own docs say
CHECK constraints must not read other rows for the same reason. Turn cross-row rules into
something the database enforces atomically: a counter on a parent row with a CHECK
(example 2), a unique index, an exclusion constraint, a conditional `UPDATE ... WHERE
booked < seats`, a row lock, or SERIALIZABLE with retries.

Second, **a rule written twice drifts.** The app wants to validate early for a friendly
message, the frontend wants instant feedback, the database enforces. Three copies of
"ignoring case" gave three answers in example 3. Keep one authoritative definition and
make the others ask it, or generate them from one source.

## Coming from C/C++/Java/Python/JS

- **Java**: Bean Validation (`@NotNull`, `@Size`) and JPA entity rules check objects in
  memory; they don't see concurrent transactions or other writers. Treat them as the
  friendly first check and the schema as the authority.
- **Python**: Django model validation runs only when your code calls `full_clean()`;
  `bulk_create` and raw SQL skip it. The database constraint is the one that cannot be
  skipped.
- **JS/TS**: zod at the trust boundary (ts.services.zod) is the right first check and the
  wrong last one: it validates shape, not uniqueness or capacity, which need the data.
  Also note that `toLowerCase()` in JavaScript applies Unicode's full case mapping and
  `toLocaleLowerCase("tr")` the Turkish one; Go's `strings.ToLower` applies neither.
- **C/C++**: an invariant in a class is protected by the type's methods within one
  process. A database is shared by processes you don't control, which is why the
  invariant has to live where they all meet.

## How it works

Under READ COMMITTED, each statement sees rows committed before it began plus its own
changes. In example 1 both sessions count bookings before either commits, so both see
zero; inserting different rows creates no conflict for PostgreSQL to detect. The trigger
version is the same read with the same snapshot.

Example 2 works because both sessions must update the same `courses` row. A row can be
updated by one transaction at a time, so B's `UPDATE` waits. When A commits, B does not
fail and does not use stale data: READ COMMITTED re-evaluates the updated row's new
version, computes `booked + 1 = 2`, and the CHECK, which only needs that one row, rejects
it. The error carries SQLSTATE 23514 and the name `courses_not_overbooked`, which is the
stable contract between schema and code.

Example 3's index works on an expression: `lower(username COLLATE "tr-x-icu")` uses ICU's
Turkish case rules, independent of the database's own C.UTF-8 locale. The app inserts
and translates the unique violation by constraint name, the same mapping go.errors.is-as-join and
go.postgres.repository-testing show for SQLSTATE 23505.

## Design notes

A practical default for each rule: write it as a constraint if it is a fact about stored
data and SQL can express it on one row or with an index; map its violation to a domain
error in the repository; pre-validate in the app or the UI only for user experience,
using the same definition. Write it as application code if it needs anything outside
the database or is a decision rather than a fact. Reach for a trigger only for things
every write path must do, such as audit and outbox rows, and keep decisions out of it.

StackLab leans on the database where it matters for the lesson runtime: the playground
role, schema-per-run and statement timeouts are enforced by PostgreSQL, not by Go code
trusting itself. Its own app data is simpler: learning progress, notes and bookmarks
have keys and NOT NULL, and the one rule with business flavour, "a lesson's status only
moves up" (viewed, ran, completed), lives twice: in a `CASE` inside `learningstore`'s
upsert and, for the offline fallback, in `RANK` in `src/features/learning/api.ts`. The two
have already drifted on an edge: marking a lesson completed again re-stamps
`completed_at` on the server (`WHEN excluded.status = 'completed' THEN now()`), while the
local version returns early and keeps the first time. Small, harmless here, and exactly
the drift this lesson warns about; a test that replays one sequence of events against
both is the cheap guard.

Cross-layer rules are where the Turkish locale bites first: case-insensitive uniqueness,
sorting, and search. Decide once which rules apply (`tr-x-icu`, `turkish` text search),
put that decision in the schema, and let the app ask the database instead of guessing.
