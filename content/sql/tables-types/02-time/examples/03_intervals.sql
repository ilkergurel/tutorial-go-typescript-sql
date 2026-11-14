-- date is a calendar day (4 bytes). date - date is a whole number of days.
SELECT date '2026-11-13' - date '2026-01-01' AS days_since_new_year,
       date '2026-11-13' + 7 AS one_week_later;

-- interval keeps months, days and microseconds apart, because months and
-- days have no fixed length. Month arithmetic clamps to the month's end.
SELECT date '2026-01-31' + interval '1 month' AS jan31_plus_1_month,
       age(date '2026-11-13', date '1990-05-20') AS age;

-- Turkey has had no DST since 2016, but Berlin moves its clocks from 02:00
-- to 03:00 on 2026-03-29, so that calendar day is 23 hours long.
SET TimeZone = 'Europe/Berlin';
SELECT timestamptz '2026-03-28 12:00' + interval '1 day'   AS plus_1_day,
       timestamptz '2026-03-28 12:00' + interval '24 hours' AS plus_24_hours,
       timestamptz '2026-03-29 12:00' - timestamptz '2026-03-28 12:00' AS elapsed;

-- now() is the transaction's start time, the same in every statement of it.
-- clock_timestamp() reads the clock each time it is called.
SELECT now() = transaction_timestamp() AS now_is_tx_start,
       clock_timestamp() > now() AS clock_has_moved_on;
