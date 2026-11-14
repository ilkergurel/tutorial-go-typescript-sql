-- LEARN(sql.windows.gaps-islands): number the rows, subtract, and consecutive rows share a key.
CREATE TABLE uploads (
  day date PRIMARY KEY            -- days the CI agent uploaded its results
);
INSERT INTO uploads VALUES
  ('2026-11-01'), ('2026-11-02'), ('2026-11-03'),
  ('2026-11-06'), ('2026-11-07'),
  ('2026-11-10'), ('2026-11-11'), ('2026-11-12'), ('2026-11-13');

-- Gaps: look at the next day; a jump of more than one day is a hole.
SELECT day + 1 AS gap_from, next_day - 1 AS gap_to,
       next_day - day - 1 AS missing_days
FROM (SELECT day, lead(day) OVER (ORDER BY day) AS next_day FROM uploads) AS d
WHERE next_day - day > 1
ORDER BY gap_from;

-- Islands: day and row number both grow by 1 inside a streak, so their
-- difference stays constant until a gap shifts it.
SELECT day,
       row_number() OVER (ORDER BY day) AS rn,
       day - row_number() OVER (ORDER BY day)::int AS island
FROM uploads
ORDER BY day;

-- Group on that constant: one row per streak.
SELECT min(day) AS first_day, max(day) AS last_day, count(*) AS days
FROM (
  SELECT day, day - row_number() OVER (ORDER BY day)::int AS island
  FROM uploads
) AS s
GROUP BY island
ORDER BY first_day;
