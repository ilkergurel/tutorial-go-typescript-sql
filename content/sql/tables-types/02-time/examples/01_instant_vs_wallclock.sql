SET TimeZone = 'Europe/Istanbul';

CREATE TABLE meetings (
  title    text        NOT NULL,
  at_tz    timestamptz NOT NULL,  -- an instant, stored as UTC
  at_local timestamp   NOT NULL   -- a wall-clock reading with no zone
);

-- The same literals, with explicit offsets, go into both columns.
INSERT INTO meetings VALUES
  ('Istanbul standup', '2026-11-13 09:00:00+03', '2026-11-13 09:00:00+03'),
  ('Berlin review',    '2026-11-13 08:30:00+01', '2026-11-13 08:30:00+01');

-- timestamptz shows each instant in the session's zone; timestamp kept the
-- digits and silently dropped the offsets.
SELECT title, at_tz, at_local FROM meetings ORDER BY at_tz;

-- So the wall-clock column puts the later meeting first.
SELECT title FROM meetings ORDER BY at_local;

-- A colleague in Berlin reads the same rows: timestamptz follows the reader.
SET TimeZone = 'Europe/Berlin';
SELECT title, at_tz, at_local FROM meetings ORDER BY at_tz;
