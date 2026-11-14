-- random() keeps its state per session. setseed() fixes the starting
-- point, so the same statements after the same seed produce the same rows.
SELECT setseed(0.42);
CREATE TABLE draw1 AS
SELECT i, random(1, 6) AS die FROM generate_series(1, 100000) AS i;

SELECT setseed(0.42);
CREATE TABLE draw2 AS
SELECT i, random(1, 6) AS die FROM generate_series(1, 100000) AS i;

-- No new seed: the sequence continues where draw2 left it.
CREATE TABLE draw3 AS
SELECT i, random(1, 6) AS die FROM generate_series(1, 100000) AS i;

-- Compare tables rather than print values: the docs promise repeatability
-- within a session, not the same numbers on another version.
SELECT NOT EXISTS (TABLE draw1 EXCEPT TABLE draw2) AS same_seed_same_rows,
       NOT EXISTS (TABLE draw1 EXCEPT TABLE draw3) AS no_reseed_same_rows,
       (SELECT min(die) || '..' || max(die) FROM draw1) AS die_range;

-- Dates in, timestamptz out: generate_series has no date variant.
SELECT pg_typeof(d) AS returns, d::date AS day
FROM generate_series(date '2026-11-01', date '2026-11-03', interval '1 day') AS d;
