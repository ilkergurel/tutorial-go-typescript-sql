SET TimeZone = 'Europe/Istanbul';

CREATE TABLE ui_events (
  id      int         PRIMARY KEY,
  user_id text        NOT NULL,
  at      timestamptz NOT NULL
);
INSERT INTO ui_events VALUES
  (1, 'ayşe',  '2026-11-13 09:00+03'), (2, 'ayşe',  '2026-11-13 09:20+03'),
  (3, 'ayşe',  '2026-11-13 09:45+03'), (4, 'ayşe',  '2026-11-13 10:10+03'),
  (5, 'ayşe',  '2026-11-13 14:00+03'), (6, 'ayşe',  '2026-11-13 14:05+03'),
  (7, 'emre',  '2026-11-13 09:00+03'), (8, 'emre',  '2026-11-13 09:31+03');

-- A session ends after 30 minutes of silence. Islands again, but the
-- break condition is a time gap instead of a changed value.
SELECT user_id, session_no,
       min(at)::time AS started, max(at)::time AS ended,
       max(at) - min(at) AS length, count(*) AS events
FROM (
  SELECT user_id, at,
         sum(new_session) OVER (PARTITION BY user_id ORDER BY at, id) AS session_no
  FROM (
    SELECT user_id, at, id,
           CASE WHEN at - lag(at) OVER (PARTITION BY user_id ORDER BY at, id)
                     <= interval '30 minutes'
                THEN 0 ELSE 1 END AS new_session
    FROM ui_events
  ) AS flagged
) AS sessions
GROUP BY user_id, session_no
ORDER BY user_id, session_no;
