-- LEARN(sql.server-side.functions): a STABLE SQL table function is a parameterized view the planner sees through.
CREATE TABLE test_runs (
  id           int  PRIMARY KEY,
  test_case_id int  NOT NULL,
  status       text NOT NULL
);
INSERT INTO test_runs
SELECT g, g % 500, CASE WHEN g % 7 = 0 THEN 'failed' ELSE 'passed' END
FROM generate_series(1, 20000) AS g;
CREATE INDEX ON test_runs (test_case_id);
ANALYZE test_runs;

-- The same function three times: SQL + STABLE, SQL with the default
-- volatility (VOLATILE), and PL/pgSQL.
CREATE FUNCTION runs_of(case_id int) RETURNS SETOF test_runs
LANGUAGE sql STABLE
AS $$ SELECT * FROM test_runs WHERE test_case_id = case_id $$;

CREATE FUNCTION runs_of_volatile(case_id int) RETURNS SETOF test_runs
LANGUAGE sql
AS $$ SELECT * FROM test_runs WHERE test_case_id = case_id $$;

CREATE FUNCTION runs_of_plpgsql(case_id int) RETURNS SETOF test_runs
LANGUAGE plpgsql STABLE
AS $$ BEGIN RETURN QUERY SELECT * FROM test_runs WHERE test_case_id = case_id; END $$;

SELECT count(*) AS failed FROM runs_of(42) WHERE status = 'failed';

EXPLAIN (COSTS OFF) SELECT count(*) FROM runs_of(42) WHERE status = 'failed';
EXPLAIN (COSTS OFF) SELECT count(*) FROM runs_of_volatile(42) WHERE status = 'failed';
EXPLAIN (COSTS OFF) SELECT count(*) FROM runs_of_plpgsql(42) WHERE status = 'failed';
