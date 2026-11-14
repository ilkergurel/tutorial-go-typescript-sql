-- Three tables that differ only in size. Each primary key is a B-tree index.
CREATE TABLE small  (id int PRIMARY KEY, pad text);
CREATE TABLE medium (id int PRIMARY KEY, pad text);
CREATE TABLE large  (id int PRIMARY KEY, pad text);
INSERT INTO small  SELECT g, 'row ' || g FROM generate_series(1, 100) AS g;
INSERT INTO medium SELECT g, 'row ' || g FROM generate_series(1, 10000) AS g;
INSERT INTO large  SELECT g, 'row ' || g FROM generate_series(1, 200000) AS g;

-- Warm-up: plan each lookup once, so the measurements below count only
-- the pages the lookup itself visits, not the planner's catalog reads.
SELECT s.pad, m.pad, l.pad
FROM small s, medium m, large l
WHERE s.id = 42 AND m.id = 42 AND l.id = 42;

-- One page per tree level, plus one heap page for the row itself.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT pad FROM small WHERE id = 42;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT pad FROM medium WHERE id = 42;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT pad FROM large WHERE id = 42;

-- A range descends once, then walks the linked leaf pages sideways.
EXPLAIN (COSTS OFF)
SELECT pad FROM large WHERE id BETWEEN 1000 AND 1999;
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT pad FROM large WHERE id BETWEEN 1000 AND 1999;
