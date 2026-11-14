-- LEARN(sql.server-side.views): a view stores a query, never rows.
CREATE TABLE requirements (
  id    int  PRIMARY KEY,
  code  text NOT NULL UNIQUE,
  title text NOT NULL
);
CREATE TABLE test_cases (
  id             int  PRIMARY KEY,
  requirement_id int  NOT NULL REFERENCES requirements,
  status         text NOT NULL  -- result of the latest run
);
INSERT INTO requirements VALUES
  (1, 'REQ-LOGIN', 'Kullanıcı girişi'),
  (2, 'REQ-EXPORT', 'CSV dışa aktarım'),
  (3, 'REQ-AUDIT', 'Denetim izi');
INSERT INTO test_cases VALUES (10, 1, 'passed'), (11, 1, 'failed'), (12, 2, 'passed');

-- One definition of "coverage" for every report, endpoint and psql session.
CREATE VIEW requirement_coverage AS
SELECT r.code,
       count(t.id) AS tests,
       count(t.id) FILTER (WHERE t.status = 'passed') AS passed
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
GROUP BY r.id, r.code;

SELECT * FROM requirement_coverage WHERE passed < tests OR tests = 0 ORDER BY code;

-- The planner expands the view like a macro and pushes the filter down.
EXPLAIN (COSTS OFF) SELECT * FROM requirement_coverage WHERE code = 'REQ-LOGIN';

-- SELECT * is expanded once, when the view is created.
CREATE VIEW requirement_list AS SELECT * FROM requirements;
ALTER TABLE requirements ADD COLUMN priority int NOT NULL DEFAULT 2;
SELECT * FROM requirement_list ORDER BY id;
SELECT trim(regexp_replace(pg_get_viewdef('requirement_list'::regclass), '\s+', ' ', 'g')) AS stored_query;

-- Views pin the columns they use.
ALTER TABLE test_cases ALTER COLUMN status TYPE varchar(10);
