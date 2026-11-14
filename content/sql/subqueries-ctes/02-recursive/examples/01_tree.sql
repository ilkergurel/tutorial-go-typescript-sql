-- LEARN(sql.subqueries-ctes.recursive): the step query reruns on the last round's rows until none come back.
CREATE TABLE requirements (
  id        int PRIMARY KEY,
  parent_id int REFERENCES requirements,   -- NULL: a top-level requirement
  code      text NOT NULL
);
INSERT INTO requirements VALUES
  (1, NULL, 'AUTH'), (2, 1, 'AUTH-LOGIN'), (3, 1, 'AUTH-RESET'),
  (4, 2, 'AUTH-LOGIN-2FA'), (5, 2, 'AUTH-LOGIN-LOCKOUT'),
  (6, NULL, 'EXPORT'), (7, 6, 'EXPORT-CSV');

-- Down the tree: AUTH and everything below it.
WITH RECURSIVE subtree AS (
  SELECT id, code, 0 AS depth              -- non-recursive term: the start
  FROM requirements
  WHERE code = 'AUTH'
  UNION ALL
  SELECT c.id, c.code, s.depth + 1         -- recursive term: children of
  FROM requirements c                      -- the rows found last round
  JOIN subtree s ON c.parent_id = s.id
) SEARCH DEPTH FIRST BY code SET ord
SELECT repeat('. ', depth) || code AS outline, depth
FROM subtree
ORDER BY ord;

-- Up the tree: the ancestors of one leaf, nearest first.
WITH RECURSIVE ancestors AS (
  SELECT id, parent_id, code, 0 AS up
  FROM requirements
  WHERE code = 'AUTH-LOGIN-2FA'
  UNION ALL
  SELECT p.id, p.parent_id, p.code, a.up + 1
  FROM requirements p
  JOIN ancestors a ON p.id = a.parent_id
)
SELECT up, code
FROM ancestors
ORDER BY up;
