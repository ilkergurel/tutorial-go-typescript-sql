---
id: design.patterns.anti-patterns
title: "Anti-patterns: Java getters/setters in Go, deep hierarchies in TS, god packages, anemic models"
track: design
module: patterns
order: 3
level: advanced
minutes: 14
since: go1.21
prerequisites: [design.patterns.classic, go.interfaces.methods, ts.narrowing.discriminated-unions]
compare_with: [go.design.packages, ts.architecture.patterns, sql.modeling.anti-patterns, go.postgres.bulk, design.architecture.business-rules]
tags: [anti-patterns, encapsulation, inheritance, set-based, n-plus-one]
summary: >
  Designs carried over from another language compile and run, and then fight you: Get/Set
  pairs and interfaces-per-struct in Go, inheritance trees in TypeScript, god packages,
  invariants scattered across services, and row-by-row loops against a set-based database.
examples:
  - id: getters-setters
    file: examples/01_getters_setters.go
    title: "Go: Get/Set pairs and a rule in a service vs a type that guards its own invariant"
    runtime: go
    check: exact
    annotations:
      - lines: "9-19"
        note: >
          Three Java habits at once: an I-prefixed interface declared next to its only
          implementation, Get on the getter, and a setter for a field that should never be
          set directly. Go's convention is the consumer declares the interface, when it
          needs one.
      - lines: "21-29"
        note: >
          The anemic model: the data has no behaviour and the rule lives in a service. Any
          code holding an IAccount can call SetBalance, so the rule is a suggestion, and
          this one forgot to reject negative amounts.
      - lines: "31-48"
        note: >
          The type owns the invariant. Unexported fields mean only this package can touch
          balance, so Withdraw is the only way to change it. Effective Go's naming: the
          getter is Balance; a setter, if one were justified, would be SetBalance.
      - lines: "55-59"
        note: >
          Withdrawing -400 deposited 400, and SetBalance(-50) skipped the service. Both
          compile, and neither leaves a trace of which caller broke the rule.
      - lines: "61-65"
        note: >
          The same three calls on Account: a bad amount and an overdraft are refused with
          errors the caller can test (errors.Is(err, ErrInsufficient)), and the balance only
          moves on success.
  - id: hierarchy-vs-union
    file: examples/02_hierarchy_vs_union.ts
    title: "TypeScript: an inheritance tree inherits the wrong answer; a union refuses to compile"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-25"
        note: >
          Behaviour is spread over three levels: label in Item and SqlLesson, minutes in
          Item and Lesson. To know what any object does you read upward through every
          ancestor, and a change to Item reaches classes you have never opened.
      - lines: "26-31"
        note: >
          The new class compiles, runs and is wrong: an exam inherits Lesson's formula and
          reports 4 minutes for 30 questions, while carrying an examples field of 0 that
          means nothing for it. Nothing forced anyone to decide what an exam's duration is.
      - lines: "37-49"
        note: >
          The variants are plain data and the behaviour is one function with a case per
          variant. The declared return type is the exhaustiveness check: every path must
          return a number, so a variant without a case is a compile error.
      - lines: "50-51"
        note: >
          SqlLesson's whole reason to exist becomes one boolean field read in one place.
          Variation that is data should be data, not a subclass.
      - lines: "60-61"
        note: >
          The failure the class tree could not give you. The message is tsc's (checked
          with tsc 7.0.2), and it points at the switch that needs a decision.
  - id: row-by-row
    file: examples/03_row_by_row.go
    title: "SQL: row by row from Go vs one statement over arrays vs one set-based UPDATE"
    runtime: go
    check: exact
    annotations:
      - lines: "13-20"
        note: >
          A pgx QueryTracer sees every Query, QueryRow and Exec, which makes "how many
          statements did this code send?" a number instead of a guess. The same hook is
          where tracing and query logging plug in.
      - lines: "39-52"
        note: >
          The ORM habit: load objects, change them in memory, save each one. 301
          statements, each waiting for its reply before the next is sent, and outside a
          transaction each UPDATE is also its own commit.
      - lines: "54-63"
        note: >
          When the rule must stay in Go, keep it there and still write once: two arrays
          and unnest turn 300 updates into one statement. pgx encodes a []string and a
          []int as PostgreSQL arrays.
      - lines: "65-69"
        note: >
          When the rule can be said in SQL, say it about all rows at once. Integer division
          truncates in both languages, so this matches newPrice exactly.
      - lines: "70-73"
        note: >
          The three tables end up identical: same result, 301 statements against 1.
use_cases:
  - "Reviewing code from a team that is new to Go or TS: these are the shapes that show up first"
  - "Deciding whether a type needs methods that guard an invariant (money, state machines, quotas) or can stay plain data (DTOs, config, rows)"
  - "Replacing an inheritance tree with a discriminated union when variants are known and behaviour is spread across levels"
  - "Finding N+1 and row-by-row loops: count statements per request with a tracer before measuring anything else"
strengths:
  - "Go's package-level privacy makes invariants cheap to enforce: unexported fields plus a few methods, with no need for Java's private/protected/package matrix"
  - "TS discriminated unions turn 'forgot a variant' into a compile error, where a class hierarchy turns it into a silently inherited default"
  - "Set-based SQL lets the database do in one statement, inside one transaction, what a loop does in hundreds of round trips"
  - "Each fix removes code: fewer interfaces, fewer classes, fewer statements"
weaknesses:
  - "Go cannot stop code inside the same package from writing unexported fields directly, so a large package weakens every invariant it holds"
  - "A union with an exhaustive switch makes adding a variant touch every switch; a class hierarchy makes adding an operation touch every class. Pick by which change is more frequent"
  - "Set-based rewrites move logic into SQL text that Go's compiler does not check; keep such statements covered by tests on a real database"
pitfalls:
  - title: "God package"
    detail: "A package named after a layer or a mood (util, common, models, services) that everything imports. It cannot be understood alone, every change recompiles its importers, and it is where import cycles start. Split by feature and move helpers next to their only caller (go.design.packages)."
  - title: "Getters and setters for every field"
    detail: "Plain data with no invariant should have exported fields and no methods. A type with an invariant should have unexported fields and operations, not setters. A Get/Set pair for every field is the worst of both: ceremony without protection."
    bad: |
      func (u *User) GetEmail() string  { return u.email }
      func (u *User) SetEmail(e string) { u.email = e }
    good: |
      type User struct{ Email string } // plain data, or:
      func (u *User) ChangeEmail(e string) error { /* validate, then set */ }
  - title: "Inheritance for code reuse"
    detail: "Extending a class to borrow two methods couples the child to every future change in the parent. In TS, prefer functions over the data, or composition; in Go there is no inheritance, and struct embedding promotes methods without making the outer type a subtype."
  - title: "N+1 queries from a loop"
    detail: "One query for the list, then one per item for its details. Fetch the details with one query (a JOIN, or WHERE id = ANY($1)) and match them up in Go."
    bad: |
      for _, o := range orders {
          conn.QueryRow(ctx, "SELECT name FROM customers WHERE id = $1", o.CustomerID).Scan(&o.Customer)
      }
    good: |
      rows, _ := conn.Query(ctx, "SELECT id, name FROM customers WHERE id = ANY($1)", customerIDs)
quiz:
  - q: "What does Effective Go say a getter for the unexported field owner should be called?"
    options: ["GetOwner", "Owner", "owner", "OwnerGetter"]
    answer: 1
    why: "Upper-case Owner exports it; putting Get in the name is neither idiomatic nor necessary. A setter, if needed, would be SetOwner."
  - q: "Exam extends Lesson and forgets to override minutes(). What happens?"
    options: ["tsc reports a missing override", "It throws at run time", "It silently inherits Lesson's formula", "minutes() returns undefined"]
    answer: 2
    why: "An overridable method with a body is a default; nothing forces a subclass to decide. The union's switch, with a declared return type, does force it."
  - q: "The row-by-row loop sent 301 statements outside a transaction. Besides round trips, what else multiplied?"
    options: ["Index rebuilds", "Commits: each UPDATE committed on its own", "Table locks", "Nothing; autocommit is free"]
    answer: 1
    why: "Without BEGIN each statement is its own transaction, so 300 updates mean 300 commits, each waiting for its WAL record to reach disk under the default synchronous_commit."
  - q: "The pricing rule exists only in Go code. What is the idiomatic way to write 300 new prices?"
    options: ["One UPDATE per row inside a transaction", "One UPDATE ... FROM unnest($1, $2) with two arrays", "Move the rule into a trigger", "300 goroutines, one UPDATE each"]
    answer: 1
    why: "The rule stays in Go and the write is one statement. A transaction around the loop fixes the commits but not the round trips. Goroutines would need a connection each (a pgx.Conn is not safe for concurrent use) and still send 300 statements."
further_reading:
  - title: "Effective Go: Getters"
    url: "https://go.dev/doc/effective_go#Getters"
  - title: "Go Code Review Comments: Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
  - title: "TypeScript Handbook: Narrowing (discriminated unions, exhaustiveness checking)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "PostgreSQL: UPDATE (the FROM clause)"
    url: "https://www.postgresql.org/docs/current/sql-update.html"
  - title: "package pgx: QueryTracer"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#QueryTracer"
---

## Concept

An anti-pattern here is a design that is good in the language it came from and fights the
one it was carried into. The code compiles and runs; the cost shows up as ceremony,
invariants nobody enforces, and changes that ripple further than they should.

- **Java getters and setters in Go.** Go has package-level privacy and no
  properties. Its answer to "how do I protect this field" is an unexported field plus
  methods that *do something* (`Withdraw`), not a `Get`/`Set` pair that exposes it
  anyway. Plain data with no invariant is a struct with exported fields and no methods.
  An `I`-prefixed interface next to its only implementation is the same habit: in Go the
  consumer declares the interface it needs.
- **Anemic models.** Data types without behaviour and every rule in a "service". It is
  not wrong in itself: rows, DTOs and JSON payloads should be plain. It becomes an
  anti-pattern when the data *has* an invariant and nothing in the type enforces it, so
  every caller must remember the rule.
- **Deep hierarchies in TypeScript.** Inheritance spreads one behaviour over several
  classes and lets new subclasses silently inherit wrong defaults. When the set of
  variants is known, a discriminated union plus functions with exhaustive switches keeps
  each behaviour in one place and makes "forgot a variant" a compile error.
- **God packages.** `util`, `common`, `models`, `services`: packages named after a layer
  or a feeling, imported by everything. In Go they also weaken encapsulation, because
  every file in a package can touch every unexported field (go.design.packages).
- **Row by row against SQL.** The database is set-based. Loading rows, changing them in
  a loop and saving each one (the ORM habit, and its cousin the N+1 query) turns one
  statement into hundreds of round trips and commits.

## Coming from C/C++/Java/Python/JS

- **Java**: JavaBeans conventions (`getX`/`setX`), interfaces per service, and abstract
  base classes are the source of most of these. Effective Go says outright that a getter
  for `owner` is `Owner`, not `GetOwner`. Records (Java 16) are closer to Go's plain data
  structs than beans are.
- **C++**: deep hierarchies with virtual functions, and the fragile base class problem,
  are familiar. `std::variant` plus `std::visit` is the C++ form of a discriminated union
  with exhaustive handling.
- **Python**: properties make getters invisible, so the habit shows up less. Python's
  `match` on dataclasses is the closest thing to a TS union switch, without the compile
  error.
- **JS**: ES2015 classes made inheritance look natural in JavaScript. React's documentation
  recommended composition over component inheritance long before hooks made function
  components the default.

## How it works

The union's exhaustiveness check comes from control-flow analysis. Inside `switch
(c.kind)`, each `case` narrows `c` to one variant; after the last case, `c` has type
`never`. With a declared return type of `number`, falling off the end of the function is
only allowed if that point is unreachable. Add a variant and the end becomes reachable,
and tsc reports TS2366. The class hierarchy has no such point: a subclass that does not
override simply uses the parent's method, which is the feature that produced the bug.

The statement counts come from round trips. pgx sends a statement and waits for its
result before the next call can use the connection, so N statements cost at least N
network round trips, plus N commits when each runs in autocommit mode. An `UPDATE ...
FROM unnest(...)` or a single set-based `UPDATE` costs one of each, however many rows it
touches.

## Design notes

Each fix has a limit worth knowing. Unexported fields protect an invariant only from
other packages; a 5,000-line package is one trust zone, which is another reason to keep
packages small. Unions make new variants expensive (every switch must change) and new
operations cheap; class hierarchies are the opposite. When third parties add variants
you don't control (plugins, a framework's components), an interface or a class is the
right tool. And set-based SQL moves logic out of the compiler's sight, so test it on a
real database (go.postgres.repository-testing).

StackLab is a fair test case. The backend has no `util` package and no interface without
a consumer; the shared packages, `platform` (database plumbing) and `runapi` (the run
request and response types), have a fixed direction: features import them, they import
no feature. The stores are concrete types used directly, which is not
anemic: they hold no invariants beyond what the schema enforces. On the frontend,
`CodeRunner` is the only class pair, and `RunState` is a discriminated union; the
learning feature's notes state is the place where three separate `useState` calls
(`body`, `loaded`, `saved`) recreate the "anything can be set" problem (ts.architecture.explicit-states).

When you review code from someone new to a language, these five shapes are the first to
look for. None of them is a bug by itself; each one makes the next bug more likely.
