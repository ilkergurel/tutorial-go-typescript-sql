CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL);
CREATE TABLE test_cases (
  id             int PRIMARY KEY,
  requirement_id int NOT NULL REFERENCES requirements,
  title          text NOT NULL
);
CREATE TABLE tags (
  requirement_id int NOT NULL REFERENCES requirements,
  tag            text NOT NULL,
  PRIMARY KEY (requirement_id, tag)
);
INSERT INTO requirements VALUES (1, 'REQ-LOGIN'), (2, 'REQ-EXPORT');
INSERT INTO test_cases VALUES (10, 1, 'Valid password'), (11, 1, 'Locked account');
INSERT INTO tags VALUES (1, 'security'), (1, 'ui'), (1, 'kvkk');

-- Two children, two LEFT JOINs: 2 tests x 3 tags = 6 rows for REQ-LOGIN,
-- so every list repeats and every count is multiplied.
SELECT r.code,
       count(t.id)                     AS test_count,
       array_agg(g.tag ORDER BY g.tag) AS tags
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
LEFT JOIN tags g       ON g.requirement_id = r.id
GROUP BY r.id
ORDER BY r.code;

-- Aggregate each child on its own. ARRAY(subquery) runs per requirement and
-- returns an empty array, not NULL, when there is nothing to collect.
SELECT r.code,
       ARRAY(SELECT t.title FROM test_cases t
             WHERE t.requirement_id = r.id ORDER BY t.title) AS tests,
       ARRAY(SELECT g.tag FROM tags g
             WHERE g.requirement_id = r.id ORDER BY g.tag)   AS tags
FROM requirements r
ORDER BY r.code;
