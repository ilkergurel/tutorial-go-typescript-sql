-- Plans print timestamptz constants in the session's time zone: pin it.
SET LOCAL TimeZone = 'Europe/Istanbul';
CREATE TABLE readings (
  sensor_id int NOT NULL,
  at        timestamptz NOT NULL,
  value     int NOT NULL
);
-- Append-only: 4 sensors, one reading each per minute, oldest rows first.
INSERT INTO readings
SELECT g % 4 + 1, timestamptz '2026-01-01 00:00+03' + (g / 4) * interval '1 minute',
       (g * 7919) % 1000
FROM generate_series(0, 19999) AS g;
ANALYZE readings;

-- How closely each column follows the physical row order (1 = exactly).
SELECT attname, round(correlation::numeric, 2) AS correlation
FROM pg_stats
WHERE schemaname = current_schema() AND tablename = 'readings'
ORDER BY attname;

-- BRIN: one min/max summary per range of 8 table pages.
CREATE INDEX readings_at_brin ON readings USING brin (at) WITH (pages_per_range = 8);
CREATE INDEX readings_at_btree ON readings (at);
SELECT pg_relation_size('readings') / 8192 AS table_pages,
       pg_relation_size('readings_at_brin') / 8192 AS brin_pages,
       pg_relation_size('readings_at_btree') / 8192 AS btree_pages;
DROP INDEX readings_at_btree;

EXPLAIN (COSTS OFF)
SELECT count(*) FROM readings WHERE at >= '2026-01-02 10:00' AND at < '2026-01-02 11:00';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT count(*) FROM readings WHERE at >= '2026-01-02 10:00' AND at < '2026-01-02 11:00';

-- Skip scan (PG 18): the index leads with sensor_id, the query never names it.
DROP INDEX readings_at_brin;
CREATE INDEX readings_sensor_at ON readings (sensor_id, at);
SELECT pg_relation_size('readings_sensor_at') / 8192 AS index_pages;
EXPLAIN (COSTS OFF)
SELECT count(*) FROM readings WHERE at >= '2026-01-02 10:00' AND at < '2026-01-02 11:00';
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT count(*) FROM readings WHERE at >= '2026-01-02 10:00' AND at < '2026-01-02 11:00';
