-- LEARN(sql.windows.over): a window function computes over related rows and keeps every row.
CREATE TABLE test_runs (
  id          int  PRIMARY KEY,
  suite       text NOT NULL,
  test_case   text NOT NULL,
  duration_ms int  NOT NULL
);
INSERT INTO test_runs VALUES
  (1, 'auth',   'Giriş',            120),
  (2, 'auth',   'Çıkış',             80),
  (3, 'auth',   'Şifre sıfırlama',  400),
  (4, 'export', 'CSV dışa aktarım', 900),
  (5, 'export', 'PDF rapor',        300);

-- GROUP BY collapses each suite into one row; the tests are gone.
SELECT suite, count(*) AS runs, sum(duration_ms) AS suite_ms
FROM test_runs
GROUP BY suite
ORDER BY suite;

-- The same aggregates with OVER keep every row and attach the group's figures.
SELECT suite, test_case, duration_ms,
       count(*)         OVER (PARTITION BY suite) AS runs,
       sum(duration_ms) OVER (PARTITION BY suite) AS suite_ms,
       round(100.0 * duration_ms / sum(duration_ms) OVER (PARTITION BY suite), 1)
                                                  AS pct_of_suite,
       sum(duration_ms) OVER ()                   AS all_ms
FROM test_runs
ORDER BY suite, id;

-- Filtering on a window result needs a subquery: compute first, filter outside.
SELECT suite, test_case, duration_ms, suite_avg
FROM (
  SELECT suite, test_case, duration_ms,
         round(avg(duration_ms) OVER (PARTITION BY suite)) AS suite_avg
  FROM test_runs
) AS t
WHERE duration_ms > suite_avg
ORDER BY suite, test_case;

-- WHERE runs before any window is computed, so it can't refer to one.
SELECT test_case FROM test_runs
WHERE duration_ms > avg(duration_ms) OVER (PARTITION BY suite);
