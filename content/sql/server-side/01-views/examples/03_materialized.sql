-- A materialized view runs its query once and stores the result.
A: CREATE MATERIALIZED VIEW pass_rates AS
   SELECT requirement, count(*) AS runs, count(*) FILTER (WHERE passed) AS passed
   FROM test_runs
   GROUP BY requirement;
A: CREATE UNIQUE INDEX ON pass_rates (requirement);

-- New runs arrive; the stored rows don't change until a refresh.
B: INSERT INTO test_runs (requirement, passed) VALUES ('REQ-LOGIN', true);
B: SELECT * FROM pass_rates ORDER BY requirement;

-- A plain REFRESH rebuilds it under an ACCESS EXCLUSIVE lock: readers wait.
A: BEGIN;
A: REFRESH MATERIALIZED VIEW pass_rates;
B: SELECT * FROM pass_rates ORDER BY requirement;
A: COMMIT;

-- CONCURRENTLY computes the new result, diffs it against the stored one
-- and applies the difference; readers keep seeing the old rows meanwhile.
B: INSERT INTO test_runs (requirement, passed) VALUES ('REQ-EXPORT', false);
A: BEGIN;
A: REFRESH MATERIALIZED VIEW CONCURRENTLY pass_rates;
B: SELECT * FROM pass_rates ORDER BY requirement;
A: COMMIT;
B: SELECT * FROM pass_rates ORDER BY requirement;
