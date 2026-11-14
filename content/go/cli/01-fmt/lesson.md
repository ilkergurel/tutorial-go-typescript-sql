---
id: go.cli.fmt
title: "fmt verbs, tabwriter tables, stdout vs stderr, exit codes"
track: go
module: cli
order: 1
level: basic
minutes: 12
since: go1.12
prerequisites: [go.basics.strings-runes, go.interfaces.implicit]
compare_with: [ts.node.cli-server, go.basics.defer]
tags: [fmt, tabwriter, stdout, stderr, exit-codes, cli]
summary: >
  fmt's verbs format any value, and widths count runes, not bytes; text/tabwriter
  aligns columns for you. A tool writes results to stdout, diagnostics to stderr,
  and reports failure through an exit code set in exactly one place.
examples:
  - id: verbs
    file: examples/01_verbs.go
    title: The verbs you actually use, and what a mistake prints
    runtime: go
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          A String method makes the type a fmt.Stringer. Inside it, convert to
          the underlying type before formatting: Sprintf("%v", c) would find the
          String method again and recurse until the stack overflows.
      - lines: "15"
        note: >
          %+v adds field names, the everyday debugging verb. %#v prints Go syntax
          you can paste into a test. %T names the dynamic type, which is what you
          want when an interface holds something unexpected.
      - lines: "17-18"
        note: >
          %v and %s call String. %.2f does not: fmt uses String only for the
          string-like verbs (%v %s %q %x %X), so a numeric verb formats the
          underlying float64.
      - lines: "20-22"
        note: >
          %q quotes and escapes, %+q also escapes everything non-ASCII. % x shows
          that İ is the two bytes c4 b0. Width 8 pads "İzmir" with three spaces:
          fmt counts runes (5), not bytes (len is 6), unlike C's printf.
      - lines: "25"
        note: >
          fmt prints maps in key-sorted order (since Go 1.12), so the output is
          stable and testable. Ranging over a map is still deliberately random.
      - lines: "27-30"
        note: >
          A wrong verb, a missing argument and an extra argument all compile and
          run; the output carries a %! marker. go vet's printf check reports all
          three lines, and go test runs that check automatically.
  - id: tabwriter
    file: examples/02_tabwriter.go
    title: Padding by bytes, by runes, and with text/tabwriter
    runtime: go
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          Padding computed with len counts bytes. İ, ğ and ı take two bytes each,
          so the İstanbul and Iğdır rows come out one and two columns short.
      - lines: "19-22"
        note: >
          A fmt width counts runes, so a fixed %-10s lines up, as long as you
          know the widest value in advance.
      - lines: "26-31"
        note: >
          tabwriter buffers tab-terminated cells, measures every column, then
          pads. The last cell of each line has no tab after it, so it is not
          padded and no line ends in spaces. Forget Flush and nothing is printed.
  - id: exit-codes
    file: examples/03_exit_codes.go
    title: "run() returns the exit code; main is one line"
    runtime: go
    check: exact
    annotations:
      - lines: "13"
        note: >
          The program takes its arguments and writers as parameters and returns
          an int. Nothing inside touches os.Args, os.Stdout or os.Exit, so a test
          calls it with buffers and checks all three outputs.
      - lines: "14-17"
        note: >
          Usage errors go to stderr with code 2, the code flag.ExitOnError uses.
          An unrecovered panic also exits with 2, and log.Fatal with 1.
      - lines: "22-23"
        note: >
          The "tool: cause" prefix tells the user which program in a pipeline
          failed. stdout stays empty, so x=$(sum 40 two) captures nothing
          instead of an error message.
      - lines: "42-43"
        note: >
          The stderr line appears in the run panel but not in the compared
          output, exactly as a pipe would skip it. os.Exit sits in main, after
          run has returned, so every defer inside run (closing files, flushing a
          bufio.Writer) has already happened. Here the code is 0.
use_cases:
  - "list and status commands that print aligned columns with text/tabwriter"
  - "%q around user input in error messages, so empty strings, trailing spaces and invisible characters show up"
  - "%+v and %T while debugging; %#v to turn a value you printed into a test fixture"
  - "A run(args, stdout, stderr) int entry point that table-driven tests call directly"
strengths:
  - "Widths and precision count runes, so %-10s aligns İstanbul and Iğdır; C's printf counts bytes and misaligns them"
  - "%v prints any value, nested structs, slices and maps included, with no toString to write as in Java; maps come out key-sorted"
  - "Format mistakes never crash or corrupt memory as they can in C: they print a %! marker, and go vet (also run by go test) flags them first"
  - "Aligned tables need no dependency: text/tabwriter is in the standard library, where Python reaches for third-party tabulate"
weaknesses:
  - "The compiler accepts any verb for any argument; only go vet catches %d with a string, so skip vet and the mistake ships"
  - "tabwriter assumes every rune is one column wide: emoji, East Asian wide characters and combining accents misalign. The package is frozen"
  - "os.Stdout has no buffer: each Printf is a write system call. A tool printing 100,000 lines needs a bufio.Writer, which C stdio and Python give you for free when stdout is a pipe"
pitfalls:
  - title: "Errors and progress on stdout"
    detail: "Anything on stdout becomes input for the next program in the pipe. Results go to stdout; usage, warnings, progress and errors go to stderr."
    bad: "fmt.Println(\"error:\", err)"
    good: "fmt.Fprintf(os.Stderr, \"mytool: %v\\n\", err)"
  - title: "os.Exit or log.Fatal deep in the code"
    detail: "Both skip deferred calls, so a deferred bufio.Writer.Flush never runs and the last block of output is lost. Return errors up to run and exit only in main."
  - title: "Forgetting tabwriter's Flush"
    detail: "The writer cannot pad a column before it has seen every row, so it holds everything until Flush. No Flush, no table."
  - title: "Exit codes outside 0-125"
    detail: "os.Exit's docs ask for [0, 125] for portability: shells use 126 and 127 themselves and report a process killed by a signal as a status above 128."
quiz:
  - q: "What does fmt.Printf(\"%d\\n\", \"42\") do?"
    options: ["Fails to compile", "Panics", "Prints %!d(string=42)", "Prints 42"]
    answer: 2
    why: "fmt reports bad verbs in its output instead of failing. go vet's printf check would have flagged the call."
  - q: "fmt.Printf(\"[%-8s]\", \"İzmir\"). How many spaces follow İzmir?"
    options: ["2", "3", "It depends on the terminal"]
    answer: 1
    why: "fmt widths count runes: İzmir is 5 runes (6 bytes), so 3 spaces pad it to 8."
  - q: "Your tool prints \"processing 3 files...\" while it works. Where should that line go?"
    options: ["stdout", "stderr", "Either; it makes no difference"]
    answer: 1
    why: "It is a diagnostic. On stdout it would end up in the file or the next command of a pipeline."
  - q: "A helper three calls deep finds a bad argument. What is the idiomatic way to end with exit code 2?"
    options: ["os.Exit(2) in the helper", "log.Fatal in the helper", "Return an error up to run, which returns 2 to main", "panic"]
    answer: 2
    why: "os.Exit and log.Fatal skip deferred cleanup, and a panic dumps a stack trace on the user. Exiting only in main keeps defers running and run testable."
further_reading:
  - title: "package fmt"
    url: "https://pkg.go.dev/fmt"
  - title: "package text/tabwriter"
    url: "https://pkg.go.dev/text/tabwriter"
  - title: "package os: Exit"
    url: "https://pkg.go.dev/os#Exit"
  - title: "go vet printf check"
    url: "https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/printf"
---

## Concept

A command-line tool talks through three channels. **stdout** carries the result: it is
what `|`, `>` and `$(...)` capture. **stderr** carries everything meant for the human
(usage, progress, warnings, errors); pipes pass it by. The **exit code** is what shells,
`&&`, Makefiles and CI read: 0 is success, anything else is failure.

`fmt` formats any value through verbs: `%v` for the default form, `%+v` and `%#v` for
more detail, `%q` for quoted strings, `%d`, `%x`, `%f` with width and precision flags.
Widths count runes, not bytes, which matters as soon as a Turkish name meets a column.
`text/tabwriter` computes column widths from the data, so tables align without you
measuring anything.

## Coming from C/C++/Java/Python/JS

- C: the verbs look familiar, but there are no length modifiers (`%ld`, `%zu`); fmt
  knows each argument's type at run time. A mismatch prints a marker instead of
  undefined behaviour. Unlike stdio, `os.Stdout` is never buffered by Go.
- Java: `String.format` ≈ `fmt.Sprintf`, `toString()` ≈ a `String() string` method,
  `System.err` ≈ `os.Stderr`, `System.exit` ≈ `os.Exit` (both skip cleanup).
- Python: `print(..., file=sys.stderr)` ≈ `fmt.Fprintln(os.Stderr, ...)`, `repr()` ≈
  `%q`/`%#v`. `sys.exit(2)` raises `SystemExit`, so `finally` blocks run; `os.Exit` does
  not run deferred calls.
- Node: `console.log`/`console.error` map to stdout/stderr. Setting `process.exitCode`
  and returning is the same idea as `run` returning a code; `process.exit()` is `os.Exit`.

## Design notes

Keep `main` to one line, `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`, and put the
program in `run`. Three things follow: deferred cleanup inside `run` always happens
before the process exits, tests drive the whole tool with `strings.Builder`s, and the
exit code is decided in exactly one place. Use 1 for "it failed" and 2 for "you called
it wrong", the code the `flag` package already uses.

When a tool prints a lot, wrap stdout once in `bufio.NewWriter` inside `run` and
`defer` its `Flush`; that is safe precisely because `run` returns normally. Write
errors as `tool: what failed: cause`, with `%q` around anything the user typed, so an
empty argument or a trailing space is visible in the message.
