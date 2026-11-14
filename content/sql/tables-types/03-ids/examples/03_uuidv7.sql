CREATE TABLE events (
  id uuid PRIMARY KEY DEFAULT uuidv7(),  -- time-ordered (PostgreSQL 18)
  n  int  NOT NULL
);
CREATE TABLE events_v4 (
  id uuid PRIMARY KEY DEFAULT uuidv4(),  -- random; same as gen_random_uuid()
  n  int  NOT NULL
);
INSERT INTO events    (n) SELECT g FROM generate_series(1, 1000) AS g;
INSERT INTO events_v4 (n) SELECT g FROM generate_series(1, 1000) AS g;

-- Generated ids differ on every run, so look at their properties.
-- Does ORDER BY id give back the insertion order?
SELECT 'uuidv7' AS generator,
       array_agg(n ORDER BY id) = array_agg(n ORDER BY n) AS id_order_is_insert_order
FROM events
UNION ALL
SELECT 'uuidv4', array_agg(n ORDER BY id) = array_agg(n ORDER BY n)
FROM events_v4
ORDER BY generator;

-- A v7 id carries its creation time; a v4 id carries nothing but randomness.
SELECT uuid_extract_version(v7) AS v7_version,
       uuid_extract_timestamp(v7) BETWEEN now() - interval '1 minute'
                                      AND now() + interval '1 minute' AS v7_time_is_now,
       uuid_extract_version(v4) AS v4_version,
       uuid_extract_timestamp(v4) AS v4_time
FROM (SELECT uuidv7() AS v7, uuidv4() AS v4) AS ids;

-- 16 bytes as uuid, 36 characters as text. And shift backdates the
-- embedded time, e.g. when backfilling ids for historic rows.
SELECT pg_column_size(uuidv7()) AS uuid_bytes,
       length(uuidv7()::text) AS text_chars,
       uuidv7(interval '-1 day') < (SELECT id FROM events ORDER BY id LIMIT 1) AS backdated_sorts_first;
