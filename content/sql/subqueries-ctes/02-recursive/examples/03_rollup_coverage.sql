CREATE TABLE requirements (
  id        int PRIMARY KEY,
  parent_id int REFERENCES requirements,
  code      text NOT NULL
);
CREATE TABLE test_cases (id int PRIMARY KEY, last_passed boolean NOT NULL);
CREATE TABLE trace_links (
  requirement_id int REFERENCES requirements,
  test_case_id   int REFERENCES test_cases,
  PRIMARY KEY (requirement_id, test_case_id)
);
INSERT INTO requirements VALUES
  (1, NULL, 'AUTH'), (2, 1, 'AUTH-LOGIN'), (3, 1, 'AUTH-RESET'),
  (4, 2, 'AUTH-LOGIN-2FA'), (6, NULL, 'EXPORT'), (7, 6, 'EXPORT-CSV');
INSERT INTO test_cases VALUES (10, true), (11, false), (12, true);
-- Test 10 verifies two requirements in the same subtree.
INSERT INTO trace_links VALUES (2, 10), (4, 10), (4, 11), (3, 12);

-- Tag every requirement with its top-level ancestor by carrying the root's
-- id down the tree, then aggregate the linked tests per root.
WITH RECURSIVE tree AS (
  SELECT id, id AS root_id FROM requirements WHERE parent_id IS NULL
  UNION ALL
  SELECT c.id, t.root_id
  FROM requirements c
  JOIN tree t ON c.parent_id = t.id
)
SELECT root.code,
       count(l.test_case_id)          AS links,
       count(DISTINCT l.test_case_id) AS tests,
       count(DISTINCT l.test_case_id) FILTER (WHERE NOT tc.last_passed) AS failing
FROM tree
JOIN requirements root  ON root.id = tree.root_id
LEFT JOIN trace_links l ON l.requirement_id = tree.id
LEFT JOIN test_cases tc ON tc.id = l.test_case_id
GROUP BY root.id
ORDER BY root.code;
