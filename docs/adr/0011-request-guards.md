# ADR-0011: Request guards on the code-executing API

Date: 2026-11-13 · Status: accepted

## Context

In local mode (ADR-0008) `POST /api/run` runs lesson code as the owner's Windows user.
Listening on 127.0.0.1 keeps other machines out, but not other websites: any page open
in the owner's browser can send requests to 127.0.0.1. The backend decoded the body as
JSON whatever its `Content-Type`, so a hostile page could send a `text/plain` POST,
which needs no CORS preflight, and run code. It couldn't read the reply, but it didn't
need to. DNS rebinding (a hostile name re-pointed at 127.0.0.1) would also let a page
read the API, notes and run history included.

## Decision

`httpapi.withRequestGuards` (server/internal/httpapi/security.go) wraps every route:

- **`http.CrossOriginProtection`** (Go 1.25) rejects unsafe requests the browser marks as
  cross-origin through `Sec-Fetch-Site` or `Origin`. 403.
- **JSON-only bodies**: a non-GET request with a body must be `application/json`, which
  a cross-site page can't send without a preflight. 415.
- **Loopback `Host` on `/api/`**: `localhost` or a loopback IP, any port, because the
  Vite proxy forwards the browser's `Host` (port 5173). This stops DNS rebinding. 403.
  Health probes are exempt because Compose probes them by container name.

## Alternatives

- **CSRF tokens**: need a session and a token endpoint for a single-user app; the browser
  already labels cross-site requests.
- **An auth token on every API call**: would also cover other local processes, but those
  already run as the owner. Revisit if the API ever listens beyond loopback.
- **CORS headers**: CORS controls reading responses, not sending simple requests.

## Consequences

- API clients must send `Content-Type: application/json` with any body. The app and
  the verify script already do; bodyless POSTs still work.
- Opening the app through a LAN IP or another hostname gets 403 from `/api/`.
- `cmd/runner` (Docker only) keeps no guards: it is on internal networks with no
  published port.
