---
id: go.cli.flag-env
title: "flag and subcommands, environment variables, stdin"
track: go
module: cli
order: 2
level: basic
minutes: 14
since: go1.13
prerequisites: [go.cli.fmt, go.files-io.reading-files]
compare_with: [ts.node.cli-server, go.testing.table-driven]
tags: [flag, cli, subcommands, environment, stdin, os]
summary: >
  A FlagSet per command parses an explicit argument slice and returns errors
  instead of exiting; environment variables feed flag defaults, so flags win.
  stdin is just an io.Reader.
examples:
  - id: flagset
    file: examples/01_flagset.go
    title: The flag syntax rules, run against a FlagSet
    runtime: go
    check: exact
    annotations:
      - lines: "17-21"
        note: >
          flag.Parse() parses os.Args into package-level globals and, on error,
          exits with code 2. A FlagSet with ContinueOnError returns the error
          instead, so the same function serves main and a table-driven test.
      - lines: "22-24"
        note: >
          XxxVar binds a flag to a struct field. The default and the help text
          are part of the definition, and flag builds the usage message from
          them: look at the stderr pane.
      - lines: "35"
        note: >
          One dash or two are the same flag, and -name=value and -name value
          both work. Typed flags parse on the spot: 1.5s becomes a
          time.Duration, and "many" is rejected as an int.
      - lines: "36"
        note: >
          Parsing stops at the first non-flag argument, so -v after a.txt is a
          file name. GNU getopt and Python's argparse accept flags after
          positionals; Go does not.
      - lines: "37"
        note: >
          A boolean flag is switched off only with -v=false: in "-v false" the
          false would be a positional argument. "--" ends the flags, so a file
          name that starts with a dash survives.
      - lines: "44-47"
        note: >
          With ContinueOnError, flag has already printed the error and the usage
          to fs.Output() (stderr by default); the caller only picks the exit
          code. -h and -help return flag.ErrHelp, which by convention exits 0.
  - id: subcommands-env
    file: examples/02_subcommands_env.go
    title: Subcommands, and an environment variable as the flag's default
    runtime: go
    check: exact
    annotations:
      - lines: "12-17"
        note: >
          A map from subcommand name to function is all the dispatch a small
          tool needs. strings.ToUpperSpecial with a nil SpecialCase has no
          overrides, so it is plain Unicode casing; unicode.TurkishCase maps
          i to İ and ı to I.
      - lines: "26-29"
        note: >
          Start from the built-in default and replace it with the environment
          value if there is one. An empty variable counts as unset here.
      - lines: "30-32"
        note: >
          One FlagSet per subcommand, named after it, so subcommands can define
          different flags. The value so far becomes the flag's default: an
          explicit -lang overwrites the environment, and -h shows the effective
          default. SetOutput sends flag's messages to the injected stderr.
      - lines: "36-40"
        note: >
          flag checks syntax, not meaning: "de" is a valid string. Validate
          after parsing and treat a bad value as a usage error, code 2.
      - lines: "55-56"
        note: >
          A map-backed getenv lets tests run without touching the process
          environment, which is global: testing's t.Setenv refuses to run in
          parallel tests for exactly that reason.
      - lines: "64-65"
        note: >
          os.LookupEnv has exactly the parameter's type. os.Setenv changes this
          process and the children it starts later, never the shell that
          started it.
  - id: stdin
    file: examples/03_stdin.go
    title: "A tiny wc: stdin is an io.Reader"
    runtime: go
    stdin: "Merhaba dünya\nİstanbul'da yağmur var\nçay mı, kahve mi?\n"
    check: exact
    annotations:
      - lines: "14-16"
        note: >
          count never mentions os.Stdin. main passes stdin, a test passes
          strings.NewReader, and a file argument would pass an *os.File: the
          same accept-a-Reader shape as in the reading-files lesson.
      - lines: "18-19"
        note: >
          ScanLines strips "\n" and a "\r" before it, so CRLF input from Windows
          tools counts the same as LF input from Unix ones.
      - lines: "22-24"
        note: >
          chars matches wc -m and bytes matches wc -c, minus the newlines. They
          differ by 5 here because ü, İ, ğ, ç and ı take two bytes each in UTF-8.
      - lines: "30-34"
        note: >
          os.Stdin is an ordinary *os.File. Run without a pipe, the program
          waits for the user to type and end the input, which looks like a hang;
          tools that default to stdin usually say so in their usage text.
use_cases:
  - "Internal tools and one-off migrations where a dependency for argument parsing isn't worth it"
  - "git-style tools with subcommands (tool import, tool export), each with its own flags"
  - "Twelve-factor services: configuration from environment variables, overridable by flags during development"
  - "Unix filters that read stdin and write stdout, so they compose with grep, sort and jq"
strengths:
  - "No dependency for small tools: flag, os and bufio are in the standard library, while the JDK has no argument parser at all"
  - "Typed flags (Int, Duration, Bool) parse and validate in one step: -timeout 1.5s arrives as a time.Duration, not a string to parse later"
  - "Usage text is generated from the definitions, so -h can't drift from the code"
  - "With a FlagSet, an injected getenv and an io.Reader, the whole CLI runs in-process in a test, with no subprocess"
weaknesses:
  - "No GNU conventions: -v and --v are the same flag, there are no short/long aliases (-v/--verbose) and no combined short flags (-xvf)"
  - "Flags must come before positional arguments; users used to GNU tools put them last and get them treated as file names"
  - "No required flags, no subcommand tree, no config-file binding: you write the checks, or reach for a third-party package once the CLI grows"
pitfalls:
  - title: "Flags after positional arguments"
    detail: "tool report.txt -v leaves -v in fs.Args(). Document flags-first usage, or check Args() for leftovers that start with a dash."
  - title: "Turning a boolean off with a separate word"
    detail: "-dry-run false sets dry-run to true and adds \"false\" as a positional argument."
    bad: "tool -dry-run false input.csv"
    good: "tool -dry-run=false input.csv"
  - title: "os.Getenv can't tell unset from empty"
    detail: "Both return \"\". Use os.LookupEnv when an empty value means something different from a missing one."
  - title: "flag.Parse in init() or in a library"
    detail: "Since Go 1.13 test flags are registered when the test binary starts, so a package that parses flags during initialization can break go test. Parse once, in main or run, and keep flag definitions out of libraries."
  - title: "Turkish text piped from Windows PowerShell 5.1"
    detail: "Text piped into a native program is encoded with $OutputEncoding, which defaults to US-ASCII there: İ and ı arrive as '?'. Set $OutputEncoding = [System.Text.UTF8Encoding]::new() first."
quiz:
  - q: "fs defines a bool flag -v. What does fs.Parse([]string{\"report.txt\", \"-v\"}) leave you with?"
    options: ["An error: flags must come first", "v=true, Args()=[report.txt]", "v=false, Args()=[report.txt -v]"]
    answer: 2
    why: "Parsing stops at the first non-flag argument without error; everything after it, -v included, is positional."
  - q: "A bool flag defaults to true. How does the user switch it off?"
    options: ["-color false", "-color=false", "-no-color"]
    answer: 1
    why: "Boolean flags take a value only in the -flag=value form; in -color false, false is a positional argument."
  - q: "The casing tool has -lang and reads CASING_LANG. Which order of precedence does it implement?"
    options: ["flag > environment > default", "environment > flag > default", "default > flag > environment"]
    answer: 0
    why: "The environment value becomes the flag's default, and an explicit flag overwrites the default."
  - q: "Why does run take getenv func(string) (string, bool) instead of calling os.Getenv?"
    options: ["os.Getenv is deprecated", "Tests can pass a map without changing the process-wide environment", "It is faster"]
    answer: 1
    why: "The environment is global to the process; os.Setenv in one test leaks into others and blocks t.Parallel. Injection keeps tests independent."
further_reading:
  - title: "package flag"
    url: "https://pkg.go.dev/flag"
  - title: "package os: LookupEnv"
    url: "https://pkg.go.dev/os#LookupEnv"
  - title: "package bufio: Scanner"
    url: "https://pkg.go.dev/bufio#Scanner"
  - title: "Go 1.13 release notes: testing flags"
    url: "https://go.dev/doc/go1.13#testingpkgtesting"
---

## Concept

A command-line program has three inputs: its arguments, its environment and its
standard input. The `flag` package turns arguments into typed values; a `FlagSet`
does it for an explicit slice, which is how you build subcommands and how you test.
`os.LookupEnv` reads the environment and tells "unset" from "empty". `os.Stdin` is an
`*os.File`, so everything that reads an `io.Reader` reads stdin.

The common shape is precedence: a built-in default, overridden by an environment
variable, overridden by a flag. With `flag` that takes three lines, because the
environment value simply becomes the flag's default.

## Coming from C/C++/Java/Python/JS

- C: GNU `getopt` permutes arguments and supports `-xvf`; `flag` does neither and
  stops at the first positional argument. `getenv` returning `NULL` for unset is what
  `os.LookupEnv`'s boolean gives you.
- Python: `argparse` is the closer match (stdlib, generated help, subparsers), but it
  accepts options after positionals. `os.environ.get` ≈ `os.LookupEnv`.
- Java: there is no argument parser in the JDK; `flag` plays the role of picocli or
  commons-cli for small tools. `System.getenv` returns `null` for unset, like
  LookupEnv's `false`.
- Node: `process.argv.slice(2)` ≈ `os.Args[1:]`, `process.env` ≈ the environment,
  `process.stdin` ≈ `os.Stdin`, but in Go reading it is plain blocking code.

## Design notes

Keep `main` as the only place that touches the process: `os.Exit(run(os.Args[1:],
os.LookupEnv, os.Stdin, os.Stdout, os.Stderr))`. Everything else takes its inputs as
parameters, so a table-driven test calls `run` with a slice, a map-backed `getenv` and a
`strings.Reader`, in parallel, with no subprocess.

Stay with `flag` while the tool is small. Since Go 1.16, `fs.Func` handles a repeatable
flag (`-H a -H b`) without writing a `flag.Value` type. When you need GNU-style
`--long`/`-s` pairs, nested subcommands, shell completion or config-file binding,
that is the point to adopt a CLI library, not before. For anything you pipe in from
Windows, remember the shell decides the encoding: make it UTF-8 and the Go side needs
nothing special.
