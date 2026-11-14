CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL UNIQUE);
INSERT INTO requirements VALUES (1, 'REQ-1'), (2, 'REQ-2'), (12, 'REQ-12');

-- 1. A comma-separated list of requirement ids in a text column.
CREATE TABLE test_cases_csv (code text PRIMARY KEY, req_ids text NOT NULL);
INSERT INTO test_cases_csv VALUES ('TC-1', '1,12'), ('TC-2', '2'), ('TC-3', '12, 99');

-- "Which tests cover requirement 1?" LIKE matches characters, not ids.
SELECT code FROM test_cases_csv WHERE req_ids LIKE '%1%' ORDER BY code;

-- 2. The same lists in a jsonb document: parsed, so matching is exact...
CREATE TABLE test_cases_doc (code text PRIMARY KEY, doc jsonb NOT NULL);
INSERT INTO test_cases_doc VALUES
  ('TC-1', '{"req_ids": [1, 12]}'),
  ('TC-2', '{"req_ids": [2]}'),
  ('TC-3', '{"req_ids": [12, 99]}');
SELECT code FROM test_cases_doc WHERE doc -> 'req_ids' @> '1' ORDER BY code;

-- ...but no foreign key reaches inside a string or a document. 99 was
-- accepted twice, and finding it means unpacking every row.
SELECT d.code, e.id AS dangling_id
FROM test_cases_doc d, jsonb_array_elements_text(d.doc -> 'req_ids') AS e(id)
WHERE e.id::int NOT IN (SELECT id FROM requirements)
ORDER BY d.code;

-- 3. A junction table: one row per link, a foreign key on each side.
CREATE TABLE test_cases (code text PRIMARY KEY);
CREATE TABLE test_case_requirements (
  test_case      text REFERENCES test_cases,
  requirement_id int  REFERENCES requirements,
  PRIMARY KEY (test_case, requirement_id)
);
CREATE INDEX ON test_case_requirements (requirement_id);
INSERT INTO test_cases VALUES ('TC-1'), ('TC-2'), ('TC-3');
INSERT INTO test_case_requirements VALUES
  ('TC-1', 1), ('TC-1', 12), ('TC-2', 2), ('TC-3', 12);
SELECT test_case FROM test_case_requirements WHERE requirement_id = 1 ORDER BY test_case;

INSERT INTO test_case_requirements VALUES ('TC-3', 99);
