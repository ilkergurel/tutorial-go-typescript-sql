SET TimeZone = 'UTC';

CREATE TABLE orders (id int PRIMARY KEY, placed_at timestamptz NOT NULL);
INSERT INTO orders VALUES
  (1, '2026-11-12 20:30:00+00'),   -- 23:30 in Istanbul
  (2, '2026-11-12 21:30:00+00');   -- 00:30 on the 13th in Istanbul

-- "Which day?" depends on whose calendar. ::date uses the session's zone;
-- AT TIME ZONE first turns the instant into Istanbul wall-clock time.
SELECT id,
       placed_at::date AS utc_day,
       (placed_at AT TIME ZONE 'Europe/Istanbul')::date AS istanbul_day
FROM orders
ORDER BY id;

-- Zone names carry history: Turkey's winter offset was +02 until 2016.
SELECT timestamp '2016-01-15 12:00' AT TIME ZONE 'Europe/Istanbul' AS winter_2016,
       timestamp '2026-01-15 12:00' AT TIME ZONE 'Europe/Istanbul' AS winter_2026;

-- Offsets written as zone strings are a trap. 'UTC+3' is POSIX syntax and
-- means three hours WEST of Greenwich. An interval means what it says.
SELECT timestamptz '2026-11-13 06:00+00' AT TIME ZONE 'Europe/Istanbul' AS by_name,
       timestamptz '2026-11-13 06:00+00' AT TIME ZONE INTERVAL '+03:00' AS by_interval,
       timestamptz '2026-11-13 06:00+00' AT TIME ZONE 'UTC+3' AS posix_utc_plus_3;

-- A bare '+03' even depends on the session (PostgreSQL 18): POSIX (UTC-3)
-- here, but Europe/Istanbul's own abbreviation is "+03", and it wins there.
SELECT timestamptz '2026-11-13 06:00+00' AT TIME ZONE '+03' AS bare_in_utc_session;
SET TimeZone = 'Europe/Istanbul';
SELECT timestamptz '2026-11-13 06:00+00' AT TIME ZONE '+03' AS bare_in_istanbul_session;
