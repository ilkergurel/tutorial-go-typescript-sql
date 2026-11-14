-- One-time setup for local mode (ADR-0008): two roles and two databases on the
-- one local PostgreSQL. Run it as the postgres superuser from the repo root:
--   & "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -f scripts/setup-local-db.sql
-- Safe to re-run: existing roles and databases are left untouched.
--
-- playground is deliberately NOT a superuser. Lesson SQL runs as this role on
-- your own machine, and a superuser can run shell commands (COPY ... TO
-- PROGRAM) and read any file. Owning its database is all sqlrun needs: each
-- run creates a throwaway schema inside a transaction that is rolled back.
--
-- Both databases use the builtin C.UTF-8 locale: code-point sort order and
-- Unicode simple case mapping, identical on every OS. Without it Windows hands
-- them the system locale, and under a Turkish locale lower('I') returns 'ı',
-- unlike the Docker images and most servers. Turkish rules stay available as
-- an explicit per-query choice (an ICU collation such as "tr-x-icu").
--
-- LC_CTYPE is the server's own default (template1's), not 'C': a few features
-- still classify characters through it rather than the builtin provider, and
-- under 'C' pg_trgm treats every non-ASCII letter (ı, ğ, ş...) as a word
-- separator. Case mapping is unaffected: it comes from the builtin locale.
-- scripts/fix-playground-ctype.sql applies this to a database created before.
--
-- The last two statements lock the playground role into its own database and
-- keep it out of the playground's public schema; re-running this file applies
-- them to a setup made before they were added.

\set ON_ERROR_STOP on

SELECT 'CREATE ROLE stacklab LOGIN PASSWORD ''stacklab'''
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'stacklab')\gexec

SELECT 'CREATE ROLE playground LOGIN PASSWORD ''playground'' NOSUPERUSER NOCREATEDB NOCREATEROLE'
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'playground')\gexec

-- CREATE DATABASE cannot run inside a DO block, hence \gexec.
SELECT format(
    'CREATE DATABASE %I OWNER %I TEMPLATE template0 ENCODING %L'
    ' LOCALE_PROVIDER builtin BUILTIN_LOCALE %L LC_COLLATE %L LC_CTYPE %L',
    db, db, 'UTF8', 'C.UTF-8', 'C',
    (SELECT datctype FROM pg_database WHERE datname = 'template1'))
FROM unnest(ARRAY['stacklab', 'playground']) AS db
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = db)\gexec

-- The Windows installer listens on every network interface. StackLab is
-- local-only, so accept connections from this machine only. Takes effect on
-- the next service restart (or reboot).
ALTER SYSTEM SET listen_addresses = 'localhost';

-- Every new database grants CONNECT and TEMPORARY to PUBLIC, so lesson code
-- (running as playground) could connect to the app's database. Its owner,
-- stacklab, keeps both privileges.
REVOKE ALL ON DATABASE stacklab FROM PUBLIC;

-- Since PostgreSQL 15, public belongs to pg_database_owner, whose one member is
-- the database's owner: playground could create objects in public, or drop it.
-- Runs work in a schema of their own with search_path set to it, so public is
-- only reached if a run escapes its sandbox; then a CREATE fails instead of
-- persisting. PUBLIC keeps USAGE. psql reuses the password for this connection.
\connect playground
ALTER SCHEMA public OWNER TO postgres;
