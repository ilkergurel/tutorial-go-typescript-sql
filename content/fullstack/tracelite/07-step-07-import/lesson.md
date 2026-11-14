---
id: fullstack.tracelite.step-07-import
title: "Step 7: File import: browser CSV upload, Go streaming validation, staging table, COPY"
track: fullstack
module: tracelite
order: 7
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-06-ui, go.files-io.csv-json, sql.files-movement.csv-staging, go.postgres.bulk]
compare_with: [sql.files-movement.copy, go.http.uploads-streams, ts.browser-dom.reading-files, sql.subqueries-ctes.recursive]
tags: [tracelite, import, csv, excel, windows-1254, utf-8, bom, copy, staging, streaming, validation]
summary: >
  The browser sends the spreadsheet as it is, labelled with its charset. Go decodes,
  splits and validates it one record at a time while pgx COPYs the records into a
  temporary staging table; SQL checks what needs other rows, and the UI gets every
  problem with its line, or the whole file imported.
examples:
  - id: upload
    file: examples/01_upload.ts
    title: "The browser side: detect the charset, send the file itself as the body"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "5-13"
        note: >
          The browser has the file, so it decides how to label it. A strict UTF-8
          decode of the first 64 KiB either works or throws; Turkish text in
          windows-1254 is practically never valid UTF-8. A wrong guess can only
          go one way, an ASCII start with windows-1254 further down, and the
          server then reports those rows as not UTF-8 instead of storing
          replacement characters.
      - lines: "17-26"
        note: >
          The charset parameter of Content-Type is HTTP's own way to say how
          text is encoded, and curl users can set it too. The body is the File:
          the browser reads it from disk while it sends, and with one file and
          no other fields there is nothing for multipart/form-data to add.
      - lines: "24"
        note: >
          In the app a 422 becomes ImportFailed, which carries the import
          problem and its rows; "How it works" shows how the form lists them.
      - lines: "29-36"
        note: >
          Harness: the iframe has no network, so fetch is replaced by a fake
          that decodes the body by its label, as the Go side does. TextDecoder
          drops a leading BOM unless ignoreBOM is set; Go's encoding/csv keeps
          it, which is why example 2 strips it by hand.
      - lines: "43-46"
        note: >
          What Excel writes. "CSV UTF-8" is the three BOM bytes, then UTF-8;
          "CSV (Comma delimited)" on Turkish Windows is windows-1254, one byte
          per Turkish letter. The table holds the letters this sheet uses; the
          rest is ASCII, which both encodings share.
      - lines: "52-57"
        note: >
          The same sheet saved both ways: different sizes, the same text after
          decoding, because each label matched its bytes.
      - lines: "59-67"
        note: >
          Two-byte ş straddles the 64 KiB cut. With { stream: true } the decoder
          keeps the first byte and waits for more; without it, the cut is an
          error and a UTF-8 file would be sent as windows-1254.
  - id: rows
    file: examples/02_rows.go
    title: "Reading Excel's CSV: decode, find the separator, number the lines, count characters"
    runtime: go
    check: exact
    annotations:
      - lines: "18-20"
        note: >
          Decoding comes first. charmap's decoder turns windows-1254 bytes into
          UTF-8 as they stream through, so everything below sees UTF-8, whatever
          Excel wrote. golang.org/x/text is outside the standard library, and
          the runner provides it.
      - lines: "21-24"
        note: >
          The header line is read off a bufio.Reader and put back in front with
          MultiReader: csv.Reader still sees the whole file, so its line numbers
          stay the file's. "CSV UTF-8" puts U+FEFF in front of "code", and
          encoding/csv leaves it there.
      - lines: "25-27"
        note: >
          The header names are fixed, so whichever of ; and , appears more often
          in it is the separator. Excel writes the Windows list separator, which
          is ; where , is the decimal separator, as on Turkish Windows.
      - lines: "28"
        note: >
          FieldsPerRecord = -1 turns off the reader's own count check, which
          would return csv.ErrFieldCount. A missing field reads as "" below, so
          a short row shows up as row problems instead of stopping the import.
      - lines: "41"
        note: >
          FieldPos(0) is the line where the record starts, counted from 1 with
          the header as line 1. That is Excel's row number, until a cell above
          contains a line break: a quoted multi-line cell spans several lines.
      - lines: "42-45"
        note: >
          Fields are looked up by header name, so the columns may come in any
          order. The real reader also refuses an unknown header such as
          "titel" before reading any row.
      - lines: "47-48"
        note: >
          Invalid UTF-8 means the label was wrong: windows-1254 bytes sent as
          utf-8. Every Turkish letter is then a problem here, instead of a
          replacement character in the database.
      - lines: "49-50"
        note: >
          RuneCountInString counts characters, like the CHECK's char_length:
          the 200 ş of LONG are 400 bytes and pass, the 201 of LONGER don't.
          len(title) > 200 would have refused LONG too. The service's other
          rules (code format, status) run here the same way.
      - lines: "62-64"
        note: >
          The same sheet as windows-1254 is about half the size: ş is two bytes
          in UTF-8 and one in windows-1254.
      - lines: "66-67"
        note: >
          After decoding, both files give the same problems: the limit is
          counted in characters after decoding, so the file's encoding never
          matters.
      - lines: "68"
        note: >
          The same bytes labelled utf-8: every title with a Turkish letter is
          reported, and nothing is silently wrong.
      - lines: "69"
        note: >
          A quote that never closes swallows the rest of the file into one
          field. csv.Reader reports it as a *csv.ParseError with the line the
          record started on; the import turns that into a row error and stops.
  - id: staging
    file: examples/03_staging.go
    title: "COPY into a temporary staging table, one record at a time"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "16-18"
        note: >
          A temporary table is private to this session, so concurrent imports
          never see each other's rows, and ON COMMIT DROP (or the rollback)
          removes it: there is nothing to clean up.
      - lines: "26"
        note: >
          In the importer this reader is example 2's: decoded, BOM stripped,
          separator detected, header checked.
      - lines: "33-44"
        note: >
          The streaming join: pgx pulls records from CopyFromFunc to fill its
          send buffer, the function pulls them from the CSV reader, and the
          reader pulls bytes from the request body. Memory holds that buffer
          and the current record, whatever the file's size. Every record goes
          in, valid or not, so the SQL checks of example 4 see every code.
      - lines: "39-40"
        note: >
          When the source fails, pgx sends CopyFail and returns the server's
          answer: a *pgconn.PgError (SQLSTATE 57014) whose message repeats the
          reader's text. errors.As can't find the *csv.ParseError in it, so
          readErr keeps the typed error, with its StartLine for the report.
      - lines: "62"
        note: >
          A ; inside quotes is part of the field, so the title arrives whole.
      - lines: "65"
        note: >
          The import's one transaction (store.WithTx in the app). Its first
          statement locks the project row FOR UPDATE, step 3's lock, so
          imports and moves in one project take turns, and an unknown project
          is a 404 before any work.
      - lines: "77"
        note: >
          Nothing is committed here. The importer runs example 4's checks, and
          either rolls back with the problems or inserts and commits.
  - id: checks
    file: examples/04_checks.sql
    title: "SQL checks over the staging table, then one INSERT for the whole tree"
    runtime: sql
    check: exact
    annotations:
      - lines: "11-17"
        note: >
          The ids are minted here by the column default, before any
          requirement exists, so the insert can resolve parents inside the
          file. This is where step 2's "the importer mints ids" happens.
      - lines: "28-33"
        note: >
          Loops: the recursive part starts from rows whose parent is empty or
          outside the file and walks down. UNION, not UNION ALL, stops at rows
          already reached. A row it never reaches is in a loop, or under one.
      - lines: "34-36"
        note: >
          A code already in the project. The importer's queries add
          project_id = $1 to every lookup in requirements; this abridged table
          holds one project.
      - lines: "38-41"
        note: >
          Duplicates within the file: a window over code finds the first line
          of each, and every later line names it.
      - lines: "43-46"
        note: >
          A parent found nowhere, neither in the file nor in the project.
      - lines: "53-60"
        note: >
          One INSERT for the whole tree. AUTH-LOGIN-2FA comes before its parent
          AUTH-LOGIN; the foreign key is checked at the end of the statement,
          when both rows exist. AUTH-LOGIN's parent is found in the project.
use_cases:
  - "Bulk-loading requirements, test cases or results that people keep in Excel"
  - "Any upload where users must see every problem with its row and column, then fix and resend"
  - "Feeds from legacy Windows tools that write the ANSI code page instead of UTF-8"
  - "Large files that must not be held in memory: the request body flows through the parser into COPY"
strengths:
  - "Constant memory: bytes flow from the request through the decoder, the CSV reader and CopyFromFunc into PostgreSQL, one record at a time"
  - "COPY sends rows in one stream, much faster than one INSERT per row, and the staging table lets SQL check against the project's data in one query"
  - "Every problem is reported at once, with line and column, unlike a loader that stops at the first bad row"
  - "One transaction makes the import all or nothing, so a half-imported tree with dangling parents can't exist"
weaknesses:
  - "The charset guess reads 64 KiB; a file whose Turkish letters start later is labelled utf-8 and comes back with row errors rather than imported"
  - "All or nothing means one bad row blocks a 5,000-row file; skipping bad rows would need a policy for their children"
  - "The project stays locked for the whole import, so moves in that project wait until it finishes"
  - "Rules live twice: in the service for messages and in the CHECKs and keys for the guarantee; both must change together"
pitfalls:
  - title: "Counting bytes instead of characters"
    detail: "len(title) is bytes. A 200-character Turkish title can be 400 bytes and must pass; use utf8.RuneCountInString, as the database's char_length does."
    bad: "if len(title) > 200 { ... }"
    good: "if utf8.RuneCountInString(title) > 200 { ... }"
  - title: "Leaving the BOM on the first header"
    detail: "encoding/csv keeps U+FEFF as part of the first field, so the header reads \"\\ufeffcode\" and a lookup by \"code\" fails. Strip it before parsing."
  - title: "Reading the file into memory first"
    detail: "io.ReadAll(r.Body) followed by csv.ReadAll holds the file twice. Stream: csv.Reader over the body, records into CopyFromFunc."
  - title: "Trusting pgx's error after a failed source"
    detail: "When the CopyFromSource returns an error, pgx returns the server's \"COPY from stdin failed\" PgError, which repeats the source's message as text only. Keep the source's error yourself to get the typed *csv.ParseError, its line and its cause."
  - title: "Inserting row by row and stopping at the first error"
    detail: "The user fixes one row, uploads, and hits the next. Stage everything, check everything, report everything, then insert in one statement."
quiz:
  - q: "Excel on Turkish Windows saves \"CSV (Comma delimited)\". What does the server receive?"
    options: ["UTF-8 with a BOM and commas", "windows-1254 bytes with ; between fields", "UTF-16 with tabs"]
    answer: 1
    why: "The ANSI code page of Turkish Windows is windows-1254, and the list separator is ; because , is the decimal separator."
  - q: "A title has 150 Turkish characters, 290 bytes in UTF-8. The limit is 200. What does the import say?"
    options: ["Too long: 290 > 200", "Fine: 150 characters", "It depends on whether the file was windows-1254"]
    answer: 1
    why: "The limit is in characters, counted after decoding; the encoding only changes the byte count."
  - q: "Why does every record go into the staging table, even one that already failed a Go check?"
    options: ["COPY can't skip rows", "So the SQL checks see every code, and its children aren't misreported as having no parent", "To make the import faster"]
    answer: 1
    why: "If a parent row with a bad title were left out, its children would get a second, misleading problem."
  - q: "The file lists AUTH-LOGIN-2FA before its parent AUTH-LOGIN. What happens on INSERT ... SELECT?"
    options: ["requirements_parent_fkey fails for the child", "Both rows are inserted: the foreign key is checked at the end of the statement", "The import sorts the rows first"]
    answer: 1
    why: "Both rows come from one statement, and the constraint is checked when it finishes, when the parent exists."
further_reading:
  - title: "package encoding/csv (Reader, FieldPos, ParseError)"
    url: "https://pkg.go.dev/encoding/csv"
  - title: "package golang.org/x/text/encoding/charmap (Windows1254)"
    url: "https://pkg.go.dev/golang.org/x/text/encoding/charmap"
  - title: "pgx v5: Conn.CopyFrom and CopyFromFunc"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#CopyFromFunc"
  - title: "package net/http: MaxBytesReader"
    url: "https://pkg.go.dev/net/http#MaxBytesReader"
  - title: "PostgreSQL docs: CREATE TABLE (TEMPORARY, ON COMMIT DROP)"
    url: "https://www.postgresql.org/docs/current/sql-createtable.html"
  - title: "MDN: TextDecoder.decode() (fatal, stream)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/TextDecoder/decode"
---

## Concept

An import has four stages, and each catches what the next can't:

1. **The browser** labels the file. It has the bytes, so it tries a strict UTF-8 decode
   of the beginning and sends `Content-Type: text/csv; charset=utf-8` or
   `charset=windows-1254`, with the file itself as the body.
2. **Go** decodes (windows-1254 to UTF-8 on the fly), strips Excel's BOM, picks the
   separator from the header, and validates each record with the service's rules (step 4)
   while it streams: code format, title length in characters, status, field count. Each
   problem gets the line Excel shows and the column's name.
3. **pgx COPY** writes every record into a temporary staging table as it is read. Nothing
   is held in memory but the current record.
4. **SQL** checks what needs other rows: codes already in the project, duplicates in the
   file, parents found nowhere, loops among the new rows. Then, if no stage found a
   problem, one `INSERT ... SELECT` creates the whole tree, in the same transaction.

The answer is `200 {"imported": n}`, or `422` with every problem from every stage and
nothing written. The four examples follow the four stages; example 2 shows the title
rule, and the service's other rules run the same way.

## Coming from C/C++/Java/Python/JS

- **Python/pandas**: `pd.read_csv(f, sep=None, encoding="cp1254")` guesses the
  separator and decodes in one call, but loads the whole file and stops at the first
  parse error. Here the same decisions are explicit and the file streams.
- **Java/Spring Batch**: reader, processor, writer with a skip policy is the same
  pipeline. TraceLite's skip policy is "none, but report all", and the writer is COPY.
- **JS (Papa Parse in the browser)**: parsing in the browser moves the work but not the
  trust; the server must validate anyway. TraceLite's browser only labels the bytes.
- **C/C++**: `iconv` in front of a hand-written tokenizer is the same layering. The
  order matters in every language: decode, then split, then count characters.

## How it works

The endpoint and its problem type, in `openapi.yaml`:

```yaml
  /api/projects/{projectId}/requirement-import:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    post:
      operationId: importRequirements
      requestBody:
        required: true
        content: { text/csv: { schema: { type: string } } }  # charset: utf-8 or windows-1254
      responses:
        "200": { description: Imported, content: { application/json: { schema: { $ref: "#/components/schemas/ImportResult" } } } }
        "404": { $ref: "#/components/responses/Problem" }
        "413": { $ref: "#/components/responses/Problem" }
        "415": { $ref: "#/components/responses/Problem" }
        "422":
          description: Nothing was imported
          content: { application/problem+json: { schema: { $ref: "#/components/schemas/ImportProblem" } } }
# components.schemas:
    ImportResult: { type: object, required: [imported], properties: { imported: { type: integer } } }
    RowError:
      type: object
      required: [line, column, detail]
      properties:
        line: { type: integer, minimum: 1 }
        column: { type: [string, "null"] }
        detail: { type: string }
    ImportProblem:
      allOf:
        - { $ref: "#/components/schemas/Problem" }
        - { type: object, required: [rows], properties: { rows: { type: array, items: { $ref: "#/components/schemas/RowError" } } } }
```

A JSON Pointer can't point into a CSV body, so each `RowError` names the line, as Excel
numbers rows, and the column by its header name (`null` for the whole line). The
handler refuses anything but CSV before reading a byte, and limits the body while it
streams:

```go
media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
if err != nil || media != "text/csv" {
	writeError(w, errNotCSV) // 415: an .xlsx is a zip file, not CSV
	return
}
body := http.MaxBytesReader(w, r.Body, 10<<20) // 10 MiB; JSON bodies get 64 KiB
n, err := s.Import(r.Context(), project, body, params["charset"])
```

A larger upload fails while COPY reads it, and `writeError` answers 413 for its
`*http.MaxBytesError`.

The file format: a header line naming the columns, in any order, `code` and `title`
required, `parent` (a code in the file or the project), `status` (empty means `draft`)
and `description` optional. Separator `;` or `,`, line ends CRLF or LF, quotes as in
RFC 4180. Step 8's export writes exactly this format, so an export can be edited in
Excel and imported into another project.

What Excel produces on Turkish Windows, for the two "Save as" choices people use:

| Save as | Bytes | Separator | Read by |
|---|---|---|---|
| CSV UTF-8 (Comma delimited) | `EF BB BF`, then UTF-8 | `;` | `charset=utf-8`, BOM stripped |
| CSV (Comma delimited) | windows-1254 | `;` | `charset=windows-1254`, decoded by x/text |

Microsoft documents the `;`: a comma decimal separator "forces Excel to use a semi-colon
for the list separator"
([Import or export text files](https://support.microsoft.com/en-us/office/import-or-export-text-txt-or-csv-files-5250ac4c-663c-47ce-937b-339e391393ba)).
The bytes were observed on 2026-11-14 with Excel for Microsoft 365 (build
16.0.20527.20196) on an en-US Windows, with Excel's own separators set to decimal `,` and
thousands `.`: both saves wrote `;` and ended every row with CRLF, while a line break
inside a quoted cell stayed a bare LF; "CSV UTF-8" started with `EF BB BF`; "CSV (Comma
delimited)" came out in the Windows ANSI code page, windows-1252 on that machine, which
has no ş, ı or İ and wrote `?` instead. Turkish Windows's ANSI code page is 1254
([code page identifiers](https://learn.microsoft.com/en-us/windows/win32/intl/code-page-identifiers)).
Going the other way, Microsoft says a UTF-8 CSV opens normally "if it was saved with BOM"
([Opening CSV UTF-8 files correctly](https://support.microsoft.com/en-us/office/opening-csv-utf-8-files-correctly-in-excel-8a935af5-3416-4edd-ba7e-3dfd2bc4a032));
double-clicked without one, `Müşteri` showed as `MÃ¼ÅŸteri`, which is why step 8's export
writes the BOM.

Characters, not bytes, at every limit: the 200-character title rule is checked after
decoding with `utf8.RuneCountInString`, the same count as PostgreSQL's `char_length`. A
title of 200 `ş` is 200 bytes in windows-1254, 400 bytes in UTF-8 and 200 characters in
both, so it passes either way.

The import's transaction, in order: lock the project row (`FOR UPDATE`, as moves do),
`CREATE TEMP TABLE import_rows (...) ON COMMIT DROP`, `COPY` from the request, run the
checks, then either roll back with the problems or insert and commit. The staging
table's `id uuid DEFAULT uuidv7()` is where step 2's "the importer mints ids" happens:
each new requirement has its id before it exists, so parents inside the file resolve by
a join on code.

Every record is staged, valid or not, so the SQL checks see every code, and a child of a
broken row isn't also reported as having no parent. PostgreSQL rejects invalid UTF-8,
which would fail the whole COPY, so a field the Go checks already reported as not UTF-8
is staged through `strings.ToValidUTF8`. A CSV parse error (an unclosed quote) stops the
import as one row error on the record's `StartLine`. Otherwise the problems from Go
(example 2) and from SQL (example 4) are merged and sorted by line, and any problem
rolls the transaction back.

The form, `src/features/import/ImportForm.tsx`, lists those rows in the user's terms. It
clears the file input after reading it: an input fires `change` only when the selection
changes, and after fixing the sheet people pick the same file again.

```tsx
<input
  type="file"
  accept=".csv,text/csv"
  onChange={(e) => {
    const file = e.currentTarget.files?.[0];
    e.currentTarget.value = ""; // the same file chosen again is a change again
    if (file) importer.mutate(file);
  }}
/>
{problem?.rows.map((row) => (
  <tr key={`${row.line} ${row.column} ${row.detail}`}>
    <td>satır {row.line}</td>
    <td>{row.column ?? "(satır)"}</td>
    <td>{row.detail}</td>
  </tr>
))}
```

## Design notes

**Validation in two places, by what each can see.** Go sees one record and knows the
line and column, so format and length checks live there, in the service shared with
the JSON API. SQL sees everything at once, so uniqueness, references and loops live
there, as set queries over the staging table. The constraints from step 2 stay as the
guarantee behind both; with every rule mirrored, the final insert can't fail on them.

**All or nothing.** A requirements tree imported half-way has children whose parents
were refused, and the second upload of the fixed file then collides with the first
half. Refusing the whole file keeps "fix the sheet, upload it again" the only workflow.
The cost is that one bad row blocks the file; the report lists every problem so that it
takes one round. A large file with many errors should cap the list (the first hundred,
and a count), which this example doesn't.

**Raw body, not multipart.** One file and no other fields means multipart would only
add a boundary to parse; with the file as the body, `r.Body` is the CSV, and the
charset travels where HTTP defines it. **Temporary, not permanent, staging.** A permanent
`import_rows` table with an import id would let the UI fetch the report later, at the
price of cleanup and of rows from concurrent imports in one table. The report fits in
the response, so the session-private table that disappears with the transaction is the
simpler choice. **COPY, not batched INSERTs.** A pgx Batch would also stream, but COPY is
one statement for the server to parse and the fastest bulk path PostgreSQL has; the
staging table is what makes its all-rows-or-error behavior harmless.
