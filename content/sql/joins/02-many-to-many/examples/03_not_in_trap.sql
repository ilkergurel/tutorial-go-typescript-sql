CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL);
-- Ad-hoc runs belong to no test case: test_case_id is NULL for them.
CREATE TABLE test_runs (
  id           int PRIMARY KEY,
  test_case_id int REFERENCES test_cases,
  passed       boolean NOT NULL
);
INSERT INTO test_cases VALUES
  (10, 'Valid password'), (11, 'Locked account'), (12, 'Reset e-mail sent');
INSERT INTO test_runs VALUES (1, 10, true), (2, 10, false), (3, 12, true);

-- Which test cases never ran? Correct, as long as no NULL is around.
SELECT title FROM test_cases
WHERE id NOT IN (SELECT test_case_id FROM test_runs)
ORDER BY id;

INSERT INTO test_runs VALUES (4, NULL, true);  -- someone ran an ad-hoc check

-- The same query now returns nothing.
SELECT title FROM test_cases
WHERE id NOT IN (SELECT test_case_id FROM test_runs)
ORDER BY id;

-- Why: x NOT IN (...) is x <> each value, ANDed. One NULL makes it unknown.
SELECT 11 NOT IN (10, 12) AS without_null, 11 NOT IN (10, 12, NULL) AS with_null;

-- NOT EXISTS asks "is there a matching row?", and a NULL never matches.
SELECT t.title FROM test_cases t
WHERE NOT EXISTS (SELECT 1 FROM test_runs r WHERE r.test_case_id = t.id)
ORDER BY t.id;

-- The same anti-join as LEFT JOIN ... IS NULL.
SELECT t.title FROM test_cases t
LEFT JOIN test_runs r ON r.test_case_id = t.id
WHERE r.id IS NULL
ORDER BY t.id;
