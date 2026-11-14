-- Runtime sql. Before your first line runs, sqlrun.Run has sent, as one message:
--   BEGIN; CREATE SCHEMA run_<nanos>_<hex>;
--   SET LOCAL search_path TO run_...; SET LOCAL statement_timeout = 5000;
--   SET LOCAL lock_timeout = 2000;
-- and after your last line it sends ROLLBACK.

-- The schema name changes on every run, so mask it.
SELECT regexp_replace(current_schema(), '^run_\d+_[0-9a-f]{8}$', 'run_<nanos>_<hex>') AS schema,
       current_setting('statement_timeout') AS statement_timeout,
       current_setting('lock_timeout') AS lock_timeout;

-- You are already inside a transaction, and it has written: CREATE SCHEMA.
SELECT pg_current_xact_id_if_assigned() IS NOT NULL AS has_xid,
       transaction_timestamp() < statement_timestamp() AS began_before_this_statement;

-- Unqualified names land in the run's schema, never in public.
CREATE TABLE notes (id int PRIMARY KEY, body text);
INSERT INTO notes VALUES (1, 'gone after ROLLBACK');
SELECT c.relnamespace::regnamespace::text = current_schema() AS in_run_schema,
       to_regclass('public.notes') AS public_notes
FROM pg_class c
WHERE c.oid = 'notes'::regclass;

-- A BEGIN of your own changes nothing: there is only one transaction.
BEGIN;
