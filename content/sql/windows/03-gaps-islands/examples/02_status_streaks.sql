CREATE TABLE test_runs (
  test_case text NOT NULL,
  build     int  NOT NULL,           -- CI build number; not every build runs every test
  status    text NOT NULL,
  PRIMARY KEY (test_case, build)
);
INSERT INTO test_runs VALUES
  ('Giriş', 101, 'passed'), ('Giriş', 102, 'passed'), ('Giriş', 103, 'failed'),
  ('Giriş', 104, 'failed'), ('Giriş', 105, 'passed'), ('Giriş', 106, 'passed'),
  ('Çıkış', 101, 'passed'), ('Çıkış', 103, 'failed'), ('Çıkış', 105, 'failed'),
  ('Çıkış', 106, 'failed');

-- Step 1: flag each row that differs from the previous one (lag), and
-- step 2: a running sum of the flags numbers the islands.
CREATE VIEW numbered AS
SELECT test_case, build, status, starts,
       sum(starts) OVER (PARTITION BY test_case ORDER BY build) AS island
FROM (
  SELECT test_case, build, status,
         CASE WHEN status IS DISTINCT FROM
                   lag(status) OVER (PARTITION BY test_case ORDER BY build)
              THEN 1 ELSE 0 END AS starts
  FROM test_runs
) AS flagged;

SELECT * FROM numbered ORDER BY test_case, build;

-- Step 3: one row per island, and the current streak of each test.
SELECT test_case, status, min(build) AS from_build, max(build) AS to_build,
       count(*) AS runs,
       island = max(island) OVER (PARTITION BY test_case) AS current
FROM numbered
GROUP BY test_case, island, status
ORDER BY test_case, from_build;

-- The tempting one-step version: lag inside sum() OVER.
SELECT test_case, build,
       sum(CASE WHEN status <> lag(status) OVER w THEN 1 ELSE 0 END) OVER w
FROM test_runs
WINDOW w AS (PARTITION BY test_case ORDER BY build);
