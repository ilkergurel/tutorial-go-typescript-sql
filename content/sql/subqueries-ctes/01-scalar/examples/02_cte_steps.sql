CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int NOT NULL REFERENCES test_cases,
  passed       boolean NOT NULL
);
INSERT INTO test_cases VALUES (10, 'Valid password'), (11, 'Locked account'), (12, 'Export CSV');
INSERT INTO test_runs VALUES
  (1, 10, true), (2, 10, false), (3, 10, true),
  (4, 11, false), (5, 11, false), (6, 12, true), (7, 12, false);

-- "Flaky tests": both passed and failed at least once. Nested derived
-- tables read inside-out, and the innermost step comes first.
SELECT t.title, f.passes, f.failures
FROM (
  SELECT test_case_id, passes, failures
  FROM (
    SELECT test_case_id,
           count(*) FILTER (WHERE passed)     AS passes,
           count(*) FILTER (WHERE NOT passed) AS failures
    FROM test_runs
    GROUP BY test_case_id
  ) AS per_case
  WHERE passes > 0 AND failures > 0
) AS f
JOIN test_cases t ON t.id = f.test_case_id
ORDER BY t.title;

-- The same query as named steps, read top to bottom.
WITH per_case AS (
  SELECT test_case_id,
         count(*) FILTER (WHERE passed)     AS passes,
         count(*) FILTER (WHERE NOT passed) AS failures
  FROM test_runs
  GROUP BY test_case_id
),
flaky AS (
  SELECT test_case_id, passes, failures
  FROM per_case
  WHERE passes > 0 AND failures > 0
)
SELECT t.title, f.passes, f.failures
FROM flaky f
JOIN test_cases t ON t.id = f.test_case_id
ORDER BY t.title;
