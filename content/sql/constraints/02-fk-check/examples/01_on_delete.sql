-- LEARN(sql.constraints.fk-check): the database refuses orphans, whoever writes.
CREATE TABLE projects (id int PRIMARY KEY, name text NOT NULL);
CREATE TABLE users    (id int PRIMARY KEY, name text NOT NULL);

CREATE TABLE requirements (
  id         int PRIMARY KEY,
  -- CASCADE: a requirement is part of its project and goes with it.
  project_id int NOT NULL REFERENCES projects ON DELETE CASCADE,
  -- SET NULL: the owner may leave; the requirement stays, unowned.
  owner_id   int REFERENCES users ON DELETE SET NULL,
  title      text NOT NULL
);
CREATE TABLE test_runs (
  id             int PRIMARY KEY,
  -- RESTRICT: test evidence never disappears as a side effect.
  requirement_id int NOT NULL REFERENCES requirements ON DELETE RESTRICT,
  passed         boolean NOT NULL
);
-- The referenced side has its key index; the referencing side gets nothing.
CREATE INDEX ON requirements (project_id);
CREATE INDEX ON requirements (owner_id);
CREATE INDEX ON test_runs (requirement_id);

INSERT INTO projects VALUES (1, 'Checkout'), (2, 'Legacy import');
INSERT INTO users VALUES (10, 'Ayşe'), (11, 'Mehmet');
INSERT INTO requirements VALUES
  (100, 1, 10, 'Pay by card'),
  (101, 1, 11, 'Pay by bank transfer'),
  (200, 2, 11, 'Import CSV');
INSERT INTO test_runs VALUES (1000, 100, true);

DELETE FROM users WHERE id = 11;     -- owner_id becomes NULL on 101 and 200
DELETE FROM projects WHERE id = 2;   -- requirement 200 is deleted with it
SELECT id, project_id, owner_id, title FROM requirements ORDER BY id;

DELETE FROM requirements WHERE id = 100;
