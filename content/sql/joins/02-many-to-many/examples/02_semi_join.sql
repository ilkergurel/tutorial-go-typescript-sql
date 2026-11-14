CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL UNIQUE);
CREATE TABLE test_cases   (id int PRIMARY KEY, last_passed boolean);  -- NULL: never run
CREATE TABLE trace_links (
  requirement_id int REFERENCES requirements,
  test_case_id   int REFERENCES test_cases,
  PRIMARY KEY (requirement_id, test_case_id)
);
INSERT INTO requirements VALUES (1, 'REQ-LOGIN'), (2, 'REQ-RESET'), (3, 'REQ-EXPORT');
INSERT INTO test_cases VALUES (10, true), (11, true), (12, NULL);
INSERT INTO trace_links VALUES (1, 10), (1, 11), (2, 11), (2, 12);

-- Which requirements have at least one test?
-- A join answers "list the pairs", so each requirement repeats per link.
SELECT r.code
FROM requirements r
JOIN trace_links l ON l.requirement_id = r.id
ORDER BY r.code;

-- EXISTS is a semi-join: each row of r at most once, first match is enough.
SELECT r.code
FROM requirements r
WHERE EXISTS (SELECT 1 FROM trace_links l WHERE l.requirement_id = r.id)
ORDER BY r.code;

-- Verified: at least one linked test, and no linked test that hasn't passed.
SELECT r.code
FROM requirements r
WHERE EXISTS (SELECT 1 FROM trace_links l WHERE l.requirement_id = r.id)
  AND NOT EXISTS (
    SELECT 1
    FROM trace_links l
    JOIN test_cases t ON t.id = l.test_case_id
    WHERE l.requirement_id = r.id AND t.last_passed IS NOT TRUE
  )
ORDER BY r.code;
