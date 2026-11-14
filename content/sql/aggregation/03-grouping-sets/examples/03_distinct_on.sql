SET TimeZone = 'Europe/Istanbul';

CREATE TABLE test_runs (
  id        int PRIMARY KEY,
  test_case text NOT NULL,
  ran_at    timestamptz NOT NULL,
  passed    boolean NOT NULL
);
INSERT INTO test_runs VALUES
  (1, 'Login',  '2026-11-12 09:00+03', true),
  (2, 'Login',  '2026-11-13 09:00+03', false),
  (3, 'Logout', '2026-11-12 10:00+03', true),
  (4, 'Export', '2026-11-13 08:00+03', false),
  (5, 'Export', '2026-11-13 08:00+03', true);  -- a retry in the same second

-- The latest run per test case: sort so the wanted row comes first in its
-- group, and DISTINCT ON keeps only that first row.
SELECT DISTINCT ON (test_case) test_case, ran_at, passed, id
FROM test_runs
ORDER BY test_case, ran_at DESC, id DESC;

-- Want the result in another order? Sort outside.
SELECT *
FROM (
  SELECT DISTINCT ON (test_case) test_case, ran_at, passed
  FROM test_runs
  ORDER BY test_case, ran_at DESC, id DESC
) AS latest
ORDER BY ran_at DESC, test_case;

-- The standard-SQL equivalent: number the rows per group, keep number 1.
SELECT test_case, ran_at, passed
FROM (
  SELECT test_case, ran_at, passed,
         row_number() OVER (PARTITION BY test_case ORDER BY ran_at DESC, id DESC) AS rn
  FROM test_runs
) AS ranked
WHERE rn = 1
ORDER BY test_case;

-- The DISTINCT ON expressions must lead the ORDER BY.
SELECT DISTINCT ON (test_case) test_case, passed FROM test_runs ORDER BY ran_at DESC;
