---
id: ts.type-level.branded
title: "Branded types (RequirementId) and making illegal states unrepresentable"
track: ts
module: type-level
order: 3
level: advanced
minutes: 15
since: ts3.5
prerequisites: [ts.type-level.conditionals, ts.mental-model.structural, ts.narrowing.discriminated-unions]
compare_with: [go.basics.types-conversion, go.design.constructors, ts.services.zod, design.patterns.anti-patterns]
tags: [branded-types, nominal-typing, unique-symbol, discriminated-unions, non-empty, typestate, over-engineering]
summary: >
  A brand gives a structural type a name the checker enforces, so a RequirementId can only
  come from the parser that validated it. Unions model each state with its own data. Both
  pay off where values get mixed up or states get confused, and stop paying beyond that.
examples:
  - id: brands
    file: examples/01_brands.ts
    title: RequirementId and TestCaseId, parsed once and never swapped
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          Structural typing compares strings with strings, so two string parameters
          are interchangeable, and so are two type aliases of string. The call is
          backwards and nothing notices.
      - lines: "9-14"
        note: >
          RequirementId is a string that also claims a property keyed by a unique
          symbol. No real string has it, so a plain string isn't assignable, and the
          two brands differ in that property's type. declare const emits nothing:
          the symbol exists only for the checker.
      - lines: "16-24"
        note: >
          Smart constructors: the regex and the cast sit in one function, so every
          RequirementId in the program has passed the check. Code that receives one
          never re-validates, which is the point of parsing at the boundary.
      - lines: "32-39"
        note: >
          Swapped arguments and raw strings are now TS2345. At run time req is just
          the string: typeof, JSON and every string method see no brand, and
          toLowerCase() returns a plain string, because a lowercased ID is no longer
          known to be valid.
      - lines: "41-44"
        note: >
          toLocaleUpperCase("tr") maps i to İ, so "REQ-LOGİN" fails the check. The
          same happens with toLocaleUpperCase() and no argument on a machine whose
          default locale is Turkish. Identifiers need locale-independent
          toUpperCase(); locale rules are for text people read.
  - id: illegal-states
    file: examples/02_illegal_states.ts
    title: Each state with its own data, and "at least one" as a type
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-14"
        note: >
          With optional fields the type admits a draft with an approver, and code
          that tests a field instead of the status believes it. Every reader has to
          know which combinations are real; the type doesn't say.
      - lines: "16-18"
        note: >
          A tuple with a required first element and a rest: [] doesn't fit, so
          "approved with no tests" can't be constructed, and evidence[0] is a string,
          not string | undefined, even under noUncheckedIndexedAccess.
      - lines: "19-26"
        note: >
          The union lists exactly three shapes. A draft has no approvedBy to set, so
          the nonsense value is a compile error, and an approval can't exist without
          evidence.
      - lines: "28-34"
        note: >
          Transitions accept the state they need and return the state they produce.
          isNonEmpty is the parser for arrays: data from a query is a plain array, and
          the guard is the one place that proves it has an element.
      - lines: "36-45"
        note: >
          The switch narrows r per case, so each branch can read only that state's
          fields. Add a fourth status without a case and the function could end
          without returning, which is TS2366 under the string return type.
  - id: typestate
    file: examples/03_typestate.ts
    title: "Where to stop: state in a type parameter vs a union of states"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-13"
        note: >
          Typestate: the status lives in a type parameter and each transition is a
          function from one state to the next. It is a real technique, used by builders
          whose steps are fixed in the source.
      - lines: "15-19"
        note: >
          When the code itself creates the value, the compiler tracks every step:
          approving a draft is TS2345.
      - lines: "21-28"
        note: >
          Most domain objects come from a database or a request, so their state is a
          run-time value: Req<Status>. It is one object type with a union-typed
          property, not a union of objects, so checking row.status narrows the
          property and leaves row as Req<Status>.
      - lines: "30-34"
        note: >
          A guard per question gets the type back. This is the cost of typestate on
          dynamic data: extra code whose only job is to convince the checker.
      - lines: "36-39"
        note: >
          The union of the states is the discriminated union from example 2, and a
          plain comparison narrows it. TS 3.5 also lets Req<Status> be assigned to it
          without a cast, because each possible status matches one member.
use_cases:
  - "IDs of different entities that travel through the same code: RequirementId, TestCaseId, UserId in services, URLs and React props"
  - "Units and currencies that are all number: meters vs feet, kuruş vs lira, milliseconds vs seconds"
  - "Validated or normalized values that are expensive or risky to re-check: an e-mail address, a slug, sanitized HTML"
  - "Workflows where each state has its own data: draft, approved with approver and evidence, rejected with a reason"
strengths:
  - "Zero run-time cost: a brand is erased, unlike a Java wrapper record that allocates an object per ID"
  - "Validation happens once, in the parser; everything downstream receives proof in the type instead of re-checking strings"
  - "Unions make impossible combinations compile errors at construction time, not bugs found by reading every if statement"
  - "Stricter than Go's named types in one respect: an untyped string constant converts to a Go `type RequirementID string` implicitly, while a literal never becomes a RequirementId without the parser"
weaknesses:
  - "A brand is a convention the compiler enforces only until someone writes `as RequirementId`; nothing at run time backs it"
  - "Data from JSON, the database and URL parameters arrives as plain strings, so every boundary needs a parse step"
  - "String methods and arithmetic drop the brand, so transformed values need re-parsing or a deliberate cast"
  - "Each extra type-level guarantee costs readability, error-message length and onboarding time; past a point the casts and guards outnumber the bugs prevented"
pitfalls:
  - title: "Forging brands with as"
    detail: "A cast at the call site skips the validation the brand stands for. Keep `as RequirementId` inside the parser module and make every other path go through it; a lint rule or code review can enforce that."
    bad: "const id = params.reqId as RequirementId;"
    good: "const id = parseRequirementId(params.reqId); if (!id) return notFound();"
  - title: "Locale-dependent case mapping for identifiers"
    detail: "toLocaleUpperCase() follows the default locale, and on a Turkish machine \"req-login\" becomes \"REQ-LOGİN\". Normalize identifiers with toUpperCase() or toLowerCase(); keep toLocaleUpperCase(\"tr\") for display text."
    bad: "const s = raw.trim().toLocaleUpperCase();"
    good: "const s = raw.trim().toUpperCase();"
  - title: "Optional fields standing in for states"
    detail: "{ status; approvedBy?; reason? } accepts combinations that never happen, and every consumer has to guess which ones are real. Write one union member per state."
  - title: "Branding everything"
    detail: "Brands on every string (Title, Description, Comment) turn ordinary code into a stream of parse calls and casts. Brand what actually gets mixed up or carries a validated invariant; leave free text as string."
  - title: "Typestate on data whose state is only known at run time"
    detail: "A Req<S> loaded from the database is Req<Status>, and checking its status doesn't narrow it. A discriminated union does the same job with a plain if."
quiz:
  - q: "id is a RequirementId holding \"REQ-LOGIN\". What does JSON.stringify({ id }) produce?"
    options: ["{\"id\":\"REQ-LOGIN\"}", "{\"id\":{\"value\":\"REQ-LOGIN\"}}", "It includes the brand symbol"]
    answer: 0
    why: "The brand exists only in the type. At run time the value is the plain string, and symbol-keyed properties wouldn't be serialized anyway."
  - q: "id is a RequirementId. What is the type of id.toLowerCase()?"
    options: ["RequirementId", "string", "Lowercase<RequirementId>"]
    answer: 1
    why: "toLowerCase is declared on string and returns string. A transformed ID isn't known to satisfy the parser's rules, so losing the brand is correct."
  - q: "row: Req<Status> where interface Req<S> { id: string; status: S }. Why does if (row.status === \"review\") approve(row) fail?"
    options:
      - "Narrowing doesn't work on string literals"
      - "Req<Status> is one object type, not a union, so the check narrows row.status but not row"
      - "approve needs an explicit type argument"
    answer: 1
    why: "Narrowing an object by a property needs a union of object types to pick from. Req<\"draft\"> | Req<\"review\"> | Req<\"approved\"> is one; Req<Status> is not."
  - q: "Which change makes \"an approved requirement without evidence\" impossible to construct?"
    options:
      - "evidence?: string[] on an approved state"
      - "evidence: string[] on an approved state"
      - "evidence: readonly [string, ...string[]] on an approved state"
    answer: 2
    why: "Only the tuple with a required first element rejects []; string[] accepts an empty array."
further_reading:
  - title: "TypeScript Handbook: Everyday Types (type aliases are only aliases)"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#type-aliases"
  - title: "TypeScript Handbook: Symbols (unique symbol)"
    url: "https://www.typescriptlang.org/docs/handbook/symbols.html"
  - title: "TypeScript Handbook: Narrowing (discriminated unions)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions"
  - title: "TypeScript 3.5 release notes: smarter union type checking"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-3-5.html"
  - title: "The Go Programming Language Specification: Assignability (untyped constants)"
    url: "https://go.dev/ref/spec#Assignability"
---

## Concept

TypeScript compares types by shape, so `type RequirementId = string` is only a second name
for `string`: any string fits, including a `TestCaseId`. A **brand** gives the type a
member no real string has, `string & { readonly [brand]: "RequirementId" }`, keyed by a
`unique symbol` that exists only for the checker. Now a plain string doesn't fit, two
brands don't fit each other, and the only way to get a `RequirementId` is a function that
checks the input and casts once. That function is the parser; everything after it trusts
the type.

The same idea applied to objects is **making illegal states unrepresentable**: instead of one
type with optional fields that allows every combination, write a union with one member per
state, each carrying exactly the data that state has. A draft can't have an approver; an
approval can't exist without evidence, and `readonly [T, ...T[]]` turns "at least one" into a
type.

This lesson also closes the type-level module, so it ends with the other half of the skill:
recognizing when an extra guarantee costs more than the bugs it prevents.

## Coming from C/C++/Java/Python/JS

- Go: `type RequirementID string` is a distinct named type, closer to a brand than a TS
  alias is, but any code can convert with `RequirementID(s)`, and an untyped string
  constant such as `"REQ-LOGIN"` is assignable without a conversion. Enforcing a
  constructor takes a struct with an unexported field.
- Java: `record RequirementId(String value)` with checks in the compact constructor is
  nominal and enforced at run time, at the price of a wrapper object per value. A brand is
  enforced only by the checker and costs nothing at run time.
- C++: the strong-typedef idiom, a wrapper template with a tag type, is the same trick: a
  type parameter that exists only to make types distinct.
- Python: `typing.NewType("RequirementId", str)` is the closest analogue: distinct for the
  type checker, the plain value at run time.
- JS: it is a string. Every guarantee here disappears when the types are stripped, which
  is why parsers sit at the boundaries where data enters.

## Design notes

Brands earn their place where values of the same base type travel together and a mix-up
is a real bug: IDs of different tables, units, amounts in different currencies, validated
strings such as slugs or sanitized HTML. Create them only in parsers at the boundary (a
route handler, an API client, a zod schema: zod's `.brand()` uses the same `unique symbol`
technique, lesson ts.services.zod), and keep the casts in those parsers. For prefixed IDs
a template literal type is a cheaper middle ground:

```ts
type RequirementCode = `REQ-${string}`;
```

Plain strings and `TC-...` IDs are rejected and literals need no cast, though it checks only
the prefix.

Then stop. A useful ladder: plain types and discriminated unions first, since they catch
most domain mistakes; brands for the values that actually get confused; typestate, phantom
parameters and type-level parsers only for APIs whose states are fixed in the source, such
as builders. Signs you went too far: casts written to satisfy your own types, guards that
exist only to convince the checker, error messages longer than the function, and types
nobody else on the team dares to change. The goal is fewer bugs per unit of reading
effort, not the most precise type that compiles.
