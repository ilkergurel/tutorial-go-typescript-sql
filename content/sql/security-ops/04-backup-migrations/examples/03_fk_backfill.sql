-- Runtime sql-nontx: every statement commits on its own, as in a migration tool.
CREATE TABLE projects (id bigint PRIMARY KEY, code text NOT NULL UNIQUE);
CREATE TABLE requirements (id bigint PRIMARY KEY, project_code text NOT NULL);
INSERT INTO projects SELECT g, 'P' || g FROM generate_series(1, 10) AS g;
INSERT INTO requirements SELECT g, 'P' || (g % 10 + 1) FROM generate_series(1, 1000) AS g;
CREATE VIEW my_locks AS
  SELECT relation::regclass AS tbl, mode FROM pg_locks
  WHERE pid = pg_backend_pid() AND locktype = 'relation'
    AND relation IN ('projects'::regclass, 'requirements'::regclass);

-- Goal: replace project_code with a real foreign key. Expand: a nullable column.
ALTER TABLE requirements ADD COLUMN project_id bigint;

-- Guard new writes now, check old rows later.
BEGIN;
ALTER TABLE requirements ADD CONSTRAINT requirements_project_fk
  FOREIGN KEY (project_id) REFERENCES projects NOT VALID;
SELECT * FROM my_locks ORDER BY tbl, mode;
COMMIT;

-- Backfill in batches, one transaction each, so no lock is held for long.
CREATE PROCEDURE backfill_project_id(batch int) LANGUAGE plpgsql AS $$
DECLARE n int;
BEGIN
  LOOP
    UPDATE requirements r SET project_id = p.id
    FROM projects p
    WHERE p.code = r.project_code
      AND r.id IN (SELECT id FROM requirements WHERE project_id IS NULL
                   ORDER BY id LIMIT batch);
    GET DIAGNOSTICS n = ROW_COUNT;
    EXIT WHEN n = 0;
    COMMIT;
    RAISE NOTICE 'committed a batch of % rows', n;
  END LOOP;
END $$;
CALL backfill_project_id(400);

-- Now check the old rows, under a lock that lets reads and writes continue.
BEGIN;
ALTER TABLE requirements VALIDATE CONSTRAINT requirements_project_fk;
SELECT * FROM my_locks ORDER BY tbl, mode;
COMMIT;
