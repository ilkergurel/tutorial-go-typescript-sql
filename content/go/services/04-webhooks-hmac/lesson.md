---
id: go.services.webhooks-hmac
title: "Webhooks with HMAC signature verification"
track: go
module: services
order: 4
level: advanced
minutes: 12
since: go1.24
prerequisites: [go.services.client, go.http.json-validation]
compare_with: [go.services.retries, design.architecture.security, fullstack.tracelite.step-10-external]
tags: [webhooks, hmac, security, signatures, replay, idempotency]
summary: >
  A webhook is an unauthenticated POST until you prove otherwise. Verify an HMAC over
  the raw body and a timestamp with hmac.Equal, reject stale timestamps, deduplicate
  by event ID, and only then parse and act.
examples:
  - id: verify
    file: examples/01_verify.go
    title: "Verify signature and freshness, and see which replay still gets through"
    runtime: go
    check: exact
    annotations:
      - lines: "16-21"
        note: >
          HMAC-SHA256 with a secret shared only by sender and receiver. The
          timestamp is part of the signed message, so an attacker can't take a
          captured request and simply put a fresh timestamp on it.
      - lines: "32-39"
        note: >
          The header may carry several v1 signatures: during a rotation the
          sender signs with both the old and the new secret. Signatures are
          compared as decoded bytes, so upper- or lowercase hex doesn't matter.
      - lines: "43-45"
        note: >
          The freshness window limits replays to five minutes. It also
          requires roughly synchronized clocks on both sides (NTP); the window
          absorbs the difference.
      - lines: "46-52"
        note: >
          hmac.Equal takes the same time whether the first or the last byte
          differs. With bytes.Equal or ==, response times leak how many leading
          bytes of a guess were right, and the MAC can be found byte by byte.
      - lines: "57"
        note: >
          verify takes "now" as a parameter, so tests and this example control
          the clock without sleeping.
      - lines: "59"
        note: >
          The receiver accepts two secrets while a rotation is in progress, so
          events signed with the old one keep verifying until the sender has
          switched.
      - lines: "74"
        note: >
          A copy of a valid request, replayed inside the window, passes every
          check here. Senders also deliver at least once on their own, so the
          event ID has to decide: see How it works.
      - lines: "76"
        note: >
          Fail closed: no header, a bad header, an old timestamp and a wrong
          MAC all end in a rejection. The reason is printed for the demo; a
          real receiver logs it and answers with a bare 403, which tells a
          prober nothing.
  - id: raw-bytes
    file: examples/02_raw_bytes.go
    title: "Why the MAC must be checked on the bytes received, not a re-encoding"
    runtime: go
    check: exact
    annotations:
      - lines: "18-20"
        note: >
          Python's json.dumps escapes non-ASCII characters by default, so ş
          travels as a six-character escape. The sender signed exactly these
          bytes.
      - lines: "22-27"
        note: >
          Decoding and re-encoding gives the same JSON value but different
          bytes: Go writes ş as UTF-8, escapes <, > and & for HTML safety, and
          sorts map keys. Any framework that hands you a parsed object instead
          of the raw body has the same problem.
      - lines: "31-33"
        note: >
          Only the received bytes verify. A receiver built on the shortcut
          rejects valid events whenever the encodings differ (here: Turkish
          letters, the HTML characters and the key order), and such a check
          tends to get "fixed" by switching verification off.
      - lines: "35-36"
        note: >
          After decoding, "Ayşe Çelik" is 10 characters and 12 bytes in UTF-8.
          On the wire it was longer still: each escape is six bytes.
use_cases:
  - "Payment, shipping and e-mail providers notifying you of events (payment succeeded, parcel delivered, message bounced)"
  - "Git hosting and CI systems triggering builds and deployments"
  - "Your own service sending webhooks to customers: the same scheme, from the signing side"
strengths:
  - "crypto/hmac and crypto/sha256 are in the standard library, and hmac.Equal is constant-time, so the whole scheme is a few dozen lines without dependencies"
  - "Verification needs no network round trip or key server, unlike checking a token against an identity provider"
  - "Reading the body as []byte first makes 'verify the raw bytes' the natural order in Go, unlike frameworks that parse JSON before your handler runs"
weaknesses:
  - "A shared secret has to be stored on both sides and rotated by hand; a leak on either side lets anyone forge events"
  - "Every provider has its own header format, signed-message layout and encoding (hex or base64), so each needs its own small verifier"
  - "HMAC proves who sent the bytes, not that the event is fresh or unique: replays need timestamps and event-ID deduplication on top"
pitfalls:
  - title: "Comparing MACs with == or bytes.Equal"
    detail: "Both stop at the first differing byte, and the timing difference can be measured over the network with enough requests. Use hmac.Equal (or crypto/subtle.ConstantTimeCompare) on the decoded bytes."
    bad: |
      if r.Header.Get("X-Signature") == hex.EncodeToString(expected) {
    good: |
      got, err := hex.DecodeString(r.Header.Get("X-Signature"))
      if err != nil || !hmac.Equal(got, expected) {
          http.Error(w, "forbidden", http.StatusForbidden)
          return
      }
  - title: "Verifying a re-encoded body"
    detail: "Key order, whitespace, escapes and number formatting all change on a round trip. Read the raw body first (with a size cap), verify it, then decode those same bytes."
  - title: "No replay protection"
    detail: "A valid request captured once (from a log, a proxy, a misconfigured debug endpoint) verifies forever unless the timestamp is signed and checked and event IDs are deduplicated."
  - title: "Doing the work inside the request"
    detail: "Slow processing makes the sender time out and redeliver, and then you process the event twice. Persist, answer 2xx, process asynchronously."
  - title: "Hard-coding one secret"
    detail: "Rotation then means downtime or a window where valid events are rejected. Accept a list of secrets, add the new one, switch the sender, remove the old one."
quiz:
  - q: "Your handler decodes the JSON body into a struct, re-marshals it and computes the HMAC of that. What happens?"
    options: ["It works: same data, same MAC", "Valid events fail whenever the sender's bytes differ from Go's encoding (key order, escapes, whitespace)", "It is faster but less secure", "Only invalid events fail"]
    answer: 1
    why: "A MAC is over bytes, not over a JSON value. Example 2 fails on Turkish characters escaped by the sender and on HTML characters escaped by Go."
  - q: "Why does the sender include a timestamp in the signed message?"
    options: ["For logging", "So the receiver can reject old requests, limiting replays to a short window", "HMAC requires a nonce", "To make the MAC longer"]
    answer: 1
    why: "Because it is signed, an attacker can't refresh it; the receiver rejects anything outside the tolerance."
  - q: "A captured, correctly signed request is replayed two minutes later, within the five-minute window. What stops it from being processed twice?"
    options: ["hmac.Equal", "MaxBytesReader", "Deduplication by event ID", "Nothing can"]
    answer: 2
    why: "The signature and timestamp are both valid, so only remembering which event IDs were already handled prevents a second effect."
further_reading:
  - title: "package crypto/hmac"
    url: "https://pkg.go.dev/crypto/hmac"
  - title: "package crypto/subtle: ConstantTimeCompare"
    url: "https://pkg.go.dev/crypto/subtle#ConstantTimeCompare"
  - title: "package net/http: MaxBytesReader"
    url: "https://pkg.go.dev/net/http#MaxBytesReader"
  - title: "package encoding/json: Marshal (HTML escaping, sorted map keys)"
    url: "https://pkg.go.dev/encoding/json#Marshal"
---

## Concept

A webhook inverts the usual direction: the other service calls *you*, with an HTTP POST
to a public URL. Anyone on the internet can send a POST to that URL, so the request
proves nothing until it is verified.

The common scheme is an **HMAC**: sender and receiver share a secret, the sender
computes `HMAC-SHA256(secret, message)` over the request and puts it in a header, and
the receiver recomputes it and compares. Matching MACs prove that the bytes came from
someone who knows the secret and were not changed on the way. Three details make it
safe in practice:

1. **The MAC covers the raw body**, exactly as received, so read the body before parsing
   it. Compare with `hmac.Equal`, which takes constant time.
2. **A timestamp is part of the signed message**, and the receiver rejects requests
   outside a small window. That limits how long a captured request can be replayed.
3. **Event IDs are deduplicated**, because senders deliver at least once and a replay
   inside the window still verifies.

Then answer 2xx quickly and process the event outside the request.

## Coming from C/C++/Java/Python/JS

- **Python**: `hmac.compare_digest(a, b)` is `hmac.Equal`. In Flask or Django, use
  `request.get_data()` or `request.body`, the raw bytes, not the parsed JSON.
- **Java**: `javax.crypto.Mac` with `HmacSHA256`, and `MessageDigest.isEqual` for the
  comparison. In Spring, take the body as `byte[]` (or `String`) rather than a
  `@RequestBody` DTO, so Jackson doesn't parse it before you verify.
- **Node**: `crypto.createHmac('sha256', secret)` and `crypto.timingSafeEqual`. Express
  needs `express.raw()` (or a `verify` callback on `express.json()`) for that route,
  because the parsed `req.body` has lost the original bytes.
- **C**: OpenSSL's `HMAC()` and `CRYPTO_memcmp` for the comparison; a plain `memcmp`
  leaks timing like `==`.

## How it works

Example 1 is the verification function. The HTTP handler around it does five things,
in this order:

1. **Read the raw body once, capped**:
   `body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))`. The MAC is
   computed over exactly these bytes (example 2 shows why), and the cap stops an
   unauthenticated sender from making you buffer gigabytes.
2. **Verify**: `verify(r.Header.Get("Webhook-Signature"), body, secrets, time.Now())`.
   Any error ends in a bare 403.
3. **Parse** the same bytes with `json.Unmarshal`, only now that they proved where they
   came from.
4. **Deduplicate by event ID.** Senders deliver at least once, and a replay inside the
   window passes `verify` (the "replayed 1 min later" line). Answer a duplicate with
   2xx too, or the sender keeps redelivering it.
5. **Store or enqueue the event and answer 2xx at once.** Senders time out and retry slow
   receivers, so the work happens afterwards, for example in a job table worker
   (fullstack.tracelite.step-11-jobs).

## Design notes

Stripe, for one, uses the shape of example 1: one header with a timestamp and one or
more versioned signatures (`t=...,v1=...`) over the string `timestamp.body`, and while a
secret is being rolled it sends one signature per active secret. Others sign only the
body (then replay protection relies on event IDs alone) or use asymmetric
signatures (Ed25519 or RSA), where you hold only a public key and nothing secret can leak
from your side; `crypto/ed25519` verifies those. Follow the provider's documentation
exactly, including hex versus base64, and write a test with a sample request from their
docs.

Keep verification as a function of `(header, body, secrets, now)`, as here: it is then
testable without HTTP and without sleeping. Wrap it in middleware for the webhook routes
only, and put the raw-body read there, so handlers receive verified bytes.

The deduplication table belongs in your database: `INSERT INTO webhook_events (id) ...
ON CONFLICT DO NOTHING` in the same transaction as the event's effect, and a 2xx for
duplicates. IP allowlists and mutual TLS are useful extra layers, but they don't replace
the signature: IP ranges change, and they say nothing about whether the body was
altered.
