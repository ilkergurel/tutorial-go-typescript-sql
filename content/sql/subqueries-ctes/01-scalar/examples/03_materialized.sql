CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int NOT NULL,
  passed       boolean NOT NULL
);

-- Referenced once and free of side effects: inlined (PG 12+). The outer
-- WHERE reaches the table, so the primary key index can answer it.
EXPLAIN (COSTS OFF)
WITH failed AS (SELECT * FROM test_runs WHERE NOT passed)
SELECT * FROM failed WHERE id = 4;

-- MATERIALIZED: the CTE is computed on its own first, every failed run,
-- and the outer filter is applied to that result.
EXPLAIN (COSTS OFF)
WITH failed AS MATERIALIZED (SELECT * FROM test_runs WHERE NOT passed)
SELECT * FROM failed WHERE id = 4;

-- Referenced twice: materialized by default, computed once and read twice.
EXPLAIN (COSTS OFF)
WITH failed AS (SELECT * FROM test_runs WHERE NOT passed)
SELECT a.id, b.id
FROM failed a
JOIN failed b ON b.test_case_id = a.test_case_id AND b.id > a.id;

-- NOT MATERIALIZED inlines it anyway: two scans of test_runs, no CTE Scan.
EXPLAIN (COSTS OFF)
WITH failed AS NOT MATERIALIZED (SELECT * FROM test_runs WHERE NOT passed)
SELECT a.id, b.id
FROM failed a
JOIN failed b ON b.test_case_id = a.test_case_id AND b.id > a.id;
