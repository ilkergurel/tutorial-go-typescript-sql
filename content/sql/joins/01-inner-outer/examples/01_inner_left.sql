-- LEARN(sql.joins.inner-outer): decide which side may be missing, and where filters go.
CREATE TABLE requirements (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE test_cases (
  id             int PRIMARY KEY,
  requirement_id int REFERENCES requirements,  -- NULL: not linked yet
  title          text NOT NULL,
  automated      boolean NOT NULL
);
INSERT INTO requirements VALUES (1, 'Login'), (2, 'Password reset'), (3, 'Export CSV');
INSERT INTO test_cases VALUES
  (10, 1,    'Valid password',     true),
  (11, 1,    'İ in the user name', false),
  (12, 2,    'Reset link expires', false),
  (13, NULL, 'Smoke test',         true);

-- INNER: matched pairs only. Export CSV and the smoke test disappear.
SELECT r.title AS requirement, t.title AS test
FROM requirements r
JOIN test_cases t ON t.requirement_id = r.id
ORDER BY r.id, t.id;

-- LEFT: every requirement, with NULLs where nothing matched.
SELECT r.title AS requirement, t.title AS test
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
ORDER BY r.id, t.id;

-- Wrong: a WHERE on the right side drops the NULL rows LEFT JOIN just added.
SELECT r.title AS requirement, t.title AS automated_test
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
WHERE t.automated
ORDER BY r.id, t.id;

-- Right: the filter is part of "what counts as a match", so it goes in ON.
SELECT r.title AS requirement, t.title AS automated_test
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id AND t.automated
ORDER BY r.id, t.id;
