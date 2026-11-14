---
id: go.composite.structs
title: "Structs, embedding, tags, value vs reference semantics"
track: go
module: composite
order: 3
level: basic
minutes: 14
since: go1.27
prerequisites: [go.composite.slices-and-maps, go.basics.pointers]
compare_with: [go.interfaces.methods, ts.objects-classes.classes, design.paradigms.polymorphism]
tags: [structs, embedding, composition, struct-tags, json, comparability]
summary: >
  A struct is a value: assignment copies its fields, shallowly. Embedding
  promotes another type's fields and methods without inheritance or virtual
  dispatch, and tags attach metadata that encoding/json and database
  libraries read through reflection.
examples:
  - id: values-compare
    file: examples/01_values_compare.go
    title: Struct values, comparison, map keys, shallow copies
    runtime: go
    check: exact
    annotations:
      - lines: "15-19"
        note: >
          The zero value is a usable Point, with no constructor. Keyed
          literals keep compiling when fields are added; positional ones like
          Point{3, 4} are fine only for small, stable types, and vet flags them
          for struct types from other packages. q := p copies the struct, so
          q.X = 100 leaves p alone.
      - lines: "21-25"
        note: >
          == compares field by field when every field is comparable, so a
          struct makes a natural composite map key, without the hashCode and
          equals pair Java needs.
      - lines: "27-32"
        note: >
          The copy is shallow: r2 got its own Name but shares the Stops array
          with r1, so r1 sees {9 9}. Route has a slice field, which also makes
          r1 == r2 a compile error: slices are not comparable.
      - lines: "34-39"
        note: >
          An anonymous struct type for a one-off shape, the standard layout of
          a table-driven test. %+v prints field names.
      - lines: "41-44"
        note: >
          struct{} occupies no memory, so map[string]struct{} is the idiomatic
          set; the comma-ok read is the membership test.
  - id: embedding
    file: examples/02_embedding.go
    title: Embedding promotes fields and methods, without virtual dispatch
    runtime: go
    check: exact
    annotations:
      - lines: "16-21"
        note: >
          This is the difference from inheritance. Summary's receiver is the
          embedded Audit, which knows nothing about Order, so a.Label() calls
          Audit.Label even when reached through o: "audit v2", not "order 7".
          There is no template method by overriding; use an interface field or
          a function value.
      - lines: "23-27"
        note: >
          An embedded field has a type and no name; its name is the type's
          name, so o.Audit is still reachable. Audit's fields and methods are
          promoted: o.By and o.Touch work as if Order declared them.
      - lines: "29-30"
        note: >
          A method declared on Order hides the promoted one with the same
          name: the shallowest depth wins, for fields and methods alike.
      - lines: "33-34"
        note: >
          New in Go 1.27: promoted fields can be keys in a composite literal.
          Before, you wrote Order{ID: 7, Audit: Audit{By: "ayşe", Version: 1}};
          the compiler reports the version needed if the go line is older, and
          go fix's embedlit modernizer rewrites the old form once it isn't.
      - lines: "36-37"
        note: >
          Touch has a pointer receiver. o is addressable, so o.Touch means
          (&o.Audit).Touch and changes o's embedded Audit in place.
  - id: tags
    file: examples/03_tags.go
    title: Struct tags drive encoding/json and database mapping
    runtime: go
    check: exact
    annotations:
      - lines: "10-17"
        note: >
          A tag is a raw string literal of key:"value" pairs; the compiler
          doesn't interpret it, and go vet's structtag check catches malformed
          ones. json renames fields; db is read by libraries such as pgx or
          sqlx.
      - lines: "13-15"
        note: >
          omitempty drops false, 0, "", nil and empty collections, but never a
          struct, so the zero Joined time is still written. omitzero (Go 1.24)
          drops zero values and uses an IsZero method when the type has one,
          as time.Time does.
      - lines: "16"
        note: >
          json:"-" keeps a field out of JSON entirely. That is the tool for
          exported fields; unexported ones are skipped anyway.
      - lines: "27-32"
        note: >
          encoding/json falls back to a case-insensitive match, so "NAME" fills
          Name. encoding/json/v2 (Go 1.27) matches exactly unless you pass
          MatchCaseInsensitiveNames. If a client sends both "name" and "NAME",
          v1 keeps whichever comes last, without an error.
      - lines: "34-39"
        note: >
          Tag.Lookup separates "absent" from "empty". This loop is, in
          miniature, how a row-to-struct mapper finds its column names.
use_cases:
  - "Domain records, request and response bodies, configuration: a struct per shape"
  - "Composite map keys: map[Point]int, map[struct{ user, day string }]int"
  - "Mixing in shared fields and helpers (timestamps, audit info, a sync.Mutex) by embedding"
  - "Mapping to JSON and database columns with tags instead of hand-written conversion code"
strengths:
  - "Values by default: a copied struct can't be changed behind your back (except through its pointer, slice and map fields), unlike Java objects, which are always references"
  - "Comparable structs work with == and as map keys with no equals/hashCode to write or get wrong"
  - "Embedding reuses fields and methods without a class hierarchy; there is no fragile base class, because the embedded type never calls back into the outer one"
  - "Tags keep wire names and column names next to the field they describe, like Java annotations but read only at runtime through reflect"
weaknesses:
  - "Copies are shallow and silent: a struct with a slice or map field shares that data with every copy"
  - "Tags are unchecked strings: a typo like json:\"naem\" or a missing colon compiles; go vet catches syntax, not misspelled names"
  - "Embedding exposes the embedded type's whole method set, so embedding *sync.Mutex or a big type widens your API by accident"
  - "Large structs passed by value cost a copy on every call; pass pointers when they are big or must be mutated"
pitfalls:
  - title: "Expecting virtual dispatch through embedding"
    detail: "A method of the embedded type always calls the embedded type's methods. If behavior must vary, put an interface or a func field in the struct."
  - title: "Modifying a copy in a range loop"
    detail: "for _, o := range orders { o.Total = 0 } changes copies. Index instead, or store pointers."
    bad: "for _, o := range orders { o.Total = 0 }"
    good: "for i := range orders { orders[i].Total = 0 }"
  - title: "omitempty on a struct or time.Time field"
    detail: "It never omits structs, so a zero time is encoded as 0001-01-01T00:00:00Z. Use omitzero (Go 1.24) or a pointer field."
  - title: "Copying a struct that contains a sync.Mutex"
    detail: "The copy gets its own, possibly locked, mutex and protects nothing. go vet's copylocks check reports it; pass such structs by pointer."
quiz:
  - q: "type Base struct{}; func (Base) Name() string { return \"base\" }; func (b Base) Hello() string { return b.Name() }. type D struct{ Base } with its own Name returning \"d\". What does D{}.Hello() return?"
    options: ["\"d\"", "\"base\"", "Compile error: ambiguous"]
    answer: 1
    why: "Hello's receiver is the embedded Base, so it calls Base.Name. Embedding promotes methods; it doesn't make them virtual."
  - q: "r2 := r1 where Route has Name string and Stops []Point. After r2.Stops[0] = p, what does r1.Stops[0] hold?"
    options: ["Its old value", "p", "It depends on capacity"]
    answer: 1
    why: "Assignment copies the slice header, not the array; both Routes share Stops."
  - q: "Which struct can be used as a map key?"
    options: ["struct{ Tags []string }", "struct{ City string; Plate int }", "struct{ Meta map[string]string }"]
    answer: 1
    why: "Map keys must be comparable; slices and maps aren't, and neither is a struct containing them."
  - q: "A time.Time field tagged json:\"at,omitempty\" holds the zero time. What does json.Marshal do?"
    options: ["Omits the field", "Writes \"0001-01-01T00:00:00Z\"", "Writes null"]
    answer: 1
    why: "omitempty doesn't apply to structs. omitzero does, using time.Time's IsZero method."
further_reading:
  - title: "The Go Programming Language Specification: Struct types"
    url: "https://go.dev/ref/spec#Struct_types"
  - title: "The Go Programming Language Specification: Composite literals"
    url: "https://go.dev/ref/spec#Composite_literals"
  - title: "Effective Go: Embedding"
    url: "https://go.dev/doc/effective_go#embedding"
  - title: "package encoding/json: Marshal (tags, omitempty, omitzero)"
    url: "https://pkg.go.dev/encoding/json#Marshal"
  - title: "package reflect: StructTag"
    url: "https://pkg.go.dev/reflect#StructTag"
---

## Concept

A struct is a sequence of named fields, and a struct value *is* its fields: no header,
no hidden pointer, no identity. Assigning, passing or returning a struct copies every
field. That copy is shallow: pointer, slice and map fields are copied as references,
so the copy shares what they point to. A struct is comparable with `==`, and usable as
a map key, when all its fields are.

**Embedding** a type (a field with a type but no name) promotes its fields and methods
to the outer struct, so `o.By` means `o.Audit.By` and `o.Touch()` means
`o.Audit.Touch()`. It is composition with shorter syntax: the embedded value never
learns about the struct around it, so there is no overriding and no virtual dispatch.
Polymorphism comes from interfaces instead.

**Tags** are string literals after a field's type. The compiler ignores them;
packages such as `encoding/json` read them through `reflect` to choose names and
options.

## Coming from C/C++/Java/Python/JS

- **Java**: there are no classes, only structs with methods. Values instead of
  references by default, `==` instead of `equals`/`hashCode`, embedding instead of
  `extends` (without `super` and without overriding), tags instead of annotations.
- **C**: a struct is laid out like a C struct and copied by value, as in C. Embedding
  is the "first member is the base struct" trick, made official and given promoted
  names.
- **C++**: closest to a struct with public members. Embedding resembles inheritance
  in which no method is `virtual`, minus the implicit upcast: an `Order` is not an
  `Audit`, you pass `o.Audit` explicitly.
- **Python/JS**: no dynamic attributes; the field set is fixed at compile time. A JS
  object spread `{...a}` is shallow in the same way a Go struct copy is.

## Design notes

Make the zero value useful (`var buf bytes.Buffer` works, so should your types), and
prefer keyed literals so adding a field never breaks callers. Embed to reuse
behavior you would otherwise forward method by method, but only when the embedded
type's whole method set belongs in your API; otherwise use a named field. When a
struct holds a mutex, slices that must not be shared, or is large, give it pointer
methods and pass `*T`. Tags are a contract with another system (an API, a table):
keep them stable, test them with a round trip, and remember that `encoding/json`
matches names loosely while `encoding/json/v2` is strict.
