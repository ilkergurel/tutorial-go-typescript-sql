-- A: a long report. It reads invoices and never touches events.
A: BEGIN ISOLATION LEVEL REPEATABLE READ;
A: SELECT count(*) AS invoices FROM invoices;

-- B: delete the newer half of events, then vacuum. Those rows fill the
-- last pages, so a successful VACUUM would hand those pages back.
B: DELETE FROM events WHERE id > 5000;
B: VACUUM events;
B: SELECT pg_relation_size('events') < (SELECT bytes FROM sizes) AS shrank;

-- A's snapshot is older than the delete: for A those rows still exist,
-- so VACUUM had to keep them. End the report and try again.
A: COMMIT;
B: VACUUM events;
B: SELECT pg_relation_size('events') < (SELECT bytes FROM sizes) AS shrank;
