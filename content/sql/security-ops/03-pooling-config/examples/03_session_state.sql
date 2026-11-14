-- Runtime sql-nontx: one server session, as a pool hands it from client to
-- client. Client 1 leaves session state behind:
SET work_mem = '256MB';
SELECT set_config('app.tenant', '1', false);
PREPARE project_by_id (int) AS SELECT $1 AS id;
CREATE TEMP TABLE import_rows (line text);
SELECT pg_advisory_lock(7310);
LISTEN requirement_changed;

-- What client 2 inherits if it gets this server session next.
SELECT (SELECT source = 'session' FROM pg_settings WHERE name = 'work_mem') AS work_mem_set,
       current_setting('app.tenant', true) AS tenant,
       (SELECT count(*) FROM pg_prepared_statements) AS prepared,
       to_regclass('pg_temp.import_rows') IS NOT NULL AS temp_table,
       (SELECT count(*) FROM pg_locks
        WHERE locktype = 'advisory' AND pid = pg_backend_pid()) AS advisory_locks,
       (SELECT count(*) FROM pg_listening_channels()) AS listening;

-- PgBouncer in session mode sends this before giving the server session
-- to another client (its default server_reset_query).
DISCARD ALL;

SELECT (SELECT source = 'session' FROM pg_settings WHERE name = 'work_mem') AS work_mem_set,
       current_setting('app.tenant', true) AS tenant,
       (SELECT count(*) FROM pg_prepared_statements) AS prepared,
       to_regclass('pg_temp.import_rows') IS NOT NULL AS temp_table,
       (SELECT count(*) FROM pg_locks
        WHERE locktype = 'advisory' AND pid = pg_backend_pid()) AS advisory_locks,
       (SELECT count(*) FROM pg_listening_channels()) AS listening;
