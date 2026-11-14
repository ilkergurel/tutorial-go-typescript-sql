---
id: inside.codebase.at-10x
title: "What I'd change at 10x scale"
track: inside
module: codebase
order: 10
level: advanced
minutes: 15
since: go1.25
prerequisites: [go.concurrency.synctest, sql.modifying.returning-upsert, inside.codebase.runner-sandbox]
compare_with: [design.architecture.styles, design.architecture.reliability, sql.modeling.multi-tenant, inside.codebase.compose-topology]
tags: [scalability, capacity, semaphore, backpressure, bundle-size, multi-user, synctest]
summary: >
  "10x" means three different things for a single-user local app: ten times the
  lessons, the runs, or the learners. Each breaks something different first: serial
  verification (and the statically imported lesson bundle, until it was split per
  lesson), the two run slots, and a schema that has no user in it.
examples:
  - id: estimate
    file: examples/01_estimate.go
    title: "Arithmetic first: today's measurements multiplied by 10"
    runtime: go
    check: exact
    annotations:
      - lines: "6-14"
        note: >
          Every input is a measurement, not a guess: the size of the generated
          bundle file, the example count in it, and the mean durationMs of 59
          server examples in a verify run. An estimate is only as honest as the
          numbers going into it.
      - lines: "26-28"
        note: >
          verify-lessons awaits each server example before sending the next, and
          runs everything twice to catch nondeterminism, so its time grows
          linearly with the content and nothing overlaps.
      - lines: "29"
        note: >
          The bundle column was the crack that users would feel: all of it was
          imported statically, so at 10x a 37 MB module would have sat in the
          main chunk. Since 2026-11-14 each lesson is a chunk of its own, and the
          main chunk holds only an index of about 0.46 KB per lesson.
      - lines: "32-33"
        note: >
          Throughput is slots divided by service time. Two slots at about a second
          per run is plenty for one person and a ceiling for anything more.
  - id: slots
    file: examples/02_slots_test.go
    title: "The two run slots under load, in fake time"
    runtime: go-test
    check: contains
    annotations:
      - lines: "12-15"
        note: >
          runexec.go's values. Two slots kept the 512 MB runner container
          honest in Docker mode and keep a burst of runs from stalling the PC in
          local mode. Both are reasons about one machine and one person.
      - lines: "25-33"
        note: >
          The admission select from Executor.Run, minus the ctx.Done case: take a
          slot, or give up after queueWait. A full buffered channel is the
          semaphore; the deferred receive gives the slot back.
      - lines: "36"
        note: >
          time.Since inside a synctest bubble reads the fake clock, so the waits
          are exact: 3.2 s for the tenth learner at 0.8 s per run, which is five
          rounds of two.
      - lines: "50-54"
        note: >
          synctest.Test (Go 1.25) runs each scenario in its own bubble; the
          clock jumps whenever every goroutine is blocked. Thirty learners and
          ten seconds of queueing take no real time.
  - id: single-user
    file: examples/03_single_user.sql
    title: "The second learner: a schema without a user"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          The real table. lesson_id alone is the key, which is exactly right
          for one learner: no joins, no user table, nothing to authenticate.
      - lines: "11-18"
        note: >
          learningstore's upsert, minus its completed_at bookkeeping. It never
          moves status backwards, so one learner's 'completed' absorbs the next
          learner's 'viewed'.
      - lines: "21-23"
        note: >
          Mehmet has never run the lesson, and the dashboard says he completed
          it. More slots or more servers can't fix this; only a different key
          can.
      - lines: "26-29"
        note: >
          The migration a second learner forces: a user column, a value for
          every existing row (the default fills them, then is dropped so new rows
          must say who they belong to), and a new primary key. notes, bookmarks,
          quiz_answers and runs need the same, and every query and handler with
          them.
use_cases:
  - "Before an \"it won't scale\" discussion: name the axis (content, runs or users) and the first number that breaks"
  - "Deciding what to fix if StackLab grows to the full curriculum and beyond, or is shared with a second learner"
  - "A template for capacity reviews: measure, multiply, find the bottleneck, then change the design"
strengths:
  - "Most of the design is fine at 10x: the stdlib HTTP server, one PostgreSQL holding a few megabytes, contentcheck reading files, the per-run schema sandbox"
  - "Admission control exists: a full queue answers 503 after 10 s instead of piling up goroutines and processes until the machine stalls"
  - "The single-user schema is a deliberate simplification (migration 003 says so), which kept every learning query a primary-key lookup"
  - "Because content is files (ADR-0004), moving lessons out of the main bundle needed no data migration, only a different generator output and loader"
weaknesses:
  - "The lesson index is still eager: about 0.46 KB per lesson in the main chunk, roughly 1 MB at ten times the curriculum (the full lessons would have been 37 MB)"
  - "verify-lessons is serial and runs everything twice: minutes today, hours at 10x, and it competes with the learner for the same two run slots"
  - "Two slots and a 10 s queue: fine for one person, 503s for a classroom"
  - "No user anywhere (schema, API, auth), lesson code running inside the backend process, and a run history table that is written on every run and read by no screen (each insert now trims it to the newest 1,000 rows)"
pitfalls:
  - title: "Scaling the wrong axis"
    detail: "More run slots don't help a second learner whose progress overwrites the first's, and a user table doesn't shrink the bundle. Name which 10x you mean before choosing the fix."
  - title: "Raising maxConcurrent as the fix"
    detail: "Each Go run is a compile plus a process; each SQL run opens its own connection (two for multisession). More slots move the bottleneck to CPU and to PostgreSQL's max_connections; size them to cores and keep the queue."
  - title: "Treating a multi-user version as a scaling exercise"
    detail: "ADR-0003 says the sandbox is void once the app is exposed beyond 127.0.0.1. A second learner on another machine needs authentication and a real isolation boundary (a container or VM per run), not tuning."
quiz:
  - q: "Thirty learners press Run at the same moment and each run takes 3 s. How many see \"server busy\"?"
    options: ["None: they queue until a slot frees", "22: two runs start every 3 s, and everyone still waiting at 10 s gets a 503", "28: only two run at all"]
    answer: 1
    why: "Batches start at 0, 3, 6 and 9 s, so 8 learners are served; the other 22 hit queueWait at 10 s."
  - q: "Which change is needed for a second learner but not for 10x the lessons?"
    options: ["Splitting the lesson bundle", "A user_id in the progress key", "Running verify-lessons in parallel"]
    answer: 1
    why: "With lesson_id as the only key, two learners share one progress row. Content size has nothing to do with it."
  - q: "Until 2026-11-14, why did the lesson bundle grow the main JavaScript chunk instead of being loaded per lesson?"
    options: ["Vite can't split JSON", "types.ts imported lessons.bundle.generated.ts statically, so the bundler had to include all of it up front", "The backend served it inline"]
    answer: 1
    why: "A static import is part of the module graph at build time. Now load.ts lists one JSON file per lesson with import.meta.glob and loads it with import(), so each lesson is its own chunk."
  - q: "What limits throughput first if verify-lessons runs while you study?"
    options: ["PostgreSQL", "The two run slots in runexec, which the learner's Run button shares", "The esbuild worker"]
    answer: 1
    why: "Both go through POST /api/run and the same Executor; verify retries 503s, the learner waits or sees \"server busy\"."
further_reading:
  - title: "package testing/synctest"
    url: "https://pkg.go.dev/testing/synctest"
  - title: "package sync: WaitGroup.Go"
    url: "https://pkg.go.dev/sync#WaitGroup.Go"
  - title: "PostgreSQL: ALTER TABLE"
    url: "https://www.postgresql.org/docs/current/sql-altertable.html"
  - title: "MDN: import() (dynamic import)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/import"
  - title: "React: lazy"
    url: "https://react.dev/reference/react/lazy"
---

## Concept

Nobody runs a personal learning app at ten times its load, so the useful question is:
ten times *what*? Three axes, each with a different first crack.

**10x the content** (about 2,300 lessons). Until 2026-11-14,
`src/features/lessons/types.ts` did `import bundleData from './lessons.bundle.generated'`:
every lesson's prose, examples, expected outputs and fixtures as one module in the main
chunk. The bundle measured on 2026-11-13 held 122 lessons in 1.98 MB, about 16 KB each,
so the planned 229 lessons needed 3.7 MB and ten times that 37 MB, downloaded and parsed
before the first page renders (example 1). That crack is fixed (Design notes, item 1).
The one that remains is `npm run verify:lessons`: it runs each example twice, one at a
time, which is minutes today and hours at 10x.

**10x the runs** (one learner, much more running, or verify running while you study).
`internal/runexec` admits two runs at once and refuses after a 10 s wait with
503 "server busy" (example 2). verify-lessons and the Run button share those two slots.

**10x the learners.** This is a change of kind, not of scale. `progress`, `notes`,
`bookmarks` and `quiz_answers` are keyed by `lesson_id` alone (example 3); there is no
user and no authentication; and in local mode lesson code runs in the backend process
as the owner's OS user (ADR-0008). ADR-0003 says plainly that exposing the app voids the
sandbox's threat model.

Two numbers hold up fine: the database (a few megabytes of progress and run history) and
the Go HTTP layer, which serves one person's clicks without noticing.

## Coming from C/C++/Java/Python/JS

- **Java**: a `ThreadPoolExecutor` with two threads, a bounded wait and a
  `RejectedExecutionException`. In Go it is a buffered channel and a `select` with a
  timer, and the rejection is a value (`*runapi.Error`), not an exception.
- **Python**: the same arithmetic as sizing Gunicorn workers: throughput is workers
  divided by service time. Queueing theory (Little's law) says the queue grows without
  bound once arrivals per second exceed that.
- **JS**: code splitting is the standard answer to a big bundle: `import()` per route
  or per lesson, `React.lazy` for components. Vite emits a separate chunk for every
  dynamic import.
- **C/C++**: a back-of-envelope estimate before a profiler. Example 1 is that envelope.

## How it works

Admission control in `runexec.Executor.Run`:

```go
select {
case e.slots <- struct{}{}:      // a free slot: run now
	defer func() { <-e.slots }()
case <-ctx.Done():               // the client went away
	return nil, busy(), nil
case <-time.After(queueWait):    // 10 s without a slot
	return nil, busy(), nil      // 503 "server busy"
}
```

With two slots and a service time of S seconds, the most the backend can admit is 2/S
runs per second, about 120 a minute for 1 s Go runs (example 1 takes the mean of 59
measured server examples: Go programs about 0.85 s, `go test` about 2.9 s, SQL about
35 ms). Anyone who waits longer than 10 s gets a 503. verify-lessons retries a 503 (up to
30 attempts); the Run button shows the error.

One budget used not to add up under load: `httpapi` extended the write deadline for
`/api/run` to 45 s up front, to cover runexec's 40 s run budget, but not the up to 10 s
spent waiting for a slot first, so a run that queued and then used its whole budget
could have its reply cut off. The handler now lifts the deadline while the run (queue
wait included) is in progress and sets a fresh 10 s one for writing the reply, so no
budget has to be added up. Docker mode still has fixed limits that leave the queue out:
`runclient`'s 45 s HTTP timeout and the runner container's 45 s `WriteTimeout`.

## Design notes

What I'd change, in the order the cracks appear:

1. **Split the lesson bundle.** Done on 2026-11-14: the main chunk keeps an index
   (titles, header facts, unconfirmed claims; the sidebar and paths read the curriculum
   module) and each lesson loads with a dynamic `import()` from a chunk of its own. The
   first chunk went from 4.7 MB to 0.48 MB of JavaScript. Serving lessons from the
   backend, as ADR-0004 planned, stays the option if search ever needs the lesson bodies.
2. **Verify only what changed.** Hash each lesson directory and skip unchanged ones,
   the way ADR-0004's planned `content_hash` upserts would have. Running server examples
   two at a time gains at most 2x, because the slots are the limit.
3. **Size the slots to the machine.** Make `maxConcurrent` configurable and default it
   from the number of cores, and give verify a lower priority than the Run button (a
   separate slot, or no retries during interactive use).
4. **Bound run history.** Done in the most boring form: `runs` gets the code and the full
   result JSON on every run, and each insert now deletes all but the newest 1,000 rows
   in the same transaction (`runstore.DefaultKeep`). Still open: no screen reads
   `/api/history`. A history screen would want the last N per lesson instead; until one
   exists, history could also be dropped.
5. **For a second learner, stop and redesign.** A users table and `user_id` in every
   learning key (example 3), authentication, and lesson execution outside the backend
   process: the Docker runner, or gVisor/Firecracker as ADR-0003 suggests. That is a
   different application with the same lessons.

What I would not change: the stdlib router, pgx, the per-run `run_` schemas, content as
files, the strategy split between browser and server runners. They are what made this
small app easy to read, and none of them is the first thing to break.
