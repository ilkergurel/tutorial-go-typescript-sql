-- LEARN(sql.subqueries-ctes.scalar): a subquery used as a value must yield at most one row.
CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int NOT NULL REFERENCES test_cases,
  run_no       int NOT NULL,       -- 1, 2, 3 ... per test case
  duration_ms  int NOT NULL,
  passed       boolean NOT NULL
);
INSERT INTO test_cases VALUES (10, 'Valid password'), (11, 'Locked account'), (12, 'Export CSV');
INSERT INTO test_runs VALUES
  (1, 10, 1, 120, true), (2, 10, 2, 900, false), (3, 10, 3, 110, true),
  (4, 11, 1, 300, false);

-- Uncorrelated: it doesn't mention the outer row, so it runs once and acts
-- as a constant.
SELECT id, duration_ms
FROM test_runs
WHERE duration_ms > (SELECT avg(duration_ms) FROM test_runs)
ORDER BY id;

-- Correlated: t.id comes from the outer row, so it is evaluated per test case.
SELECT t.title,
       (SELECT count(*) FROM test_runs r WHERE r.test_case_id = t.id) AS runs,
       (SELECT r.passed FROM test_runs r
        WHERE r.test_case_id = t.id
        ORDER BY r.run_no DESC LIMIT 1) AS last_passed
FROM test_cases t
ORDER BY t.id;

-- The plan shows it: a SubPlan that the outer scan calls per row.
EXPLAIN (COSTS OFF)
SELECT t.title,
       (SELECT count(*) FROM test_runs r WHERE r.test_case_id = t.id) AS runs
FROM test_cases t;

-- Without ORDER BY ... LIMIT 1 this fails, but only for test cases that
-- have more than one run. The others would have worked.
SELECT t.title,
       (SELECT r.passed FROM test_runs r WHERE r.test_case_id = t.id) AS passed
FROM test_cases t
ORDER BY t.id;
