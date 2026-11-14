---
id: go.files-io.csv-json
title: "CSV and JSON: encoding/csv, encoding/json (and json/v2, Go 1.27)"
track: go
module: files-io
order: 6
level: intermediate
minutes: 15
since: go1.27
prerequisites: [go.composite.structs, go.files-io.composing]
compare_with: [sql.files-movement.csv-staging, ts.services.zod, go.http.json-validation, sql.json.jsonb-operators]
tags: [csv, json, json-v2, encoding, excel, utf-8, streaming]
summary: >
  encoding/csv reads and writes RFC 4180 records; Turkish Excel adds ';', a BOM
  or windows-1254. encoding/json streams with Decoder and defers decoding with
  RawMessage. Go 1.27 ships encoding/json/v2, stricter by default.
examples:
  - id: csv-excel
    file: examples/01_csv_excel.go
    title: "CSV from and for Turkish Excel: ';', BOM, decimal comma, windows-1254"
    runtime: go
    check: exact
    annotations:
      - lines: "16-20"
        note: >
          With Turkish regional settings ',' is the decimal separator, so Excel
          separates CSV fields with ';'. Saving as "CSV UTF-8" adds a byte
          order mark; plain "CSV (Comma delimited)" ("Virgülle ayrılmış") is
          written in windows-1254 instead.
      - lines: "23-28"
        note: >
          Comma is the field separator: any rune except CR, LF and U+FFFD.
          ReadAll loads every record; for large files call Read in a loop,
          which holds one record at a time. Quoted fields, doubled quotes and
          CRLF are handled for you.
      - lines: "29"
        note: >
          encoding/csv doesn't strip the BOM, so the first header is
          U+FEFF followed by Müşteri and a lookup by "Müşteri" fails. Strip it
          from the first field, or peek at the first three bytes with
          bufio.Reader before parsing.
      - lines: "33-34"
        note: >
          1.250,50 is Turkish notation for 1250.50, and strconv only knows the
          English form. Remove the thousands dots, swap the comma, and parse
          money into integer kuruş rather than float64.
      - lines: "37-39"
        note: >
          In windows-1254, 0xDD is İ and 0xDE is Ş; followed by ASCII letters,
          neither byte forms valid UTF-8, so Go shows replacement characters. Decode such files with
          golang.org/x/text/encoding/charmap, which is not in the standard
          library.
      - lines: "42-45"
        note: >
          For Excel: write the BOM yourself, then set Comma and UseCRLF.
          Without the BOM, a double-clicked UTF-8 file opens in the ANSI code
          page and ş turns into two odd characters (ÅŸ in windows-1252).
      - lines: "46-49"
        note: >
          The writer quotes the field that contains ';' and doubles the quotes
          inside it. csv.Writer is buffered: WriteAll flushes and returns any
          error; if you call Write yourself, finish with Flush, which returns
          nothing, and then check w.Error().
  - id: json-stream
    file: examples/02_json_stream.go
    title: Streaming a large array with Decoder, RawMessage for mixed payloads
    runtime: go
    check: exact
    annotations:
      - lines: "12-15"
        note: >
          json.RawMessage keeps the bytes of data undecoded. The envelope is
          decoded once, and the switch decides which struct the payload
          becomes; unknown types keep their raw JSON for logging or
          forwarding.
      - lines: "19"
        note: >
          JSON numbers are decoded into whatever type the field has. Keep money
          in integer kuruş (or a decimal string): float64 can't represent
          values like 0.10 exactly.
      - lines: "36-39"
        note: >
          Token reads one syntactic piece, here the opening bracket, and More
          reports whether another element follows. Decode then reads one
          element. Memory is bounded by the largest element, not the whole
          array.
      - lines: "44-59"
        note: >
          Each payload is decoded only after its type is known. With
          interface-based decoding (map[string]any) you'd get float64 numbers
          and type assertions everywhere instead.
      - lines: "61-63"
        note: >
          Reading the closing bracket proves the array really ended. On a
          truncated upload either Decode or this Token fails (which one
          depends on the implementation), so a cut-off file is never accepted
          as a shorter one.
  - id: json-v2
    file: examples/03_json_v2.go
    title: encoding/json vs encoding/json/v2 on the same input
    runtime: go
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          Both packages are named json, so one import needs an alias. In Go
          1.27, encoding/json is itself implemented on top of v2 with options
          that keep the old behavior.
      - lines: "14"
        note: >
          v2 redefines omitempty as "omit if it encodes as an empty JSON value"
          (null, "", {}, []), so 0 is kept. Use omitzero, which means the same
          in v1 and v2, to drop zero numbers and booleans.
      - lines: "17-32"
        note: >
          Classify errors with errors.Is and errors.As. The v2 docs say the
          text of its errors may change over time, and v1 messages may differ
          since 1.27 moved v1 onto the v2 implementation.
      - lines: "35-38"
        note: >
          A nil slice: v1 writes null, v2 writes []. Frontends that call
          .map() on the field stop crashing; consumers that check for null
          must be updated.
      - lines: "40-50"
        note: >
          On the same bytes v1 keeps the last duplicate, replaces the invalid
          byte with U+FFFD and ignores the unknown member, all silently. v2
          rejects duplicates and invalid UTF-8 by default; unknown members
          take RejectUnknownMembers (v1: Decoder.DisallowUnknownFields).
      - lines: "52-54"
        note: >
          v1 always sorted map keys; v2 writes them in unspecified order unless
          you pass Deterministic(true). Snapshot tests and signatures over JSON
          need that option.
use_cases:
  - "Importing bank statements, e-invoice exports and price lists that users export from Excel"
  - "Generating CSV reports that open correctly in Turkish Excel with a double click"
  - "Streaming large JSON exports or NDJSON logs without loading them into memory"
  - "Webhook or event envelopes whose payload shape depends on a type field (RawMessage, jsontext.Value)"
strengths:
  - "Decoding goes straight into typed structs with the standard library alone, where Python's json.load gives you dicts and Java needs Jackson or Gson as a dependency"
  - "Decoder and csv.Reader read from any io.Reader, so a 2 GB upload streams with flat memory"
  - "v2 rejects duplicate names and invalid UTF-8 by default, the inputs where two parsers can disagree about what a document means"
  - "v1 and v2 share one implementation in 1.27, so both can live in one program and a migration can go type by type"
weaknesses:
  - "The standard library has no charset conversion: windows-1254 needs golang.org/x/text"
  - "encoding/csv returns [][]string; types, headers and validation are yours to write"
  - "Two JSON packages with different defaults: moving a struct from v1 to v2 changes nil slices, omitempty, map order and name matching"
  - "Reflection-based: a misspelled tag compiles and silently leaves a field empty"
pitfalls:
  - title: "The BOM in the first header"
    detail: "Excel's \"CSV UTF-8\" starts with EF BB BF, which becomes part of the first field. Strip it before mapping headers to columns."
    bad: "idx := slices.Index(header, \"Müşteri\") // -1: the field is BOM + \"Müşteri\""
    good: "header[0] = strings.TrimPrefix(header[0], \"\\xEF\\xBB\\xBF\")"
  - title: "Reading windows-1254 as UTF-8"
    detail: "Every İ, Ş, Ğ, ı, ş, ğ becomes U+FFFD (v1) or an error (v2, PostgreSQL). Check utf8.Valid on a sample and decode the stream instead."
    good: "r := csv.NewReader(charmap.Windows1254.NewDecoder().Reader(f)) // golang.org/x/text/encoding/charmap"
  - title: "Parsing 1.250,50 with strconv"
    detail: "strconv.ParseFloat fails on Turkish number formatting, and float64 is the wrong type for money anyway. Normalize the string and parse integer kuruş."
  - title: "time.Duration under v2"
    detail: "v1 encodes a Duration as nanoseconds; v2 has no default representation and returns an error. Pass jsonv1.FormatDurationAsNano(true) (the option lives in encoding/json), or wrap the field in a type with its own MarshalJSON and UnmarshalJSON."
  - title: "Expecting v2 to sort maps"
    detail: "Without json.Deterministic(true), the same map can encode to different bytes on each run."
quiz:
  - q: "A Turkish user uploads an Excel \"CSV UTF-8\" file. record[0][0] of the header doesn't equal \"Müşteri\" although it prints like it. Why?"
    options: ["Excel uses NFD normalization", "The field starts with a byte order mark", "csv.Reader lowercases headers"]
    answer: 1
    why: "The BOM (U+FEFF) is invisible when printed, but encoding/csv keeps it as part of the first field."
  - q: "{\"name\":\"a\",\"name\":\"b\"} is unmarshaled into a struct. What do v1 and v2 do by default?"
    options: ["Both keep \"b\"", "v1 keeps \"b\" silently, v2 returns an error", "Both return an error"]
    answer: 1
    why: "v2 rejects duplicate names unless you pass jsontext.AllowDuplicateNames(true)."
  - q: "Why read the closing ] with dec.Token() after the dec.More() loop?"
    options: ["To free the buffer", "To detect a truncated or malformed array", "Decode fails otherwise"]
    answer: 1
    why: "The loop can end without seeing ]: with the pre-1.27 implementation, More just returns false on a broken stream. The final Token is the explicit check."
  - q: "Is encoding/json/v2 usable in Go 1.27 without setting GOEXPERIMENT?"
    options: ["Yes, it ships enabled; GOEXPERIMENT=nojsonv2 opts out", "No, it still needs GOEXPERIMENT=jsonv2", "Only through golang.org/x"]
    answer: 0
    why: "It was experimental in Go 1.25 and 1.26 and became available by default in 1.27."
further_reading:
  - title: "package encoding/csv"
    url: "https://pkg.go.dev/encoding/csv"
  - title: "package encoding/json (see \"Migrating to v2\")"
    url: "https://pkg.go.dev/encoding/json"
  - title: "package encoding/json/v2"
    url: "https://pkg.go.dev/encoding/json/v2"
  - title: "Go 1.27 release notes: encoding/json/v2"
    url: "https://go.dev/doc/go1.27"
---

## Concept

**CSV.** `csv.NewReader(r)` parses RFC 4180 records from any `io.Reader`: quoted fields,
doubled quotes, embedded newlines, and CRLF turned into LF. Set `Comma` for `;` or tab files,
`FieldsPerRecord` to enforce a column count, `LazyQuotes` for sloppy producers.
`csv.NewWriter(w)` is buffered: `Flush` (or `WriteAll`), then check `Error`. It works on bytes
and knows nothing about character sets.

**JSON (v1).** `json.Marshal`/`Unmarshal` map structs through field tags. For streams, use
`json.NewDecoder(r)`: `Decode` reads one value at a time (NDJSON), and `Token` plus `More`
walk into a large array element by element. `json.RawMessage` holds a value's bytes to
decode later. `Decoder.DisallowUnknownFields` makes decoding strict.

**JSON v2 (Go 1.27).** `encoding/json/v2` (`Marshal`, `Unmarshal`, `MarshalWrite`,
`UnmarshalRead`, all taking options) and `encoding/json/jsontext` (tokens, `Value`, a
streaming `Encoder`/`Decoder`) were behind `GOEXPERIMENT=jsonv2` in Go 1.25 and 1.26 and ship
enabled in 1.27. The v1 package now runs on the v2 implementation with options that keep v1
behavior; `GOEXPERIMENT=nojsonv2` restores the old implementation for now. v1 is not
deprecated, and you don't have to migrate.

## Coming from C/C++/Java/Python/JS

- Python: `csv.reader(f, delimiter=";")` ≈ `csv.NewReader` with `Comma`; Python's
  `encoding="utf-8-sig"` strips the BOM, Go has no such switch. `json.load` builds dicts;
  Go decodes straight into typed structs.
- Java: Jackson's `@JsonProperty` ≈ struct tags; Jackson's streaming `JsonParser` ≈
  `Decoder.Token`. Jackson fails on unknown properties by default; both Go packages ignore
  them unless you ask (`DisallowUnknownFields`, `RejectUnknownMembers`).
- JS: `JSON.parse` keeps the last duplicate key and has no schema, which is why the TS track
  validates with zod. Go v1 behaves like `JSON.parse` on duplicates; v2 refuses them.
- C/C++: no standard library support at all; Go's tag-driven mapping replaces hand-written
  visitors or code generators.

## How it works

The v1/v2 differences that change output or acceptance by default, from the `encoding/json`
"Migrating to v2" docs (v1 first; the option that controls it in parentheses):

- Name matching: case-insensitive vs case-sensitive (`MatchCaseInsensitiveNames`).
- Duplicate names: last one wins vs error (`jsontext.AllowDuplicateNames`).
- Invalid UTF-8: replaced with U+FFFD vs error (`jsontext.AllowInvalidUTF8`).
- nil slice and map: `null` vs `[]` and `{}` (`FormatNilSliceAsNull`, `FormatNilMapAsNull`).
- `omitempty`: omits empty Go values (0, false, "") vs empty JSON values (null, "", {}, []).
- Map keys: sorted vs unspecified order (`Deterministic`).
- `time.Duration`: nanoseconds vs an error (`FormatDurationAsNano` in `encoding/json`).
- `<`, `>` and `&` in strings: escaped for HTML vs written as is (`jsontext.EscapeForHTML`).

In v2, `json.RawMessage`'s role is played by `jsontext.Value` (v1's `RawMessage` is now an
alias of it), and the streaming loop is `jsontext.NewDecoder` with `json.UnmarshalDecode`.
`UnmarshalRead` reads its reader to EOF and expects exactly one value, unlike v1's
`Decoder.Decode`. If you tried the experiment in 1.25 or 1.26: in 1.27 the `inline` tag
option is called `embed`, and the `format` and `unknown` options are gone. Only `format`
fails loudly; a leftover `inline` or `unknown` is ignored, so the field is quietly no
longer flattened.

## Design notes

Treat imports as a trust boundary. For CSV from users: sniff the encoding (`utf8.Valid` on
the first few KB, allowing for a character cut off at the end of the sample), decode
windows-1254 if needed, strip the BOM, map headers by name rather than position, and
validate each row into a typed struct with its line number (`Reader.FieldPos`) in the error;
or load the raw rows into a staging table and let SQL validate them. For JSON, new code can
use v2 for its stricter defaults. To move existing types, call v2 with
`jsonv1.DefaultOptionsV1()` first and add v2 behaviors one option at a time after it (later
options win), with a round-trip test per type. Whatever you pick, decode into structs at the
edge and keep `map[string]any` out of the core.

Where the Excel facts come from. Microsoft documents two: a comma decimal separator
"forces Excel to use a semi-colon for the list separator"
([Import or export text files](https://support.microsoft.com/en-us/office/import-or-export-text-txt-or-csv-files-5250ac4c-663c-47ce-937b-339e391393ba)),
and a UTF-8 CSV opens normally "if it was saved with BOM"
([Opening CSV UTF-8 files correctly](https://support.microsoft.com/en-us/office/opening-csv-utf-8-files-correctly-in-excel-8a935af5-3416-4edd-ba7e-3dfd2bc4a032)).
The rest was observed on 2026-11-14 with Excel for Microsoft 365 (build 16.0.20527.20196)
on an en-US Windows, with Excel's own separators set to decimal `,` and thousands `.`, as
Windows's tr-TR locale has them. "CSV UTF-8" starts with `EF BB BF` and writes `;`,
`1250,5` and CRLF row ends. "CSV (Comma delimited)" is written in the Windows ANSI code
page: windows-1252 on that machine, where ş, ı and İ became `?`. Turkish Windows's ANSI
code page is 1254
([code page identifiers](https://learn.microsoft.com/en-us/windows/win32/intl/code-page-identifiers)).
So a file with `?` in place of Turkish letters was saved as plain CSV on a non-Turkish
Windows, and no decoder brings them back: ask for "CSV UTF-8". Double-clicked, a UTF-8 file
without a BOM showed `Müşteri` as `MÃ¼ÅŸteri`, its bytes read as windows-1252.

