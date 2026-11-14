-- Device labs. Each tester works in one lab; each test case is run in
-- each lab by one tester:  (test_code, lab) -> tester,  tester -> lab.
CREATE TABLE lab_assignments (
  test_code text NOT NULL,
  lab       text NOT NULL,
  tester    text NOT NULL,
  PRIMARY KEY (test_code, lab)
);
INSERT INTO lab_assignments VALUES
  ('TC-1', 'android', 'Ayşe'), ('TC-2', 'android', 'Ayşe'), ('TC-1', 'ios', 'Mehmet');

-- 3NF holds, yet "Ayşe works in android" is stored twice. She moves to
-- iOS and one of her rows is missed.
UPDATE lab_assignments SET lab = 'ios' WHERE tester = 'Ayşe' AND test_code = 'TC-2';
SELECT tester, string_agg(lab, ', ' ORDER BY lab) AS labs
FROM lab_assignments
GROUP BY tester
HAVING count(DISTINCT lab) > 1;

-- BCNF: tester -> lab moves to a table keyed by its determinant.
CREATE TABLE testers (
  name text PRIMARY KEY,
  lab  text NOT NULL,
  UNIQUE (name, lab)                     -- a target for the pair below
);
CREATE TABLE assignments (
  test_code text NOT NULL,
  tester    text NOT NULL,
  lab       text NOT NULL,               -- a copy the FK keeps honest
  PRIMARY KEY (test_code, lab),          -- still one tester per test per lab
  FOREIGN KEY (tester, lab) REFERENCES testers (name, lab) ON UPDATE CASCADE
);
INSERT INTO testers VALUES ('Ayşe', 'android'), ('Mehmet', 'ios'), ('Zeynep', 'android');
INSERT INTO assignments VALUES
  ('TC-1', 'Ayşe', 'android'), ('TC-2', 'Ayşe', 'android'), ('TC-1', 'Mehmet', 'ios');

-- Moving Ayşe is one UPDATE, and the cascade rewrites every copy.
UPDATE testers SET lab = 'web' WHERE name = 'Ayşe';
SELECT test_code, tester, lab FROM assignments ORDER BY test_code, lab;

-- A copy that disagrees with its source is refused.
INSERT INTO assignments VALUES ('TC-3', 'Zeynep', 'ios');
