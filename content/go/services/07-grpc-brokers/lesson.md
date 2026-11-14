---
id: go.services.grpc-brokers
title: "gRPC/protobuf and message brokers: when to choose them (concepts)"
track: go
module: services
order: 7
level: intermediate
minutes: 13
since: go1.21
prerequisites: [go.services.client, go.services.retries]
compare_with: [go.services.listen-notify, sql.transactions.skip-locked, design.architecture.styles]
tags: [grpc, protobuf, message-brokers, kafka, rabbitmq, delivery-semantics]
summary: >
  gRPC trades HTTP/JSON's readability for a generated, typed contract, binary
  protobuf, streaming and deadlines that travel with each call. A broker trades a
  direct call for time decoupling, at the price of at-least-once delivery that
  your consumers must make idempotent.
examples:
  - id: wire-format
    file: examples/01_wire_format.go
    title: "Protobuf's wire format by hand: tags, varints, and skipping unknown fields"
    runtime: go
    check: exact
    annotations:
      - lines: "8-12"
        note: >
          Every field on the wire carries a wire type, so a reader always knows
          how long a field is, even one it has never heard of. Groups (3, 4)
          are deprecated; 1 and 5 are fixed 64- and 32-bit values.
      - lines: "14-17"
        note: >
          A record is a tag, (field_number << 3) | wire_type, and a value.
          encoding/binary's varint is the same base-128 encoding protobuf uses:
          7 bits per byte, high bit set while more bytes follow.
      - lines: "19-23"
        note: >
          Strings are length-prefixed UTF-8, and the length counts bytes:
          "Ayşe" is 4 characters and 5 bytes (12 05 41 79 c5 9f 65 in the
          output).
      - lines: "25-26"
        note: >
          Generated code from google.golang.org/protobuf does all of this for
          you, including stopping on truncated input instead of slicing out of
          range. The point here is what it does, not that you should write it.
      - lines: "38-40"
        note: >
          A length-delimited value is skipped by its length alone, whatever it
          contains: a string, raw bytes or a whole nested message.
      - lines: "52-53"
        note: >
          The heart of protobuf's compatibility story: an old reader skips a
          field it doesn't know, using only the wire type. Names never appear
          on the wire, which is why renumbering a field breaks every client
          and renaming one breaks none.
      - lines: "59-64"
        note: >
          08 96 01 is field 1 holding 150, the same bytes the protobuf
          encoding guide uses as its first example. The whole message is 14
          bytes.
      - lines: "65-66"
        note: >
          The same data as JSON is 51 bytes, most of them field names. JSON is
          self-describing and readable with curl; protobuf needs the .proto
          file to mean anything.
      - lines: "68-71"
        note: >
          Version 2 of the schema adds field 4. A v1 reader still decodes the
          message and ignores the note: adding fields is a compatible change,
          as long as numbers are never reused.
  - id: at-least-once
    file: examples/02_at_least_once.go
    title: "At-least-once delivery: a crash before the ack, a redelivery, an idempotent consumer"
    runtime: go
    check: exact
    annotations:
      - lines: "10-13"
        note: >
          A toy model of how SQS-style queues behave (RabbitMQ and Kafka
          differ in mechanism, not in outcome): a message is only gone once it
          is acknowledged.
      - lines: "28"
        note: >
          The visibility timeout: if a consumer doesn't ack in time, the queue
          assumes it died and hands the message to someone else. The broker
          can't tell a crashed consumer from a slow one.
      - lines: "41-45"
        note: >
          m2 comes back after m3. Redelivery reorders messages; consumers that
          need order must handle it themselves (or use an ordered partition,
          as in Kafka).
      - lines: "46-50"
        note: >
          Deduplication by message ID turns at-least-once delivery into an
          effect that happens exactly once. The processed set must be durable
          and written in the same transaction as the effect, or a crash
          between the two reopens the gap.
      - lines: "54-58"
        note: >
          The crash happens after the work and before the ack, the one window
          no broker can close for you. Acking before the work moves the risk:
          a crash then loses the message (at-most-once).
use_cases:
  - "gRPC: chatty internal service-to-service calls, polyglot teams that want one contract, streaming between backend services"
  - "Brokers: work that may happen later (e-mails, thumbnails, exports), absorbing traffic bursts, one event consumed by several independent services"
  - "Kafka-style logs: event streams that consumers need to replay, such as rebuilding a search index or an analytics pipeline"
strengths:
  - "gRPC generates typed clients and servers from one .proto file, so a field rename fails the build, not production, unlike hand-written JSON clients"
  - "gRPC sends the caller's deadline with each call (the grpc-timeout header) and supports cancellation, so budgets propagate across services as lesson 1 wished plain HTTP did"
  - "Protobuf's field numbers make adding fields backward and forward compatible by design"
  - "A broker lets producers keep working while consumers are down, and lets consumers scale independently"
weaknesses:
  - "gRPC needs a code generation toolchain (protoc and plugins) and isn't readable with curl; browsers need gRPC-Web and a proxy"
  - "Binary payloads and HTTP/2 make debugging and HTTP caching harder than with REST/JSON"
  - "A broker is one more stateful system to run, monitor and back up, and every consumer has to handle duplicates and reordering"
  - "Asynchronous flows are harder to trace and to test end to end than a call stack"
pitfalls:
  - title: "Reusing or renumbering protobuf fields"
    detail: "Old clients decode the new field's bytes as the old field's meaning, silently. Never change a field's number or type; mark removed numbers and names as reserved in the .proto file."
  - title: "Writing to the database and publishing to the broker separately"
    detail: "A crash between the two leaves them disagreeing: an order without its event, or an event for an order that rolled back. Write the event to an outbox table in the same transaction, and relay it to the broker afterwards."
  - title: "Assuming exactly-once delivery"
    detail: "Brokers deliver at least once in practice. Make consumers idempotent (a processed-IDs table, upserts, natural keys) instead of trusting a delivery guarantee."
  - title: "Adding a broker for a job queue PostgreSQL could run"
    detail: "A jobs table with FOR UPDATE SKIP LOCKED plus LISTEN/NOTIFY wake-ups handles a lot of work with transactional guarantees and no new infrastructure. Reach for a broker when you need its specific strengths: fan-out to many services, replay, or very high throughput."
quiz:
  - q: "A v1 client receives a protobuf message that carries a field number it doesn't know. What happens?"
    options: ["Decoding fails", "The field is skipped, using its wire type to find its length", "The field is mapped by name to a similar field", "The client crashes"]
    answer: 1
    why: "Every record's wire type says how to skip it. That is what makes adding fields a compatible change."
  - q: "A consumer charges a card and then crashes before acknowledging the message. What does an at-least-once broker do?"
    options: ["Drops the message", "Delivers it again later, so the card is charged twice unless the consumer deduplicates", "Rolls back the charge", "Marks it as processed"]
    answer: 1
    why: "The broker only knows the message wasn't acked. Idempotency is the consumer's job."
  - q: "Which is the strongest reason to pick gRPC over REST/JSON for an API?"
    options: ["A public API for browser clients", "Internal, high-volume calls between services in several languages that share a typed contract and need streaming", "Being able to test it with curl", "HTTP caching by CDNs"]
    answer: 1
    why: "Generated clients, a binary format, streaming and deadline propagation pay off between services; the other options favour REST/JSON."
further_reading:
  - title: "package encoding/binary: varints"
    url: "https://pkg.go.dev/encoding/binary#AppendUvarint"
  - title: "Go blog: A new Go API for Protocol Buffers"
    url: "https://go.dev/blog/protobuf-apiv2"
  - title: "package google.golang.org/grpc"
    url: "https://pkg.go.dev/google.golang.org/grpc"
  - title: "package google.golang.org/protobuf/proto"
    url: "https://pkg.go.dev/google.golang.org/protobuf/proto"
  - title: "PostgreSQL: SELECT ... FOR UPDATE SKIP LOCKED (the locking clause)"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE"
---

## Concept

Two separate decisions hide in this lesson's title.

**How services call each other.** REST with JSON over HTTP is the default: readable,
cacheable, callable from a browser and from curl. **gRPC** is an RPC framework from
Google: you describe services and messages in a `.proto` file, generate client and
server code for each language, and calls travel as binary **protobuf** messages over
HTTP/2. gRPC offers four call shapes (unary, server streaming, client streaming and
bidirectional streaming), standard status codes, cancellation, and a deadline that the
client sends with every call. In Go, a gRPC handler receives a `context.Context` that
carries the caller's deadline, so lesson 1's budgets cross service boundaries by
themselves.

**Whether they should call each other at all.** A direct call couples both sides in
time: if the payment service is down, checkout is down. A **message broker** sits in
between. The producer writes a message and moves on, consumers process at their own
pace, and bursts queue up instead of overloading anyone. The cost is delivery
semantics: brokers deliver **at least once** (a message whose ack is lost comes back),
often out of order after a redelivery, so every consumer must be idempotent. "Exactly
once" in practice means at-least-once delivery plus deduplication in the consumer.

Brokers come in three broad shapes: **queues** (RabbitMQ, Amazon SQS), where each
message goes to one consumer and disappears once acked; **logs** (Kafka), which keep
messages for a retention period, track each consumer group's offset, allow replay, and
guarantee order only within a partition; and lightweight **pub/sub** (core NATS), which
is fast and at-most-once unless you add its persistence layer (JetStream).

## Coming from C/C++/Java/Python/JS

- **Java**: gRPC-Java and Spring's `@KafkaListener` or `@RabbitListener` are the usual
  stack. A listener method that throws typically gets the message retried (how often
  depends on the configured error handler), so the idempotency requirement is the same
  as in example 2.
- **Python**: `grpcio` generates the same stubs. Celery is a broker under a task-queue
  API, and it shows the trade-off in one setting: by default a task is acknowledged
  before it runs (a crash loses it); with `acks_late` it is acknowledged afterwards (a
  crash runs it twice).
- **JS**: `@grpc/grpc-js` on Node; in the browser you use REST/JSON or gRPC-Web, because
  browser JavaScript can't drive gRPC's HTTP/2 framing directly.
- **C/C++**: the protobuf runtime is a library you link; the encoding itself is plain
  enough to write by hand in any language, as example 1 does.

## How it works

On the wire, a gRPC call is an HTTP/2 `POST` to `/package.Service/Method` with content
type `application/grpc`. Each message is prefixed with 5 bytes, a compression flag and a
4-byte big-endian length (the length-prefix framing from lesson 5). The deadline travels
in the `grpc-timeout` request header, and the call's status (`grpc-status`) arrives in
HTTP/2 trailers, even when it is OK.

Protobuf messages are a sequence of records, each a tag and a value; the tag is
`(field_number << 3) | wire_type`. Field names exist only in the `.proto` file and the
generated code. That makes messages small and evolvable (add fields with new numbers;
old readers skip them), and it makes a message unreadable without its schema.

## Design notes

For a single team with one Go backend and a React frontend, REST/JSON with a published
OpenAPI description is the boring, correct default (lesson ts.services.openapi on the
TypeScript side). gRPC earns its toolchain when many services in several languages call
each other a lot, or when you need streaming between backends.

Before introducing a broker, check what PostgreSQL already gives you: a jobs table with
`FOR UPDATE SKIP LOCKED` for work queues, `LISTEN/NOTIFY` to wake workers (lesson 6),
and an outbox table for events that must leave the database exactly when a transaction
commits. Add Kafka, RabbitMQ or NATS when you need what they are built for: fan-out to
many independent services, replaying history, or throughput beyond one database.

Whatever carries the messages, the consumer design is the same: idempotent handlers,
deduplication in the same transaction as the effect, explicit handling of reordering,
and a dead-letter destination for messages that keep failing.
