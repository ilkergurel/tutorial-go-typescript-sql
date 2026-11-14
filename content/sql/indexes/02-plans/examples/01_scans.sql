-- LEARN(sql.indexes.plans): one table, the planner's whole scan menu.
CREATE TABLE readings (
  id        int PRIMARY KEY,
  sensor_id int NOT NULL,
  value     int NOT NULL
);
INSERT INTO readings
SELECT g, g % 200, (g * 7919) % 10000 FROM generate_series(1, 20000) AS g;
CREATE INDEX readings_value ON readings (value);
CREATE INDEX readings_sensor ON readings (sensor_id);
ANALYZE readings;

-- Most of the table: read all of it.
EXPLAIN (COSTS OFF) SELECT * FROM readings WHERE value > 1000;

-- One row: descend the index, then fetch the row from the table.
EXPLAIN (COSTS OFF) SELECT * FROM readings WHERE id = 4242;

-- Only indexed columns are needed: the table may not be visited at all.
EXPLAIN (COSTS OFF) SELECT id FROM readings WHERE id BETWEEN 10000 AND 10099;

-- 200 scattered rows: collect their locations, then visit pages in order.
EXPLAIN (COSTS OFF) SELECT * FROM readings WHERE value < 100;

-- Two conditions on two indexes, combined as bitmaps.
EXPLAIN (COSTS OFF) SELECT * FROM readings WHERE value < 20 OR sensor_id = 7;

-- The index already delivers rows in order: no sort, and LIMIT stops it early.
EXPLAIN (COSTS OFF) SELECT * FROM readings ORDER BY value LIMIT 5;

-- "May not": these rows are not yet known visible to everyone, so each one
-- is still checked in the table.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT id FROM readings WHERE id BETWEEN 10000 AND 10099;
