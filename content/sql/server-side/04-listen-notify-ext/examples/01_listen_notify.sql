-- Runtime sql-nontx: autocommit, like psql, so each statement is its own
-- transaction unless BEGIN says otherwise.
LISTEN test_runs;
SELECT pg_listening_channels() AS listening;

-- LISTEN and UNLISTEN are transactional: they take effect at COMMIT.
BEGIN;
LISTEN requirements;
SELECT pg_listening_channels() AS listening ORDER BY 1;
COMMIT;
SELECT pg_listening_channels() AS listening ORDER BY 1;

BEGIN;
UNLISTEN *;
ROLLBACK;
SELECT count(*) AS still_listening FROM pg_listening_channels();

-- NOTIFY takes a literal payload; pg_notify() takes any expression.
-- This session listens on test_runs, so it gets these back itself; the
-- runner just doesn't display asynchronous messages (a pgx program does).
NOTIFY test_runs, 'run 42 finished';
SELECT pg_notify('test_runs', json_build_object('run', 42, 'status', 'failed')::text);

-- Payloads must be shorter than 8000 bytes: bytes, not characters.
SELECT octet_length(repeat('ş', 3999)) AS bytes_3999, octet_length(repeat('ş', 4000)) AS bytes_4000;
SELECT pg_notify('test_runs', repeat('ş', 3999));
SELECT pg_notify('test_runs', repeat('ş', 4000));
