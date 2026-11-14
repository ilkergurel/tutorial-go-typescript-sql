-- One server (a "cluster") holds many databases. Some catalogs are
-- shared by all of them, the rest exist once per database.
SELECT relname AS catalog, relisshared AS shared_by_all_databases
FROM pg_class
WHERE relname IN ('pg_authid', 'pg_database', 'pg_namespace', 'pg_class')
ORDER BY relisshared DESC, relname;

-- Roles are cluster-wide, so a database is owned by a role, and so is
-- every schema in it. A new database's public belongs to pg_database_owner
-- (PG 15+); StackLab's setup gave the playground's public to postgres.
CREATE SCHEMA reporting;
SELECT nspname AS schema, pg_get_userbyid(nspowner) AS owner
FROM pg_namespace
WHERE nspname IN ('public', 'reporting')
ORDER BY nspname;

SELECT datname AS database, pg_get_userbyid(datdba) AS owner
FROM pg_database
WHERE datname = current_database();

-- A connection sees exactly one database. Naming another one is an
-- error, not a join: there is no cross-database query.
SELECT * FROM stacklab.public.users;
