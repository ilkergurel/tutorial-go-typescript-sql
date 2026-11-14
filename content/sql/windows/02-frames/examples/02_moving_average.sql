CREATE TABLE nightly_builds (
  built_on date PRIMARY KEY,
  minutes  int  NOT NULL
);
INSERT INTO nightly_builds VALUES
  ('2026-11-02', 10), ('2026-11-03', 12), ('2026-11-04', 11),
  ('2026-11-05', 30), ('2026-11-06', 13),
  -- no builds on the weekend of 11-07 and 11-08
  ('2026-11-09', 14), ('2026-11-10', 12);

-- "Last three builds" (ROWS) vs "last three days" (RANGE with an interval).
-- count(*) over the same frame shows how many rows each average used.
SELECT built_on, minutes,
       round(avg(minutes) OVER last3_rows, 1) AS avg_3_builds,
       count(*)           OVER last3_rows     AS n_builds,
       round(avg(minutes) OVER last3_days, 1) AS avg_3_days,
       count(*)           OVER last3_days     AS n_days
FROM nightly_builds
WINDOW last3_rows AS (ORDER BY built_on ROWS BETWEEN 2 PRECEDING AND CURRENT ROW),
       last3_days AS (ORDER BY built_on
                      RANGE BETWEEN '2 days' PRECEDING AND CURRENT ROW)
ORDER BY built_on;

-- A baseline that excludes the current row: the three builds before it.
SELECT built_on, minutes, baseline,
       minutes > 2 * baseline AS spike
FROM (
  SELECT built_on, minutes,
         round(avg(minutes) OVER (ORDER BY built_on
                                  ROWS BETWEEN 3 PRECEDING AND 1 PRECEDING), 1)
           AS baseline
  FROM nightly_builds
) AS b
ORDER BY built_on;

-- RANGE with an offset measures distance on one value, so it needs exactly
-- one ORDER BY column.
SELECT built_on,
       avg(minutes) OVER (ORDER BY built_on, minutes
                          RANGE BETWEEN '2 days' PRECEDING AND CURRENT ROW)
FROM nightly_builds;
