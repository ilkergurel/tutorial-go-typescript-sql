---
id: sql.tables-types.text-numbers
title: "text vs varchar(n), integer vs bigint, numeric for money (never float)"
track: sql
module: tables-types
order: 1
level: basic
minutes: 12
since: pg12
prerequisites: [sql.basics.mental-model]
compare_with: [go.basics.strings-runes, go.basics.types-conversion, ts.everyday-types.primitives]
tags: [types, text, varchar, integer, bigint, numeric, money, utf8]
summary: >
  text and varchar(n) perform the same in PostgreSQL, and lengths count characters,
  not bytes. Use bigint wherever integer's 2.1 billion could run out, and numeric (or
  integer minor units) for money, because a float can't even hold 0.10 exactly.
examples:
  - id: text
    file: examples/01_text.sql
    title: "Characters vs bytes, and what varchar(n) actually enforces"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          Same storage and speed: varchar(n) is text plus a length check on write.
          Use it when the limit is a real rule, not as a guess; a guessed limit
          you later raise is a migration (cheap, see the ALTER lesson), a guessed
          limit you hit in production is an outage.
      - lines: "13-15"
        note: >
          'şükrü' is 5 characters and 8 bytes (ş and ü take 2 bytes each in
          UTF-8), and it fits varchar(6). length() agrees with Python's len() and
          Go's utf8.RuneCountInString; Go's len() and C's strlen() report what
          octet_length() does.
      - lines: "19-20"
        note: >
          The database's builtin C.UTF-8 collation upper-cases ipekç to IPEKÇ:
          the dotless capital, wrong in Turkish. COLLATE "tr-x-icu" applies
          Turkish rules and gives İPEKÇ. The same choice decides how ILIKE and
          lower() treat I and ı.
      - lines: "25"
        note: >
          A cast is an instruction to fit the value, so it truncates to 6
          characters without a word. Data that passes through such a cast loses
          its tail silently.
      - lines: "28"
        note: >
          Storing is different: an over-long value fails with SQLSTATE 22001 and,
          as in any transaction, nothing after it runs. Better an error than a
          silently shortened name.
  - id: integers
    file: examples/02_integers.sql
    title: "integer vs bigint: division, widening, and overflow"
    runtime: sql
    check: exact
    annotations:
      - lines: "7"
        note: >
          Two integers divide as integers and the fraction is dropped. One numeric
          operand (2.0) makes the whole expression numeric, which picks a generous
          scale for the result.
      - lines: "16"
        note: >
          sum(integer) returns bigint and sum(bigint) returns numeric, so totals
          don't overflow. avg() of integers returns numeric, which is why avg(x)
          beats sum(x) / count(*).
      - lines: "20"
        note: >
          SQLSTATE 22003, integer out of range. C would hit undefined behaviour,
          Java and Go would wrap to -2147483648 and keep going. Postgres stops, and
          the real fix is the column type: bigint costs 4 more bytes per row,
          widening a big table later costs a full rewrite.
  - id: money
    file: examples/03_money.sql
    title: "numeric vs double precision for money"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-5"
        note: >
          In Python and JS, 0.1 is a binary float (in Go, once it lands in a
          float64). In Postgres a literal with a decimal point is numeric, so 0.1 + 0.2 = 0.3 holds until you cast to
          float8. Since PostgreSQL 12 floats print their shortest exact form
          (0.30000000000000004); older versions rounded to 15 digits and printed
          0.3, which hid the error.
      - lines: "8"
        note: >
          numeric(12, 2) means 12 significant digits, 2 of them after the point:
          amounts up to 9,999,999,999.99. Choose the scale per field: 2 for TL
          prices, more for exchange rates or unit prices.
      - lines: "15-18"
        note: >
          Ten float payments of 0.10 add up to 0.9999999999999999, and an equality
          check against 1 fails. numeric adds them in decimal and gets 1.00, the
          same answer an accountant would.
      - lines: "22-24"
        note: >
          numeric rounds ties away from zero (19.995 becomes 20.00, 2.5 becomes 3);
          float8 rounds 2.5 to 2 (ties to even). If a business rule says how to
          round, apply it explicitly before storing instead of leaving it to the
          column's scale.
      - lines: "27"
        note: >
          Eleven digits before the point don't fit numeric(12, 2): SQLSTATE 22003,
          numeric field overflow. Decimals round; whole digits never get dropped.
use_cases:
  - "text: names, emails, addresses, descriptions: anything without a business-defined maximum"
  - "varchar(n), or text with a CHECK: codes whose length is a real rule (ISO country codes, an IBAN's 34 characters)"
  - "bigint: primary keys, counters, byte sizes, anything that grows for years or gets summed"
  - "numeric(p, s): prices, balances, tax and exchange rates; integer minor units (kuruş in a bigint) when you never need fractions of the smallest unit"
  - "double precision: measurements and scientific values where a tiny relative error is acceptable"
strengths:
  - "text has no penalty against varchar(n), so you don't guess maximum lengths up front; the docs call char(n) usually the slowest of the three"
  - "Length limits count characters, so varchar(6) means the same for 'ada' and 'şükrü', unlike a C char[6] or Go's len()"
  - "Integer overflow is an error (SQLSTATE 22003), not the silent wraparound of Java and Go or C's undefined behaviour"
  - "numeric is exact decimal arithmetic at the column level, like Java's BigDecimal; JS numbers and Go's float64 have no exact decimal type"
weaknesses:
  - "numeric arithmetic is much slower than integer or float arithmetic; aggregating millions of rows notices it"
  - "integer columns run out at 2,147,483,647, and widening one later rewrites the whole table under an exclusive lock"
  - "Character limits are not byte limits: varchar(255) can hold up to 4 bytes per character, which matters for client buffers and index size limits"
  - "Integer division truncates silently, so sum(x) / count(*) quietly loses the fraction"
pitfalls:
  - title: "Money in double precision"
    detail: "Sums drift (ten times 0.10 is 0.9999999999999999) and equality checks fail. The docs' Monetary Types page says it directly: floats should not be used to handle money."
    bad: "price double precision NOT NULL"
    good: "price numeric(12, 2) NOT NULL  -- or price_kurus bigint NOT NULL"
  - title: "Counting on varchar(n) to reject long values"
    detail: "Plain inserts fail with 22001, but an explicit cast (value::varchar(50)) truncates without an error. If the limit protects something, check it in a CHECK constraint, not through a cast."
  - title: "integer keys and counters"
    detail: "serial and integer ids stop at 2,147,483,647. Moving to bigint later is ALTER COLUMN ... TYPE bigint, a full rewrite of table and indexes. Start with bigint."
  - title: "Averages by integer division"
    detail: "Both operands integer means integer division."
    bad: "SELECT sum(score) / count(*) FROM results"
    good: "SELECT avg(score) FROM results  -- or sum(score)::numeric / count(*)"
  - title: "Using length() for a byte budget"
    detail: "length() counts characters. For anything measured in bytes (a protocol field, a fixed-size buffer in a C client) use octet_length(): Turkish letters take 2 bytes, emoji 4."
  - title: "The money type"
    detail: "Its output is formatted by the database's lc_monetary setting, and the docs warn that dumps may not load into a database with a different setting. numeric plus a currency column is portable and explicit."
quiz:
  - q: "Which column type keeps 0.10 + 0.20 exactly equal to 0.30?"
    options: ["real", "double precision", "numeric(12, 2)", "Any of them: Postgres rounds for you"]
    answer: 2
    why: "Only numeric stores decimal digits exactly; real and double precision are binary floats and can't represent 0.1."
  - q: "handle is varchar(6). What happens with INSERT ... VALUES ('şükrüm'), 6 characters and 9 bytes?"
    options: ["It is stored", "Error: value too long", "It is truncated to 6 bytes"]
    answer: 0
    why: "varchar(n) limits characters, not bytes, so 6 characters fit regardless of their UTF-8 length."
  - q: "hits is an integer column holding 2147483647. What does UPDATE ... SET hits = hits + 1 do?"
    options: ["Wraps to -2147483648", "Promotes the column to bigint", "Fails with integer out of range"]
    answer: 2
    why: "Postgres checks integer arithmetic and raises SQLSTATE 22003; it neither wraps nor changes the column type."
  - q: "What does SELECT -7 / 2 return?"
    options: ["-3.5", "-4", "-3"]
    answer: 2
    why: "Two integer operands give integer division, truncated toward zero as in Go, C and Java (Python's // would give -4)."
further_reading:
  - title: "PostgreSQL docs: Character Types"
    url: "https://www.postgresql.org/docs/current/datatype-character.html"
  - title: "PostgreSQL docs: Numeric Types"
    url: "https://www.postgresql.org/docs/current/datatype-numeric.html"
  - title: "PostgreSQL docs: Monetary Types"
    url: "https://www.postgresql.org/docs/current/datatype-money.html"
  - title: "PostgreSQL docs: Collation Support"
    url: "https://www.postgresql.org/docs/current/collation.html"
---

## Concept

Three type families cover most columns, and each wrong choice fails in its own way:

- **Strings.** `text`, `varchar(n)` and `char(n)` share one implementation. `varchar(n)`
  is `text` plus a length check, counted in characters. `char(n)` pads with spaces and,
  per the docs, is usually the slowest of the three. Too long is an error on insert, but
  a silent truncation in an explicit cast.
- **Whole numbers.** `smallint` (2 bytes), `integer` (4 bytes, ±2.1 billion) and `bigint`
  (8 bytes, ±9.2 × 10¹⁸). Arithmetic that leaves the range is an error, never a wrap.
- **Fractions.** `numeric(p, s)` is exact decimal arithmetic; `real` and `double
  precision` are IEEE 754 binary floats, fast but unable to store 0.1 exactly.

The goal of this lesson is a reflex: `text` unless a length is a rule, `bigint` unless
the range is provably small, `numeric` whenever someone will add the values up and
expect the books to balance.

## Coming from C/C++/Java/Python/JS

- **C/C++:** `char name[6]` is 6 bytes; `varchar(6)` is 6 characters, and there is no
  buffer to overflow: too long is an error. Signed overflow, undefined behaviour in C,
  is a clean SQLSTATE 22003 here.
- **Go:** `len(s)` counts bytes, like `octet_length()`; `utf8.RuneCountInString` counts
  what `length()` counts. Postgres `integer` is always 32 bits, so Go's `int` (64-bit on
  your machine) pairs with `bigint`, not `integer`.
- **Java:** `int`/`long`/`BigDecimal` map to `integer`/`bigint`/`numeric`. Java wraps on
  overflow; Postgres refuses.
- **Python:** ints never overflow, so Python code is the most likely to meet `integer out
  of range` for the first time in the database. `/` on two integers here behaves like
  C's, not like Python 3's.
- **JS/TS:** `number` is a float64. A `bigint` above 2⁵³ − 1 or a `numeric` amount parsed
  into a `number` loses precision, which is why drivers commonly hand them to JS as
  strings. Keep them strings (or `BigInt`) until you format them.

## Design notes

Lengths: default to `text`. When a length is a business rule, say so with a constraint,
`CHECK (char_length(code) = 2)`, or with `varchar(n)` if you prefer; raising a `varchar`
limit later is a catalog-only change, as the ALTER TABLE lesson shows. What to avoid is the
reflexive `varchar(255)` copied from other databases: it documents nothing and rejects the
one long name you didn't anticipate.

Integers: `bigint` for keys, counters and sizes. The 4 bytes you save with `integer` are
rarely worth the day you hit 2,147,483,647, because widening a column rewrites the table.

Money: `numeric(12, 2)` (scale per currency and field) or an integer number of minor units
(`price_kurus bigint`), always with the currency stored next to it when there could ever be
more than one. Never `double precision`, and not the `money` type, whose formatting follows
the server's `lc_monetary` setting. Rounding is a business rule: numeric rounds ties away
from zero, banks sometimes round to even, so make the rounding step explicit where a rule
applies instead of leaving it to the column's scale.

Turkish text: lengths are safe (characters, not bytes), but case mapping depends on the
collation. The database default here maps `i` to `I`; use `COLLATE "tr-x-icu"` where Turkish
casing matters, as the search lessons do for ILIKE and full-text search.
