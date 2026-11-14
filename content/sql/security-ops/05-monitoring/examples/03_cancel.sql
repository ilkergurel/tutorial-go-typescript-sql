A: INSERT INTO sessions VALUES ('A', pg_backend_pid());

-- A starts a report that runs far longer than anyone wants.
A: SELECT pg_sleep(4);
B: SELECT state, wait_event_type, wait_event, query
   FROM pg_stat_activity WHERE pid = (SELECT pid FROM sessions WHERE name = 'A');
B: SELECT pg_cancel_backend(pid) AS cancelled FROM sessions WHERE name = 'A';

-- The guard that doesn't need a human: a statement timeout.
A: SET statement_timeout = '200ms';
A: SELECT pg_sleep(1);
A: SELECT 'the session is still usable' AS status;
