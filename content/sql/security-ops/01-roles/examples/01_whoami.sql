-- LEARN(sql.security-ops.roles): the sandbox role owns its database and nothing beyond it.
-- Role attributes are cluster-wide switches, separate from object privileges.
SELECT rolname, rolsuper, rolcreaterole, rolcreatedb, rolbypassrls
FROM pg_roles
WHERE rolname = current_user;

-- Roles whose privileges this session has without SET ROLE.
SELECT rolname FROM pg_roles
WHERE pg_has_role(current_user, oid, 'USAGE')
ORDER BY rolname;

-- Runs a statement and reports a permission error instead of aborting.
-- The EXCEPTION block is a subtransaction: a failure undoes only that statement.
CREATE FUNCTION try(stmt text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN
  EXECUTE stmt;
  RETURN 'allowed';
EXCEPTION WHEN insufficient_privilege THEN
  RETURN SQLERRM;
END $$;

SELECT stmt AS "statement", try(stmt) AS outcome
FROM unnest(ARRAY[
  'CREATE ROLE tracelite_app LOGIN',
  'ALTER ROLE playground BYPASSRLS',
  'SET ROLE pg_read_all_stats',
  $$COPY (SELECT 1) TO PROGRAM 'whoami'$$,
  $$SELECT pg_read_file('postgresql.conf')$$,
  'SELECT rolpassword FROM pg_authid',
  'SET log_min_duration_statement = 0',
  'CREATE SCHEMA tracelite',
  'DROP SCHEMA public'
]) WITH ORDINALITY AS t (stmt, n)
ORDER BY n;
