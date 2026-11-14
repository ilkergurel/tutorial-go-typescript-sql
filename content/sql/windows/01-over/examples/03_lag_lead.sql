SET TimeZone = 'Europe/Istanbul';

CREATE TABLE test_runs (
  id          int         PRIMARY KEY,
  test_case   text        NOT NULL,
  ran_at      timestamptz NOT NULL,
  status      text        NOT NULL,
  duration_ms int         NOT NULL
);
INSERT INTO test_runs VALUES
  (1, 'Giriş', '2026-11-10 09:00+03', 'passed', 120),
  (2, 'Giriş', '2026-11-11 09:00+03', 'passed', 130),
  (3, 'Giriş', '2026-11-12 09:00+03', 'failed', 940),
  (4, 'Giriş', '2026-11-13 09:00+03', 'passed', 125),
  (5, 'Çıkış', '2026-11-11 18:30+03', 'failed',  80),
  (6, 'Çıkış', '2026-11-13 09:05+03', 'failed',  85);

-- Each run next to the previous and next run of the same test case.
SELECT test_case, ran_at::date AS day, status,
       lag(status) OVER w                    AS prev_status,
       lag(status, 1, 'first run') OVER w    AS prev_or_label,
       duration_ms - lag(duration_ms) OVER w AS delta_ms,
       lead(ran_at) OVER w - ran_at          AS until_next_run
FROM test_runs
WINDOW w AS (PARTITION BY test_case ORDER BY ran_at, id)
ORDER BY test_case, ran_at, id;

-- Regressions: a run that failed right after a pass of the same test.
SELECT test_case, ran_at
FROM (
  SELECT test_case, ran_at, status,
         lag(status) OVER (PARTITION BY test_case ORDER BY ran_at, id) AS prev
  FROM test_runs
) AS r
WHERE prev = 'passed' AND status = 'failed'
ORDER BY ran_at;
