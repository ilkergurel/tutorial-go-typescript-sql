CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL);
CREATE TABLE test_cases   (id int PRIMARY KEY, code text NOT NULL);
INSERT INTO requirements VALUES (1, 'REQ-1');
INSERT INTO test_cases   VALUES (1, 'TC-1'), (2, 'TC-2');

-- Polymorphic association: target_type says which table target_id
-- points into. No foreign key can express "one of these tables".
CREATE TABLE comments_poly (
  id          int  PRIMARY KEY,
  target_type text NOT NULL,         -- 'requirement' or 'test_case'
  target_id   int  NOT NULL,
  body        text NOT NULL
);
INSERT INTO comments_poly VALUES
  (1, 'requirement', 1, 'Needs a 3-D Secure case'),
  (2, 'test_case',   2, 'Flaky on Android'),
  (3, 'test_case',   9, 'Points at nothing');
DELETE FROM test_cases WHERE id = 2;   -- comment 2 is now an orphan

-- Every read joins every possible target and sorts it out per row.
SELECT p.id, p.target_type, p.target_id, coalesce(r.code, t.code) AS target
FROM comments_poly p
LEFT JOIN requirements r ON p.target_type = 'requirement' AND r.id = p.target_id
LEFT JOIN test_cases t   ON p.target_type = 'test_case'   AND t.id = p.target_id
ORDER BY p.id;

-- Exclusive arc: one real foreign key per target, exactly one of them set.
CREATE TABLE comments (
  id             int  PRIMARY KEY,
  requirement_id int  REFERENCES requirements ON DELETE CASCADE,
  test_case_id   int  REFERENCES test_cases   ON DELETE CASCADE,
  body           text NOT NULL,
  CONSTRAINT comments_one_target
    CHECK (num_nonnulls(requirement_id, test_case_id) = 1)
);
CREATE INDEX ON comments (requirement_id);
CREATE INDEX ON comments (test_case_id);
INSERT INTO comments (id, requirement_id, body) VALUES (1, 1, 'Needs a 3-D Secure case');
INSERT INTO comments (id, test_case_id, body)   VALUES (2, 1, 'Passes on iOS 18');
DELETE FROM test_cases WHERE id = 1;   -- its comment goes with it
SELECT id, requirement_id, test_case_id, body FROM comments ORDER BY id;

INSERT INTO comments (id, test_case_id, body) VALUES (3, 9, 'Points at nothing');
