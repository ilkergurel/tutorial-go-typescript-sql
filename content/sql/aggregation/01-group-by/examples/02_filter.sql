CREATE TABLE test_runs (
  id          int PRIMARY KEY,
  suite       text NOT NULL,
  status      text NOT NULL,
  duration_ms int NOT NULL         -- 0 for skipped runs
);
INSERT INTO test_runs VALUES
  (1, 'auth',   'passed',  120), (2, 'auth',   'failed',  340),
  (3, 'auth',   'passed',  110), (4, 'export', 'skipped',   0),
  (5, 'export', 'passed',  900), (6, 'search', 'failed',   75),
  (7, 'search', 'failed',   80), (8, 'search', 'skipped',   0);

-- One pass over the table, one row per suite, one column per status.
SELECT suite,
       count(*) FILTER (WHERE status = 'passed')  AS passed,
       count(*) FILTER (WHERE status = 'failed')  AS failed,
       count(*) FILTER (WHERE status = 'skipped') AS skipped,
       count(*)                                   AS total
FROM test_runs
GROUP BY suite
ORDER BY suite;

-- The same counts the portable way: CASE inside the aggregate.
SELECT suite,
       sum(CASE WHEN status = 'passed' THEN 1 ELSE 0 END) AS passed,
       count(CASE WHEN status = 'failed' THEN 1 END)      AS failed
FROM test_runs
GROUP BY suite
ORDER BY suite;

-- count() returns bigint, and bigint / bigint truncates toward zero.
SELECT suite,
       count(*) FILTER (WHERE status = 'passed') / count(*) AS int_ratio,
       round(100.0 * count(*) FILTER (WHERE status = 'passed') / count(*), 1)
         AS pass_pct
FROM test_runs
GROUP BY suite
ORDER BY suite;

-- A FILTER that matches no row: count says 0, sum says NULL.
SELECT suite,
       count(*)         FILTER (WHERE status = 'failed') AS failed_runs,
       sum(duration_ms) FILTER (WHERE status = 'failed') AS failed_ms,
       coalesce(sum(duration_ms) FILTER (WHERE status = 'failed'), 0)
         AS failed_ms_or_0
FROM test_runs
GROUP BY suite
ORDER BY suite;
