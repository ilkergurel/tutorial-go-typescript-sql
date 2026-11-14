-- LEARN(sql.modern.16-to-18): three PG 18 features in two table definitions.
-- WITHOUT OVERLAPS needs GiST equality for the int column: btree_gist.
CREATE EXTENSION btree_gist;

-- Temporal primary key: many rows per requirement, but no two periods overlap.
CREATE TABLE requirement_versions (
  requirement_id int,
  valid          daterange,
  title          text NOT NULL,
  PRIMARY KEY (requirement_id, valid WITHOUT OVERLAPS)
);
INSERT INTO requirement_versions VALUES
  (1, '[2026-01-01,2026-06-01)', 'Giriş'),
  (1, '[2026-06-01,)',           'Giriş, SSO dahil');

CREATE TABLE test_runs (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),   -- time-ordered UUID
  requirement_id int  NOT NULL,
  ran            daterange NOT NULL,
  duration_ms    int  NOT NULL,
  duration_s     numeric(6,2) GENERATED ALWAYS AS (duration_ms / 1000.0),  -- virtual
  -- Temporal foreign key: the requirement must exist for the whole period.
  FOREIGN KEY (requirement_id, PERIOD ran)
    REFERENCES requirement_versions (requirement_id, PERIOD valid)
);
INSERT INTO test_runs (requirement_id, ran, duration_ms) VALUES
  (1, '[2026-03-10,2026-03-11)', 1500),
  (1, '[2026-05-31,2026-06-02)', 2250);  -- spans both versions: still covered

SELECT uuid_extract_version(id) AS uuid_version, ran, duration_ms, duration_s
FROM test_runs
ORDER BY ran;

-- The catalog: a GiST index behind the temporal key, NOT NULL as a
-- constraint row (contype n), and a virtual generated column.
SELECT c.conrelid::regclass AS tbl, c.conname, c.contype,
       CASE c.contype WHEN 'p' THEN am.amname END AS key_index
FROM pg_constraint c
LEFT JOIN pg_class i ON i.oid = c.conindid
LEFT JOIN pg_am am ON am.oid = i.relam
WHERE c.conrelid IN ('requirement_versions'::regclass, 'test_runs'::regclass)
ORDER BY c.conrelid::regclass::text, c.conname;
SELECT attname, attgenerated FROM pg_attribute
WHERE attrelid = 'test_runs'::regclass AND attgenerated <> '';

-- An overlapping period is rejected like an exclusion-constraint conflict.
INSERT INTO requirement_versions VALUES (1, '[2026-05-01,2026-07-01)', 'Çakışan');
