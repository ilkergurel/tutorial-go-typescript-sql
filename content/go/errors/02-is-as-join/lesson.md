---
id: go.errors.is-as-join
title: "errors.Is, errors.As, errors.Join; sentinels vs error types"
track: go
module: errors
order: 2
level: intermediate
minutes: 14
since: go1.26
prerequisites: [go.errors.errors-as-values]
compare_with: [ts.errors.try-catch, ts.errors.result, go.interfaces.assertions, design.paradigms.errors]
tags: [errors, sentinel, errors-is, errors-as, errors-join, wrapping]
summary: >
  Sentinels answer "which failure?", error types carry data, and errors.Is,
  errors.As and errors.AsType find either anywhere in a wrapped tree. Whatever
  you wrap with %w becomes your API, so translate driver errors at the boundary.
examples:
  - id: sentinel-vs-type
    file: examples/01_sentinel_vs_type.go
    title: Sentinels with errors.Is, error types with errors.AsType and errors.As
    runtime: go
    check: exact
    annotations:
      - lines: "8-9"
        note: >
          A sentinel is matched by identity, so its message text plays no part.
          Exporting one makes it API: callers will depend on it. io.EOF and
          sql.ErrNoRows are the standard library's best-known sentinels.
      - lines: "11-16"
        note: >
          An error type carries fields the caller can act on. Error has a
          pointer receiver, so *ValidationError is the error type, and it is
          the type to ask for below.
      - lines: "29-32"
        note: >
          == compares only the outermost value, the wrapper fmt.Errorf built on
          line 23. errors.Is walks the chain, comparing each link with == or
          asking the link's own Is method.
      - lines: "34-38"
        note: >
          A type assertion has the same blind spot as ==: it sees only the
          outer wrapper, so one more layer of context breaks it. Never
          type-assert errors you didn't just create.
      - lines: "40-42"
        note: >
          errors.AsType (Go 1.26) returns the first *ValidationError in the
          chain, already typed. errors.AsType[ValidationError] wouldn't
          compile: the value type has no Error method.
      - lines: "43-46"
        note: >
          errors.As does the same through a pointer to a target variable; code
          written before Go 1.26 uses it. Passing verr instead of &verr panics
          at run time, and go vet's errorsas check reports it.
  - id: join
    file: examples/02_join.go
    title: errors.Join and several %w build a tree that Is and As search
    runtime: go
    check: exact
    annotations:
      - lines: "26-29"
        note: >
          Limits on names and places should count characters. len counts UTF-8
          bytes: "Şanlıurfa" is 9 runes but 11 bytes, and a byte limit of 10
          would reject a valid Turkish city.
      - lines: "30"
        note: >
          Collect, then join once. errors.Join drops nil values and returns
          nil when nothing is left, so the valid path (line 40) needs no
          special case.
      - lines: "34-35"
        note: >
          A joined error prints one line per error, which suits logs. An API
          that returns a list of field errors can get the parts through the
          Unwrap() []error method that Join's result implements.
      - lines: "36-39"
        note: >
          Is and As search the whole tree depth-first, so each check finds its
          error whichever branch holds it.
      - lines: "42-47"
        note: >
          Since Go 1.20 fmt.Errorf accepts several %w, and the result unwraps
          to all of them. errors.Unwrap only follows Unwrap() error, so it
          returns nil here: inspect with Is and As, never by hand.
  - id: translate
    file: examples/03_translate.go
    title: Translating a driver error at the repository boundary
    runtime: go
    check: exact
    annotations:
      - lines: "8-12"
        note: >
          A stand-in so the example needs no database. The real type is pgx's
          *pgconn.PgError, and SQLSTATE 23505 is unique_violation in
          PostgreSQL's error code table.
      - lines: "22-28"
        note: >
          %w passes the driver type through, so callers can, and will, write
          errors.AsType[*pgconn.PgError]. Your service now depends on pgx, and
          changing the storage breaks code that never touched SQL.
      - lines: "34-36"
        note: >
          The translation: %w on ErrEmailTaken, %v on the driver error. The
          message keeps the SQLSTATE and constraint name for the log; the only
          link callers can inspect is the sentinel you chose to promise.
      - lines: "37-39"
        note: >
          Other driver errors stay opaque too: the caller logs them and answers
          500. If callers must detect something from below, such as
          context.Canceled, wrap that one on purpose.
      - lines: "44-48"
        note: >
          The same text, a different API: errors.Is succeeds only on the
          translated error, and only the leaky one exposes the driver type.
use_cases:
  - "Sentinels for conditions callers branch on: ErrNotFound, ErrEmailTaken, io.EOF, sql.ErrNoRows"
  - "Error types when callers need data: the field that failed validation, a retry-after duration, an HTTP status from an upstream API"
  - "errors.Join for form validation that reports every field at once, and for cleanup that fails after the main operation failed"
  - "Translating pgx or HTTP-client errors into domain errors inside the repository or client package"
strengths:
  - "Matching is by identity or type, never by message text, so messages can change (or be localized) without breaking callers, unlike parsing exception messages"
  - "errors.Is and errors.As walk any depth of wrapping, so each layer can add context without breaking inspection further up"
  - "Exposure is an explicit choice at each call site: %w makes the cause inspectable, %v keeps only its text, and the message reads the same either way"
  - "errors.AsType is checked at compile time: asking for a type that doesn't implement error fails to build instead of panicking"
weaknesses:
  - "Nothing tells the caller which sentinels and types a function can return; it lives in doc comments, unlike Java's throws clause"
  - "Every exported sentinel and error type is API surface you have to keep"
  - "Trees from errors.Join and several %w have no single cause: errors.Unwrap returns nil, and order of the search matters when two links match"
pitfalls:
  - title: "Comparing with == or asserting the type"
    detail: "err == ErrNotFound and err.(*MyErr) both fail as soon as any layer wraps the error. Use errors.Is and errors.AsType (errors.As before Go 1.26)."
  - title: "Matching on err.Error() text"
    detail: "Messages are not API, and OS text differs by platform and language: on Windows, Go asks for the English message and falls back to the system language when it isn't available. Match sentinels and types instead."
  - title: "Passing a non-pointer to errors.As"
    detail: "errors.As(err, verr) panics at run time; it needs &verr. go vet's errorsas check catches it, and errors.AsType makes the mistake impossible."
  - title: "Wrapping driver errors with %w out of habit"
    detail: "Callers start depending on *pgconn.PgError and SQLSTATE codes. Translate to your own sentinel with %w and keep the driver error as %v text."
quiz:
  - q: "err := fmt.Errorf(\"load: %w\", ErrNotFound). What does err == ErrNotFound return?"
    options: ["true", "false", "It doesn't compile"]
    answer: 1
    why: "err is the wrapper fmt.Errorf created; only errors.Is looks inside it."
  - q: "When is an error type better than a sentinel?"
    options: ["When the caller needs data about the failure, such as which field", "Always: types are faster", "When the error is returned from many places", "Never: sentinels are idiomatic"]
    answer: 0
    why: "A sentinel only identifies the failure; a type carries fields the caller can use."
  - q: "err := errors.Join(errA, errB). What does errors.Unwrap(err) return?"
    options: ["errA", "errB", "nil", "[errA errB]"]
    answer: 2
    why: "errors.Unwrap only calls Unwrap() error; Join's result has Unwrap() []error, which Is and As use instead."
  - q: "Your repository gets a *pgconn.PgError with code 23505. What should it return?"
    options: ["The PgError unchanged", "fmt.Errorf(\"create user: %w\", err)", "fmt.Errorf(\"create user: %w: %v\", ErrEmailTaken, err)", "errors.New(err.Error())"]
    answer: 2
    why: "Callers get an inspectable domain sentinel and the full text for logs, without depending on the driver's type."
further_reading:
  - title: "package errors"
    url: "https://pkg.go.dev/errors"
  - title: "The Go Blog: Working with Errors in Go 1.13 (Whether to wrap)"
    url: "https://go.dev/blog/go1.13-errors#whether-to-wrap"
  - title: "Go 1.20 Release Notes: wrapping multiple errors"
    url: "https://go.dev/doc/go1.20#errors"
  - title: "Go 1.26 Release Notes: errors.AsType"
    url: "https://go.dev/doc/go1.26#errorspkgerrors"
  - title: "PostgreSQL: Error codes (Appendix A)"
    url: "https://www.postgresql.org/docs/current/errcodes-appendix.html"
---

## Concept

There are two ways to make an error something callers can inspect. A sentinel is a
package-level value, `var ErrNotFound = errors.New("not found")`, checked with
`errors.Is(err, ErrNotFound)`. It answers "which failure was it?". An error type is a
struct with an `Error` method, checked with `errors.AsType[*ValidationError](err)` (Go
1.26) or `errors.As(err, &target)`. It answers "which failure, and with what details?".

Both functions search the chain `%w` builds, and since Go 1.20 that chain can be a tree:
`errors.Join` and `fmt.Errorf` with several `%w` produce errors with several children,
searched depth-first. A type can also customize matching with an `Is(error) bool` method.
That is why the previous lesson's `errors.Is(err, fs.ErrNotExist)` worked on Linux and
Windows alike: `syscall.Errno` has an `Is` method that maps each OS's own "file not
found" codes to `fs.ErrNotExist`.

The design rule comes from the Go blog: wrapping with `%w` makes the wrapped error part
of your API. Wrap what you are willing to support; format the rest with `%v`.

## Coming from C/C++/Java/Python/JS

- Java: `catch (ValidationException e)` looks only at the thrown exception;
  `errors.AsType[*ValidationError](err)` is that catch plus a walk down `getCause()`.
  `errors.Is` is the same walk looking for one specific instance. `errors.Join` plays
  the role of suppressed exceptions, but as an ordinary value.
- Python: `except ValidationError as e:` maps to `AsType`, `raise DomainError() from e`
  to `%w`, and `ExceptionGroup` (3.11) to `errors.Join`.
- JS/TS: `err instanceof ValidationError` plus `err.cause` (ES2022) is the same search
  done by hand; `AggregateError` corresponds to `errors.Join`.
- C: errno values are sentinels without a chain, and `strerror` text was never meant to be
  matched either.

## Design notes

Prefer, in this order: opaque errors (the caller only checks `err != nil`), then sentinels
for the few conditions callers really branch on, then error types when they need data.
Every exported sentinel and type is a promise, so add them when a caller needs one, not
in advance. Translate at package boundaries: the repository turns `*pgconn.PgError` with
SQLSTATE 23505 into `ErrEmailTaken`, the HTTP layer turns `ErrEmailTaken` into 409, and
neither knows the other's vocabulary. The next lesson puts that mapping in one place and
adds the panics that slip past it. Never match on `err.Error()` text: on Windows, Go
asks the OS for English messages and falls back to the system language, so the same
failure can read differently on different machines.
