---
id: fullstack.tracelite.step-08-export
title: "Step 8: File export: streaming CSV/JSON download and a traceability report"
track: fullstack
module: tracelite
order: 8
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-07-import, go.http.uploads-streams, go.services.pagination-streaming, go.files-io.csv-json]
compare_with: [sql.files-movement.dump-generate, ts.browser-dom.generate, go.http.server-lifecycle, sql.windows.over]
tags: [tracelite, export, streaming, csv, json-v2, jsontext, content-disposition, http, abort, excel]
summary: >
  Exports stream: pgx rows are written as they arrive, through csv.Writer or a jsontext
  Encoder, so memory stays flat whatever the project's size. Everything that can fail
  early is checked before the 200; after it, a failure aborts the connection so the
  download fails instead of looking complete.
examples:
  - id: csv
    file: examples/01_csv.go
    title: "The CSV export for Excel: checked before the first byte, then written row by row"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "21-27"
        note: >
          Everything that can fail cheaply comes before the first byte: the
          project, and its name for the file name. Until something is written, a
          failure is still an ordinary problem response. TraceLite parses the id
          with uuid.Parse first; here a malformed id simply fails the lookup,
          which ends in the same 404.
      - lines: "28-29"
        note: >
          The columns and order of step 7's import, so a file exported, edited
          in Excel and imported elsewhere round-trips. ORDER BY code is the
          requirements_code_key index's order.
      - lines: "30"
        note: >
          Deferred calls run while a panic unwinds, including the
          ErrAbortHandler panic below, so the pooled connection is released
          either way.
      - lines: "31-35"
        note: >
          pgx sends the query and returns; an error in it arrives with the first
          row. Reading that row before setting any header turns a broken query
          (a typo, a missing permission) into a 500, not a 200 with an empty
          file. Example 2 shows both outcomes.
      - lines: "37-38"
        note: >
          mime.FormatMediaType writes a non-ASCII parameter in RFC 2231's
          encoded form, filename*=utf-8''%C3%96deme..., which browsers decode
          to "Ödeme gereksinimleri.csv". A filename here also overrides the one
          an <a download> attribute suggests.
      - lines: "39-41"
        note: >
          The BOM is what makes Excel read the bytes as UTF-8 when the file is
          double-clicked: Microsoft documents that a UTF-8 CSV with one opens
          normally, and without it Excel for Microsoft 365 showed Müşteri as
          MÃ¼ÅŸteri.
          ';' is the separator Excel expects with Turkish regional settings,
          and CRLF is RFC 4180's line break and what Excel writes itself.
      - lines: "43-48"
        note: >
          One row in memory at a time: scan it, hand it to csv.Writer, forget
          it. csv.Writer and the ResponseWriter each buffer a few KiB and pass
          the bytes on whenever a buffer fills, so the download runs while the
          query is still reading, without any explicit Flush. TraceLite's
          handler also passes title and description through excelText (Design
          notes); this example leaves it out to stay within one screen.
      - lines: "49-53"
        note: >
          After the first byte the status is already 200. The only honest move
          left is to abort; example 2 shows what returning normally would do
          instead. The cause goes to the log, because the client can no longer
          be told.
      - lines: "67-72"
        note: >
          The recorder captures the response a browser would save: a BOM, ';'
          between fields, quotes around the description that contains ';', and
          CRLF after every record.
  - id: abort
    file: examples/02_abort.go
    title: "A failure half-way: returning leaves a clean short file, ErrAbortHandler a failed download"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "19-25"
        note: >
          The first-row check of example 1. A query that fails at once, here a
          misspelled table, is still an ordinary 500 because nothing has been
          written yet.
      - lines: "34-39"
        note: >
          After the first byte the status line is on the wire. Returning
          normally makes net/http end the chunked body properly, so the client
          keeps a complete-looking, shorter file. Panicking with
          http.ErrAbortHandler closes the connection without the final chunk,
          and net/http logs no stack trace for this particular value.
      - lines: "48-50"
        note: >
          A stand-in failure: 0 / (1500 - n) is 0 until row 1,500, which
          divides by zero after 1,499 rows have gone out. Real exports break
          the same way on a statement timeout or a restarted database.
      - lines: "68-69"
        note: >
          Both broken downloads are chunked, with no Content-Length: the server
          never had the whole file, so it never knew the size. The returning
          handler's file ends cleanly after 1,499 rows, and nothing tells the
          user that 501 are missing. The aborted transfer ends with unexpected
          EOF, which a browser shows as a failed download; it also lost what
          was still in net/http's buffer, down to half a line, which doesn't
          matter once the download counts as failed.
  - id: report-query
    file: examples/03_report_query.sql
    title: "The report query: tree order, one row per requirement and test, coverage on every row"
    runtime: sql
    check: exact
    setup: |
      CREATE TABLE requirements (id uuid PRIMARY KEY, project_id uuid NOT NULL,
        parent_id uuid REFERENCES requirements, code text COLLATE "C" NOT NULL, title text NOT NULL);
      CREATE TABLE test_cases (id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL,
        code text COLLATE "C" NOT NULL);
      CREATE TABLE trace_links (project_id uuid NOT NULL, requirement_id uuid NOT NULL REFERENCES requirements,
        test_case_id uuid NOT NULL REFERENCES test_cases, PRIMARY KEY (project_id, requirement_id, test_case_id));
      CREATE TABLE test_runs (id uuid PRIMARY KEY DEFAULT uuidv7(), test_case_id uuid NOT NULL REFERENCES test_cases,
        result text NOT NULL, finished_at timestamptz NOT NULL);
      SET LOCAL TimeZone = 'UTC'; -- finished_at then reads like the report's JSON times
      -- Project Ödeme of the lesson data (steps 6-10).
      INSERT INTO requirements (id, project_id, parent_id, code, title)
      SELECT ('01937b6e-0000-7000-8000-00000000000' || n)::uuid, '01937b6e-0000-7000-8000-00000000000a',
             ('01937b6e-0000-7000-8000-00000000000' || parent)::uuid, code, title
      FROM (VALUES (1, NULL, 'AUTH', 'Kimlik doğrulama'), (2, 1, 'AUTH-LOGIN', 'Parola ile giriş'),
        (3, 2, 'AUTH-LOGIN-2FA', 'İki adımlı doğrulama'), (4, 2, 'AUTH-LOGIN-LOCKOUT', 'Hesap kilitleme'),
        (5, 1, 'AUTH-RESET', 'Parola sıfırlama'), (6, NULL, 'EXPORT', 'Dışa aktarma'),
        (7, 6, 'EXPORT-CSV', 'CSV indir')) AS v (n, parent, code, title);
      INSERT INTO test_cases (project_id, code)
      SELECT '01937b6e-0000-7000-8000-00000000000a', 'TC-' || n FROM generate_series(1, 4) n;
      INSERT INTO trace_links SELECT r.project_id, r.id, t.id FROM requirements r JOIN test_cases t
        ON (r.code, t.code) IN (('AUTH-LOGIN-2FA', 'TC-1'), ('AUTH-LOGIN-2FA', 'TC-2'),
          ('AUTH-LOGIN-LOCKOUT', 'TC-1'), ('AUTH-LOGIN-LOCKOUT', 'TC-4'), ('AUTH-RESET', 'TC-3'));
      INSERT INTO test_runs (test_case_id, result, finished_at)
      SELECT t.id, v.result, v.at::timestamptz FROM test_cases t JOIN (VALUES
        ('TC-1', 'failed', '2026-11-12 10:00+03'), ('TC-1', 'passed', '2026-11-13 09:00+03'),
        ('TC-2', 'passed', '2026-11-13 09:00+03'), ('TC-3', 'passed', '2026-11-12 10:00+03'),
        ('TC-3', 'failed', '2026-11-13 09:00+03')) AS v (code, result, at) ON t.code = v.code;
    annotations:
      - lines: "4"
        note: >
          PREPARE with a $1 parameter runs the query exactly as the Go store
          sends it. The tables and the data of project Ödeme are in this
          example's setup.
      - lines: "5-9"
        note: >
          Step 4's tree: SEARCH DEPTH FIRST BY code produces the depth-first
          order (ord) in which the report lists requirements, with each
          requirement's depth.
      - lines: "11"
        note: >
          The Go side needs to know which requirements are headings: one with
          children and no tests of its own doesn't count in the summary (step
          6's roll-up rule).
      - lines: "12-15"
        note: >
          Step 2's coverage rule, computed with window aggregates over
          PARTITION BY the requirement, so every row of a requirement carries
          its coverage and the rule stays in SQL, out of Go. count(*) FILTER
          rather than bool_and, which ignores NULLs.
      - lines: "20-21"
        note: >
          The latest run of each linked test, ordered by finished_at DESC, id
          DESC as everywhere in TraceLite. TC-4 never ran, so its result and
          finished_at are NULL; TC-1's failed run of the day before is ignored.
      - lines: "23"
        note: >
          The rows of one requirement are adjacent, so the Go side can stream
          them: a requirement is complete when the next row's code differs.
  - id: report
    file: examples/04_report.go
    title: "The report as streamed JSON: jsontext tokens around one requirement at a time, summary last"
    runtime: go
    check: exact
    annotations:
      - lines: "14-35"
        note: >
          The report's wire types. result and finishedAt are null for a test
          that never ran, never absent. hasChildren is unexported, so
          encoding/json/v2 leaves it out of the JSON.
      - lines: "38-43"
        note: >
          jsontext.Encoder writes tokens and values as a stream and checks the
          grammar: a value in the wrong place, or an EndArray without its
          BeginArray, is an error instead of broken JSON. It writes to w when
          its buffer is sufficiently full or a top-level value is complete.
          cmp.Or returns the first non-nil error; every call still runs, which
          is harmless here because these tokens are valid in this order.
      - lines: "46-49"
        note: >
          On failure the closing tokens are never written. Closing the array
          and the object would turn a partial report into valid JSON that a
          client accepts; JSON that stops in the middle fails to parse. The
          handler then aborts as in example 2.
      - lines: "50"
        note: >
          json.MarshalEncode writes a whole Go value at the current position,
          with the struct tags and options of encoding/json/v2. Only the
          current requirement is in memory.
      - lines: "53-57"
        note: >
          The summary counts as step 6's tree does: every requirement adds to
          requirements, but a heading without tests of its own adds nothing to
          counted or to the coverage counts.
      - lines: "59-62"
        note: >
          A streamed document can only put aggregates after the data they sum.
          Putting summary first would need a second query or a buffered body.
      - lines: "67-78"
        note: >
          A stand-in for the report query's rows (example 3), already grouped
          per requirement; the times are UTC because Go converts pgx's local
          times with .UTC() before they reach the wire.
      - lines: "79"
        note: >
          The output is the document as the client receives it: one line, with
          the newline jsontext writes after a complete top-level value.
use_cases:
  - "Spreadsheet exports that people open in Excel, edit and import again"
  - "Audit and traceability reports handed to a reviewer or a certification body"
  - "Any download whose size grows with the data: run histories, logs, backups of a project"
  - "Feeding another tool with NDJSON or JSON without paging through an API"
strengths:
  - "Memory stays flat with the project's size: one row in Go, a few KiB in each buffer, unlike building a []byte or a slice of all rows first"
  - "Bytes leave as soon as a buffer fills, while the query is still running, so the browser shows the download at once instead of after the whole query"
  - "jsontext checks the document's grammar while it is written, so a streaming bug can't silently produce invalid JSON"
  - "An aborted response makes the failure visible to the browser and to curl, where a short file with a clean end would not be"
weaknesses:
  - "After the 200 there is no way to send a problem; the client learns only that the transfer broke, and the cause is in the server's log"
  - "A slow download holds a pooled connection and an open query for its whole duration, and a long-open snapshot holds back VACUUM"
  - "Summaries can only come after the data in a single streamed document"
  - "A server with an http.Server WriteTimeout cuts long downloads at that deadline; a streaming handler then has to extend its own with ResponseController.SetWriteDeadline (TraceLite's server sets no WriteTimeout)"
pitfalls:
  - title: "Returning normally after a mid-stream error"
    detail: "net/http then ends the chunked body properly and the client keeps a truncated file that looks complete. Panic with http.ErrAbortHandler after logging the cause."
    bad: "if err := rows.Err(); err != nil { log.Print(err); return }"
    good: "if err := rows.Err(); err != nil { log.Print(err); panic(http.ErrAbortHandler) }"
  - title: "Setting headers or a status after the first write"
    detail: "The first Write sends the status line and headers. Content-Disposition set after it is silently dropped, and WriteHeader(500) only logs \"superfluous response.WriteHeader call\"."
  - title: "Downloading with fetch and a Blob"
    detail: "await res.blob() holds the whole file in the page's memory before the user sees it. A plain same-origin link with the download attribute lets the browser stream to disk."
    bad: "const blob = await (await fetch(url)).blob(); a.href = URL.createObjectURL(blob)"
    good: "<a href={`/api/projects/${projectId}/requirement-export`} download>CSV indir</a>"
  - title: "Closing the JSON after an error"
    detail: "Writing the closing brackets in a deferred cleanup turns any partial report into valid JSON. Only the success path closes the document."
  - title: "A non-ASCII file name in plain filename"
    detail: "Raw UTF-8 bytes in filename=\"Ödeme.csv\" are outside what the plain parameter is defined to carry; RFC 6266 points to filename* for other characters. mime.FormatMediaType writes that form (filename*=utf-8''...) for you."
quiz:
  - q: "The export's query fails on row 1,500, after 1,499 rows were sent. Which status does the client see?"
    options: ["500, because the handler calls writeError", "200: the status went out with the first bytes", "206 Partial Content"]
    answer: 1
    why: "The status line and headers are sent before the first byte of the body; nothing later can change them."
  - q: "What does panic(http.ErrAbortHandler) change for the client compared with returning?"
    options: ["Nothing; both end the response", "The connection closes without the final chunk, so the client gets a read error instead of a clean end", "The server sends a 500 trailer"]
    answer: 1
    why: "A normal return lets net/http terminate the chunked body; the abort leaves it unterminated, which every HTTP client reports as an error."
  - q: "Why does the report put summary after requirements?"
    options: ["JSON requires aggregates last", "The counts are only known after the last row has been streamed", "jsontext sorts members"]
    answer: 1
    why: "Streaming writes each requirement as it is read; the totals exist only at the end. Summary first would need a second query or a buffer."
  - q: "Why does the handler call rows.Next() before setting any header?"
    options: ["To learn the number of rows for Content-Length", "So an error in the query itself becomes a 500 problem rather than a 200 with an empty body", "pgx requires it before Scan"]
    answer: 1
    why: "pgx reports query errors with the first row. Reading it first is the last moment a status can still be chosen."
further_reading:
  - title: "package net/http: ErrAbortHandler"
    url: "https://pkg.go.dev/net/http#ErrAbortHandler"
  - title: "package net/http: ResponseController (Flush, SetWriteDeadline)"
    url: "https://pkg.go.dev/net/http#ResponseController"
  - title: "package encoding/json/jsontext (Encoder, Decoder, Token)"
    url: "https://pkg.go.dev/encoding/json/jsontext"
  - title: "package mime: FormatMediaType"
    url: "https://pkg.go.dev/mime#FormatMediaType"
  - title: "MDN: Content-Disposition"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Disposition"
  - title: "MDN: <a> download attribute"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/a"
---

## Concept

TraceLite has two downloads, both streamed:

- **`GET /api/projects/{projectId}/requirement-export`**: every requirement as CSV in
  step 7's import format, written for Excel on Turkish Windows (BOM, `;`, CRLF).
- **`GET /api/projects/{projectId}/traceability-report`**: the traceability report as one
  JSON document: each requirement in tree order with its coverage and the latest run of
  each linked test, then a summary.

Streaming means each row goes from pgx to the client as it is read: `rows.Next()`,
`Scan`, write, next. Neither the rows nor the file exist in memory as a whole, so a
project of 100,000 requirements costs the same memory as one of 10.

The price is that the status is decided before the data. A handler checks everything it
can first (the project, the first row of the query), then commits to `200`. If something
breaks later, the only honest move left is to abort the connection, so the browser
reports a failed download instead of saving a short file.

## Coming from C/C++/Java/Python/JS

- **Java/Spring**: `StreamingResponseBody` and a JDBC `ResultSet` with a fetch size are
  the same pipeline. Spring can't change the status after the first flush either.
- **Python/Django**: `StreamingHttpResponse` over a generator, with
  `queryset.iterator()` so the ORM doesn't cache every row, is the same idea; an exception
  inside the generator also leaves the client with a truncated body.
- **Node/Express**: `pipeline(cursorStream, csvStringifier, res)` streams with
  backpressure; destroying `res` on error plays the role of `ErrAbortHandler`.
- **Go with encoding/json v1**: `json.NewEncoder(w).Encode(all)` streams the bytes but
  needs `all` in memory first. jsontext writes the document piece by piece.

## How it works

The two endpoints in `openapi.yaml`:

```yaml
  /api/projects/{projectId}/requirement-export:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: exportRequirements
      responses:
        "200":
          description: Step 7's CSV format, UTF-8 with BOM, ';' and CRLF
          headers: { Content-Disposition: { schema: { type: string } } }
          content: { text/csv: { schema: { type: string } } }
        "404": { $ref: "#/components/responses/Problem" }
  /api/projects/{projectId}/traceability-report:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: getTraceabilityReport
      responses:
        "200":
          description: The report, streamed
          headers: { Content-Disposition: { schema: { type: string } } }
          content: { application/json: { schema: { $ref: "#/components/schemas/TraceabilityReport" } } }
        "404": { $ref: "#/components/responses/Problem" }
# components.schemas:
    TraceabilityReport:
      type: object
      required: [project, requirements, summary]
      properties:
        project: { type: object, required: [name], properties: { name: { type: string } } }
        requirements:
          type: array
          items:
            type: object
            required: [code, title, depth, coverage, tests]
            properties:
              code: { $ref: "#/components/schemas/Code" }
              title: { type: string }
              depth: { type: integer, minimum: 0 }
              coverage: { type: string, enum: [uncovered, failing, passing, incomplete] }
              tests:
                type: array
                items:
                  type: object
                  required: [code, result, finishedAt]
                  properties:
                    code: { $ref: "#/components/schemas/Code" }
                    result: { type: [string, "null"], enum: [passed, failed, skipped, null] }
                    finishedAt: { type: [string, "null"], format: date-time }
        summary:
          type: object
          required: [requirements, counted, failing, uncovered, incomplete, passing]
          properties:
            requirements: { type: integer }
            counted: { type: integer, description: "requirements that are not headings without tests" }
            failing: { type: integer }
            uncovered: { type: integer }
            incomplete: { type: integer }
            passing: { type: integer }
```

The order of operations in both handlers:

1. Load the project (404 if the id is malformed or unknown), start the query and read its
   first row. Any error so far is an ordinary problem response.
2. Set `Content-Type` and `Content-Disposition: attachment; filename*=utf-8''…`.
3. Write rows as they are read. csv.Writer, jsontext.Encoder and the ResponseWriter each
   buffer a few KiB and pass the bytes on whenever a buffer fills; nothing waits for the
   end of the query.
4. On success, end normally (the JSON handler writes the closing tokens and the summary
   first). On failure, log the cause and `panic(http.ErrAbortHandler)`.

The report handler is example 1's handler with the report query (example 3) in place of
the export query and `writeReport` (example 4) in place of the CSV loop. Between the two
sits a small grouping step: the query returns one row per requirement and linked test,
and a requirement's item is complete when the next row's code differs, so only one
requirement's rows are in memory at a time.

In the browser the download is a plain link, same origin through the Vite proxy (step 1),
so no JavaScript holds the file and the response's file name wins:

```tsx
<a href={`/api/projects/${projectId}/requirement-export`} download>CSV indir</a>
```

These responses are files, not API data: they don't go through step 5's `request()`,
whose zod parse would need the whole body in memory.

## Design notes

**Stream, or generate a file?** Streaming needs no storage and starts at once, but a
failure costs the whole download and a slow client holds a database connection. The
alternative is step 11's job table: generate the report into a file in the background,
then serve the finished file with a real `Content-Length` and resumable range requests.
TraceLite streams the CSV export (fast, read once) and moves the report to a job when it
gets slow.

**No explicit Flush.** The buffers already pass bytes on as they fill, so a large export
leaves the server continuously. `http.NewResponseController(w).Flush()` would only push
out the last few KiB earlier, which matters for a query that produces rows slowly;
flushing after every row would cost a chunk header and a system call per row.

**One query, read once.** Each export is a single query, so its rows come from one
snapshot and can't mix data from before and after a concurrent edit. Paging through the
API's keyset pages would release the connection between pages, at the price of a report
that may straddle an edit. For exports made of plain columns, PostgreSQL's
`COPY (query) TO STDOUT WITH (FORMAT csv)` through `PgConn().CopyTo` would skip
the Go per-row code entirely; TraceLite writes the CSV in Go because the BOM, the
separator and the step 7 column order are decisions it wants in one readable place.

**Aborting is the contract.** HTTP/1.1 chunked transfer and HTTP/2 both let a client tell
a finished body from a broken one, but only if the server breaks it. A trailer with a
checksum, or a final "end" record, would also make truncation visible, and only to
clients that check; an aborted connection is noticed by every client, including a
browser's download manager.

**Formulas in cells.** Excel turns a CSV cell that starts with `=`, `+`, `-` or `@` into
a formula when the rest parses as one, quoted or not. A requirement titled
`=HYPERLINK(...)` becomes a live link in the exported file, and one titled `- madde`
shows `#NAME?`. For files meant for reading, OWASP's
[CSV Injection](https://owasp.org/www-community/attacks/CSV_Injection) page recommends,
for Excel, a tab at the start of such a quoted field. A `'` prefix, the older advice, shows
in the cell, and OWASP calls it "not reliable in Microsoft Excel after saving and
re-opening the CSV file". Either prefix becomes part of the value. TraceLite's export uses
the tab, for the two free-text columns (codes, parents and statuses can't start with these
characters: their CHECKs and the import's rules allow letters first):

```go
// excelText is OWASP's advice for Excel: a tab before a leading =, +, - or @ keeps
// the cell text. Step 7's import trims it again with the other whitespace.
func excelText(s string) string {
	if strings.IndexAny(s, "=+-@") == 0 {
		return "\t" + s
	}
	return s
}

err = out.Write([]string{code, excelText(title), parent, status, excelText(description)})
```

csv.Writer quotes a field that starts with whitespace, so the tab lands inside the quoted
field as OWASP describes: `- madde` goes out as `"\t- madde"`. The trade-off: the file no
longer holds such a title byte for byte, and a program other than step 7's import has to
trim the tab. In return a title survives an edit in Excel. Without the tab, Excel reads
`-1+1` as a formula and saving as CSV writes its result, `0`; with it, Excel keeps the
cell as text and writes the tab back, and step 7's `strings.TrimSpace` restores the title.
A `'` prefix would come back as part of the title. Checked with Go 1.27.4 on 2026-11-14:
`=HYPERLINK(...)`, `- madde`, `+90 …` and `@ali` gained the tab, `a=b` and ordinary titles
didn't, and every title came back unchanged after `TrimSpace`.

Observed on 2026-11-14 by opening a CSV in Excel for Microsoft 365 (build
16.0.20527.20196): `=1+1`, `+1+1`, `-1+1` and `@SUM(1,1)` became formulas, also inside
double quotes; `-5` stayed a number and `+90 532 000 00 00` and `@ali` stayed text. The
tab-prefixed and `'`-prefixed cells stayed text, and still did after one save as "CSV
UTF-8" and a reopen.
