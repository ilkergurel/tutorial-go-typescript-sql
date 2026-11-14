-- LEARN(sql.modeling.temporal): a key that holds at every point in time, not just now.
-- = on an int column inside a GiST index needs btree_gist.
CREATE EXTENSION btree_gist;

-- Who owns which requirement, and when. valid_at is the business period.
CREATE TABLE requirement_owners (
  requirement_id int,
  owner          text NOT NULL,
  valid_at       daterange,
  PRIMARY KEY (requirement_id, valid_at WITHOUT OVERLAPS)   -- PG 18
);
INSERT INTO requirement_owners VALUES
  (1, 'Ayşe',   '[2026-01-01,2026-06-01)'),
  (1, 'Mehmet', '[2026-06-01,)'),            -- no upper bound: until further notice
  (2, 'Zeynep', '[2026-03-01,)');

-- As of a date: who owned what on 15 March?
SELECT requirement_id, owner
FROM requirement_owners
WHERE valid_at @> date '2026-03-15'
ORDER BY requirement_id;

-- Can takes over requirement 2 on 1 October: close the open row, then
-- open a new one. In the other order the INSERT would overlap.
UPDATE requirement_owners
SET valid_at = daterange(lower(valid_at), '2026-10-01')
WHERE requirement_id = 2 AND valid_at @> date '2026-10-01';
INSERT INTO requirement_owners VALUES (2, 'Can', '[2026-10-01,)');

SELECT requirement_id, owner, valid_at
FROM requirement_owners
ORDER BY requirement_id, lower(valid_at);

-- The key is a GiST index, like an exclusion constraint's.
SELECT c.relname AS index_name, am.amname AS method
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
JOIN pg_am am   ON am.oid = c.relam
WHERE i.indrelid = 'requirement_owners'::regclass;

-- Two owners of requirement 1 during May: the periods overlap.
INSERT INTO requirement_owners VALUES (1, 'Selin', '[2026-05-01,2026-07-01)');
