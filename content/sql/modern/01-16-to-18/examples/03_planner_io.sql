CREATE TABLE test_runs (
  status text NOT NULL,  -- three distinct values
  ran_on date NOT NULL
);
INSERT INTO test_runs
SELECT (ARRAY['failed', 'passed', 'skipped'])[g % 3 + 1], date '2026-01-01' + g % 300
FROM generate_series(1, 300000) AS g;
CREATE INDEX test_runs_status_ran_on ON test_runs (status, ran_on);
ANALYZE test_runs;
SELECT pg_relation_size('test_runs_status_ran_on') / 8192 AS index_pages;

-- PG 18 skip scan: the query never names status, the index's first column.
SELECT count(*) FROM test_runs WHERE ran_on = '2026-06-01';
-- Run it again under EXPLAIN ANALYZE, which in PG 18 also reports Index
-- Searches and, without asking, Buffers.
EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF)
SELECT count(*) FROM test_runs WHERE ran_on = '2026-06-01';

-- PG 18 asynchronous I/O: how this server is configured.
SELECT current_setting('io_method') AS io_method,
       current_setting('io_workers') AS io_workers,
       current_setting('effective_io_concurrency') AS effective_io_concurrency;
