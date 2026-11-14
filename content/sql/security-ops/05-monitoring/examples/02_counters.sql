CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int NOT NULL,
  status       text NOT NULL
);
INSERT INTO test_runs
SELECT g, g % 500, CASE WHEN g % 10 = 0 THEN 'failed' ELSE 'passed' END
FROM generate_series(1, 20000) AS g;
ANALYZE test_runs;

-- This transaction's counters for the table. pg_stat_user_tables has the same
-- columns, accumulated from every session since the last statistics reset.
CREATE VIEW counters AS
  SELECT seq_scan, seq_tup_read, idx_scan, idx_tup_fetch, n_tup_ins
  FROM pg_stat_xact_user_tables WHERE relname = 'test_runs';
SELECT * FROM counters;

-- A dashboard asks for one test case's runs. No index on test_case_id yet.
SELECT count(*) FROM test_runs WHERE test_case_id = 42;
SELECT count(*) FROM test_runs WHERE test_case_id = 43;
SELECT * FROM counters;

CREATE INDEX test_runs_case ON test_runs (test_case_id);
SELECT count(*) FROM test_runs WHERE test_case_id = 42;
SELECT count(*) FROM test_runs WHERE test_case_id = 43;
SELECT * FROM counters;

-- The shared view hasn't seen any of it yet.
SELECT seq_scan, idx_scan FROM pg_stat_user_tables WHERE relname = 'test_runs';
