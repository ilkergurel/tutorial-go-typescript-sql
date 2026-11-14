---
id: sql.querying.patterns
title: "LIKE/ILIKE/regex, CASE, string and date functions"
track: sql
module: querying
order: 3
level: basic
minutes: 12
since: pg9.0
prerequisites: [sql.querying.null-logic]
compare_with: [go.basics.strings-runes]
tags: [patterns, case, to_char, ilike]
summary: >
  Compute in the database instead of in the app: case-insensitive matching
  with ILIKE, conditional labels with CASE, and formatted output with to_char.
examples:
  - id: patterns
    file: examples/01_patterns.sql
    title: "ILIKE, CASE labels and to_char formatting"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-15"
        note: >
          ILIKE lowercases both sides with the database's case rules. StackLab's
          databases use the builtin C.UTF-8 locale, so Ş/ş and İ/i fold, but
          Turkish I/ı never pair up: 'ISPARTA' ILIKE 'ısparta' is false.
          Appending COLLATE "tr-x-icu" switches to Turkish rules.
      - lines: "18-21"
        note: >
          A searched CASE without ELSE yields NULL when no branch matches, which
          is why Quiet day gets a NULL label.
      - lines: "26"
        note: >
          Dy and Mon always print English abbreviations. The TM prefix (TMDy,
          TMMon) translates them using the server's lc_time setting instead.
use_cases:
  - "Search boxes: ILIKE '%term%' over names and titles"
  - "Reports: CASE labels, to_char formatted dates, grouped totals"
strengths:
  - "Shaping happens once, in the database, close to the data and its indexes"
  - "CASE works everywhere an expression does: SELECT, WHERE, ORDER BY, aggregates"
  - "to_char formats dates in the query itself; with the TM prefix, month and day names follow the server's lc_time"
weaknesses:
  - "Leading-wildcard ILIKE cannot use a btree index; large tables need pg_trgm or full-text search"
  - "ILIKE follows the database's case rules: under the default (non-Turkish) mapping, I and ı never match; Turkish rules need COLLATE \"tr-x-icu\""
  - "Heavy string formatting in SQL moves presentation logic away from the app"
pitfalls:
  - title: "CASE without ELSE returning NULL"
    detail: "Rows matching no branch come back NULL, which then fails comparisons and concatenation downstream. Add an ELSE when 'other' means something."
  - title: "Trusting to_char for machine-readable output"
    detail: "to_char is presentation and depends on session settings; keep timestamps raw in the wire format and format in the client when the app must compute on them."
quiz:
  - q: "Which operator matches case-insensitively?"
    options: ["LIKE", "ILIKE", "SIMILAR TO", "~"]
    answer: 1
    why: "ILIKE is PostgreSQL's case-insensitive LIKE; standard SQL has no equivalent."
  - q: "What does a CASE expression without ELSE return when no branch matches?"
    options: ["0", "Empty string", "NULL", "Error"]
    answer: 2
    why: "A missed CASE yields NULL, which then interacts with three-valued logic downstream."
further_reading:
  - title: "PostgreSQL pattern matching documentation"
    url: "https://www.postgresql.org/docs/current/functions-matching.html"
  - title: "PostgreSQL conditional expressions (CASE)"
    url: "https://www.postgresql.org/docs/current/functions-conditional.html"
---

## Concept

Every row you ship to the app is a row the app must re-shape. Postgres can do
the shaping: `ILIKE` for case-insensitive search, `CASE` for conditional labels,
`to_char` for formatted dates, and the string functions (`upper`, `left`,
`split_part`) for light text munging. The result reads like a tiny report
written in SQL.

`CASE` comes in two forms — searched (`CASE WHEN cond THEN ...`) and simple
(`CASE x WHEN v THEN ...`) — and both return NULL when nothing matches and no
ELSE is given, which connects back to three-valued logic.

## Coming from C/C++/Java/Python/JS

If you reach for the application layer for formatting, this is the lesson that
moves that code. Python: think of `CASE` as a per-row conditional expression
(`x if cond else y`); `to_char` is `strftime` with Postgres's own format
specifiers — note they differ from strftime's codes. Java: `String.format`
lives in the database now, and the format survives whichever client queries
next. JavaScript: this is the work you'd otherwise do in a `.map()` after
fetching — doing it in SQL means less data over the wire and one less place
for the formatting to drift.

## Design notes

Postgres deliberately blurs "expression" and "report": CASE is an expression, so
it sorts, filters and aggregates like any other value; to_char exists so that a
report can come out of the database already formatted (with TM-prefixed patterns,
in the server's lc_time language). The split to keep in mind
is data versus rendering — keep the wire format raw (timestamptz, plain text)
and reach for these functions for evidence-shaped queries and human output, not
for storage formats.
