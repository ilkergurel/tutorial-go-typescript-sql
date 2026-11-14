CREATE TABLE test_cases (
  id         int  PRIMARY KEY,
  project_id int  NOT NULL,
  title      text NOT NULL,
  status     text NOT NULL
);
INSERT INTO test_cases VALUES
  (1, 1, 'Giriş', 'failed'), (2, 1, 'Çıkış', 'passed'), (3, 2, 'Fatura', 'failed');

-- One table, no aggregates, DISTINCT or LIMIT: automatically updatable.
CREATE VIEW project1_cases AS
SELECT id, project_id, title, status FROM test_cases WHERE project_id = 1;

-- Writes through the view see only the view's rows.
UPDATE project1_cases SET status = 'retest' WHERE status = 'failed';
SELECT id, project_id, status FROM test_cases ORDER BY id;

-- Without a check option, the view accepts rows it would not show.
INSERT INTO project1_cases VALUES (4, 2, 'Rapor', 'failed');
SELECT count(*) AS visible_through_view FROM project1_cases WHERE id = 4;

-- WITH CHECK OPTION: every written row must satisfy the view's WHERE.
CREATE OR REPLACE VIEW project1_cases AS
SELECT id, project_id, title, status FROM test_cases WHERE project_id = 1
WITH CHECK OPTION;
UPDATE project1_cases SET status = 'passed' WHERE id = 1;
UPDATE project1_cases SET project_id = 2 WHERE id = 2;
