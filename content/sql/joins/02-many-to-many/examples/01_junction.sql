-- LEARN(sql.joins.many-to-many): the junction table is the relationship.
CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL UNIQUE);
CREATE TABLE test_cases   (id int PRIMARY KEY, title text NOT NULL);

-- One row per link. The composite key forbids duplicate links, and deleting
-- either end deletes its links.
CREATE TABLE trace_links (
  requirement_id int REFERENCES requirements ON DELETE CASCADE,
  test_case_id   int REFERENCES test_cases   ON DELETE CASCADE,
  PRIMARY KEY (requirement_id, test_case_id)
);
-- The key's index serves requirement -> tests; test -> requirements needs this.
CREATE INDEX ON trace_links (test_case_id);

INSERT INTO requirements VALUES (1, 'REQ-LOGIN'), (2, 'REQ-RESET'), (3, 'REQ-EXPORT');
INSERT INTO test_cases VALUES
  (10, 'Valid password'), (11, 'Locked account'), (12, 'Reset e-mail sent');
INSERT INTO trace_links VALUES (1, 10), (1, 11), (2, 11), (2, 12);

-- Linking the same pair again is a no-op, not a second row.
INSERT INTO trace_links VALUES (1, 10) ON CONFLICT DO NOTHING;

-- Two joins walk the relationship: requirement -> link -> test case.
SELECT r.code, t.title
FROM requirements r
JOIN trace_links l ON l.requirement_id = r.id
JOIN test_cases t  ON t.id = l.test_case_id
ORDER BY r.code, t.title;

-- Test 11 covers two requirements; deleting it removes both of its links.
DELETE FROM test_cases WHERE id = 11;
SELECT requirement_id, test_case_id FROM trace_links ORDER BY 1, 2;
