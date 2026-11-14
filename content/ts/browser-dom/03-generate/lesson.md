---
id: ts.browser-dom.generate
title: "Generating and downloading files (Blob + object URL); drag and drop; clipboard"
track: ts
module: browser-dom
order: 3
level: intermediate
minutes: 14
since: es2021
prerequisites: [ts.browser-dom.dom, ts.browser-dom.reading-files]
compare_with: [go.files-io.csv-json, sql.files-movement.copy, fullstack.tracelite.step-08-export]
tags: [files, blob, csv, download, drag-and-drop, clipboard, intl]
summary: >
  Build the content, wrap it in a Blob, and click an <a download> pointing at
  an object URL: that is a file export with no server. For Turkish Excel, write
  semicolons, decimal commas and a UTF-8 BOM, sorted with Intl.Collator("tr").
examples:
  - id: build-csv
    file: examples/01_build_csv.ts
    title: "A CSV writer for Turkish Excel, with Turkish sorting"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-15"
        note: >
          Three rows need quoting: a quote inside a note, a semicolon, and a line
          break. Real exports always contain them eventually.
      - lines: "17-21"
        note: >
          A column is a header plus an accessor. Generic over the row type, so
          toCsv works for any T and each accessor is checked against Order.
      - lines: "23-24"
        note: >
          With Turkish settings the comma is the decimal mark, so Excel splits
          fields on semicolons. Numbers use the same convention: 1234,5 with no
          thousands separator.
      - lines: "26-31"
        note: >
          The RFC 4180 rules with ; as the separator: quote a field that holds
          the separator, a quote or a line break, and double the quotes inside.
          Go's encoding/csv applies the same rules for you.
      - lines: "33-39"
        note: >
          RFC 4180 ends lines with CRLF. The header row goes through cell() too,
          because header text can contain the separator as well.
      - lines: "41-46"
        note: >
          Plain sort() compares UTF-16 code units, so Ç and Ş land after Z.
          Intl.Collator("tr") follows the Turkish alphabet: Ayşe, Çağla, Şule,
          Zeynep.
      - lines: "54"
        note: >
          Only for display: the ⏎ marks each CRLF so the line endings are visible
          in the output. Line 38 is what goes into the file.
  - id: download-blob
    file: examples/02_download_blob.ts
    title: "Blob, object URL, <a download>, revoke"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-14"
        note: >
          The whole download recipe: an object URL for the Blob, a temporary
          link with a download attribute, a click from code, then cleanup.
      - lines: "7"
        note: >
          download works for same-origin, blob: and data: URLs. The name is a
          suggestion: browsers replace / and \ with _ and may adjust the rest.
      - lines: "11-13"
        note: >
          An object URL keeps its Blob alive until revoked. A page that exports
          often and never revokes holds every export in memory.
      - lines: "16-23"
        note: >
          Sandbox stand-in: the lesson iframe has no allow-downloads, so this
          listener cancels the click and reports what would be saved.
      - lines: "25-27"
        note: >
          The BOM goes first, as its own part. The charset in the type matters
          when the Blob is opened in a browser tab, not to Excel.
      - lines: "29-34"
        note: >
          37 UTF-16 code units become 45 bytes: Blob encodes strings as UTF-8,
          like a Go string. text() decodes as UTF-8 and drops a leading BOM, so
          check the bytes, not the text, when you test an export.
  - id: drop-clipboard
    file: examples/03_drop_clipboard.ts
    title: "A drop zone for CSV files, a copy handler, and the async Clipboard API"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          Elements reject drops by default. Cancelling dragover is what makes
          this one a drop target; without it, drop never fires.
      - lines: "11-14"
        note: >
          Cancel drop too, or the browser may open the dropped file itself.
          dataTransfer.files holds File objects, the same type an input gives
          you. Filter by name: file.type is guessed from the extension and is
          often empty for CSV.
      - lines: "15-23"
        note: >
          dispatchEvent and the browser call listeners synchronously and never
          await them. Keep the promise if anything must wait for the reading.
      - lines: "26-37"
        note: >
          Synthetic events (isTrusted is false) run the same listeners, which is
          how this example and UI tests simulate a drop. dispatchEvent returns
          false when a listener cancelled the event.
      - lines: "39-53"
        note: >
          A copy listener can replace what gets copied: setData, then
          preventDefault. Tab-separated rows paste into Excel as separate
          cells; semicolons don't split there.
      - lines: "55-61"
        note: >
          navigator.clipboard.writeText is the modern API, but it needs a user
          gesture or a permission, and in an iframe a permissions policy too.
          Here it rejects with NotAllowedError, which real code must handle.
use_cases:
  - "Exporting the table the user is looking at, filtered and sorted, as CSV for Excel"
  - "Saving generated JSON, SVG or text reports without a server endpoint"
  - "Drop zones for CSV import, read with File.text() or stream() as in the reading-files lesson"
  - "Copy buttons and copy handlers that put spreadsheet-friendly, tab-separated text on the clipboard"
strengths:
  - "A download takes three lines and no server: where a Go handler sets Content-Disposition, the page clicks an <a download>"
  - "Blob encodes strings as UTF-8 and reports the exact byte size, so a BOM or a size limit is easy to check before saving"
  - "File inputs, drag and drop and paste all deliver File objects, so one import function serves all three"
weaknesses:
  - "The whole export is built in memory; for very large data, stream it from the server instead"
  - "The browser decides the download UX: it may prompt for a location or adjust the file name"
  - "Clipboard writes need a user gesture or permission, and synthetic events never count as one"
  - "There is no CSV writer in the platform: quoting and number formats are yours to get right"
pitfalls:
  - title: "Comma-separated CSV for a Turkish Excel user"
    detail: "With Turkish regional settings Excel splits on semicolons, so a comma-separated file opens as one column. Choose the separator per audience."
    bad: "rows.map((r) => r.join(\",\")).join(\"\\n\")"
    good: "rows.map((r) => r.map(cell).join(\";\") + \"\\r\\n\").join(\"\")"
  - title: "Forgetting the BOM"
    detail: "Without it, Excel may read the file in the Windows code page and show Ã¼ for ü. Prepend \"\\uFEFF\" for Excel-bound files only; other tools may keep it as part of the first header."
  - title: "Never revoking object URLs"
    detail: "Each URL keeps its Blob alive for the life of the document. Revoke once the download has started."
  - title: "Not cancelling dragover"
    detail: "drop never fires on an element that didn't cancel dragover, and an uncancelled drop lets the browser open the file."
  - title: "Sorting Turkish text with sort() or the en collator"
    detail: "Code-unit order puts Ç, Ö, Ş, İ after Z. new Intl.Collator(\"tr\").compare follows the Turkish alphabet, with I before İ."
quiz:
  - q: "Your drop zone never receives drop events. What is missing?"
    options: ["draggable=\"true\" on the zone", "preventDefault() in a dragover listener", "A file input inside the zone"]
    answer: 1
    why: "Elements reject drops by default; cancelling dragover is what makes one a drop target."
  - q: "A Blob was built from [\"\\uFEFF\", csv]. Does (await blob.text()).startsWith(\"\\uFEFF\") return true?"
    options: ["Yes", "No: text() decodes as UTF-8 and strips a leading BOM"]
    answer: 1
    why: "UTF-8 decoding removes the BOM. Read the first bytes with arrayBuffer() to confirm it is there."
  - q: "With ; as the separator, how must the field zil bozuk; arayın be written?"
    options: ["As is", "Wrapped in double quotes", "With the ; replaced by ,"]
    answer: 1
    why: "A field that contains the separator, a quote or a line break must be quoted; inner quotes are doubled."
  - q: "How should Çağla and Zeynep be ordered for a Turkish reader, and what produces that order?"
    options: ["Zeynep, Çağla: plain sort()", "Çağla, Zeynep: new Intl.Collator(\"tr\").compare"]
    answer: 1
    why: "Ç comes right after C in the Turkish alphabet; sort() compares code units and puts U+00C7 after Z."
further_reading:
  - title: "MDN: Blob"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Blob"
  - title: "MDN: URL.createObjectURL()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/URL/createObjectURL_static"
  - title: "MDN: <a> element, download attribute"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/a"
  - title: "MDN: File drag and drop"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/HTML_Drag_and_Drop_API/File_drag_and_drop"
  - title: "MDN: Clipboard API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API"
  - title: "MDN: Intl.Collator"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/Collator"
---

## Concept

A browser-side export has three steps: build the content, wrap it in a `Blob` with a
media type, and hand it to the browser through an **object URL** on an `<a download>` that
your code clicks. The browser then saves it like any other download. Nothing is uploaded.

CSV meant for a Turkish user's Excel needs a few deliberate choices: `;` as the separator,
a decimal comma, RFC 4180 quoting, UTF-8 with a byte order mark, and rows sorted with
`Intl.Collator("tr")`. Lines end in CRLF, as RFC 4180 and Excel's own CSV files have them,
though Excel reads LF too.

Microsoft documents the separator: the list separator comes from the Windows region
settings, and a comma decimal separator "forces Excel to use a semi-colon for the list
separator"
([Import or export text files](https://support.microsoft.com/en-us/office/import-or-export-text-txt-or-csv-files-5250ac4c-663c-47ce-937b-339e391393ba)).
It also says a UTF-8 CSV opens normally "if it was saved with BOM"
([Opening CSV UTF-8 files correctly](https://support.microsoft.com/en-us/office/opening-csv-utf-8-files-correctly-in-excel-8a935af5-3416-4edd-ba7e-3dfd2bc4a032)).
The rest was observed on 2026-11-14 with Excel for Microsoft 365 (build 16.0.20527.20196)
on an en-US Windows: double-clicked, a UTF-8 file without a BOM showed `Müşteri` as
`MÃ¼ÅŸteri` (its bytes read as windows-1252), and a BOM file with LF line ends opened
correctly. Tab-separated text pasted from the clipboard landed in separate cells, while
`;`-separated text stayed in one; the pasted `12,5` became the number 12.5 only with
Excel's decimal separator set to `,`.

The other direction reuses `File`. A drop hands you a `DataTransfer` whose `files` are the
same objects a file input gives you. The clipboard has two APIs: the synchronous `copy`,
`cut` and `paste` events, whose handlers read or replace `clipboardData`, and the
asynchronous `navigator.clipboard`, which needs a user gesture or a permission.

## Coming from C/C++/Java/Python/JS

- **Go**: `encoding/csv` with `w.Comma = ';'` and `w.UseCRLF = true` does the quoting for
  you; the browser has no CSV writer, so `cell()` in example 1 is the whole library. On a
  server, `Content-Disposition: attachment; filename=...` does what `<a download>` does.
- **Python**: `csv.writer(f, delimiter=";")` with `open(path, "w", encoding="utf-8-sig",
  newline="")` writes the BOM for you.
- **Java**: `OutputStreamWriter` with UTF-8 never writes a BOM; you write `\uFEFF`
  yourself, exactly as here.
- **JS/Node**: `fs.writeFile` takes a path. A web page never gets one: it can only offer a
  download, or ask the user for a file with `showSaveFilePicker()`, which only Chromium
  browsers implement.

## How it works

`URL.createObjectURL(blob)` registers the Blob in the document under a string such as
`blob:https://example.com/4f0c...`. As long as that URL is registered, the Blob can't be
garbage-collected; `revokeObjectURL` removes the entry. A `data:` URL is the alternative
for tiny files: the content is inside the URL string itself, so there is nothing to revoke,
but it grows with the content.

`new Blob(parts)` encodes string parts as UTF-8 and concatenates the bytes. That is why
`"\uFEFF"` becomes `EF BB BF`, and why `blob.size` counts bytes while `string.length`
counts UTF-16 code units.

## Design notes

Keep format decisions in one profile per audience. "CSV for Excel in Turkey" means
semicolons, decimal commas and a BOM. "CSV for machines", such as a Postgres `COPY ... CSV`
or a Go importer, means commas, decimal points and no BOM. Mixing the two is how a
`1234,5` turns into two columns.

Exported user text can contain cells starting with `=`, `+`, `-` or `@`. Excel turns such
a cell into a formula when the rest parses as one, quoted or not, so `=HYPERLINK(...)`
from another user becomes a live link. LibreOffice Calc's text import does it only with
its "Evaluate formulas" option checked
([LibreOffice help](https://help.libreoffice.org/latest/en-US/text/shared/00/00000208.html)).
When the data comes from other users, OWASP's
[CSV Injection](https://owasp.org/www-community/attacks/CSV_Injection) page recommends, for
Excel, a tab character at the start of such a field, inside its quotes. The older advice, a
single quote, shows in the cell, and OWASP calls it "not reliable in Microsoft Excel after
saving and re-opening the CSV file". Either prefix becomes part of the value, so a program
that reads the file back has to strip it again.

Observed on 2026-11-14 by opening a CSV in Excel for Microsoft 365 (build
16.0.20527.20196): `=1+1`, `+1+1`, `-1+1` and `@SUM(1,1)` became formulas, also inside
double quotes; `-5` stayed a number and `+90 532 000 00 00` stayed text. The tab-prefixed
and quote-prefixed cells stayed text, and still did after one save as "CSV UTF-8" and a
reopen.

Build exports in the browser when the data is already on screen and modest in size. For
large or sensitive exports, stream from the server instead: a Go handler writing through
`csv.Writer` straight to the response never holds the whole file in memory (the TraceLite
export step).
