---
id: go.basics.strings-runes
title: "Strings, bytes and runes — UTF-8 all the way down"
track: go
module: basics
order: 3
level: basic
minutes: 15
since: go1.0
prerequisites: []
compare_with: [sql.querying.patterns]
tags: [strings, unicode, utf8, range]
summary: >
  A Go string is a read-only byte slice holding UTF-8. len counts bytes, range
  walks runes, and case conversion is locale-blind — which matters if your
  users write Turkish.
examples:
  - id: len-range
    file: examples/01_len_range.go
    title: "len counts bytes; range counts runes"
    runtime: go
    check: exact
    annotations:
      - lines: "11"
        note: >
          len(s) is the byte length: 4 for "DİL", because İ encodes as two
          bytes. utf8.RuneCountInString counts the 3 characters.
      - lines: "13-16"
        note: >
          range decodes UTF-8 and yields each rune with the byte offset where it
          starts: 0, 1, 3. The offsets skip 2, the second byte of İ.
      - lines: "18-22"
        note: >
          Indexing returns a single byte (0xC4, half of İ). Decode from that
          offset to get the whole character back.
  - id: turkish-case
    file: examples/02_turkish_case.go
    title: "Case conversion ignores locale until you ask for Turkish"
    runtime: go
    check: exact
    annotations:
      - lines: "9-13"
        note: >
          strings.ToLower maps I to i whatever the user's language. The stdlib
          has the Turkish and Azeri mapping as unicode.TurkishCase: I becomes
          ı and İ becomes i (strings.ToUpperSpecial goes the other way).
      - lines: "21-24"
        note: >
          Naive lowercasing turns both DIL and DİL into "dil", so a search for
          one finds the other. strings.EqualFold has no Turkish mode either:
          EqualFold("DİL", "dil") is false.
use_cases:
  - "Processing user input that is not pure ASCII: names, cities, addresses"
  - "Deciding between byte offsets (wire formats, file slices) and rune counts (UI limits)"
  - "Understanding why len('İ') == 2 and range treats it as one rune"
strengths:
  - "One encoding everywhere: source and string literals are UTF-8 and the standard library assumes it; no separate wide-char type"
  - "range over a string just works for iteration; no manual decoder loops"
  - "Explicit byte/rune split makes costs visible instead of hidden in a locale setting"
weaknesses:
  - "len and slicing are byte-based; s[:1] can cut a rune in half and produce invalid UTF-8"
  - "Locale rules are opt-in: Turkish needs strings.ToLowerSpecial(unicode.TurkishCase, s), other languages golang.org/x/text/cases, and EqualFold has no Turkish mode"
  - "Rune is an alias for int32, not a distinct type — the name suggests more safety than it delivers"
pitfalls:
  - title: "Indexing a string gives a byte, not a character"
    detail: "s[1] on \"DİL\" returns the first byte of İ (0xC4), which prints as garbage. Use []rune(s) or utf8.DecodeRuneInString when you need a whole character."
  - title: "Counting characters with len"
    detail: "len(s) is bytes. For UI length limits use utf8.RuneCountInString(s) — 'DİL' is len 4 but 3 runes."
quiz:
  - q: "What is len(\"DİL\")?"
    options: ["3", "4", "5", "Depends on locale"]
    answer: 1
    why: "İ encodes as two UTF-8 bytes, so the byte length is 4 even though there are 3 runes."
  - q: "What does range yield for each iteration over a string?"
    options: ["Byte index and byte", "Byte index and rune", "Rune index and rune", "Rune only"]
    answer: 1
    why: "range decodes UTF-8 and yields the starting byte index of each rune plus the rune itself — which is why indices skip over continuation bytes."
  - q: "On a machine with a Turkish OS locale, what does strings.ToLower(\"ISTANBUL\") return?"
    options: ["istanbul", "ıstanbul", "It depends on the locale"]
    answer: 0
    why: "ToLower never consults the OS locale, so you always get 'istanbul'. Turkish rules need strings.ToLowerSpecial(unicode.TurkishCase, s)."
further_reading:
  - title: "Go blog: Strings, bytes, runes and characters in Go"
    url: "https://go.dev/blog/strings"
  - title: "unicode/utf8 package documentation"
    url: "https://pkg.go.dev/unicode/utf8"
---

## Concept

A Go string is a read-only slice of bytes, and those bytes are UTF-8. That
single decision explains most of this lesson: `len` counts bytes, indexing
gives bytes, and the `range` clause is the one construct that decodes runes
for you. A rune is a Unicode code point — `int32` under the alias — and the
conversion cost between "what the user sees" (runes) and "what the machine
stores" (bytes) is always visible in Go.

The second half is about case: Go's `strings` functions do simple, locale-blind
mapping. That is predictable and fast, but wrong for Turkish, where uppercase
`I` lowercases to dotless `ı` and dotted `İ` lowercases to plain `i`. The
standard library has that mapping as `unicode.TurkishCase`, used through
`strings.ToLowerSpecial` and `strings.ToUpperSpecial`.

## Coming from C/C++/Java/Python/JS

C gives you `char*` and no encoding opinion — you already know the pain Go
avoids. Java and C# hide UTF-16 behind a `char` type, so you get a *different*
surprise (surrogate pairs) instead. Python 3 and JavaScript strings are
code-point-ish abstractions where indexing usually does what you mean; Go is
closer to the metal: bytes in memory, explicit decode on demand. If you have
fought `wchar_t` or ICU locale structs, Go's "one encoding, no guesses" stance
is a relief: Turkish casing is one stdlib call, and `golang.org/x/text` covers
other languages' rules when you need them.

## Design notes

The stdlib refuses to guess locales, so `strings.ToLower` never consults the
OS locale — the same binary behaves identically in Istanbul and Oslo. The
trade-off: correctness for Turkish users is *your* job, via
`strings.ToLowerSpecial(unicode.TurkishCase, s)` wherever text is normalized
for comparison or search. This matches Go's general philosophy of explicit
dependencies over ambient environment.
