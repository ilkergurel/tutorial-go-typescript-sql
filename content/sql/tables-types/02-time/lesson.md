---
id: sql.tables-types.time
title: "timestamptz vs timestamp, date, interval; time zones done right"
track: sql
module: tables-types
order: 2
level: basic
minutes: 14
since: pg18
prerequisites: [sql.tables-types.text-numbers]
compare_with: [sql.querying.patterns, go.postgres.querying, sql.modeling.temporal]
tags: [types, timestamptz, timestamp, time-zones, interval, date]
summary: >
  timestamptz stores an instant and shows it in the reader's zone; timestamp stores
  wall-clock digits and silently drops any offset you give it. Use timestamptz for
  events, zone names like Europe/Istanbul instead of offsets, and AT TIME ZONE
  wherever a calendar day matters.
examples:
  - id: instant-vs-wallclock
    file: examples/01_instant_vs_wallclock.sql
    title: "The same input in timestamptz and timestamp, read from two cities"
    runtime: sql
    check: exact
    annotations:
      - lines: "1"
        note: >
          The session's TimeZone decides how timestamptz values are displayed and
          how input without an offset is read. The server default differs between
          machines (it is Europe/Istanbul on this one), so examples and
          applications set it explicitly.
      - lines: "5-6"
        note: >
          Both types take 8 bytes with microsecond resolution. Despite its name,
          timestamp with time zone stores no zone: it converts the input to UTC
          and converts back to the session's zone on output.
      - lines: "11-12"
        note: >
          The docs say it plainly: in a timestamp without time zone, PostgreSQL
          silently ignores any time zone indication. at_local keeps 09:00 and
          08:30 as typed; the +03 and +01 are gone.
      - lines: "19"
        note: >
          The review started at 07:30 UTC, after the standup at 06:00 UTC, but
          its wall-clock digits are smaller, so it sorts first. Every
          comparison, range query and duration on at_local has the same problem.
      - lines: "22-23"
        note: >
          Same rows, Berlin session: at_tz now reads 07:00+01 and 08:30+01, the
          same instants in Berlin time. at_local can't follow, because it never
          knew which zone its digits belonged to.
  - id: zones
    file: examples/02_zones.sql
    title: "Calendar days, zone history, and offsets that mean the opposite"
    runtime: sql
    check: exact
    annotations:
      - lines: "11-12"
        note: >
          timestamptz AT TIME ZONE 'Europe/Istanbul' gives the wall-clock time in
          Istanbul (a timestamp), and only then is it cut to a date. Order 2 was
          placed on the 12th in UTC but on the 13th in Istanbul: a daily sales
          report that uses ::date silently follows the server's zone instead.
      - lines: "17-18"
        note: >
          The other direction: timestamp AT TIME ZONE reads the digits as
          Istanbul time and returns the instant, shown here in UTC. Noon in
          January was 10:00 UTC in 2016 (+02) and is 09:00 UTC now (+03). A zone
          name gets both right; a hard-coded +03 gets 2016 wrong.
      - lines: "22-24"
        note: >
          The zone name and INTERVAL '+03:00' agree on 09:00. 'UTC+3' is a
          POSIX-style specification, whose sign means the opposite of ISO 8601:
          three hours west of Greenwich, so 03:00. No error, just a wrong answer.
      - lines: "28-30"
        note: >
          Same expression, two answers: 03:00 in a UTC session, 09:00 in an
          Istanbul session. In PostgreSQL 18 the current zone's own IANA
          abbreviations are recognized first, and tzdata's abbreviation for Turkey
          is literally "+03"; elsewhere the string falls through to POSIX rules.
          Spell zones as names or intervals and the question never comes up.
  - id: intervals
    file: examples/03_intervals.sql
    title: "date and interval arithmetic, DST, and what now() means"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          date minus date is an integer number of days, and date plus an integer
          is a date. No time of day and no zone: the right type for birthdays,
          due dates and holidays.
      - lines: "7"
        note: >
          date + interval returns a timestamp. January 31 plus one month is
          February 28: the docs specify that the last day of the month is used
          when the day doesn't exist. Go's AddDate normalizes instead (October 31
          plus one month is December 1).
      - lines: "8"
        note: >
          age() answers in years, months and days, which is what a human means
          by "how old"; plain subtraction of timestamps would answer in days and
          hours.
      - lines: "13-15"
        note: >
          '1 day' means the same local time tomorrow (12:00); '24 hours' means
          elapsed time (13:00, because an hour was skipped). The subtraction shows
          the day lasted 23:00:00. That is why interval stores months, days and
          microseconds as separate fields.
      - lines: "19-20"
        note: >
          now(), current_timestamp and transaction_timestamp() are the
          transaction's start time. A batch insert with DEFAULT now() stamps every
          row identically, however long it runs; clock_timestamp() is the actual
          time at each call.
use_cases:
  - "timestamptz: created_at, paid_at, sent_at, log lines: anything that happened at an instant"
  - "date: birthdays, due dates, public holidays, billing days: a calendar day with no time or zone"
  - "timestamp plus a zone-name column: future appointments promised in local time ('Mondays 09:00 in Istanbul'), where the wall clock is the contract even if the zone's rules change"
  - "interval: durations, subscription lengths ('1 month'), retry back-offs, retention windows"
strengths:
  - "One stored instant reads correctly in Istanbul and in Berlin; the conversion happens at the edges, like Java's Instant or Go's time.Time"
  - "IANA zone names carry history: Europe/Istanbul knows it was +02 in the winter of 2016 and +03 since"
  - "interval holds calendar units and elapsed time in one value; Go splits them into AddDate and time.Duration"
  - "Both timestamp types cost 8 bytes with microsecond resolution, so the correct choice is free"
weaknesses:
  - "The name misleads: timestamp with time zone stores no zone. Keep the user's zone in its own column if you need it"
  - "Display depends on each session's TimeZone, so two clients print the same row differently unless they set it"
  - "Comparing timestamp with timestamptz converts implicitly through the session's zone, so results can change with a SET"
  - "Zone strings are parsed leniently: 'UTC+3' is accepted and means UTC−3"
pitfalls:
  - title: "timestamp for events"
    detail: "Offsets in the input are dropped and DEFAULT now() is stored as the server's local wall clock. The moment a second zone is involved (a server, a colleague, Docker running in UTC), the values become ambiguous."
    bad: "created_at timestamp NOT NULL DEFAULT now()"
    good: "created_at timestamptz NOT NULL DEFAULT now()"
  - title: "Offsets instead of zone names"
    detail: "'+03' or INTERVAL '+03:00' is right for Istanbul today and wrong for its winters before 2016; 'UTC+3' is POSIX syntax for UTC−3; a bare '+03' changes meaning with the session's zone. Store and use names: Europe/Istanbul."
  - title: "Daily reports with ::date"
    detail: "Casting timestamptz to date uses the session's zone, so evening orders in Istanbul land on the wrong day when the server or pool runs in UTC."
    bad: "GROUP BY placed_at::date"
    good: "GROUP BY (placed_at AT TIME ZONE 'Europe/Istanbul')::date"
  - title: "now() in long transactions"
    detail: "now() is frozen at transaction start. A job that runs for ten minutes in one transaction stamps everything with its first second; use clock_timestamp() where the actual time matters."
  - title: "Future local appointments stored as timestamptz"
    detail: "The instant is computed with today's zone rules. When a country changes them, as Turkey did in 2016 by staying on +03 for good, appointments stored as instants drift by an hour. Store the local time plus the zone name and compute the instant when needed."
  - title: "time with time zone"
    detail: "A time of day with an offset but no date can't account for DST. The docs explicitly do not recommend timetz; use timestamptz, or time plus a zone name."
quiz:
  - q: "TimeZone is Europe/Istanbul. You insert '2026-11-13 10:00+01' into a timestamp (without time zone) column. What is stored?"
    options: ["2026-11-13 10:00:00", "2026-11-13 12:00:00", "2026-11-13 09:00:00 UTC"]
    answer: 0
    why: "A timestamp without time zone ignores any zone in its input; only the digits are kept."
  - q: "What does a timestamptz column store?"
    options: ["The instant plus the zone it was written in", "The instant (internally UTC); the zone is applied on output", "The local wall-clock digits"]
    answer: 1
    why: "Input is converted to UTC and the original zone is not retained; output converts to the session's TimeZone."
  - q: "The session's TimeZone is UTC. Which expression gives the Istanbul calendar day of placed_at (a timestamptz)?"
    options: ["placed_at::date", "(placed_at AT TIME ZONE 'UTC+3')::date", "(placed_at AT TIME ZONE 'Europe/Istanbul')::date"]
    answer: 2
    why: "AT TIME ZONE with a zone name yields Istanbul wall-clock time first. ::date alone uses UTC, and 'UTC+3' is POSIX for UTC−3."
  - q: "Why can ts + interval '1 day' differ from ts + interval '24 hours' for a timestamptz?"
    options: ["It can't; they are equal", "Across a DST change a calendar day lasts 23 or 25 hours", "interval rounds to whole days"]
    answer: 1
    why: "'1 day' keeps the local time of day; '24 hours' adds elapsed time. They differ exactly when the day isn't 24 hours long."
further_reading:
  - title: "PostgreSQL docs: Date/Time Types"
    url: "https://www.postgresql.org/docs/current/datatype-datetime.html"
  - title: "PostgreSQL docs: Date/Time Functions and Operators (AT TIME ZONE)"
    url: "https://www.postgresql.org/docs/current/functions-datetime.html"
  - title: "PostgreSQL docs: Date/Time Configuration Files (time zone abbreviations)"
    url: "https://www.postgresql.org/docs/current/datetime-config-files.html"
  - title: "PostgreSQL docs: POSIX Time Zone Specifications"
    url: "https://www.postgresql.org/docs/current/datetime-posix-timezone-specs.html"
---

## Concept

There are two different things people call "a time", and Postgres has a type for each:

- An **instant**: the moment a payment went through, the same moment everywhere on
  Earth. That is `timestamptz` (`timestamp with time zone`). Input is converted to UTC,
  output is converted to the session's `TimeZone`. No zone is stored.
- A **wall-clock reading**: "09:00", with no idea which clock it was read from. That is
  `timestamp` (`timestamp without time zone`). It stores the digits you gave it and
  silently ignores any offset in the input.

Around them: `date` (a calendar day, 4 bytes), `interval` (a duration kept as months, days
and microseconds, because months and days have no fixed length), and the `AT TIME ZONE`
operator that converts between instants and wall clocks for a named zone.

The most common Postgres data-type mistake is storing events in `timestamp`. It works
while every server, client and user shares one zone, and quietly corrupts ordering and
durations the day one of them doesn't.

## Coming from C/C++/Java/Python/JS

- **Java:** `timestamptz` ≈ `Instant`, `timestamp` ≈ `LocalDateTime`, `date` ≈
  `LocalDate`, `interval` ≈ `Period` and `Duration` in one value.
- **Python:** aware vs naive `datetime`. A naive datetime sent to a `timestamptz` column is
  read in the session's zone, which is usually not what you meant.
- **Go:** `time.Time` is an instant with a `Location` for display, which is `timestamptz`.
  Go has no zoneless wall-clock type. `time.Duration` is `'24 hours'`; `AddDate` is the
  calendar arithmetic, but it normalizes 2026-01-31 + 1 month to March 3 where Postgres
  clamps to February 28.
- **JS:** a `Date` is milliseconds since the epoch, an instant; `toString()` shows it in
  the browser's zone, as `timestamptz` output follows the session's.
- **C:** `time_t` is the instant, `struct tm` the wall clock.

## Design notes

Events get `timestamptz`, without exception. Set the session's `TimeZone` per connection
(`UTC` for services is a good default) so nothing depends on the server's configuration;
this machine's server default is `Europe/Istanbul`, a Docker image's is often UTC.

Keep a user's zone as a name (`Europe/Istanbul`), never as an offset. Names carry the
rule history; offsets don't, and string offsets are parsed by rules (POSIX signs, and in
PostgreSQL 18 the current zone's abbreviations) that surprise everyone once.

Future appointments are the exception to "always timestamptz". A meeting promised for
09:00 Istanbul time next spring is a wall-clock promise; store `timestamp` plus the zone
name, and compute the instant when you need it. Turkey's 2016 switch to permanent +03 is the
textbook case of rules changing under stored instants.

Reports by day, week or month: convert first, then truncate:
`date_trunc('day', placed_at AT TIME ZONE 'Europe/Istanbul')`. The whole lesson in one
habit: decide whose calendar you are counting in, and say it in the query.
