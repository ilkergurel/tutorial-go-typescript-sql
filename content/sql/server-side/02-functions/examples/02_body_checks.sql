CREATE TABLE test_runs (test_case_id int NOT NULL, status text NOT NULL);
INSERT INTO test_runs VALUES (1, 'failed'), (1, 'passed'), (1, 'failed'), (2, 'passed');

-- 1. Body as a string: checked at CREATE, stored as text, parsed again when run.
CREATE FUNCTION failures_text(case_id int) RETURNS bigint
LANGUAGE sql STABLE
AS $$ SELECT count(*) FROM test_runs WHERE test_case_id = case_id AND status = 'failed' $$;

-- 2. SQL-standard body (PG 14): parsed now, stored as a parse tree,
--    with real dependencies on the table and its columns.
CREATE FUNCTION failures(case_id int) RETURNS bigint
LANGUAGE sql STABLE
RETURN (SELECT count(*) FROM test_runs WHERE test_case_id = case_id AND status = 'failed');

SELECT failures_text(1) AS text_body, failures(1) AS standard_body;

-- Rename a column both functions use.
ALTER TABLE test_runs RENAME COLUMN status TO outcome;
SELECT failures(1) AS standard_body_after_rename;
SELECT substring(pg_get_functiondef('failures'::regproc) FROM 'RETURN [^\n]*') AS stored_body;
SELECT failures_text(1);
