---
id: go.basics.types-conversion
title: "Basic types and explicit conversion"
track: go
module: basics
order: 2
level: basic
minutes: 12
since: go1.0
prerequisites: [go.basics.variables]
compare_with: [ts.everyday-types.primitives, sql.tables-types.text-numbers]
tags: [types, conversion, integers, strconv, named-types]
summary: >
  Go never converts between numeric types for you, not even int to int64. Every
  conversion is visible, and named types like Celsius carry meaning the
  compiler enforces.
examples:
  - id: explicit
    file: examples/01_explicit.go
    title: Mixing numeric types needs a conversion, and conversions have rules
    runtime: go
    check: exact
    annotations:
      - lines: "6-9"
        note: >
          int and int64 are distinct types even where int is 64 bits wide. The
          commented line is a compile error; the conversion makes the widening
          visible at the exact spot it happens.
      - lines: "11-13"
        note: >
          The same for float64 and int. Untyped constants are the exception:
          price * 3 compiles, because the literal 3 takes on float64.
      - lines: "15-16"
        note: >
          Float to int truncates toward zero: -2.9 becomes -2, not -3. Use
          math.Floor or math.Round first when you mean those.
      - lines: "18-19"
        note: >
          Narrowing a variable keeps the low bits: 200 fits uint8 but reads as
          -56 in int8. With a constant, int8(200) is a compile error instead.
      - lines: "21-23"
        note: >
          Unsigned arithmetic wraps modulo 2^32 here. Nothing panics on integer
          overflow in Go, signed or unsigned.
      - lines: "25-26"
        note: >
          Integer division truncates toward zero and % takes the sign of the
          dividend: -3 and -1. Python floors instead and gives -4 and 1.
  - id: named-types
    file: examples/02_named_types.go
    title: Named types, time.Duration and strconv
    runtime: go
    check: exact
    annotations:
      - lines: "9-12"
        note: >
          Celsius and Fahrenheit share float64 underneath, so converting between
          them is allowed, but only when written out. Assigning one to the
          other is the bug the type exists to catch.
      - lines: "20-23"
        note: >
          time.Duration is a named int64 counting nanoseconds. An int variable
          must be converted before multiplying, and a bare 2 is two nanoseconds.
      - lines: "25-26"
        note: >
          strconv.Itoa formats digits; string(rune(65)) is the code point "A".
          Writing string(n) for an int n is the classic mix-up, and go vet
          flags it.
      - lines: "27-28"
        note: >
          Parsing returns the zero value plus an error that names the function
          and the input: no exception, no NaN.
      - lines: "30-32"
        note: >
          strconv follows Go syntax, not a locale. A Turkish decimal comma
          ("3,14") is a syntax error, so normalize user input at the boundary.
use_cases:
  - "Converting between int sizes at API boundaries (database int64 ids, len() int results)"
  - "Named types for units and identifiers: type UserID int64, type Celsius float64"
  - "time.Duration arithmetic for timeouts and backoff"
  - "strconv for parsing and formatting numbers from flags, query strings and CSV"
strengths:
  - "No implicit conversions means no silent truncation or sign changes hidden in arithmetic, unlike C's usual arithmetic conversions"
  - "Named types give unit and ID safety for free: passing a UserID where an OrderID is expected doesn't compile, unlike TypeScript's structural number aliases"
  - "Parsing errors are values that say what failed and on which input"
weaknesses:
  - "Conversions add noise in numeric code, e.g. float64(len(xs)) in every average"
  - "Integer overflow wraps silently at run time; there is no checked arithmetic as in Rust's debug builds"
  - "Named types over the same base convert freely when written out, so a careless Fahrenheit(c) still compiles"
pitfalls:
  - title: "string(n) on an integer"
    detail: "It produces the character with that code point, not decimal digits. Use strconv.Itoa or fmt.Sprint."
    bad: "s := string(65) // \"A\""
    good: "s := strconv.Itoa(65) // \"65\""
  - title: "Expecting float-to-int to round"
    detail: "int(f) truncates toward zero. Round explicitly with math.Round when you need it."
  - title: "Narrowing without a range check"
    detail: "int32(x) on an out-of-range int64 keeps the low bits and changes the value. Check the range first when the input isn't under your control."
quiz:
  - q: "var a int = 1; var b int64 = 2. Does a + b compile?"
    options: ["Yes, int is 64 bits on amd64", "No: mismatched types int and int64", "Only with go vet disabled"]
    answer: 1
    why: "Distinct types never mix in arithmetic, whatever their size on the platform."
  - q: "What does fmt.Println(int(-2.9)) print?"
    options: ["-3", "-2", "-2.9"]
    answer: 1
    why: "The fraction is discarded: truncation toward zero."
  - q: "What is time.Duration(5)?"
    options: ["5 seconds", "5 milliseconds", "5 nanoseconds"]
    answer: 2
    why: "Duration counts nanoseconds; write 5 * time.Second for seconds."
further_reading:
  - title: "The Go Programming Language Specification: Conversions"
    url: "https://go.dev/ref/spec#Conversions"
  - title: "The Go Programming Language Specification: Arithmetic operators"
    url: "https://go.dev/ref/spec#Arithmetic_operators"
  - title: "package strconv"
    url: "https://pkg.go.dev/strconv"
---

## Concept

Go's basic types are `bool`, `string`, sized integers (`int8`…`int64`, `uint8`…`uint64`,
plus `int`/`uint` at the platform's word size), `float32`/`float64`, and aliases `byte`
(`uint8`) and `rune` (`int32`). Values of different types never mix: you write the
conversion `T(x)` yourself. A *named type* such as `type Celsius float64` is a new type
with the same underlying representation; it converts to and from `float64` explicitly,
never implicitly.

## Coming from C/C++/Java/Python/JS

- C/C++: no integer promotion and no usual arithmetic conversions. `int + int64` is a
  compile error, so a sign or width change always shows up as a conversion in the code.
- Java widens `int` to `long` automatically; Go doesn't widen at all. There is no
  boxing either: no `Integer` objects.
- Python: Go integers have a fixed size and wrap on overflow, and integer `/` and `%`
  truncate toward zero where Python's `//` and `%` floor.
- JS/TS: no single `number` type. TS `type UserId = number` is only an alias, while Go's
  `type UserID int64` is a distinct type the compiler checks.

## Design notes

Convert at the boundary and keep one type inside: parse strings with `strconv` where
input enters, convert database `int64` ids once in the store layer, and let named types
(`UserID`, `Celsius`, `time.Duration`) carry meaning through the rest of the code. When
a conversion narrows a value you don't control, check the range first: Go wraps silently
on overflow. For user-facing numbers in Turkish (decimal comma, dot as thousands
separator), normalize the text before `strconv.ParseFloat`, which only accepts Go syntax.
