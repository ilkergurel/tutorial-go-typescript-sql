-- LEARN(sql.windows.frames): the frame decides which rows an aggregate OVER (...) sees.
CREATE TABLE runs (
  id      int  PRIMARY KEY,
  project text NOT NULL,
  ran_on  date NOT NULL,
  failed  int  NOT NULL          -- failed tests in this run
);
INSERT INTO runs VALUES
  (1, 'TraceLite', '2026-11-09', 3),
  (2, 'TraceLite', '2026-11-10', 1),
  (3, 'TraceLite', '2026-11-10', 4),   -- a second run on the same day
  (4, 'TraceLite', '2026-11-12', 2),
  (5, 'StackLab',  '2026-11-10', 5),
  (6, 'StackLab',  '2026-11-11', 0);

-- ORDER BY inside OVER turns sum into a running sum. The default frame
-- (RANGE) ends at the current row's last peer; ROWS ends at the row itself.
SELECT project, ran_on, id, failed,
       sum(failed) OVER (PARTITION BY project ORDER BY ran_on) AS range_default,
       sum(failed) OVER (PARTITION BY project ORDER BY ran_on, id
                         ROWS UNBOUNDED PRECEDING)             AS rows_running,
       sum(failed) OVER (PARTITION BY project)                 AS project_total
FROM runs
ORDER BY project, ran_on, id;

-- last_value sees the frame, not the partition: with the default frame the
-- "last" row is the current one. Widen the frame to get the latest run.
SELECT project, id, failed,
       last_value(failed) OVER (PARTITION BY project ORDER BY ran_on, id)
         AS last_default,
       last_value(failed) OVER (PARTITION BY project ORDER BY ran_on, id
                                ROWS BETWEEN UNBOUNDED PRECEDING
                                         AND UNBOUNDED FOLLOWING)
         AS latest_failed
FROM runs
ORDER BY project, id;
