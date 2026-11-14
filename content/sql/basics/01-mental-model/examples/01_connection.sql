-- LEARN(sql.basics.mental-model): a connection is one server process, one role, one database.

-- The server started a process for this connection. pg_stat_activity
-- has one row per server process; this row is yours.
SELECT backend_type, datname AS database, usename AS role, state
FROM pg_stat_activity
WHERE pid = pg_backend_pid();

-- The same facts from SQL functions. session_user is who logged in;
-- current_user is whose privileges apply (SET ROLE changes it).
SELECT current_database(), session_user, current_user;

-- Inside the database, the search path decides where unqualified names
-- resolve. The sandbox schema is renamed every run, so its suffix is
-- masked as <id>. current_schemas(true) adds the implicit entries.
SELECT regexp_replace(current_schema(), '_\d+_[0-9a-f]+$', '_<id>')
         AS current_schema,
       regexp_replace(current_schemas(true)::text, '_\d+_[0-9a-f]+', '_<id>')
         AS effective_path;
