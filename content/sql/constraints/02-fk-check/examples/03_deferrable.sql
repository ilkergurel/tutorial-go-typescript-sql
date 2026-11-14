-- Checked at the end of each statement instead of row by row.
CREATE TABLE steps (
  id  int PRIMARY KEY,
  pos int NOT NULL,
  CONSTRAINT steps_pos_key UNIQUE (pos) DEFERRABLE INITIALLY IMMEDIATE
);
INSERT INTO steps VALUES (1, 1), (2, 2), (3, 3);
-- Mid-statement, two rows share pos 2. Only the end state is checked.
UPDATE steps SET pos = pos + 1;
SELECT id, pos FROM steps ORDER BY pos;

-- Checked at COMMIT, unless SET CONSTRAINTS asks for it sooner.
CREATE TABLE requirements (id int PRIMARY KEY, title text NOT NULL);
CREATE TABLE test_cases (
  id             int PRIMARY KEY,
  requirement_id int NOT NULL,
  CONSTRAINT test_cases_requirement_fk FOREIGN KEY (requirement_id)
    REFERENCES requirements DEFERRABLE INITIALLY DEFERRED
);

INSERT INTO test_cases VALUES (10, 1);          -- child first: no parent yet
INSERT INTO requirements VALUES (1, 'Login');   -- the parent arrives
SET CONSTRAINTS test_cases_requirement_fk IMMEDIATE;  -- check now: consistent

SET CONSTRAINTS test_cases_requirement_fk DEFERRED;
INSERT INTO test_cases VALUES (11, 2);          -- orphan, accepted for now
SET CONSTRAINTS test_cases_requirement_fk IMMEDIATE;
