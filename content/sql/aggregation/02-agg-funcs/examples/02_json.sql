CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL);
CREATE TABLE test_cases (
  id             int PRIMARY KEY,
  requirement_id int NOT NULL REFERENCES requirements,
  title          text NOT NULL,
  status         text NOT NULL
);
INSERT INTO requirements VALUES (1, 'REQ-LOGIN'), (2, 'REQ-EXPORT');
INSERT INTO test_cases VALUES
  (10, 1, 'Valid password', 'passed'), (11, 1, 'Locked account', 'failed');

-- One JSON document per requirement, tests nested. REQ-EXPORT has no tests:
-- the LEFT JOIN gives it one all-NULL row, and that row becomes an object.
SELECT r.code,
       jsonb_agg(jsonb_build_object('id', t.id, 'title', t.title) ORDER BY t.id)
         AS tests
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
GROUP BY r.id
ORDER BY r.code;

-- Fixed: FILTER drops the padding row, coalesce turns "no rows" into [].
SELECT jsonb_build_object(
         'code', r.code,
         'tests', coalesce(
           jsonb_agg(jsonb_build_object('id', t.id, 'title', t.title)
                     ORDER BY t.id) FILTER (WHERE t.id IS NOT NULL),
           '[]'::jsonb)
       ) AS doc
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
GROUP BY r.id
ORDER BY r.code;

-- A JSON object instead of an array: key/value pairs from a grouped subquery.
SELECT jsonb_object_agg(status, n) AS tests_by_status
FROM (SELECT status, count(*) AS n FROM test_cases GROUP BY status) AS s;
