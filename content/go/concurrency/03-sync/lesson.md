---
id: go.concurrency.sync
title: "sync: WaitGroup (and wg.Go), Mutex/RWMutex, Once/OnceValue, atomic"
track: go
module: concurrency
order: 3
level: intermediate
minutes: 14
since: go1.25
prerequisites: [go.concurrency.goroutines-channels]
compare_with: [sql.transactions.row-locks, design.paradigms.concurrency, ts.async.event-loop]
tags: [concurrency, sync, mutex, waitgroup, atomic, once]
summary: >
  When goroutines share memory instead of passing it, package sync guards it:
  WaitGroup waits for them, Mutex and RWMutex serialise access, OnceValue runs
  initialisation exactly once, and sync/atomic handles lone counters and flags.
examples:
  - id: waitgroup-mutex
    file: examples/01_waitgroup_mutex.go
    title: wg.Go starts and counts goroutines; a Mutex guards the shared map
    runtime: go
    check: exact
    annotations:
      - lines: "13-16"
        note: >
          The mutex sits next to the data it guards, and both are unexported:
          callers go through add and can't forget the lock. The zero Mutex is
          unlocked and ready, so there is no constructor.
      - lines: "19-21"
        note: >
          Lock, then defer Unlock on the next line, so an early return or a
          panic can't leave it locked. Go's Mutex isn't reentrant: calling add
          from inside add would block forever, where Java's synchronized would
          let the same thread in again.
      - lines: "34-38"
        note: >
          wg.Go (Go 1.25) adds 1 before the goroutine starts and calls Done when
          the function returns. The older wg.Add(1); go func() { defer wg.Done()
          ... }() form is what go fix's waitgroupgo analyzer rewrites into this.
          Each iteration has its own line variable (Go 1.22), so no copy is needed.
      - lines: "40-46"
        note: >
          Every goroutine's return happens before Wait returns, so their writes
          are visible here and reading the map without the lock is safe. Before
          Wait, it would be a data race.
  - id: once-values
    file: examples/02_once_values.go
    title: OnceValues for lazy initialisation, errors included
    runtime: go
    check: exact
    annotations:
      - lines: "13"
        note: >
          atomic.Int32 (Go 1.19) is the typed form: the value can only be
          reached through atomic methods, unlike a plain int32 handed to
          atomic.AddInt32 and read directly elsewhere.
      - lines: "17-20"
        note: >
          OnceValues (Go 1.21) returns a function that runs f at most once.
          Callers that arrive while it runs wait for it, then everyone gets
          the same values. It replaces sync.Once plus package-level variables
          for the result and the error.
      - lines: "22-29"
        note: >
          Each goroutine writes its own element of a pre-sized slice. Distinct
          elements are distinct memory locations, so this needs no lock and
          is race-free.
      - lines: "34-45"
        note: >
          The error is cached too: three calls, one dial. If a failed attempt
          should be retried later, OnceValues is the wrong tool; keep the
          connection in a mutex-guarded field that stays nil until a dial
          succeeds.
  - id: rwmutex-atomic
    file: examples/03_rwmutex_atomic.go
    title: RWMutex for read-mostly data, atomics for counters
    runtime: go
    check: exact
    annotations:
      - lines: "11-16"
        note: >
          RWMutex admits many readers or one writer. It helps only when reads
          dominate: a waiting Lock blocks new RLock calls, so with frequent
          writes readers queue behind every writer anyway.
      - lines: "19-26"
        note: >
          Several readers hold RLock at the same time, so c.hits++ on a plain
          int here would be a data race. RLock protects reading the map, not
          writing the counters; that's why the counters are atomics.
      - lines: "31-33"
        note: >
          Writes take the exclusive lock. Concurrent writes to a Go map can
          crash the program with "fatal error: concurrent map writes", a fatal
          error rather than a panic, so recover can't catch it.
      - lines: "39-50"
        note: >
          The 100 goroutines interleave differently on every run, yet the
          totals are fixed: 50 readers times two hits, 50 misses for "fr",
          and 2 + 50 entries.
use_cases:
  - "WaitGroup: wait for a batch of goroutines that report through shared memory or not at all; when they can fail, use errgroup (lesson 6)"
  - "Mutex: a map or struct that many request goroutines update, such as an in-memory cache, a session table or per-client rate-limit state"
  - "RWMutex: read-mostly data such as a routing table or a feature-flag set reloaded once a minute"
  - "OnceValue/OnceValues: lazy initialisation of templates, compiled regexps or a client, instead of init() or a hand-written nil check under a lock"
  - "atomic: counters, flags, and swapping in a new immutable config snapshot with atomic.Pointer[T]"
strengths:
  - "Zero values are ready to use: var mu sync.Mutex and var wg sync.WaitGroup need no constructor, unlike Java's new ReentrantLock()"
  - "go vet catches copied locks (copylocks) and wg.Add called inside the new goroutine (waitgroup); javac and C++ compilers check neither"
  - "OnceValue is lazy, thread-safe initialisation in one line, without Java's double-checked locking and its volatile subtleties"
  - "Atomics are sequentially consistent, the semantics of C++'s default std::atomic and Java's volatile, so there are no memory orderings to choose"
weaknesses:
  - "Nothing ties a mutex to the data it guards. Unlike Rust's Mutex<T>, the compiler can't tell you a field was read without the lock; only the race detector can, at run time"
  - "No reentrancy: a goroutine that locks a mutex it already holds blocks forever"
  - "No lock timeouts like Java's tryLock(timeout, unit); TryLock exists, but its own docs call correct uses rare"
pitfalls:
  - title: "Copying a value that contains a lock"
    detail: "A value receiver or a struct passed by value copies the mutex, and each copy locks independently, which protects nothing. Use pointer receivers; go vet's copylocks check reports the copy."
    bad: "func (c cache) get(k string) string { c.mu.RLock(); defer c.mu.RUnlock(); return c.data[k] }"
    good: "func (c *cache) get(k string) string { c.mu.RLock(); defer c.mu.RUnlock(); return c.data[k] }"
  - title: "Calling wg.Add inside the goroutine"
    detail: "Wait may run before the goroutine has called Add and return early. Call Add before the go statement, or let wg.Go do both; go vet's waitgroup analyzer (Go 1.25) reports the misplaced Add."
    bad: "go func() { wg.Add(1); defer wg.Done(); work() }()"
    good: "wg.Go(work)"
  - title: "Locking the same mutex twice in one call chain"
    detail: "A locked method that calls another locked method on the same value blocks forever. Split it into a public method that locks and an unexported helper that assumes the lock is held."
    bad: "func (s *Store) Move(a, b string) { s.mu.Lock(); defer s.mu.Unlock(); s.Delete(a); s.Put(b) } // Delete and Put lock s.mu too"
    good: "func (s *Store) Move(a, b string) { s.mu.Lock(); defer s.mu.Unlock(); s.deleteLocked(a); s.putLocked(b) }"
  - title: "Holding a lock across I/O"
    detail: "An HTTP call or a database query under a mutex serialises every caller behind the slowest request. Copy what you need, unlock, then do the I/O."
quiz:
  - q: "A method holds mu.RLock() and increments a plain int field. Is that safe?"
    options: ["Yes, it holds a lock", "No: several readers can hold RLock at once, so the increment races", "Only when GOMAXPROCS is 1"]
    answer: 1
    why: "RLock is shared. Writes need Lock, or the counter needs to be an atomic."
  - q: "The function passed to sync.OnceValues returned an error on its first call. What do later calls return?"
    options: ["The same error, without calling the function", "They call the function again", "A zero value and nil"]
    answer: 0
    why: "OnceValues runs the function once and caches both results, the error included."
  - q: "A method holding s.mu.Lock() calls another method of s that also calls s.mu.Lock(). What happens?"
    options: ["It works, the lock is reentrant", "Lock panics", "The goroutine blocks forever"]
    answer: 2
    why: "Go's Mutex has no owner and no reentrancy, so the second Lock waits for an Unlock that never comes."
  - q: "Many goroutines increment one shared counter. What is the simplest correct choice?"
    options: ["A plain int", "atomic.Int64", "sync.Map"]
    answer: 1
    why: "A lone counter is what atomics are for. A plain int races, and sync.Map is a map for specific access patterns, not a counter."
further_reading:
  - title: "package sync"
    url: "https://pkg.go.dev/sync"
  - title: "package sync/atomic"
    url: "https://pkg.go.dev/sync/atomic"
  - title: "The Go Memory Model"
    url: "https://go.dev/ref/mem"
  - title: "Go 1.25 release notes: WaitGroup.Go and the waitgroup vet check"
    url: "https://go.dev/doc/go1.25"
---

## Concept

Lesson 1 passed data between goroutines through channels. The alternative is to share
memory and guard it. Package `sync` and `sync/atomic` provide the tools:

- **`WaitGroup`**: wait until a set of goroutines has finished. `wg.Go(f)` (Go 1.25)
  starts `f` in a goroutine and counts it; `wg.Wait()` blocks until all have returned.
- **`Mutex`**: one goroutine at a time in the critical section. **`RWMutex`**: many
  readers or one writer.
- **`Once`**, **`OnceFunc`**, **`OnceValue`**, **`OnceValues`** (the last three since
  Go 1.21): run something exactly once, on first use, safely from any goroutine.
- **`sync/atomic`**: typed atomics such as `atomic.Int64`, `atomic.Bool` and
  `atomic.Pointer[T]` (Go 1.19) for single values.

The types (`WaitGroup`, `Mutex`, `RWMutex`, `Once`, the atomics) are ready to use as zero
values and must not be copied after first use.

## Coming from C/C++/Java/Python/JS

- **Java**: `Mutex` ≈ `ReentrantLock` without the reentrancy, and there is no
  `synchronized` keyword, so a lock is always an explicit field. `WaitGroup` ≈
  `CountDownLatch` with a counter you can increase. `OnceValue` replaces lazy holders
  and double-checked locking. `atomic.Int64` ≈ `AtomicLong`.
- **C/C++**: `Mutex` ≈ `std::mutex`, with `defer mu.Unlock()` in place of
  `std::lock_guard`. Go atomics are always sequentially consistent, like
  `std::atomic` with the default `memory_order_seq_cst`; there is no relaxed mode.
- **Python**: `threading.Lock` maps directly. The difference: goroutines run in parallel
  on all cores, with no GIL serialising them, so a missing lock corrupts data quickly.
- **JS**: one event loop never needs locks between callbacks. Go's goroutines are
  preemptively scheduled across threads, so any shared write needs one of these tools.

## How it works

The guarantees come from the Go memory model, which defines "synchronizes before":
an `Unlock` synchronizes before the next `Lock`, the return of a `wg.Go` function before
the `Wait` it releases, and the return of a `Once` function before any `once.Do` call
returns. Without such an edge, a write in one goroutine and a read in another are a
**data race**: the read may see the old value or the new one, and for multiword values
such as strings, slices and interfaces a torn mix that can corrupt memory. Lesson 6
shows how the race detector finds them.

Typed atomics are the safer form. `var n int64` plus `atomic.AddInt64(&n, 1)` compiles
fine next to a plain `n++` somewhere else; `var n atomic.Int64` has no plain access at
all. In the Go 1.27 toolchain, `go fix`'s `atomictypes` analyzer rewrites the old form
into the new one.

## Design notes

Channels or mutexes? Use a channel when ownership moves, such as handing a job to a
worker or passing a result back. Use a mutex when several goroutines need short access
to the same structure, like a cache or a counter table. "Share memory by communicating"
is the default, not a ban: a mutex around a map is simpler than a goroutine that owns
the map and serves requests over channels.

Keep critical sections small and free of I/O. Put the mutex in the struct it protects,
unexported, with methods that lock, so callers can't forget. Prefer `Mutex` over
`RWMutex` until reads clearly dominate.

`sync.Map` is not a general replacement for a map plus a mutex. Its docs say most code
should use a plain map, and name the two cases it is optimized for: keys written once
and read many times, and goroutines working on disjoint key sets. Otherwise a plain map
with a mutex keeps type safety and lets you maintain other invariants under the same
lock.
