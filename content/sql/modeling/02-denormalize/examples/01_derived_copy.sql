-- LEARN(sql.modeling.denormalize): a copy is fine once you know what keeps it honest.
CREATE TABLE test_cases (
  id          int  PRIMARY KEY,
  code        text NOT NULL UNIQUE,
  -- Denormalized: derivable from test_runs, stored so the list needs no join.
  last_result text,
  last_run_on date
);
CREATE TABLE test_runs (
  id           int  PRIMARY KEY,
  test_case_id int  NOT NULL REFERENCES test_cases,
  run_on       date NOT NULL,
  result       text NOT NULL
);
INSERT INTO test_cases (id, code) VALUES (1, 'TC-1'), (2, 'TC-2');

-- The one write path: record the run and move the copy forward, atomically.
WITH run AS (
  INSERT INTO test_runs VALUES (100, 1, '2026-11-02', 'failed')
  RETURNING test_case_id, run_on, result
)
UPDATE test_cases c
SET last_result = run.result, last_run_on = run.run_on
FROM run
WHERE c.id = run.test_case_id
  AND (c.last_run_on IS NULL OR c.last_run_on <= run.run_on);

-- A bulk import that doesn't know about the copy.
INSERT INTO test_runs VALUES (101, 1, '2026-11-09', 'passed'), (102, 2, '2026-11-09', 'passed');

-- Reconcile: recompute from the source and list where the copy disagrees.
CREATE VIEW latest_runs AS
  SELECT DISTINCT ON (test_case_id) test_case_id, run_on, result
  FROM test_runs
  ORDER BY test_case_id, run_on DESC, id DESC;

SELECT c.code, c.last_result AS stored, l.result AS actual
FROM test_cases c
LEFT JOIN latest_runs l ON l.test_case_id = c.id
WHERE c.last_result IS DISTINCT FROM l.result
ORDER BY c.code;

-- Repair from the same definition.
UPDATE test_cases c
SET last_result = l.result, last_run_on = l.run_on
FROM latest_runs l
WHERE l.test_case_id = c.id;
SELECT code, last_result, last_run_on FROM test_cases ORDER BY code;
