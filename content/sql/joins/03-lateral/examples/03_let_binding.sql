CREATE TABLE test_runs (
  id          int PRIMARY KEY,
  started_at  timestamp NOT NULL,
  finished_at timestamp NOT NULL,
  budget_ms   int NOT NULL
);
INSERT INTO test_runs VALUES
  (1, '2026-11-13 09:00:00', '2026-11-13 09:00:01.5', 2000),
  (2, '2026-11-13 09:05:00', '2026-11-13 09:05:04',   3000),
  (3, '2026-11-13 09:10:00', '2026-11-13 09:10:09',   3000);

-- Name an expression once, then use it in SELECT, WHERE and ORDER BY.
SELECT r.id, m.ms, m.ms - r.budget_ms AS over_budget_ms
FROM test_runs r
CROSS JOIN LATERAL (
  SELECT (extract(epoch FROM r.finished_at - r.started_at) * 1000)::int AS ms
) AS m
WHERE m.ms > r.budget_ms
ORDER BY over_budget_ms DESC;

-- A SELECT-list alias can't do that: WHERE is evaluated before SELECT.
SELECT id, (extract(epoch FROM finished_at - started_at) * 1000)::int AS ms
FROM test_runs
WHERE ms > budget_ms;
