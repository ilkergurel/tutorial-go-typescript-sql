---
id: go.cli.slog
title: "log/slog: levels, attributes, handlers, JSON output"
track: go
module: cli
order: 4
level: intermediate
minutes: 14
since: go1.26
prerequisites: [go.cli.fmt, go.interfaces.implicit]
compare_with: [go.http.middleware, go.concurrency.context]
tags: [slog, logging, json, observability, levels]
summary: >
  log/slog (Go 1.21) logs a message plus typed key-value attributes; a Handler
  turns each record into text for people or JSON for machines. Loggers carry
  context with With, levels change at run time with a LevelVar.
examples:
  - id: text-json
    file: examples/01_text_json.go
    title: The same calls through TextHandler and JSONHandler
    runtime: go
    check: exact
    annotations:
      - lines: "10-17"
        note: >
          ReplaceAttr sees every attribute before it is written, including the
          built-in time, level and msg. Returning a zero Attr drops it. The same
          hook renames keys or masks personal data for a whole handler.
      - lines: "20"
        note: >
          Before SetDefault, the top-level functions go through the old log
          package: stderr, its timestamp format, nothing in the compared output.
      - lines: "23-28"
        note: >
          The call sites don't know the format; the Handler decides. Switching a
          service to JSON is a one-line change at startup. The Debug call
          produces nothing, because a handler's default minimum level is Info.
      - lines: "29"
        note: >
          Alternating keys and values. TextHandler quotes a value only when it
          contains spaces, non-printing characters, quotes or "=", so İzmir
          stays bare and "Ayşe Yılmaz" is quoted. JSON keeps the UTF-8 as is,
          without \u escapes.
      - lines: "30-32"
        note: >
          Typed constructors (slog.Duration, slog.Int) can't get out of step
          the way loose key-value pairs can. Group nests attributes: db.table in
          text, a nested object in JSON. A Duration is "1.5s" in text but
          nanoseconds (1500000000) in JSON.
      - lines: "33"
        note: >
          An error value is written as its Error() string in both formats. Use a
          consistent key such as "err" so you can search for it.
  - id: with-levels
    file: examples/02_with_levels.go
    title: "With, WithGroup, a LevelVar, and LogValuer for redaction"
    runtime: go
    check: exact
    annotations:
      - lines: "8-19"
        note: >
          A type with a LogValue method decides its own log form, and the
          built-in handlers resolve it before writing. Token logs as REDACTED and
          User as its ID, without the e-mail, and no call site has to remember.
      - lines: "29-31"
        note: >
          A LevelVar is a Leveler you can change while the program runs; its
          zero value is Info. HandlerOptions.Level also accepts a fixed level
          such as slog.LevelWarn.
      - lines: "34"
        note: >
          With returns a new logger that adds request_id and method to every
          record. Pass reqLog down the request instead of repeating the IDs at
          each call.
      - lines: "36-39"
        note: >
          The first Debug is dropped, the second appears: same code, level
          changed by level.Set with no restart.
      - lines: "41"
        note: >
          WithGroup prefixes the keys of later attributes: payment.amount in
          text, a "payment" object in JSON. Attributes added earlier by With
          stay outside the group.
      - lines: "43-44"
        note: >
          A key without a value compiles; the handler prints !BADKEY. go vet's
          slog check reports the call ("missing a final value").
  - id: multi-default
    file: examples/03_multi_default.go
    title: "One logger, two audiences: NewMultiHandler (Go 1.26) and SetDefault"
    runtime: go
    check: exact
    annotations:
      - lines: "17-22"
        note: >
          Each handler has its own minimum level and format. stdout stands in
          for a log collector here; a CLI whose stdout carries results sends
          all logs to stderr instead.
      - lines: "23"
        note: >
          NewMultiHandler (Go 1.26) passes each record to every handler that is
          enabled for its level. Before 1.26 this took a small hand-written
          Handler. SetDefault makes the logger the target of slog.Info and friends.
      - lines: "25-26"
        note: >
          Both records reach the JSON handler; only the warning reaches the text
          handler, which is the line in the stderr pane.
      - lines: "28-29"
        note: >
          After SetDefault, log.Printf output becomes an Info record on the same
          handler, so old code joins the structured stream unchanged while you
          migrate it.
use_cases:
  - "Services: one JSON object per line on stdout, for a collector such as Loki, Elasticsearch or CloudWatch"
  - "Request-scoped loggers: logger.With(\"request_id\", id) created in middleware and passed down"
  - "Raising verbosity in production for a while with a LevelVar, without a restart"
  - "Migrating a codebase from log.Printf gradually: SetDefault routes the old calls through the new handler"
strengths:
  - "In the standard library since Go 1.21, so most programs need no logging dependency; in Java, java.util.logging, SLF4J and Log4j still compete"
  - "Call sites are format-agnostic: the Handler chooses text or JSON, and switching is one line at startup"
  - "Redaction lives on the type (LogValuer), not at hundreds of call sites"
  - "Logger.LogAttrs with typed attributes avoids allocations, so logging on hot paths stays cheap"
weaknesses:
  - "Key-value arguments are ...any: a missing value compiles and prints !BADKEY; only go vet catches it"
  - "Arguments are evaluated before the level check: Debug(\"x\", \"dump\", expensive()) pays for expensive() even when Debug is off"
  - "The built-in handlers ignore the context.Context: getting a request or trace ID from ctx into the output needs a small custom Handler"
  - "TextHandler joins group keys with dots without escaping, so a.b.c is ambiguous to a parser; use JSON when machines read the logs"
pitfalls:
  - title: "Unpaired keys and values"
    detail: "The variadic list is untyped. Run go vet, whose slog check catches it (the vet subset that go test runs does not include it), or use slog.String, slog.Int and friends."
    bad: "logger.Info(\"login\", \"user\")"
    good: "logger.Info(\"login\", slog.String(\"user\", name))"
  - title: "Paying for disabled debug logs"
    detail: "Arguments are computed before the handler checks the level. Wrap expensive values in a LogValuer, or guard the call with logger.Enabled(ctx, slog.LevelDebug)."
  - title: "Durations in JSON are nanoseconds"
    detail: "slog.Duration becomes an integer in JSON (1500000000) but 1.5s in text. Pick a unit in the key (took_ms) or convert in ReplaceAttr so dashboards agree."
  - title: "Secrets and personal data in logs"
    detail: "Log an e-mail address once and it is in every backup of your log store. Give sensitive types a LogValue method, or mask keys in ReplaceAttr."
quiz:
  - q: "JSONHandler logs slog.Duration(\"took\", 1500*time.Millisecond). What appears in the output?"
    options: ["\"took\":\"1.5s\"", "\"took\":1500000000", "\"took\":1.5"]
    answer: 1
    why: "A Duration is an int64 count of nanoseconds, and the JSON handler writes the number; TextHandler writes the String form."
  - q: "logger.Info(\"login\", \"user\") has a key without a value. What happens?"
    options: ["Compile error", "Panic at run time", "The record is logged with !BADKEY=user; go vet warns"]
    answer: 2
    why: "The arguments are ...any, so the compiler accepts it. The handler marks the stray value, and vet's slog check reports the call."
  - q: "Where does a type-wide rule like \"never log the API token\" belong?"
    options: ["In a LogValue method on the token type", "At each call site", "In a comment above the struct"]
    answer: 0
    why: "The built-in handlers resolve LogValuer values before writing, so the rule holds wherever the value is logged."
  - q: "After slog.SetDefault(l), what happens to log.Printf calls?"
    options: ["They still print in the old format to stderr", "They are dropped", "They become Info records on l's handler"]
    answer: 2
    why: "SetDefault also redirects the log package's default logger to l's Handler, at the level set by SetLogLoggerLevel (Info by default)."
further_reading:
  - title: "package log/slog"
    url: "https://pkg.go.dev/log/slog"
  - title: "Structured Logging with slog (Go blog)"
    url: "https://go.dev/blog/slog"
  - title: "Go 1.26 release notes"
    url: "https://go.dev/doc/go1.26"
---

## Concept

A structured log record is a level, a message and a list of typed attributes, not one
formatted string. `log/slog` (Go 1.21) separates the two halves: a `Logger` collects
records at call sites, and a `Handler` decides which records to keep and how to write
them. The standard library ships `TextHandler` (`key=value` lines for people) and
`JSONHandler` (one JSON object per line for machines). Because messages stay constant
and the variables become attributes, you can search for `msg="payment failed"` and
group by `err` instead of writing regular expressions over prose.

## Coming from C/C++/Java/Python/JS

- Java: `Logger` ≈ an SLF4J logger, `Handler` ≈ a Logback appender plus encoder,
  `With` ≈ MDC values, but explicit and passed as a value instead of thread-local.
- Python: `logging` with a JSON formatter and `extra={...}`; `logger.With` is closer to
  structlog's `bind`. Levels are numbers in both (slog: Debug -4, Info 0, Warn 4, Error 8).
- Node: very close to pino: one JSON object per line, child loggers (`pino.child`) ≈
  `With`, levels filtered before formatting.
- C/C++: no standard equivalent; spdlog-style sinks are the `Handler` idea.

## Design notes

Configure logging once in `main`: build the handler (JSON in production, text in a
terminal) with a `LevelVar` as its level, and pass the `*slog.Logger` to the
components that log, as a constructor parameter. `slog.SetDefault` is for the parts you
don't control, such as libraries still calling `log.Printf`. Request-scoped attributes
come from `With` in the HTTP middleware.

Keep messages constant and put variables in attributes. Choose key names once
(`err`, `user_id`, `request_id`) and stick to them: the log store indexes keys, not
your intentions. For a CLI, logs and diagnostics go to stderr so stdout stays clean for
results, as in the fmt lesson; for a service, the collector usually reads stdout.
