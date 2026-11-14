-- One-time fix for a playground database created by an older setup-local-db.sql
-- with LC_CTYPE 'C', where pg_trgm treats Turkish letters as word separators.
-- Recreates ONLY the playground database (lesson runs keep nothing there);
-- the stacklab database with your progress and notes is not touched.
-- Run as the postgres superuser from the repo root:
--   & "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -f scripts/fix-playground-ctype.sql

\set ON_ERROR_STOP on

SELECT datctype = 'C' AS needs_fix FROM pg_database WHERE datname = 'playground' \gset

\if :needs_fix
  -- WITH (FORCE) ends open lesson sessions; a run in flight just fails once.
  DROP DATABASE playground WITH (FORCE);
  SELECT format(
      'CREATE DATABASE playground OWNER playground TEMPLATE template0 ENCODING %L'
      ' LOCALE_PROVIDER builtin BUILTIN_LOCALE %L LC_COLLATE %L LC_CTYPE %L',
      'UTF8', 'C.UTF-8', 'C', (SELECT datctype FROM pg_database WHERE datname = 'template1'))\gexec
  \echo playground recreated
\else
  \echo playground already has a real LC_CTYPE; nothing to do
\endif

SELECT datname, datlocale, datcollate, datctype FROM pg_database WHERE datname = 'playground';
