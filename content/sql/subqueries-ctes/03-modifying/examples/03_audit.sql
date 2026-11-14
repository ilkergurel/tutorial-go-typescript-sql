CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL, status text NOT NULL);
CREATE TABLE status_changes (
  test_case_id int  NOT NULL REFERENCES test_cases,
  old_status   text NOT NULL,
  new_status   text NOT NULL
);
INSERT INTO test_cases VALUES
  (1, 'Login', 'failed'), (2, 'Logout', 'passed'), (3, 'Export', 'retest');

-- Change rows and write the audit trail in one statement. RETURNING old.*
-- and new.* (PG 18) hand both versions of each row to the next step.
WITH changed AS (
  UPDATE test_cases SET status = 'passed'
  WHERE id IN (1, 2, 3) AND status <> 'passed'
  RETURNING id, old.status AS old_status, new.status AS new_status
), logged AS (
  INSERT INTO status_changes (test_case_id, old_status, new_status)
  SELECT id, old_status, new_status FROM changed
  RETURNING *
)
SELECT * FROM logged ORDER BY test_case_id;

-- Logout was already passed: not updated, so not logged.
SELECT id, status FROM test_cases ORDER BY id;
