CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL, status text NOT NULL);
INSERT INTO test_cases VALUES
  (1, 'Login', 'failed'), (2, 'Logout', 'passed'), (3, 'Export', 'failed');

-- Every part of the statement reads the same snapshot, taken before any of
-- it ran. The table still shows the old status; RETURNING shows the new one.
WITH reopened AS (
  UPDATE test_cases SET status = 'retest'
  WHERE status = 'failed'
  RETURNING id, status
)
SELECT t.id, t.status AS in_table, r.status AS returned
FROM test_cases t
LEFT JOIN reopened r ON r.id = t.id
ORDER BY t.id;

-- The next statement sees the change.
SELECT id, status FROM test_cases ORDER BY id;

-- Get-or-create a tag. If the INSERT adds a row, only RETURNING can show it;
-- if the tag already exists, the INSERT returns nothing and the SELECT finds it.
CREATE TABLE tags (id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name text UNIQUE NOT NULL);
INSERT INTO tags (name) VALUES ('security');

WITH ins AS (
  INSERT INTO tags (name) VALUES ('kvkk') ON CONFLICT (name) DO NOTHING
  RETURNING id, 'created' AS how
)
SELECT id, how FROM ins
UNION ALL
SELECT id, 'found' FROM tags WHERE name = 'kvkk';

WITH ins AS (
  INSERT INTO tags (name) VALUES ('security') ON CONFLICT (name) DO NOTHING
  RETURNING id, 'created' AS how
)
SELECT id, how FROM ins
UNION ALL
SELECT id, 'found' FROM tags WHERE name = 'security';

-- Data-modifying WITH is allowed only on the top-level statement.
SELECT count(*)
FROM (WITH gone AS (DELETE FROM tags RETURNING id) SELECT id FROM gone) AS s;
