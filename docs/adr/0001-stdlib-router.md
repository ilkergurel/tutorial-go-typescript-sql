# ADR-0001: Standard-library `net/http` router, no web framework

Date: 2026-11-12 · Status: accepted

## Context

StackLab's backend needs routing, middleware and JSON handling. It is also a teaching
artifact: the backend's own source is lesson material for the Go track.

## Decision

Use the standard library `net/http` with Go 1.22+ `ServeMux` method-and-path patterns
(`GET /api/lessons/{id}`), hand-written middleware as function wrappers, and
`encoding/json` helpers. No web framework (no chi, gin, echo, fiber).

## Alternatives

- **chi / gin / echo**: less boilerplate for middleware chains and route groups, but
  they hide exactly the machinery the lessons teach, and add a dependency for what
  `ServeMux` now covers (method matching, wildcards, precedence).
- **Generated frameworks (goa, oapi-codegen server stubs)**: the OpenAPI contract stays
  hand-maintained for the TypeScript side; generating the server too would obscure the
  handler→service→store flow the lessons walk through.

## Consequences

- Every routing/middleware concept in the curriculum (§8.1 module 11) is demonstrated by
  the app's own code, readable in the Inside StackLab track.
- We write ~150 lines of small helpers (JSON encode, error mapping, middleware chain)
  that a framework would provide. Accepted: the helpers are themselves lessons.
- If routing needs outgrow `ServeMux`, revisiting requires an ADR.
