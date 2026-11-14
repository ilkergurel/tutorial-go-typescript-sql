-- LEARN(sql.security-ops.monitoring): when it's slow, first ask who is waiting, on what, and for whom.
A: INSERT INTO sessions VALUES ('A', pg_backend_pid());
B: INSERT INTO sessions VALUES ('B', pg_backend_pid());

-- A starts a transfer, then stalls before COMMIT (a slow HTTP call, say).
A: BEGIN;
A: UPDATE accounts SET balance = balance - 10 WHERE id = 1;
B: SELECT s.name, a.state, a.wait_event_type, a.wait_event, a.query
   FROM pg_stat_activity a JOIN sessions s USING (pid)
   WHERE s.name = 'A';

-- B wants the same row and has to wait.
B: UPDATE accounts SET balance = balance + 10 WHERE id = 1;
A: SELECT s.name, a.state, a.wait_event_type, a.wait_event,
          (SELECT array_agg(b.name) FROM sessions b
           WHERE b.pid = ANY (pg_blocking_pids(a.pid))) AS blocked_by
   FROM pg_stat_activity a JOIN sessions s USING (pid)
   WHERE s.name = 'B';
A: SELECT l.locktype, l.mode, l.granted
   FROM pg_locks l JOIN sessions s USING (pid)
   WHERE s.name = 'B' AND NOT l.granted;
A: COMMIT;
