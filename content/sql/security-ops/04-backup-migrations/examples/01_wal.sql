-- LEARN(sql.security-ops.backup-migrations): PITR replays the WAL, so what isn't in the WAL can't come back.
-- Every change to an ordinary table is written to the write-ahead log (WAL)
-- first. Archiving ships that log; point-in-time recovery replays it.
CREATE TABLE audit_log (id int PRIMARY KEY, event text NOT NULL);
-- UNLOGGED skips the WAL: faster writes, no crash safety, no recovery.
CREATE UNLOGGED TABLE import_scratch (id int PRIMARY KEY, event text NOT NULL);

-- How many WAL records a statement wrote, from EXPLAIN (ANALYZE, WAL).
CREATE FUNCTION wal_records(stmt text) RETURNS bigint LANGUAGE plpgsql AS $$
DECLARE plan json;
BEGIN
  EXECUTE 'EXPLAIN (ANALYZE, WAL, FORMAT JSON) ' || stmt INTO plan;
  RETURN (plan -> 0 -> 'Plan' ->> 'WAL Records')::bigint;
END $$;

SELECT wal_records($$INSERT INTO audit_log
                     SELECT g, 'login ' || g FROM generate_series(1, 1000) AS g$$) AS logged,
       wal_records($$INSERT INTO import_scratch
                     SELECT g, 'login ' || g FROM generate_series(1, 1000) AS g$$) AS unlogged;

-- Both tables hold 1000 rows now; after a crash or a restore, only one would.
SELECT relname, relpersistence, (SELECT count(*) FROM audit_log) AS audit_rows,
       (SELECT count(*) FROM import_scratch) AS scratch_rows
FROM pg_class WHERE relname IN ('audit_log', 'import_scratch') ORDER BY relname;

-- A named WAL position to recover to, taken just before a risky migration.
SELECT pg_create_restore_point('before-migration-0042');
