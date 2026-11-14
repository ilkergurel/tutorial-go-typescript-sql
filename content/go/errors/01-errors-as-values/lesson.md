---
id: go.errors.errors-as-values
title: "Errors as values: wrapping with %w and adding context"
track: go
module: errors
order: 1
level: basic
minutes: 10
since: go1.13
prerequisites: [go.basics.functions]
compare_with: [ts.errors.try-catch, design.paradigms.errors, sql.constraints.keys]
tags: [errors, wrapping, fmt]
summary: >
  Errors are ordinary values you return and check. Wrap them with %w to add
  context without losing the cause, and inspect them later with errors.Is.
examples:
  - id: wrap
    file: examples/01_wrap.go
    title: Wrapping adds context; errors.Is still finds the cause
    runtime: go
    check: contains
    annotations:
      - lines: "10-15"
        note: >
          os.ReadFile already says "open users.txt" — but not who wanted it or
          why. Wrapping adds the loadConfig frame; it doesn't discard anything.
      - lines: "14"
        note: >
          %w (not %v) is what makes wrapping work: it embeds the original error
          so errors.Is/As can unwrap. %v would only format text.
      - lines: "21"
        note: >
          The text after the last colon comes from the operating system: "no
          such file or directory" on Linux and macOS, "The system cannot find
          the file specified." on Windows. That is why this example's check
          only looks at the parts Go produced.
      - lines: "23-24"
        note: >
          errors.Is walks the whole chain with ==, so a sentinel like
          fs.ErrNotExist matches even three wrappers deep, on every OS. A string
          match on the message would be brittle and locale-hostile.
use_cases:
  - "Boundary functions that translate low-level errors into your domain"
  - "Adding what the operation was doing (query, file, service) without losing WHY it failed"
strengths:
  - "Unlike Java checked exceptions, the compiler won't force you to handle the error — errcheck-style linters fill that role"
  - "Wrapping keeps one error chain instead of exception hierarchies; the cause and the context travel together"
  - "Sentinels like fs.ErrNotExist are comparable values: errors.Is is precise string-free matching"
weaknesses:
  - "Nothing enforces checking a returned error; ignoring one is a silent bug"
  - "Deeply wrapped chains can produce noisy messages if every layer adds context"
  - "No stack trace by default: you add context deliberately instead"
pitfalls:
  - title: "Wrapping with %v instead of %w"
    detail: "The message looks right, but errors.Is/As can no longer see the cause. Always %w when you want unwrapping."
  - title: "Checking err == fs.ErrNotExist on a wrapped error"
    detail: "== only matches the immediate value. Wrapped errors must be inspected with errors.Is."
  - title: "Adding context twice"
    detail: "\"open users.txt: open users.txt\". Wrap where you add new information, not to re-log what the message already says."
quiz:
  - q: "Which verb preserves the wrapped error for errors.Is?"
    options: ["%v", "%s", "%w"]
    answer: 2
    why: "%w embeds the error value; %v and %s only format its message into the new string."
  - q: "errors.Is(err, fs.ErrNotExist) on err = fmt.Errorf(\"load: %w\", fmt.Errorf(\"open: %w\", fs.ErrNotExist))?"
    options: ["false — it's wrapped twice", "true — Is unwraps the chain", "true only with errors.As"]
    answer: 1
    why: "errors.Is walks Unwrap() repeatedly until it finds a match or the chain ends."
further_reading:
  - title: "Error handling and Go"
    url: "https://go.dev/blog/errors-are-values"
  - title: "Working with Errors in Go 1.13"
    url: "https://go.dev/blog/go1.13-errors"
  - title: "package errors"
    url: "https://pkg.go.dev/errors"
---

## Concept

A Go error is an ordinary value returned alongside your result: `value, err := doThing()`.
You check it, add context with `fmt.Errorf` and `%w`, and pass it up. `errors.Is` and
`errors.As` inspect the wrapped chain without knowing how many layers added context.

## Coming from C/C++/Java/Python/JS

- Java/Python/JS: exceptions unwind for you. In Go the `if err != nil` check is the unwind
  — visible, local, and impossible to trigger from a line you didn't expect to throw.
- C: errno plus return codes, but the error carries a message and an identity (`==`-able
  sentinels) instead of a global mutable code.
- Rust: `Result<T, E>` with `?` ≈ `(T, error)` with `if err != nil`; Go's `return err` is
  the explicit version of `?`.

## Design notes

Add context at each layer that knows something new ("loading config", "querying orders for
user 42"), and let `errors.Is`/`As` answer inspection questions at the top. Handle the error
once: either respond to it meaningfully or wrap and pass it up — logging it *and* returning
it produces duplicate log lines for one failure. This lesson's strategy across layers
(sentinel vs typed errors, where to recover) is its own later lesson.
