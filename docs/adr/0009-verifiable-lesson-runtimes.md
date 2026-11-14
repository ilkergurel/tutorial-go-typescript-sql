# ADR-0009: Every runtime implemented, every example verified

Date: 2026-11-13 · Status: accepted

## Context

The first end-to-end run in local mode showed lessons that could never pass: SQL runs
returned structured results but no text, so "matches expected" compared against an
empty stdout; the multisession runner ignored `setup`, ran strictly in sequence and
could not show a lock wait or a deadlock; `ts-node` and `sql-nontx` had no runner; the
browser runner rejected top-level `await` and cut async output off 50 ms after load.
Nothing ran the examples, so hand-written expected files and annotation line numbers
had drifted from the code.

## Decision

- **SQL text output.** sqlrun renders results as deterministic text (command tag, then
  notices and an indented table; `NULL` spelled out) into stdout, so one judge works for
  every runtime. Cells travel as `string | null`; NULL never looks like `''`.
- **Three SQL modes, one sandbox idea.** Each run gets a `run_<nanos>_<rand>` schema:
  rolled back (`sql`), or dropped afterwards (`sql-nontx`, `sql-multisession`), with a
  janitor for schemas a crash left behind.
- **Concurrent multisession.** Two sessions run like two terminals: a step still running
  after 400 ms is reported as waiting and the script continues; results print in a fixed
  causal order (the awaited step first, then what it unblocked). Real lock waits and
  deadlocks become visible and deterministic.
- **ts-node** runs Node's type stripping (`--experimental-strip-types`, harmless on Node
  23.6+, needed on 22) in gorun, which already owns the process hygiene.
- **Browser runs end when the module has finished and no timer is pending**, not after a
  fixed delay; examples run as ES modules (top-level `await`) under a CSP with no network.
- **`npm run verify:lessons`** runs every example the way the Run button does (headless
  Edge/Chrome for ts-browser) and judges it with the UI's own `judge.ts`, twice, plus
  annotation range checks. `--write-expected` records reviewed output.

## Alternatives

- **Compare SQL structurally in the UI**: expected files would need a JSON format nobody
  can read or write by hand. Text is what psql users read anyway.
- **Sequential multisession with a short `lock_timeout`**: simpler, but "B waits until A
  commits" would appear as an error and a deadlock could never form.
- **Re-implement runners in Node for verification**: would check a copy of the runtime
  rules, not the code learners use.

## Consequences

- Expected files are generated, then reviewed; never hand-written for `exact` checks.
- The multisession transcript depends on a 400 ms block window being well under
  PostgreSQL's 1 s `deadlock_timeout`; a very slow machine could reorder a deadlock demo.
- Verification needs the app running (`npm start`) and Edge or Chrome installed.
