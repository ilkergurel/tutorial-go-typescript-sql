-- LEARN(sql.joins.lateral): a subquery that runs per row and may return many rows.
CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int NOT NULL REFERENCES test_cases,
  finished_at  timestamp NOT NULL,
  passed       boolean NOT NULL
);
CREATE INDEX ON test_runs (test_case_id, finished_at DESC);
INSERT INTO test_cases VALUES (10, 'Login'), (11, 'Password reset'), (12, 'Export CSV');
INSERT INTO test_runs VALUES
  (1, 10, '2026-11-10 09:00', true),
  (2, 10, '2026-11-11 09:00', false),
  (3, 10, '2026-11-12 09:00', true),
  (4, 11, '2026-11-12 10:00', true);

-- The last two runs per test case. The subquery reads t.id: that is LATERAL.
-- LEFT JOIN ... ON true keeps test cases that have no runs at all.
SELECT t.title, last.finished_at, last.passed
FROM test_cases t
LEFT JOIN LATERAL (
  SELECT r.finished_at, r.passed
  FROM test_runs r
  WHERE r.test_case_id = t.id
  ORDER BY r.finished_at DESC
  LIMIT 2
) AS last ON true
ORDER BY t.id, last.finished_at DESC;

-- Without the keyword, a FROM subquery cannot see its neighbours.
SELECT t.title, last.finished_at
FROM test_cases t
LEFT JOIN (
  SELECT r.finished_at FROM test_runs r
  WHERE r.test_case_id = t.id
  ORDER BY r.finished_at DESC LIMIT 2
) AS last ON true;
