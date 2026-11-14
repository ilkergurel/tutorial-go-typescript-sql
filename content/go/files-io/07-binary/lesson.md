---
id: go.files-io.binary
title: "Binary data, byte order, gzip and zip"
track: go
module: files-io
order: 7
level: advanced
minutes: 15
since: go1.23
prerequisites: [go.files-io.composing, go.files-io.directories, go.basics.types-conversion]
compare_with: [ts.browser-dom.reading-files, go.services.tcp-exec, ts.node.fs-streams]
tags: [binary, encoding-binary, endianness, gzip, zip, compression, security]
summary: >
  encoding/binary turns fixed-size integers and structs into bytes in an
  explicit byte order. gzip streams are an io.Writer and io.Reader you stack
  on files; zip archives need random access and untrusted entry names.
examples:
  - id: byte-order
    file: examples/01_byte_order.go
    title: Decoding a sensor packet by offsets and with binary.Read
    runtime: go
    check: exact
    annotations:
      - lines: "13-20"
        note: >
          Only sized types: uint8, int16, uint32, [2]byte. A plain int has no
          fixed size, and binary.Read/Write reject it with "some values are
          not fixed-sized".
      - lines: "26-27"
        note: >
          The same four bytes are 123456 or 1088553216 depending on the order
          you read them in. Network protocols and Modbus registers are
          big-endian, plenty of device protocols are little-endian, and x86
          and ARM machines are little-endian in practice. The order is a
          property of the protocol, never of the machine.
      - lines: "29-34"
        note: >
          binary.Read fills the fields in declaration order and never inserts
          padding, so binary.Size is the wire size: 12 here. A C struct with a
          uint8 followed by a uint32 is usually 8 bytes in memory; through
          encoding/binary it is 5.
      - lines: "35-36"
        note: >
          0xFF83 read into an int16 is -125: two's complement comes for free
          when the field has a signed type. Scaled integers (tenths of a
          degree) are how devices avoid floats on the wire.
      - lines: "38-44"
        note: >
          binary.Append (Go 1.23) appends the encoding to a slice; Encode and
          Decode work on a byte slice you already have. Before 1.23 you
          wrote into a bytes.Buffer with binary.Write.
      - lines: "46-48"
        note: >
          A packet cut short returns io.ErrUnexpectedEOF; io.EOF means no
          bytes at all. On a TCP stream that's how you tell "connection
          closed between packets" from "closed in the middle of one".
      - lines: "50-53"
        note: >
          A uvarint uses 7 bits per byte and the top bit as "more follows", so
          5 takes one byte and 2^32 takes five. Signed values use Varint, which
          zigzag-encodes them so small negative numbers stay short.
  - id: gzip
    file: examples/02_gzip.go
    title: gzip as a Writer and a Reader, its Latin-1 header, and Close
    runtime: go
    check: exact
    annotations:
      - lines: "21-26"
        note: >
          gzip.Writer wraps the file; fmt.Fprintf doesn't know it is
          compressing. Close flushes the compressor and writes the footer, a
          CRC-32 and the length; closing the file does neither.
      - lines: "38-49"
        note: >
          gzip.Reader is an io.Reader, so the Scanner from the reading-files
          lesson works unchanged and memory stays flat. The CRC is checked
          when the stream ends, so a damaged file fails at sc.Err, after
          you've already seen its lines: treat data as tentative until EOF.
      - lines: "52-55"
        note: >
          The gzip header's name and comment must fit in Latin-1 (U+0001 to
          U+00FF). ü and ç fit; Ş, ş, Ğ, ğ, İ and ı don't, so a Turkish file
          name in the header fails at the first Write. Leave Name empty or
          transliterate it.
      - lines: "57-63"
        note: >
          Flush makes the data so far decodable (useful on a network stream),
          but without Close there's no footer: the reader returns every byte
          and then io.ErrUnexpectedEOF.
  - id: zip
    file: examples/03_zip.go
    title: Writing a zip, reading raw entries, extracting safely with CopyFS
    runtime: go
    check: exact
    annotations:
      - lines: "17-24"
        note: >
          Create compresses with Deflate and returns an io.Writer for that one
          entry; finish writing it before the next Create. Nothing checks the
          name, so this archive really contains ../../evil.txt, the classic
          "zip slip" payload.
      - lines: "25-30"
        note: >
          The central directory, the index of all entries, lives at the end of
          the file and is written by zw.Close. Without it the archive can't be
          opened. Then close the file and check that error too.
      - lines: "32-40"
        note: >
          zip.OpenReader needs random access because it reads that index
          first. zr.File holds names as stored: Go sets the UTF-8 flag (bit
          0x800) for Şubat.csv automatically, and the hostile name is there
          verbatim. Joining zf.Name onto a target directory yourself is the
          vulnerability.
      - lines: "42-46"
        note: >
          zip.Reader is also an fs.FS whose listing cleans stored names: a
          leading / and leading ../ are dropped, so ../../evil.txt appears as
          evil.txt while zf.Name above keeps the original.
          os.CopyFS (Go 1.23) extracts through that view and won't overwrite
          existing files.
use_cases:
  - "Device and industrial protocols over serial or TCP: fixed headers, big-endian fields, scaled integers"
  - "File formats with binary headers: PNG chunks, WAV, custom record files"
  - "Compressed exports and logs (.csv.gz) streamed to disk, S3 or an HTTP response"
  - "Zip uploads and downloads: bundling several reports, importing an archive a user sends"
strengths:
  - "Byte order is an explicit argument, so you can't accidentally depend on the machine's endianness the way a C memcpy into a struct does"
  - "gzip, flate and zlib are plain io.Writer/io.Reader wrappers, so csv.NewReader(zr) or json.NewDecoder(zr) works directly; Java needs an InputStreamReader between GZIPInputStream and a text parser because bytes and characters are separate hierarchies"
  - "zip.Reader is an fs.FS: fs.WalkDir, http.FileServerFS and os.CopyFS work on an archive without extracting it"
weaknesses:
  - "binary.Read/Write use reflection; for hot paths decode by offsets with BigEndian.Uint32 and friends"
  - "Compressed bytes are not stable across Go releases (Go 1.27 changed the flate encoder), so never hash or compare compressed output in tests"
  - "zip needs io.ReaderAt and the total size, so it can't be read from a pipe or a streaming HTTP body; use archive/tar plus gzip for true streams"
pitfalls:
  - title: "Extracting with filepath.Join(dir, zf.Name)"
    detail: "Entry names are attacker-controlled: ../../ climbs out of dir. Extract through the fs.FS view (os.CopyFS), or check filepath.IsLocal and use os.Root. GODEBUG=zipinsecurepath=0 makes OpenReader report such names as ErrInsecurePath."
    bad: "dst := filepath.Join(dir, zf.Name) // ../../etc/cron.d/x"
    good: "err := os.CopyFS(dir, zr) // names come from the cleaned fs.FS view"
  - title: "Trusting declared sizes"
    detail: "UncompressedSize64 is a number from the archive; a zip bomb lies or simply expands to gigabytes. Wrap each entry in io.LimitReader and enforce a total budget."
  - title: "Forgetting Close on gzip.Writer or zip.Writer"
    detail: "gzip loses its footer and readers report unexpected EOF; zip loses its central directory and can't be opened at all."
  - title: "Using int in binary structs"
    detail: "int is 32 or 64 bits depending on the platform, so encoding/binary refuses it. Use int32, uint16 and the like, matching the protocol spec."
quiz:
  - q: "The bytes 00 01 E2 40 are documented as a big-endian uint32. What does binary.LittleEndian.Uint32 give?"
    options: ["123456", "1088553216", "An error"]
    answer: 1
    why: "Byte order is a reading convention, not something the bytes carry; reading big-endian data little-endian silently gives a different number."
  - q: "You wrote a .csv.gz with gzip.NewWriter(f), called zw.Flush() and f.Close(). What does a reader see?"
    options: ["The full file, no error", "All the data, then io.ErrUnexpectedEOF", "Nothing: gzip.NewReader fails"]
    answer: 1
    why: "Flush emits the compressed data; only Close writes the footer with the CRC and length."
  - q: "Why can't you read a zip archive directly from an HTTP response body?"
    options: ["zip is not an io.Reader format; it needs io.ReaderAt and the size, because the index is at the end", "HTTP bodies are always gzip-encoded", "zip.Reader only opens files by name"]
    answer: 0
    why: "The central directory sits at the end of the file. Save the body to a temp file (or buffer it within a size limit) first, or use tar.gz for streaming."
  - q: "A gzip header Name of \"Şubat.csv\" makes the first Write fail. Why?"
    options: ["gzip headers are ASCII only", "Header strings must be in Latin-1, and Ş is outside it", "Names must end in .gz"]
    answer: 1
    why: "RFC 1952 stores the name in ISO 8859-1; Go accepts U+0001 to U+00FF and rejects the rest."
further_reading:
  - title: "package encoding/binary"
    url: "https://pkg.go.dev/encoding/binary"
  - title: "package compress/gzip"
    url: "https://pkg.go.dev/compress/gzip"
  - title: "package archive/zip"
    url: "https://pkg.go.dev/archive/zip"
  - title: "Go 1.27 release notes: compress/flate output changes"
    url: "https://go.dev/doc/go1.27"
---

## Concept

- **encoding/binary** converts between Go values and bytes. `binary.BigEndian` and
  `binary.LittleEndian` read and write `uint16/32/64` at given offsets (`Uint32(b)`,
  `PutUint32(b, v)`, `AppendUint32`). `binary.Read`, `Write`, and since Go 1.23 `Append`,
  `Encode` and `Decode`, handle whole fixed-size structs and slices, packed field by field.
  `Uvarint`/`Varint` give the variable-length integers protobuf also uses.
- **compress/gzip** wraps an `io.Writer` (`gzip.NewWriter`) or `io.Reader`
  (`gzip.NewReader`); everything you know about streams applies. `Close` finishes the file.
- **archive/zip** reads and writes multi-file archives. Reading needs random access
  (`zip.OpenReader` or `zip.NewReader(io.ReaderAt, size)`); entries open as readers, and the
  archive is also an `fs.FS`.

## Coming from C/C++/Java/Python/JS

- C/C++: instead of casting a buffer to a packed struct (and fighting alignment,
  endianness and strict aliasing), you name the byte order and the decoder copies field by
  field. `binary.BigEndian.Uint32` is `ntohl` plus the load.
- Java: `ByteBuffer.order(ByteOrder.BIG_ENDIAN).getInt()` ≈ `BigEndian.Uint32`;
  `GZIPOutputStream` ≈ `gzip.Writer`, but in Go the same `io.Writer` interface covers it, no
  separate stream class tree. `ZipFile` ≈ `zip.OpenReader`.
- Python: `struct.unpack(">2sBBIhH", pkt)` is exactly the `Reading` struct (`>` =
  big-endian); `gzip.open` and `zipfile.ZipFile` ≈ these packages. Python's
  `ZipFile.extractall` sanitizes names; Go's equivalent is extracting through the `fs.FS`
  view with `os.CopyFS`.
- JS: `DataView.getUint32(offset, littleEndian)` takes the byte order as a flag too;
  `DecompressionStream("gzip")` ≈ `gzip.NewReader`. There's no zip in the platform.

## How it works

A gzip file is a 10-byte header (optionally followed by a name and a comment in ISO
8859-1), a DEFLATE stream, and an 8-byte footer with the CRC-32 and the uncompressed length. The reader
decompresses as you read and checks the footer at the end, which is why errors in a
damaged file arrive last. Several gzip members can be concatenated, and `gzip.Reader` reads
through all of them unless you call `Multistream(false)`.

A zip file is a sequence of local entries (header plus compressed data) followed by the
**central directory**, an index of all entries, and an end record pointing to it. Readers
start from the end, which is why `zip` needs `io.ReaderAt` and the size, why
`zip.Writer.Close` is essential, and why an entry can be read without decompressing the
others. Bit 11 of an entry's flags marks the name as UTF-8; Go's writer sets it for
non-ASCII names, and its reader reports names from tools that used a legacy code page
through `FileHeader.NonUTF8`.

Go 1.27 changed the DEFLATE encoder for speed, so gzip, zip and PNG output from 1.27 can
differ byte for byte from 1.26. Decompressing still gives identical data; only the
compressed bytes moved, which is why these examples print contents, never compressed sizes.

## Design notes

Write protocol decoders as functions over `[]byte` (or an `io.Reader` plus
`io.ReadFull` for one frame), with the byte order and field offsets taken from the spec, and
test them against captured packets. Prefer explicit offsets for hot paths and
`binary.Read` for readability elsewhere. For compressed exports, stack writers
(`csv.Writer` → `gzip.Writer` → file or `http.ResponseWriter`; `csv.Writer` already
buffers) and finish them in the order you write through them: flush the CSV writer, close
the gzip writer, then close the file. Treat every archive from outside as hostile: extract through the
`fs.FS` view or an `os.Root`, limit bytes per entry and in total, and cap the entry count.

The archive/zip docs only say that `Reader.Open` takes paths "with no leading / or ../
elements"; they don't say how stored names are listed. That comes from the source:
`toValidName` in `archive/zip/reader.go`, called from `initFileList`, turns backslashes
into slashes, cleans the path, then strips a leading `/` and every leading `../`, while
`File.Name` keeps the name as stored. A run on Go 1.27.4 (2026-11-14) showed it: entries
`../../evil.txt` and `/abs.txt` walked as `evil.txt` and `abs.txt`, and
`Open("evil.txt")` worked. The zip example's exact output pins this.
