-- The same 10000 rows and the same updates, in two tables: one packed full
-- (the default fillfactor, 100), one that leaves half of every page free.
CREATE TEMP TABLE packed (id int PRIMARY KEY, hits int NOT NULL, note text NOT NULL);
CREATE TEMP TABLE roomy (id int PRIMARY KEY, hits int NOT NULL, note text NOT NULL)
  WITH (fillfactor = 50);
INSERT INTO packed SELECT g, 0, repeat('x', 100) FROM generate_series(1, 10000) AS g;
INSERT INTO roomy SELECT * FROM packed;
CREATE TABLE sizes AS
  SELECT pg_relation_size('packed') AS packed, pg_relation_size('roomy') AS roomy;

-- hits has no index, so an update that finds room on its own page is HOT.
-- No VACUUM anywhere below.
UPDATE packed SET hits = hits + 1;
UPDATE packed SET hits = hits + 1;
UPDATE packed SET hits = hits + 1;
UPDATE roomy SET hits = hits + 1;
UPDATE roomy SET hits = hits + 1;
UPDATE roomy SET hits = hits + 1;

SELECT round(pg_relation_size('packed')::numeric / packed, 1) AS packed_growth,
       round(pg_relation_size('roomy')::numeric / roomy, 1) AS roomy_growth,
       round(roomy::numeric / packed, 1) AS roomy_start_vs_packed
FROM sizes;
