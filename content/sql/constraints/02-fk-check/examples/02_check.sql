CREATE TABLE test_runs (
  id          int PRIMARY KEY,
  status      text NOT NULL CHECK (status IN ('passed', 'failed', 'skipped')),
  duration_ms int CHECK (duration_ms >= 0),
  started_at  timestamp NOT NULL,
  finished_at timestamp,
  -- Table constraints may compare columns; a name makes the error readable.
  CONSTRAINT finished_after_start CHECK (finished_at >= started_at),
  -- NULL handling written out, because a CHECK lets unknown through.
  CONSTRAINT duration_unless_skipped
    CHECK (status = 'skipped' OR duration_ms IS NOT NULL)
);

INSERT INTO test_runs VALUES
  (1, 'passed',  1200, '2026-11-13 10:00', '2026-11-13 10:00:01.2'),
  (2, 'skipped', NULL, '2026-11-13 10:05', NULL);

SELECT id, status, duration_ms, finished_at FROM test_runs ORDER BY id;

INSERT INTO test_runs VALUES
  (3, 'failed', 50, '2026-11-13 11:00', '2026-11-13 10:59');
