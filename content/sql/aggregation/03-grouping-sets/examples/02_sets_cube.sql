CREATE TABLE test_runs (id int PRIMARY KEY, suite text NOT NULL, status text NOT NULL);
INSERT INTO test_runs VALUES
  (1, 'auth', 'passed'), (2, 'auth', 'failed'), (3, 'auth', 'passed'),
  (4, 'search', 'failed'), (5, 'search', 'failed');

-- Two unrelated breakdowns and the total, as one result. Without grouping
-- sets this is three GROUP BY queries glued together with UNION ALL.
SELECT suite, status, count(*) AS runs
FROM test_runs
GROUP BY GROUPING SETS ((suite), (status), ())
ORDER BY GROUPING(suite, status), suite, status;

-- CUBE (a, b) is every subset: (a, b), (a), (b) and ().
SELECT GROUPING(suite, status) AS level, suite, status, count(*) AS runs
FROM test_runs
GROUP BY CUBE (suite, status)
ORDER BY level, suite, status;

-- One scan of the table feeds all the groupings.
EXPLAIN (COSTS OFF)
SELECT suite, status, count(*)
FROM test_runs
GROUP BY GROUPING SETS ((suite), (status), ());
