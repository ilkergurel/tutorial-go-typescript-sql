-- Runtime sql-nontx: autocommit, so CALL runs in a transaction of its own.
CREATE TABLE test_runs (id int PRIMARY KEY, ran_on date NOT NULL);
CREATE TABLE test_runs_archive (LIKE test_runs);
INSERT INTO test_runs
SELECT g, date '2026-01-01' + g / 100 FROM generate_series(1, 1000) AS g;

-- A procedure may COMMIT: each batch becomes durable on its own, so a
-- big cleanup never holds its locks for the whole run.
CREATE PROCEDURE archive_runs(cutoff date, batch_size int)
LANGUAGE plpgsql
AS $$
DECLARE
  moved int;
BEGIN
  LOOP
    WITH batch AS (
      DELETE FROM test_runs
      WHERE id IN (SELECT id FROM test_runs WHERE ran_on < cutoff ORDER BY id LIMIT batch_size)
      RETURNING *
    )
    INSERT INTO test_runs_archive SELECT * FROM batch;
    GET DIAGNOSTICS moved = ROW_COUNT;
    EXIT WHEN moved = 0;
    COMMIT;
    RAISE NOTICE 'archived % runs', moved;
  END LOOP;
END
$$;

CALL archive_runs('2026-01-04', 120);
SELECT (SELECT count(*) FROM test_runs) AS live,
       (SELECT count(*) FROM test_runs_archive) AS archived;

-- Inside a transaction block the caller owns the transaction, so the
-- procedure may not end it.
BEGIN;
CALL archive_runs('2026-01-08', 120);
