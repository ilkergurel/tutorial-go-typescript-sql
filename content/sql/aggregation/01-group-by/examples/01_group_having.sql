-- LEARN(sql.aggregation.group-by): one row per group; WHERE filters rows, HAVING filters groups.
CREATE TABLE test_runs (
  id          int PRIMARY KEY,
  suite       text NOT NULL,
  status      text NOT NULL,
  duration_ms int              -- NULL: the run was skipped, nothing was timed
);
INSERT INTO test_runs VALUES
  (1, 'auth',   'passed',  120), (2, 'auth',   'failed',  340),
  (3, 'auth',   'passed',  110), (4, 'export', 'skipped', NULL),
  (5, 'export', 'passed',  900), (6, 'search', 'failed',   75),
  (7, 'search', 'failed',   80), (8, 'search', 'skipped', NULL);

-- One output row per suite. count(*) counts rows; count(col) and avg skip NULLs.
SELECT suite,
       count(*)                AS runs,
       count(duration_ms)      AS timed,
       round(avg(duration_ms)) AS avg_ms
FROM test_runs
GROUP BY suite
ORDER BY suite;

-- WHERE picks rows before grouping; HAVING picks groups after aggregation.
SELECT suite, count(*) AS failures
FROM test_runs
WHERE status = 'failed'
GROUP BY suite
HAVING count(*) >= 2
ORDER BY suite;

-- No GROUP BY: all input rows form one group, so even zero rows give one row.
SELECT count(*) AS runs, sum(duration_ms) AS total_ms
FROM test_runs
WHERE suite = 'billing';

-- With GROUP BY, zero input rows make zero groups.
SELECT suite, count(*) AS runs
FROM test_runs
WHERE suite = 'billing'
GROUP BY suite;

-- Every selected column must be grouped or aggregated. Which status per suite?
SELECT suite, status, count(*) FROM test_runs GROUP BY suite;
