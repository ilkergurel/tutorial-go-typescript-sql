-- The spreadsheet from example 1 is loaded by this example's setup.
-- One table per determinant: each dependency becomes a key.
CREATE TABLE requirements (
  code  text PRIMARY KEY,                -- req_code -> req_title
  title text NOT NULL
);
CREATE TABLE test_cases (
  code     text PRIMARY KEY,             -- test_code -> req_code
  req_code text NOT NULL REFERENCES requirements
);
CREATE TABLE testers (
  name  text PRIMARY KEY,                -- tester -> tester_email
  email text NOT NULL UNIQUE
);
CREATE TABLE test_runs (                 -- only what needs the whole key
  test_code text NOT NULL REFERENCES test_cases,
  run_on    date NOT NULL,
  tester    text NOT NULL REFERENCES testers,
  result    text NOT NULL CHECK (result IN ('passed', 'failed')),
  PRIMARY KEY (test_code, run_on)
);
CREATE TABLE test_case_tags (            -- one value per field
  test_code text REFERENCES test_cases,
  tag       text,
  PRIMARY KEY (test_code, tag)
);

-- Two titles for REQ-1 can't both be loaded: decide. Newest wins here.
INSERT INTO requirements
SELECT DISTINCT ON (req_code) req_code, req_title
FROM sheet
ORDER BY req_code, run_on DESC;
INSERT INTO test_cases SELECT DISTINCT test_code, req_code FROM sheet;
INSERT INTO testers SELECT DISTINCT tester, tester_email FROM sheet;
INSERT INTO test_runs SELECT test_code, run_on, tester, result FROM sheet;
INSERT INTO test_case_tags
SELECT DISTINCT test_code, trim(tag)
FROM sheet, string_to_table(tags, ',') AS tag;

-- Joins rebuild the spreadsheet: the decomposition lost nothing.
SELECT r.test_code, r.run_on, c.req_code, q.title AS req_title, r.tester, r.result
FROM test_runs r
JOIN test_cases c   ON c.code = r.test_code
JOIN requirements q ON q.code = c.req_code
ORDER BY r.run_on, r.test_code;

SELECT test_code FROM test_case_tags WHERE tag = 'payment' ORDER BY test_code;

-- The contradiction from example 1 can no longer be stored.
INSERT INTO requirements VALUES ('REQ-1', 'Pay by card');
