---
id: go.tooling.project-layout
title: "Modules, packages, internal/, and the cmd/ layout"
track: go
module: tooling
order: 2
level: basic
minutes: 13
since: go1.22
prerequisites: [go.tooling.go-command]
compare_with: [ts.objects-classes.modules, ts.architecture.folders, go.design.packages]
tags: [packages, modules, visibility, exported, internal, cmd, go-work]
summary: >
  Go has two visibility levels, decided by the first letter of a name, plus
  internal/ directories that the go command enforces. Applications put
  binaries under cmd/ and everything else under internal/.
examples:
  - id: exported-names
    file: examples/01_exported_names.go
    title: Exported means "starts with an upper-case letter", and libraries obey it
    runtime: go
    check: exact
    annotations:
      - lines: "11-15"
        note: >
          No public or private keywords: Name and City are exported, password
          is visible only inside this package, in every file of it. There is
          no class-private or protected level.
      - lines: "20-25"
        note: >
          encoding/json is another package, so it cannot touch password and
          silently leaves it out. A struct whose fields are all lower-case
          marshals to {} and never gets filled by Unmarshal.
      - lines: "27-28"
        note: >
          fmt also uses reflection, but only reads, and it prints unexported
          fields too. Logging a struct with %+v can leak a secret that
          json.Marshal would have dropped.
      - lines: "30-32"
        note: >
          reflect applies the same rule through StructField.IsExported.
          reflect.TypeFor[T]() (Go 1.22) gets the type without needing a value.
      - lines: "34-37"
        note: >
          The spec says "a Unicode upper-case letter (category Lu)". İ is one,
          so İlçe is exported; dotless ı is lower-case. Names starting with _
          or with a letter that has no case, such as 日, can never be exported.
          (The %-6s width counts runes, not bytes, so İlçe pads correctly.)
  - id: cmd-internal
    file: examples/02_cmd_internal.go
    title: "cmd/ for binaries, internal/ for shared code, and an unexported name"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "11-14"
        note: >
          A typical application module. Each directory under cmd/ is a package
          main that builds one binary named after the directory; keep main
          small (configuration, wiring) and put the logic in internal/.
      - lines: "35-41"
        note: >
          Package vat's API is Of. round is an implementation detail that
          other packages cannot call, so it can change without breaking them.
      - lines: "53-54"
        note: >
          Three packages, one module. A binary is addressed by its directory:
          go run ./cmd/shopd here, go install example.com/shop/cmd/shopd@latest
          for someone installing it from elsewhere.
      - lines: "55"
        note: >
          go build ./... type-checks every package. cmd/peek reaches for an
          unexported function and fails to compile; the message names the
          rule. Importing internal/vat itself is fine: both packages live in
          example.com/shop.
  - id: workspace
    file: examples/03_workspace.go
    title: go.work joins two local modules; internal/ still stops outsiders
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "11-12"
        note: >
          A library module and an app that uses it, both on disk. Without a
          workspace the app would need a require line and a published version,
          or a replace directive pointing at ../shop.
      - lines: "44-51"
        note: >
          go.work: a go line and one use directive per module directory. go
          work init ./app ./shop writes this file for you; go work use adds
          more modules later.
      - lines: "60"
        note: >
          No go.work yet, and app/go.mod doesn't require example.com/shop, so
          the import cannot be resolved. The error suggests go get, which
          would need a published module.
      - lines: "62-66"
        note: >
          The go command looks for go.work in the current directory and its
          parents. Both modules are now main modules: the import resolves to
          the code on disk, with no edit to either go.mod.
      - lines: "67"
        note: >
          internal/ is enforced across modules: only example.com/shop and the
          packages below it may import example.com/shop/internal/vat. A
          workspace does not lift that rule.
use_cases:
  - "Application: cmd/<binary>/main.go for each executable, internal/<feature>/ for the code, nothing importable by other modules"
  - "Library: the package at the module root (import example.com/lib), helpers in internal/ so you can change them without breaking users"
  - "Hiding a field or helper: start its name with a lower-case letter; the compiler enforces it at every use site in other packages"
  - "Developing a library and an app together: go work init ./app ./lib, without touching either go.mod"
strengths:
  - "Visibility shows at every use site: shop.Gross is public and round is private without looking at a declaration, unlike Java modifiers or TypeScript export lists"
  - "internal/ is enforced by the go command, unlike Python's _module convention or a JS package's undocumented files"
  - "No build configuration: the directory tree is the package graph, so a new binary is a new directory under cmd/"
  - "go.work replaces the old habit of temporary replace directives in go.mod, which tended to get committed by accident"
weaknesses:
  - "Only two visibility levels plus internal/: no protected and no class-private, so a struct's invariants are guarded from other packages but not from the rest of its own package"
  - "Changing visibility means renaming the identifier at every use site; gopls rename does it, plain search and replace may not"
  - "Import paths are absolute (module path + directory), so moving a package rewrites every import of it"
pitfalls:
  - title: "Lower-case struct fields and encoding/json"
    detail: "json.Marshal skips unexported fields and Unmarshal never sets them, without an error. Export the field and control its JSON name with a tag."
    bad: "type User struct { name string }"
    good: "type User struct { Name string `json:\"name\"` }"
  - title: "Printing structs that hold secrets"
    detail: "fmt's %v and %+v print unexported fields. Give the type a String or slog LogValue method that redacts, or log selected fields only."
  - title: "Assuming sub-directories share privileges"
    detail: "example.com/shop/vat cannot see unexported names of example.com/shop. Nesting directories means nothing to the compiler; only the internal/ rule depends on paths."
  - title: "Committing go.work"
    detail: "A checked-in go.work can override a developer's own workspace and makes CI test versions users won't get. Keep it local, or run CI with GOWORK=off."
quiz:
  - q: "Which of these identifiers is exported?"
    options: ["_Config", "config", "İl", "日本"]
    answer: 2
    why: "İ (U+0130) is in Unicode category Lu, so İl is exported. _ is not a letter, and 日 has no case, so names starting with it can never be exported."
  - q: "Module example.com/shop has a package in internal/vat. Who can import it?"
    options: ["Any module that requires example.com/shop", "example.com/shop and the packages below it", "Only the package example.com/shop", "Any module in the same go.work"]
    answer: 1
    why: "Code in or below internal/ is importable only by code rooted at internal's parent, here example.com/shop. A workspace does not change that."
  - q: "json.Marshal(struct{ name string; Age int }{\"Ayşe\", 30}) returns?"
    options: ["{\"name\":\"Ayşe\",\"Age\":30}", "{\"Age\":30}", "A compile error"]
    answer: 1
    why: "encoding/json only sees exported fields; name is dropped without an error."
further_reading:
  - title: "Organizing a Go module"
    url: "https://go.dev/doc/modules/layout"
  - title: "The Go Programming Language Specification: Exported identifiers"
    url: "https://go.dev/ref/spec#Exported_identifiers"
  - title: "Go Modules Reference: Workspaces"
    url: "https://go.dev/ref/mod#workspaces"
  - title: "Tutorial: Getting started with multi-module workspaces"
    url: "https://go.dev/doc/tutorial/workspaces"
  - title: "Effective Go: Names"
    url: "https://go.dev/doc/effective_go#names"
---

## Concept

A package is a directory; every file in it declares the same package name and sees
everything the others declare. Across packages there are exactly two levels of
visibility, and the name itself says which: an identifier whose first character is
an upper-case letter is **exported**, anything else is visible only inside its
package. That covers types, functions, constants, variables, struct fields and
methods alike.

The go command adds one more rule based on paths: a package in or below a directory
named `internal` can be imported only by code rooted at the parent of that
directory. That is how a module keeps packages for itself while still splitting its
code into many packages.

The common layout follows from those two rules:

```
shop/                     module example.com/shop
├── go.mod
├── cmd/
│   ├── shopd/main.go     go build ./cmd/shopd  -> binary "shopd"
│   └── shopctl/main.go
└── internal/
    ├── orders/           example.com/shop/internal/orders
    └── vat/
```

A library puts its public package at the module root instead, so users import
`example.com/lib`, and keeps helpers in `internal/`. A workspace file, `go.work`,
lets several local modules build together while you change them in step.

## Coming from C/C++/Java/Python/JS

- **Java**: exported ≈ `public`, unexported ≈ package-private. There is no `protected`
  and no class-private: every file in the package can touch a struct's lower-case
  fields. Like Java, `com.x.shop.vat` gets no special access to `com.x.shop`.
  `internal/` plays the role of packages a JPMS module does not export.
- **C/C++**: no headers to curate. Capitalization does what `static` and the header's
  contents do in C.
- **Python**: `_name` is a convention that anyone can ignore; Go's rule is checked by
  the compiler. There is no `__init__.py`: the directory is the package.
- **JS/TS**: the capital letter replaces the `export` keyword, `internal/` replaces a
  `package.json` `exports` map, and `go.work` resembles npm or pnpm workspaces.

## Design notes

Start with one package and split when a part has a clear name and API, by feature
(`internal/orders`, `internal/billing`) rather than by layer (`models`, `utils`).
Callers write the package name at every call, so `vat.Of` reads well and
`vat.VATOf` stutters. Import cycles are compile errors, which pushes the dependency
graph into layers whether you planned them or not. The official layout guide does not
use a `pkg/` directory; it is a community habit, and `internal/` gives the protection
`pkg/` only implies. Treat `go.work` as a local tool: it is for editing modules
together, and CI should build each module the way its users will (`GOWORK=off`).
