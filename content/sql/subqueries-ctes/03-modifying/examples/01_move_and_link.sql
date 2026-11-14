-- LEARN(sql.subqueries-ctes.modifying): RETURNING turns a write into rows the rest of the statement can read.
CREATE TABLE test_runs (
  id        int PRIMARY KEY,
  test_case text NOT NULL,
  ran_on    date NOT NULL,
  passed    boolean NOT NULL
);
CREATE TABLE test_runs_archive (LIKE test_runs INCLUDING ALL);
INSERT INTO test_runs VALUES
  (1, 'Login', '2026-10-02', true), (2, 'Login', '2026-11-12', false),
  (3, 'Export', '2026-10-20', true), (4, 'Export', '2026-11-13', true);

-- Move October's runs to the archive in one statement: a run can't be
-- deleted without being archived, or archived twice.
WITH moved AS (
  DELETE FROM test_runs
  WHERE ran_on < date '2026-11-01'
  RETURNING *
)
INSERT INTO test_runs_archive
SELECT * FROM moved;

SELECT 'live' AS place, id FROM test_runs
UNION ALL
SELECT 'archive', id FROM test_runs_archive
ORDER BY place, id;

-- A parent and its children in one round trip: generated ids flow from
-- the first INSERT's RETURNING into the second.
CREATE TABLE requirements (
  id   int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  code text NOT NULL UNIQUE
);
CREATE TABLE test_cases (
  id             int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  requirement_id int NOT NULL REFERENCES requirements,
  title          text NOT NULL
);
WITH req AS (
  INSERT INTO requirements (code) VALUES ('REQ-EXPORT') RETURNING id
), tests AS (
  INSERT INTO test_cases (requirement_id, title)
  SELECT req.id, t.title
  FROM req, (VALUES ('CSV header row'), ('UTF-8 with BOM for Excel')) AS t(title)
  RETURNING requirement_id, id, title
)
SELECT * FROM tests ORDER BY id;
